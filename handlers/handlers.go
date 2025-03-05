package handlers

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"time"
	"worker/jobs"

	"github.com/gin-gonic/gin"
	pb "github.com/prorobot-ai/grpc-protos/gen/crawler"
	"google.golang.org/grpc"
)

// StartWorkerHandler starts a new job
func StartWorkerHandler(c *gin.Context) {
	var request struct {
		URL   string `json:"url"`
		Depth int    `json:"depth"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	jobID, err := jobs.HireCrawler(request.URL, request.Depth)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"job_id": jobID})
}

func StartGRPCWorkerHandler(c *gin.Context) {
	var request struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Connect to gRPC server
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client := pb.NewCrawlerServiceClient(conn)

	// Call gRPC StartCrawl
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	jobID := uint64(time.Now().Unix())        // Get the current Unix timestamp as uint64
	jobIDStr := strconv.FormatUint(jobID, 10) // Convert uint64 to string

	resp, err := client.StartCrawl(ctx, &pb.CrawlRequest{Url: request.URL, JobId: jobIDStr})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start crawl job"})
		return
	}

	log.Println(resp)

	c.JSON(http.StatusCreated, gin.H{"job_id": jobIDStr})
}

// JobStatusHandler returns the status of a job
func JobStatusHandler(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	status, processed, total, err := jobs.GetJobStatus(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, jobs.JobStatus{
		JobID:     jobID,
		Status:    status,
		Processed: processed,
		Total:     total,
	})
}

// ListJobsHandler returns all jobs
func ListJobsHandler(c *gin.Context) {
	jobsList, err := jobs.ListJobs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs"})
		return
	}

	c.JSON(http.StatusOK, jobsList)
}

// JobResultsHandler returns the results of a completed job
func JobResultsHandler(c *gin.Context) {
	jobID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	results, err := jobs.GetJobResults(jobID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, results)
}

// StatusHandler returns the status of the API
func StatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "online",
		"message": "API is running",
	})
}
