package server

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"
	"worker/database"
	"worker/worker"

	pb "github.com/prorobot-ai/grpc-protos/gen/crawler"
)

// CrawlerServer implements the gRPC service
type CrawlerServer struct {
	pb.UnimplementedCrawlerServiceServer
	mu         sync.Mutex
	jobStreams map[uint64]pb.CrawlerService_StartCrawlServer
}

// NewCrawlerServer initializes a new CrawlerServer instance
func NewCrawlerServer() *CrawlerServer {
	return &CrawlerServer{
		jobStreams: make(map[uint64]pb.CrawlerService_StartCrawlServer),
	}
}

// StartCrawl handles incoming gRPC crawl requests
func (s *CrawlerServer) StartCrawl(req *pb.CrawlRequest, stream pb.CrawlerService_StartCrawlServer) error {
	log.Printf("Received Crawl Request for URL: %s", req.Url)

	// Create Job in Database
	job, err := database.CreateJob(1) // Priority 1 (default)
	if err != nil {
		log.Printf("❌ Failed to create job: %v", err)
		return err
	}
	jobID := job.ID

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Store stream for progress updates
	s.mu.Lock()
	s.jobStreams[jobID] = stream
	s.mu.Unlock()

	// Set up progress callback
	progressCallback := NewProgressCallback(s, jobID, cancel)

	// Configure and start worker
	config := worker.WorkerConfig{MaxLinks: 16}
	newWorker := worker.NewWorker(jobID, req.Url, config, progressCallback)

	// Start worker asynchronously
	done := make(chan struct{})
	go func() {
		newWorker.Start()
		close(done) // Signal job completion
	}()

	// Keep the gRPC stream open while job runs
	return s.manageJobLifecycle(jobID, stream, ctx, done)
}

// manageJobLifecycle keeps the gRPC stream open until the job is done
func (s *CrawlerServer) manageJobLifecycle(jobID uint64, stream pb.CrawlerService_StartCrawlServer, ctx context.Context, done chan struct{}) error {
	for {
		select {
		case <-ctx.Done():
			log.Printf("❌ Job %d cancelled due to client disconnection", jobID)
			return nil // End gRPC safely
		case <-done:
			log.Printf("✅ Job %d completed successfully", jobID)
			return nil // Close gRPC stream
		case <-time.After(5 * time.Second):
			err := stream.Send(&pb.CrawlResponse{
				JobId:   strconv.FormatUint(jobID, 10),
				Message: "Heartbeat: job still running",
			})
			if err != nil {
				log.Printf("❌ Failed to send heartbeat: %v", err)
			}
		}
	}
}
