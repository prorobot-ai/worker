package server

import (
	"log"
	"net"
	"time"

	pb "github.com/prorobot-ai/grpc-protos/gen/crawler"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Implement the gRPC service
type CrawlerServer struct {
	pb.UnimplementedCrawlerServiceServer
}

// StartCrawl handles incoming gRPC crawl requests (Streaming Response)
func (s *CrawlerServer) StartCrawl(req *pb.CrawlRequest, stream pb.CrawlerService_StartCrawlServer) error {
	log.Printf("🕷 Received Crawl Request for URL: %s", req.Url)

	// Simulate progress
	progressMessages := []string{
		"Job received",
		"Fetching page...",
		"Parsing content...",
		"Finished crawling",
	}

	for _, msg := range progressMessages {
		err := stream.Send(&pb.CrawlResponse{
			Message: msg,
			JobId:   req.JobId,
		})
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Second) // Simulate processing time
	}
	return nil
}

// GetJobStatus fetches the status of a crawling job (Streaming Response)
func (s *CrawlerServer) GetJobStatus(req *pb.JobStatusRequest, stream pb.CrawlerService_GetJobStatusServer) error {
	log.Printf("📡 Fetching job status for: %s", req.JobId)

	// Simulate status updates
	statusUpdates := []string{"Queued", "Crawling", "Processing", "Completed"}
	for _, status := range statusUpdates {
		err := stream.Send(&pb.JobStatusResponse{
			JobId:  req.JobId,
			Status: status,
		})
		if err != nil {
			return err
		}
		time.Sleep(2 * time.Second) // Simulate processing time
	}
	return nil
}

// Function to start the gRPC server
func StartGRPCServer() {
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("❌ Failed to listen: %v", err)
	}

	server := grpc.NewServer()
	pb.RegisterCrawlerServiceServer(server, &CrawlerServer{})

	// Register reflection service on gRPC server
	reflection.Register(server)

	log.Println("🚀 gRPC server running on port 50051")
	if err := server.Serve(listener); err != nil {
		log.Fatalf("❌ Failed to serve: %v", err)
	}
}
