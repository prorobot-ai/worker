package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

var (
	visited  = make(map[string]bool) // Track visited URLs
	mu       sync.Mutex              // Mutex to protect shared resources
	wg       sync.WaitGroup          // WaitGroup to wait for all goroutines to finish
	counter  int                     // Counter to track the number of links crawled
	maxLinks = 64                    // Maximum number of links to crawl
)

func crawl(url string) {
	defer wg.Done()

	// Check if the maximum number of links has been crawled
	mu.Lock()
	if counter >= maxLinks {
		mu.Unlock()
		return
	}
	mu.Unlock()

	// Check if the URL has already been visited
	mu.Lock()
	if visited[url] {
		mu.Unlock()
		return
	}
	visited[url] = true
	mu.Unlock()

	// Increment the counter
	mu.Lock()
	counter++
	fmt.Printf("Crawling (%d/%d): %s\n", counter, maxLinks, url)
	mu.Unlock()

	// Fetch the URL
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching URL:", err)
		return
	}
	defer resp.Body.Close()

	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		fmt.Println("Error parsing HTML:", err)
		return
	}

	// Extract links and crawl them
	doc.Find("a").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists {
			mu.Lock()
			if counter >= maxLinks {
				mu.Unlock()
				return
			}
			mu.Unlock()

			wg.Add(1)
			go crawl(href) // Recursively crawl links
		}
	})
}

func main() {
	startURL := "https://prorobot.ai/hashtags" // Replace with your target URL
	wg.Add(1)
	go crawl(startURL)
	wg.Wait()
	fmt.Println("Crawling completed.")
}
