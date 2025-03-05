package server

import (
	"context"
	"log"
	"strconv"
	"worker/database"

	pb "github.com/prorobot-ai/grpc-protos/gen/crawler"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewProgressCallback returns a function to update job progress
func NewProgressCallback(s *CrawlerServer, jobID uint64, cancel context.CancelFunc) func(uint64, string) {
	return func(jobID uint64, message string) {
		s.mu.Lock()
		defer s.mu.Unlock()

		stream, exists := s.jobStreams[jobID]
		if !exists {
			log.Printf("⚠️ Stream for jobID %d not found", jobID)
			return
		}

		jobIDStr := strconv.FormatUint(jobID, 10)
		err := stream.Send(&pb.CrawlResponse{JobId: jobIDStr, Message: message})
		if err != nil {
			// Detect client disconnect
			if grpcErr, ok := err.(interface{ GRPCStatus() *status.Status }); ok {
				if grpcErr.GRPCStatus().Code() == codes.Canceled {
					log.Printf("🛑 Client disconnected: Cancelling job %d", jobID)
					cancel()
					delete(s.jobStreams, jobID)
					return
				}
			}
			log.Printf("❌ Error sending progress update: %v", err)
		}

		// ✅ Update database status
		if message == "Job completed" {
			database.UpdateJobStatus(jobID, "completed")
		} else if message == "Job started" {
			database.UpdateJobStatus(jobID, "in_progress")
		}
	}
}
