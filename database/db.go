package database

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Job represents a scheduled crawling task
type Job struct {
	ID          uint64     `gorm:"primaryKey"`
	Status      string     `gorm:"type:varchar(20);default:'queued'"` // queued, in_progress, completed, failed
	Priority    int        `gorm:"default:1"`                         // 1 = low, 2 = medium, 3 = high
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	StartedAt   *time.Time // Nullable, records when the job starts
	CompletedAt *time.Time // Nullable, records when the job finishes
	Pages       []Page     `gorm:"foreignKey:JobID"` // One-to-Many Relationship
}

// Page represents a crawled webpage
type Page struct {
	ID        uint   `gorm:"primaryKey"`
	JobID     uint64 `gorm:"index"` // Foreign key to jobs
	URL       string `gorm:"unique"`
	Title     string
	Content   string         `gorm:"type:text"`
	Metadata  datatypes.JSON `gorm:"type:jsonb"` // Store structured metadata
	CreatedAt time.Time      `gorm:"autoCreateTime"`
}

// InitDatabase initializes the PostgreSQL database connection from environment variables
func InitDatabase() {
	_ = godotenv.Load() // Load .env file if available
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL environment variable not set")
	}

	var err error
	DB, err = gorm.Open(postgres.Open(databaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto Migrate the schema
	err = DB.AutoMigrate(&Job{}, &Page{})
	if err != nil {
		log.Fatalf("Failed to migrate database schema: %v", err)
	}
}

// CreateJob adds a new job entry
func CreateJob(priority int) (*Job, error) {
	job := &Job{Priority: priority, Status: "queued"}
	if err := DB.Create(job).Error; err != nil {
		return nil, err
	}
	return job, nil
}

// GetJob retrieves a job by ID
func GetJob(jobID uint64) (*Job, error) {
	var job Job
	if err := DB.Preload("Pages").First(&job, jobID).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

// GetAllJobs retrieves all jobs
func GetAllJobs() ([]Job, error) {
	var jobs []Job
	if err := DB.Order("id DESC").Preload("Pages").Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// UpdateJobStatus updates the job's status
func UpdateJobStatus(jobID uint64, status string) error {
	return DB.Model(&Job{}).Where("id = ?", jobID).Update("status", status).Error
}

func AddPage(jobID uint64, url, title, content string, metadata map[string]interface{}) error {
	metadataJSON, err := json.Marshal(metadata) // Convert map to JSON
	if err != nil {
		return err
	}

	page := &Page{
		JobID:    jobID,
		URL:      url,
		Title:    title,
		Content:  content,
		Metadata: datatypes.JSON(metadataJSON), // Store JSON in PostgreSQL
	}
	return DB.Create(page).Error
}

// DeleteJob removes a job and its associated pages from the database.
func DeleteJob(jobID uint64) error {
	// Begin transaction to ensure atomicity
	tx := DB.Begin()

	// Delete associated pages first
	if err := tx.Where("job_id = ?", jobID).Delete(&Page{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Delete the job itself
	if err := tx.Where("id = ?", jobID).Delete(&Job{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Commit transaction
	return tx.Commit().Error
}
