// Package jobs defines the core domain types for crawl jobs, results, errors,
// and summaries shared across CLI and daemon execution modes.
package jobs

import "time"

// CrawlJob represents an incoming request to crawl a single domain.
type CrawlJob struct {
	JobID          string    `json:"job_id"`
	Domain         string    `json:"domain"`
	AllowedDomains []string  `json:"allowed_domains"`
	MaxPages       int       `json:"max_pages"`
	MaxDepth       int       `json:"max_depth"`
	OutputFormat   string    `json:"output_format"`
	DiscoveryMode  string    `json:"discovery_mode"`
	RequestedAt    time.Time `json:"requested_at"`
}

// CrawledPage holds the extraction result for a single page.
type CrawledPage struct {
	JobID            string     `json:"job_id"`
	URL              string     `json:"url"`
	CanonicalURL     string     `json:"canonical_url"`
	Title            string     `json:"title"`
	Byline           string     `json:"byline,omitempty"`
	SiteName         string     `json:"site_name,omitempty"`
	Language         string     `json:"language,omitempty"`
	PublishedTime    *time.Time `json:"published_time,omitempty"`
	RelativePath     string     `json:"relative_path"`
	MarkdownContent  string     `json:"markdown_content"`
	TextContent      string     `json:"text_content"`
	Excerpt          string     `json:"excerpt,omitempty"`
	WordCount        int        `json:"word_count"`
	ContentHash      string     `json:"content_hash"`
	HTTPStatus       int        `json:"http_status"`
	CrawledAt        time.Time  `json:"crawled_at"`
	ExtractionMethod string     `json:"extraction_method"`
	ContentType      string     `json:"content_type"`
}

// ManifestRow is the per-page metadata written to CSV or SQLite manifests.
type ManifestRow struct {
	URL          string    `json:"url"`
	CanonicalURL string    `json:"canonical_url"`
	RelativePath string    `json:"relative_path"`
	Title        string    `json:"title"`
	WordCount    int       `json:"word_count"`
	ContentHash  string    `json:"content_hash"`
	CrawledAt    time.Time `json:"crawled_at"`
	Status       string    `json:"status"`
}

// CrawlSummary reports aggregate statistics for a completed crawl job.
type CrawlSummary struct {
	JobID           string    `json:"job_id"`
	Domain          string    `json:"domain"`
	PagesDiscovered int       `json:"pages_discovered"`
	PagesVisited    int       `json:"pages_visited"`
	PagesExtracted  int       `json:"pages_extracted"`
	Errors          int       `json:"errors"`
	StartedAt       time.Time `json:"started_at"`
	FinishedAt      time.Time `json:"finished_at"`
}

// CrawlError records a per-page error encountered during crawling or extraction.
type CrawlError struct {
	JobID      string    `json:"job_id"`
	URL        string    `json:"url"`
	HTTPStatus int       `json:"http_status,omitempty"`
	ErrorMsg   string    `json:"error"`
	Phase      string    `json:"phase"` // "fetch", "extract", "convert"
	OccurredAt time.Time `json:"occurred_at"`
}
