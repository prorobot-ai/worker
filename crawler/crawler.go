package crawler

import (
	"log"
	"net/http"
	"sync"
	"time"

	"crawler/database"

	"github.com/PuerkitoBio/goquery"
)

// CrawlResult holds data extracted from a page.
type CrawlResult struct {
	URL     string
	Title   string
	Content string
}

// CrawlerConfig allows custom configuration for the crawler.
type CrawlerConfig struct {
	MaxLinks      int               // Maximum number of links to crawl
	RequestDelay  time.Duration     // Delay between requests
	CustomHeaders map[string]string // Optional HTTP headers for requests
}

// Crawler struct to manage crawl state
type Crawler struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	visited  map[string]bool
	counter  int
	Config   CrawlerConfig
	JobID    uint
	StartURL string
	Results  []CrawlResult
}

// NewCrawler initializes a new crawler instance with custom config
func NewCrawler(jobID uint, startURL string, config CrawlerConfig) *Crawler {
	return &Crawler{
		visited:  make(map[string]bool),
		Config:   config,
		JobID:    jobID,
		StartURL: startURL,
		Results:  make([]CrawlResult, 0),
	}
}

// Start begins the crawling process
func (c *Crawler) Start() {
	log.Printf("Starting crawl job %d for URL: %s", c.JobID, c.StartURL)
	database.UpdateJobStatus(c.JobID, "in_progress")

	c.wg.Add(1)
	go c.crawl(c.StartURL)
	c.wg.Wait()

	database.UpdateJobStatus(c.JobID, "completed")
}

// crawl processes a single URL
func (c *Crawler) crawl(url string) {
	defer c.wg.Done()

	c.mu.Lock()
	if c.counter >= c.Config.MaxLinks || c.visited[url] {
		c.mu.Unlock()
		return
	}
	c.visited[url] = true
	c.counter++
	log.Printf("Crawling (%d/%d): %s", c.counter, c.Config.MaxLinks, url)
	c.mu.Unlock()

	// Apply delay if set in configuration
	if c.Config.RequestDelay > 0 {
		time.Sleep(c.Config.RequestDelay)
	}

	// Create HTTP request with custom headers if provided
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating HTTP request: %v", err)
		return
	}
	for key, value := range c.Config.CustomHeaders {
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
	database.AddPage(c.JobID, url, title, doc.Text(), metadata)

	// Save result to the crawler results
	c.mu.Lock()
	c.Results = append(c.Results, CrawlResult{
		URL:     url,
		Title:   title,
		Content: content,
	})
	c.mu.Unlock()

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			c.mu.Lock()
			if c.counter >= c.Config.MaxLinks {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()

			c.wg.Add(1)
			go c.crawl(href)
		}
	})
}

// Counter safely returns the current number of processed links
func (c *Crawler) Counter() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counter
}

// GetStatus returns the current processed count and the maximum number of links.
func (c *Crawler) GetStatus() (processed int, total int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.counter, c.Config.MaxLinks
}
