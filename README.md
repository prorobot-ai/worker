# ProRobot Crawler

A concurrent web crawler written in Go (Golang) designed to crawl websites efficiently while respecting basic crawling policies. The crawler stops automatically after crawling a specified number of links (default: 64).

![Go Version](https://img.shields.io/badge/go-%3E%3D1.21-blue)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Features

- **Concurrent crawling**: Uses Go's goroutines for parallel processing of URLs.
- **Kill switch**: Automatically stops after crawling `n` links (configurable).
- **Duplicate URL prevention**: Tracks visited URLs to avoid reprocessing.
- **HTML parsing**: Extracts links using the `goquery` library.
- **Simple CLI**: Easy to use with minimal configuration.

### **Testing Instructions**

Below are step-by-step instructions to test the web crawler application.

---

### **1. Prerequisites**

- **Go**: Ensure Go is installed on your system.
- **Dependencies**: Install the required Go packages.

```bash
go get github.com/PuerkitoBio/goquery
go get github.com/mattn/go-sqlite3
```

---

### **2. Run the Server**

Start the server by running the following command in the terminal:

```bash
go run main.go
```

The server will start on `http://localhost:8080`.

---

### **3. Test the API Endpoints**

#### **Start a New Crawl Job**

Send a POST request to start a new crawling job:

```bash
curl -X POST http://localhost:8080/crawl \
  -H "Content-Type: application/json" \
  -d '{"url":"https://prorobot.ai/hashtags"}'
```

**Response**:
```json
{"job_id": "1623751234567890000"}
```

- Save the `job_id` for further testing.

---

#### **Check Job Status**

Use the `job_id` to check the status of a crawling job:

```bash
curl http://localhost:8080/jobs/{job_id}/status
```

Replace `{job_id}` with the actual job ID.

**Example**:
```bash
curl http://localhost:8080/jobs/1623751234567890000/status
```

**Response**:
```json
{"job_id": "1623751234567890000", "status": "running", "processed": 15, "total": 64}
```

---

#### **List All Jobs**

Retrieve a list of all jobs (both active and completed):

```bash
curl http://localhost:8080/jobs
```

**Response**:
```json
[
  {"job_id": "1623751234567890000", "status": "running", "processed": 15, "total": 64},
  {"job_id": "1623751234567890001", "status": "completed", "processed": 64, "total": 64}
]
```

---

#### **Get Job Results**

Retrieve the results of a completed job:

```bash
curl http://localhost:8080/jobs/{job_id}/results
```

Replace `{job_id}` with the actual job ID.

**Example**:
```bash
curl http://localhost:8080/jobs/1623751234567890000/results
```

**Response**:
```json
[
  {"url": "https://prorobot.ai/hashtags", "title": "Example Page", "content": "Lorem ipsum..."},
  ...
]
```

---

### **4. Expected Behavior**

1. **Starting a Job**:
   - A new job is created, and a `job_id` is returned.
   - The job begins crawling the provided URL.

2. **Checking Job Status**:
   - If the job is running, the status will be `"running"` with the number of processed links.
   - If the job is completed, the status will be `"completed"`.

3. **Listing All Jobs**:
   - Returns a list of all jobs with their `job_id`, `status`, `processed`, and `total` links.

4. **Retrieving Job Results**:
   - If the job is completed, returns the crawled data (URL, title, and content).
   - If the job is still running, returns a `"processing"` status.

---

### **5. Notes**

- **Database**: The SQLite database (`crawl.db`) will be created in the same directory as the application. You can inspect it using SQLite tools:
  ```bash
  sqlite3 crawl.db
  sqlite> .schema crawl_results
  sqlite> SELECT * FROM crawl_results LIMIT 10;
  ```

- **Concurrency**: Multiple jobs can run simultaneously. Each job is tracked independently.

- **Error Handling**: If a job ID is invalid or not found, the API will return a `404 Not Found` error.

---

### **6. Example Workflow**

1. Start a new job:
   ```bash
   curl -X POST http://localhost:8080/crawl -H "Content-Type: application/json" -d '{"url":"https://prorobot.ai/hashtags"}'
   ```

2. Check the job status:
   ```bash
   curl http://localhost:8080/jobs/1623751234567890000/status
   ```

3. List all jobs:
   ```bash
   curl http://localhost:8080/jobs
   ```

4. Retrieve results after the job completes:
   ```bash
   curl http://localhost:8080/jobs/1623751234567890000/results
   ```

---

This testing guide ensures you can verify all functionality of the web crawler application. Let me know if you need further assistance! 🚀