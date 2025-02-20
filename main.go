package main

import (
	"log"
	"os"

	"crawler/database"
	"crawler/handlers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	// Initialize database
	database.InitDatabase()

	// Set up Gin router
	router := gin.Default()

	// Job routes
	jobRoutes := router.Group("/jobs")
	{
		jobRoutes.POST("", handlers.StartCrawlHandler)
		jobRoutes.GET("", handlers.ListJobsHandler)
		jobRoutes.GET(":id/status", handlers.JobStatusHandler)
		jobRoutes.GET(":id/results", handlers.JobResultsHandler)
	}

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server starting on :%s", port)
	log.Fatal(router.Run(":" + port))
}
