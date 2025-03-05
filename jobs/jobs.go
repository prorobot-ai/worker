package jobs

import (
	"sync"
	"time"
	"worker/database"
	"worker/worker"
)

// JobManager manages active workers
var activeWorkers sync.Map // map[uint]*worker.Worker

// HireCrawler starts a new crawling job
func HireCrawler(url string, depth int) (uint64, error) {
	// Create a new job in the database
	job, err := database.CreateJob(1) // Default priority
	if err != nil {
		return 0, err
	}

	config := worker.WorkerConfig{
		MaxLinks:     64,
		RequestDelay: 0 * time.Second,
		CustomHeaders: map[string]string{
			"User-Agent": "ProRobot/1.0",
		},
	}

	newWorker := worker.NewWorker(job.ID, url, config, nil)
	StoreJob(job.ID, newWorker)

	// Run the worker in a goroutine
	go func() {
		newWorker.Start()
		RemoveJob(job.ID)
	}()

	return job.ID, nil
}

// GetJobStatus fetches the status of an active or completed job
func GetJobStatus(jobID uint64) (string, int, int, error) {
	// Check if the job is still active
	if val, exists := activeWorkers.Load(jobID); exists {
		cr := val.(*worker.Worker)
		processed, total := cr.GetStatus()
		return "running", processed, total, nil
	}

	// If not active, fetch from database
	job, err := database.GetJob(jobID)
	if err != nil {
		return "", 0, 0, err
	}

	return job.Status, len(job.Pages), len(job.Pages), nil
}

// ListJobs returns all active and completed jobs
func ListJobs() ([]JobStatus, error) {
	var jobs []JobStatus
	activeJobIDs := make(map[uint64]bool)

	// Track active jobs
	activeWorkers.Range(func(key, value interface{}) bool {
		jobID := key.(uint64)
		cr := value.(*worker.Worker)
		processed, total := cr.GetStatus()

		jobs = append(jobs, JobStatus{
			JobID:     jobID,
			Status:    "in_progress",
			Processed: processed,
			Total:     total,
		})

		activeJobIDs[jobID] = true
		return true
	})

	// Fetch completed jobs
	dbJobs, err := database.GetAllJobs()
	if err != nil {
		return nil, err
	}

	for _, job := range dbJobs {
		if activeJobIDs[job.ID] {
			continue
		}
		jobs = append(jobs, JobStatus{
			JobID:     job.ID,
			Status:    job.Status,
			Processed: len(job.Pages),
			Total:     len(job.Pages),
		})
	}

	return jobs, nil
}

// GetJobResults returns the crawled pages for a completed job
func GetJobResults(jobID uint64) ([]database.Page, error) {
	job, err := database.GetJob(jobID)
	if err != nil {
		return nil, err
	}
	return job.Pages, nil
}

// StoreJob registers a new worker
func StoreJob(jobID uint64, w *worker.Worker) {
	activeWorkers.Store(jobID, w)
}

// GetJob retrieves a running job
func GetJob(jobID uint64) (*worker.Worker, bool) {
	val, exists := activeWorkers.Load(jobID)
	if !exists {
		return nil, false
	}
	return val.(*worker.Worker), true
}

// RemoveJob removes a completed or canceled job
func RemoveJob(jobID uint64) {
	activeWorkers.Delete(jobID)
}

// JobStatus struct for API response
type JobStatus struct {
	JobID     uint64 `json:"job_id"`
	Status    string `json:"status"`
	Processed int    `json:"processed"`
	Total     int    `json:"total"`
}
