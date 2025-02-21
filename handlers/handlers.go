package handlers

import (
	"crawler/crawler"
	"crawler/database"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// JobStatus represents the status of a crawl job.
type JobStatus struct {
	JobID     uint   `json:"job_id"`
	Status    string `json:"status"`
	Processed int    `json:"processed"`
	Total     int    `json:"total"`
}

// 🏃 Active crawlers in memory
var activeCrawlers sync.Map // map[string]*crawler.Crawler

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

	config := crawler.CrawlerConfig{
		MaxLinks:     64,
		RequestDelay: 0 * time.Second, // 360 * time.Second
		CustomHeaders: map[string]string{
			"User-Agent": "ProRobot/1.0",
		},
	}

	newCrawler := crawler.NewCrawler(job.ID, request.URL, config)
	activeCrawlers.Store(job.ID, newCrawler)
	go func() {
		newCrawler.Start()
		activeCrawlers.Delete(job.ID)
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

	if val, exists := activeCrawlers.Load(uint(jobID)); exists {
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

// ListJobsHandler returns a list of all jobs, combining both active and completed jobs.
func ListJobsHandler(c *gin.Context) {
	var jobs []JobStatus // List to hold all job statuses (active + completed)

	// ✅ Step 1: Track active jobs (crawlers)
	activeJobIDs := make(map[uint]bool) // Tracks active job IDs (as strings)

	// Iterate over the active crawlers
	activeCrawlers.Range(func(key, value interface{}) bool {
		jobID := key.(uint)                // Extract the job ID from the key
		cr := value.(*crawler.Crawler)     // Type assertion to get the Crawler instance
		processed, total := cr.GetStatus() // Get current crawling status from the Crawler

		// Add active job details
		jobs = append(jobs, JobStatus{
			JobID:     jobID,
			Status:    "in_progress",
			Processed: processed,
			Total:     total,
		})

		activeJobIDs[jobID] = true // Track active job IDs to avoid duplication
		return true                // Continue iterating through the map
	})

	// ✅ Step 2: Fetch completed jobs from the database
	dbJobs, err := database.GetAllJobs()
	if err == nil {
		for _, job := range dbJobs {
			// 🚫 Skip jobs that are already active
			if activeJobIDs[job.ID] {
				continue
			}

			// Add completed job from the database
			jobs = append(jobs, JobStatus{
				JobID:     job.ID,
				Status:    job.Status,
				Processed: len(job.Pages), // Assuming `Pages` stores crawled results
				Total:     len(job.Pages),
			})
		}
	} else {
		// ❌ Handle database fetch errors
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs from the database"})
		return
	}

	// ✅ Step 3: Return the combined job list as JSON
	c.JSON(http.StatusOK, jobs)
}

// StatusHandler returns the status of the API
func StatusHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "online",
		"message": "API is running",
	})
}
