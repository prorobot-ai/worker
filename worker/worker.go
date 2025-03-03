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

// Worker struct to manage crawl state
type Worker struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	visited  map[string]bool
	counter  int
	Config   WorkerConfig
	JobID    uint
	StartURL string
	Results  []WorkerResult
}

// NewWorker initializes a new worker instance with custom config
func NewWorker(jobID uint, startURL string, config WorkerConfig) *Worker {
	return &Worker{
		visited:  make(map[string]bool),
		Config:   config,
		JobID:    jobID,
		StartURL: startURL,
		Results:  make([]WorkerResult, 0),
	}
}

// Start begins the crawling process
func (c *Worker) Start() {
	log.Printf("Starting crawl job %d for URL: %s", c.JobID, c.StartURL)
	database.UpdateJobStatus(c.JobID, "in_progress")

	c.wg.Add(1)
	go c.crawl(c.StartURL)
	c.wg.Wait()

	database.UpdateJobStatus(c.JobID, "completed")
}

// worker processes a single URL
func (w *Worker) crawl(url string) {
	defer w.wg.Done()

	w.mu.Lock()
	if w.counter >= w.Config.MaxLinks || w.visited[url] {
		w.mu.Unlock()
		return
	}
	w.visited[url] = true
	w.counter++
	log.Printf("Crawling (%d/%d): %s", w.counter, w.Config.MaxLinks, url)
	w.mu.Unlock()

	// Apply delay if set in configuration
	if w.Config.RequestDelay > 0 {
		time.Sleep(w.Config.RequestDelay)
	}

	// Create HTTP request with custom headers if provided
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return
	}
	for key, value := range w.Config.CustomHeaders {
		req.Header.Set(key, value)
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

	// Save the page to the database
	database.AddPage(w.JobID, url, title, doc.Text(), metadata)

	// Save result to the worker results
	w.mu.Lock()
	w.Results = append(w.Results, WorkerResult{
		URL:     url,
		Title:   title,
		Content: content,
	})
	w.mu.Unlock()

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

// Counter safely returns the current number of processed links
func (w *Worker) Counter() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.counter
}

// GetStatus returns the current processed count and the maximum number of links.
func (w *Worker) GetStatus() (processed int, total int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.counter, w.Config.MaxLinks
}
