package main

import (
	"log"
	"net/http"

	"crawler/db"
	"crawler/handlers"
)

func main() {
	// Initialize the database.
	database := db.InitDB("./crawl.db")
	defer database.Close()

	// Set up routes. Here we use http.ServeMux.
	mux := http.NewServeMux()
	mux.HandleFunc("/crawl", handlers.StartCrawlHandler(database))
	mux.HandleFunc("/jobs", handlers.ListJobsHandler(database))
	// For job status and results, we expect a query parameter like ?id=JOB_ID.
	mux.HandleFunc("/jobs/{id}/status", handlers.JobStatusHandler(database))
	mux.HandleFunc("/jobs/{id}/results", handlers.JobResultsHandler(database))

	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
