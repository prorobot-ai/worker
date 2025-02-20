package crawler

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

// CrawlResult holds data extracted from a page.
type CrawlResult struct {
	URL     string
	Title   string
	Content string
}

// Crawler represents a single crawling job.
type Crawler struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	visited  map[string]bool
	counter  int
	MaxLinks int
	JobID    string
	StartURL string
	Results  []CrawlResult
	DB       *sql.DB
}

// NewCrawler creates a new Crawler instance.
func NewCrawler(startURL string, db *sql.DB) *Crawler {
	return &Crawler{
		visited:  make(map[string]bool),
		MaxLinks: 64,
		StartURL: startURL,
		Results:  make([]CrawlResult, 0),
		DB:       db,
	}
}

// Start begins the crawling process.
func (c *Crawler) Start() {
	c.wg.Add(1)
	go c.crawl(c.StartURL)
	c.wg.Wait()
	c.saveToDB()
}

// crawl processes a single URL.
func (c *Crawler) crawl(url string) {
	defer c.wg.Done()

	// Check if we've hit our link limit or already visited this URL.
	c.mu.Lock()
	if c.counter >= c.MaxLinks || c.visited[url] {
		c.mu.Unlock()
		return
	}
	c.visited[url] = true
	c.counter++
	fmt.Printf("Crawling (%d/%d): %s\n", c.counter, c.MaxLinks, url)
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
	content := doc.Find("body").Text()

	c.mu.Lock()
	c.Results = append(c.Results, CrawlResult{
		URL:     url,
		Title:   title,
		Content: content,
	})
	c.mu.Unlock()

	// Enqueue all found links.
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

// saveToDB stores the crawling results into the database.
func (c *Crawler) saveToDB() {
	tx, err := c.DB.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		return
	}

	stmt, err := tx.Prepare(`INSERT INTO crawl_results (job_id, url, title, content) VALUES (?, ?, ?, ?)`)
	if err != nil {
		log.Printf("Error preparing statement: %v", err)
		tx.Rollback()
		return
	}
	defer stmt.Close()

	for _, result := range c.Results {
		_, err := stmt.Exec(c.JobID, result.URL, result.Title, result.Content)
		if err != nil {
			log.Printf("Error inserting data: %v", err)
			tx.Rollback()
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
	}
}

// Counter safely returns the current number of processed links.
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
