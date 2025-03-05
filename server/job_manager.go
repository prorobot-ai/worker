package server

import (
	"sync"

	pb "github.com/prorobot-ai/grpc-protos/gen/crawler"
)

// JobManager handles active job streams
type JobManager struct {
	mu         sync.Mutex
	jobStreams map[uint64]pb.CrawlerService_StartCrawlServer
}

// NewJobManager initializes a new JobManager instance
func NewJobManager() *JobManager {
	return &JobManager{
		jobStreams: make(map[uint64]pb.CrawlerService_StartCrawlServer),
	}
}

// StoreJobStream stores a gRPC stream for a job
func (jm *JobManager) StoreJobStream(jobID uint64, stream pb.CrawlerService_StartCrawlServer) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	jm.jobStreams[jobID] = stream
}

// RemoveJobStream removes a job stream when the job is done
func (jm *JobManager) RemoveJobStream(jobID uint64) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	delete(jm.jobStreams, jobID)
}

// GetJobStream retrieves a stored job stream
func (jm *JobManager) GetJobStream(jobID uint64) (pb.CrawlerService_StartCrawlServer, bool) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	stream, exists := jm.jobStreams[jobID]
	return stream, exists
}
