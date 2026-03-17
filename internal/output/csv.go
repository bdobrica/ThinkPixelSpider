package output

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/bdobrica/ThinkPixelSpider/internal/jobs"
)

// csvHeader is the column order written to the manifest CSV.
var csvHeader = []string{
	"url",
	"canonical_url",
	"relative_path",
	"title",
	"word_count",
	"content_hash",
	"crawled_at",
	"status",
}

// CSVManifestWriter writes manifest rows to a CSV file.
type CSVManifestWriter struct {
	mu     sync.Mutex
	file   *os.File
	writer *csv.Writer
}

// NewCSVManifestWriter creates (or truncates) the file at path and writes
// the CSV header row.
func NewCSVManifestWriter(path string) (*CSVManifestWriter, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir for manifest: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create manifest %s: %w", path, err)
	}

	w := csv.NewWriter(f)
	if err := w.Write(csvHeader); err != nil {
		f.Close()
		return nil, fmt.Errorf("write CSV header: %w", err)
	}

	return &CSVManifestWriter{file: f, writer: w}, nil
}

// WriteRow appends a single manifest row to the CSV.
func (m *CSVManifestWriter) WriteRow(_ context.Context, row jobs.ManifestRow) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	record := []string{
		row.URL,
		row.CanonicalURL,
		row.RelativePath,
		row.Title,
		strconv.Itoa(row.WordCount),
		row.ContentHash,
		row.CrawledAt.UTC().Format(time.RFC3339),
		row.Status,
	}
	return m.writer.Write(record)
}

// Close flushes any buffered data and closes the underlying file.
func (m *CSVManifestWriter) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.writer.Flush()
	if err := m.writer.Error(); err != nil {
		m.file.Close()
		return err
	}
	return m.file.Close()
}
