package crawler

import (
	"log"
	"net/http"
	"sync"
	"time"

	"crawler/database"

	"github.com/PuerkitoBio/goquery"
)

// Crawler struct to manage crawl state
type Crawler struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	visited  map[string]bool
	counter  int
	MaxLinks int
	JobID    uint
	StartURL string
}

// NewCrawler initializes a new crawler instance
func NewCrawler(jobID uint, startURL string, maxLinks int) *Crawler {
	return &Crawler{
		visited:  make(map[string]bool),
		MaxLinks: maxLinks,
		JobID:    jobID,
		StartURL: startURL,
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
	if c.counter >= c.MaxLinks || c.visited[url] {
		c.mu.Unlock()
		return
	}
	c.visited[url] = true
	c.counter++
	log.Printf("Crawling (%d/%d): %s", c.counter, c.MaxLinks, url)
	c.mu.Unlock()

	resp, err := http.Get(url)
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
	metadata := map[string]interface{}{
		"status":    resp.StatusCode,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	database.AddPage(c.JobID, url, title, doc.Text(), metadata)

	doc.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			c.mu.Lock()
			if c.counter >= c.MaxLinks {
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
	return c.counter, c.MaxLinks
}
