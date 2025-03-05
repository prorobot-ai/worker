package worker

import (
	"log"
	"net/http"
	"sync"
	"time"

	"worker/database"

	"github.com/PuerkitoBio/goquery"
)

// WorkerResult holds data extracted from a page.
type WorkerResult struct {
	URL     string
	Title   string
	Content string
}

// WorkerConfig allows custom configuration for the worker.
type WorkerConfig struct {
	MaxLinks      int               // Maximum number of links to crawl
	RequestDelay  time.Duration     // Delay between requests
	CustomHeaders map[string]string // Optional HTTP headers for requests
}

type WorkerStatusCallback func(jobID uint64, message string)

// NewWorker initializes a new worker instance with custom config
func NewWorker(jobID uint64, startURL string, config WorkerConfig, cb WorkerStatusCallback) *Worker {
	return &Worker{
		visited:  make(map[string]bool),
		Config:   config,
		JobID:    jobID,
		StartURL: startURL,
		Results:  make([]WorkerResult, 0),
		StatusCb: cb,
	}
}

// Worker struct to manage crawl state
type Worker struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	visited  map[string]bool
	counter  int
	Config   WorkerConfig
	JobID    uint64
	StartURL string
	Results  []WorkerResult
	StatusCb WorkerStatusCallback
}

// Start begins the crawling process
func (w *Worker) Start() {
	log.Printf("Starting crawl job %d for URL: %s", w.JobID, w.StartURL)
	database.UpdateJobStatus(w.JobID, "in_progress")

	if w.StatusCb != nil {
		w.StatusCb(w.JobID, "Job started")
	}

	w.wg.Add(1)
	go w.crawl(w.StartURL)
	w.wg.Wait()

	if w.StatusCb != nil {
		w.StatusCb(w.JobID, "Job completed")
	}

	database.UpdateJobStatus(w.JobID, "completed")
}

// Crawl a single URL and store it in the database
func (w *Worker) crawl(url string) {
	defer w.wg.Done()

	w.mu.Lock()
	if w.counter >= w.Config.MaxLinks || w.visited[url] {
		w.mu.Unlock()
		return
	}
	w.visited[url] = true
	w.counter++
	w.mu.Unlock()

	// Send progress message
	if w.StatusCb != nil {
		w.StatusCb(w.JobID, "Crawling: "+url)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error fetching URL: %v", err)
		return
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return
	}

	title := doc.Find("title").Text()
	content := doc.Find("body").Text()

	metadata := map[string]interface{}{
		"status":    resp.StatusCode,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Store page in database
	database.AddPage(w.JobID, url, title, content, metadata)

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			w.mu.Lock()
			if w.counter >= w.Config.MaxLinks {
				w.mu.Unlock()
				return
			}
			w.mu.Unlock()

			w.wg.Add(1)
			go w.crawl(href)
		}
	})
}

// GetStatus returns the current processed count and the maximum number of links.
func (w *Worker) GetStatus() (processed int, total int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.counter, w.Config.MaxLinks
}
