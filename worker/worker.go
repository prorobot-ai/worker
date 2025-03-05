package worker

import (
	"log"
	"net/http"
	"net/url"
	"strings"
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

// WorkerStatusCallback defines a function signature for status reporting.
type WorkerStatusCallback func(jobID uint64, message string)

// Cancel stops the worker immediately
func (w *Worker) Cancel() {
	w.mu.Lock()
	w.canceled = true
	w.mu.Unlock()
}

// NewWorker initializes a new worker instance with custom config
func NewWorker(jobID uint64, startURL string, config WorkerConfig, cb WorkerStatusCallback) *Worker {
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		log.Fatalf("Invalid start URL: %v", err)
	}

	return &Worker{
		visited:  make(map[string]bool),
		Config:   config,
		JobID:    jobID,
		StartURL: startURL,
		Results:  make([]WorkerResult, 0),
		StatusCb: cb,
		Host:     parsedURL.Host, // Store the base domain to filter links
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
	Host     string // Base host (e.g., "example.com")
	canceled bool
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
func (w *Worker) crawl(urlStr string) {
	defer w.wg.Done()

	// Normalize URL and filter out external domains
	absoluteURL := w.resolveURL(urlStr)
	if absoluteURL == "" {
		return
	}

	w.mu.Lock()
	if w.counter >= w.Config.MaxLinks || w.visited[absoluteURL] {
		w.mu.Unlock()
		return
	}
	w.visited[absoluteURL] = true
	w.counter++
	w.mu.Unlock()

	// Send progress message
	if w.StatusCb != nil {
		w.StatusCb(w.JobID, "Crawling: "+absoluteURL)
	}

	req, err := http.NewRequest("GET", absoluteURL, nil)
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
	database.AddPage(w.JobID, absoluteURL, title, content, metadata)

	// Store result in WorkerResult
	w.mu.Lock()
	w.Results = append(w.Results, WorkerResult{
		URL:     absoluteURL,
		Title:   title,
		Content: content,
	})
	w.mu.Unlock()

	// Extract and queue internal links
	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			resolvedURL := w.resolveURL(href)
			if resolvedURL != "" {
				w.wg.Add(1)
				go w.crawl(resolvedURL)
			}
		}
	})
}

// **resolveURL ensures that URLs are absolute and belong to the same domain**
func (w *Worker) resolveURL(href string) string {
	parsedBase, err := url.Parse(w.StartURL)
	if err != nil {
		return ""
	}

	parsedURL, err := url.Parse(href)
	if err != nil {
		return ""
	}

	// Convert relative URLs to absolute URLs
	resolvedURL := parsedBase.ResolveReference(parsedURL)

	// Ignore external domains
	if resolvedURL.Host != w.Host {
		return ""
	}

	// Ignore mailto, tel, javascript, and fragment (#) links
	if strings.HasPrefix(resolvedURL.String(), "mailto:") ||
		strings.HasPrefix(resolvedURL.String(), "tel:") ||
		strings.HasPrefix(resolvedURL.String(), "javascript:") ||
		strings.Contains(resolvedURL.String(), "#") {
		return ""
	}

	return resolvedURL.String()
}

// GetStatus returns the current processed count and the maximum number of links.
func (w *Worker) GetStatus() (processed int, total int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.counter, w.Config.MaxLinks
}
