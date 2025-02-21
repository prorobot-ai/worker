package handlers

import (
	"crawler/crawler"
	"crawler/database"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

// JobStatus represents the status of a crawl job.
type JobStatus struct {
	JobID     uint   `json:"job_id"`
	Status    string `json:"status"`
	Processed int    `json:"processed"`
	Total     int    `json:"total"`
}

var activeJobs sync.Map // Tracks active crawling jobs

// StartCrawlHandler starts a new crawling job.
func StartCrawlHandler(c *gin.Context) {
	var request struct {
		URL   string `json:"url"`
		Depth int    `json:"depth"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	job, err := database.CreateJob(1) // Default priority
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	cr := crawler.NewCrawler(job.ID, request.URL, 64)
	activeJobs.Store(job.ID, cr)
	go func() {
		cr.Start()
		activeJobs.Delete(job.ID)
	}()

	c.JSON(http.StatusCreated, gin.H{"job_id": job.ID})
}

// JobStatusHandler returns the status of a specific job.
func JobStatusHandler(c *gin.Context) {
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	if val, exists := activeJobs.Load(uint(jobID)); exists {
		cr := val.(*crawler.Crawler)
		processed, total := cr.GetStatus()
		c.JSON(http.StatusOK, JobStatus{JobID: uint(jobID), Status: "running", Processed: processed, Total: total})
		return
	}

	job, err := database.GetJob(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, JobStatus{JobID: job.ID, Status: job.Status, Processed: len(job.Pages), Total: len(job.Pages)})
}

// JobResultsHandler returns the results of a completed job.
func JobResultsHandler(c *gin.Context) {
	jobID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := database.GetJob(uint(jobID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job.Pages)
}

// ListJobsHandler returns a list of all jobs with their statuses.
func ListJobsHandler(c *gin.Context) {
	var jobs []JobStatus

	activeJobs.Range(func(key, value interface{}) bool {
		cr := value.(*crawler.Crawler)
		processed, total := cr.GetStatus()
		jobs = append(jobs, JobStatus{JobID: key.(uint), Status: "running", Processed: processed, Total: total})
		return true
	})

	dbJobs, err := database.GetAllJobs()
	if err == nil {
		for _, job := range dbJobs {
			jobs = append(jobs, JobStatus{JobID: job.ID, Status: job.Status, Processed: len(job.Pages), Total: len(job.Pages)})
		}
	}

	c.JSON(http.StatusOK, jobs)
}

// StatusHandler responds with the health status of the API
func StatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "online",
		"message": "API is running",
	})
}
