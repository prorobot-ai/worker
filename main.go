package main

import (
	"log"
	"os"
	"sync"

	"worker/database"
	"worker/handlers"
	"worker/server"

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

	// API Status route
	router.GET("/status", handlers.StatusHandler) // ✅ Status handler route

	// Job routes
	jobRoutes := router.Group("/jobs")
	{
		jobRoutes.POST("", handlers.StartWorkerHandler)
		jobRoutes.GET("", handlers.ListJobsHandler)
		jobRoutes.GET(":id/status", handlers.JobStatusHandler)
		jobRoutes.GET(":id/results", handlers.JobResultsHandler)
		jobRoutes.DELETE(":id", handlers.DeleteJobHandler)
	}

	// Run both HTTP and gRPC servers concurrently
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		log.Printf("🌍 HTTP Server running on :%s", port)
		if err := router.Run(":" + port); err != nil {
			log.Fatalf("❌ HTTP Server failed: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		server.StartGRPCServer() // ✅ Use the imported function
	}()

	wg.Wait()
}
