package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"crawler/crawler"
)

// JobStatus represents the status of a crawl job.
type JobStatus struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	Processed int    `json:"processed"`
	Total     int    `json:"total"`
}

var activeJobs sync.Map // Tracks active crawling jobs

// StartCrawlHandler starts a new crawling job.
func StartCrawlHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		jobID := fmt.Sprintf("%d", time.Now().UnixNano())
		cr := crawler.NewCrawler(req.URL, db)
		cr.JobID = jobID

		activeJobs.Store(jobID, cr)
		go func() {
			cr.Start()
			activeJobs.Delete(jobID)
		}()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"job_id": jobID})
	}
}

// JobStatusHandler returns the status of a specific job.
func JobStatusHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("id")

		// Check if the job is active.
		if val, exists := activeJobs.Load(jobID); exists {
			cr := val.(*crawler.Crawler)
			processed, total := cr.GetStatus()
			json.NewEncoder(w).Encode(JobStatus{
				JobID:     jobID,
				Status:    "running",
				Processed: processed,
				Total:     total,
			})
			return
		}

		// Otherwise, query the database for a completed job.
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM crawl_results WHERE job_id = ?", jobID).Scan(&count)
		if err != nil {
			http.Error(w, "Job not found", http.StatusNotFound)
			return
		}

		if count > 0 {
			json.NewEncoder(w).Encode(JobStatus{
				JobID:     jobID,
				Status:    "completed",
				Processed: count,
				Total:     count,
			})
		} else {
			http.Error(w, "Job not found", http.StatusNotFound)
		}
	}
}

// JobResultsHandler returns the results of a completed job.
func JobResultsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobID := r.PathValue("id")

		// If the job is still active, return a processing status.
		if _, exists := activeJobs.Load(jobID); exists {
			w.WriteHeader(http.StatusAccepted)
			json.NewEncoder(w).Encode(map[string]string{"status": "processing"})
			return
		}

		rows, err := db.Query(`SELECT url, title, content FROM crawl_results WHERE job_id = ? ORDER BY created_at`, jobID)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var results []crawler.CrawlResult
		for rows.Next() {
			var res crawler.CrawlResult
			if err := rows.Scan(&res.URL, &res.Title, &res.Content); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			results = append(results, res)
		}

		if len(results) == 0 {
			http.Error(w, "No results found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

// ListJobsHandler returns a list of all jobs with their statuses.
func ListJobsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var jobs []JobStatus

		// Add active jobs.
		activeJobs.Range(func(key, value interface{}) bool {
			cr := value.(*crawler.Crawler)
			processed, total := cr.GetStatus()
			jobs = append(jobs, JobStatus{
				JobID:     key.(string),
				Status:    "running",
				Processed: processed,
				Total:     total,
			})
			return true
		})

		// Add completed jobs from the database.
		rows, err := db.Query(`SELECT job_id, COUNT(*) as processed FROM crawl_results GROUP BY job_id`)
		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var jobID string
			var processed int
			if err := rows.Scan(&jobID, &processed); err != nil {
				continue
			}
			jobs = append(jobs, JobStatus{
				JobID:     jobID,
				Status:    "completed",
				Processed: processed,
				Total:     processed,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	}
}
