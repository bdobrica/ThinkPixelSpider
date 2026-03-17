package output

import (
	"context"
	"encoding/csv"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bdobrica/ThinkPixelSpider/internal/jobs"
)

func TestCSVManifestWriter_WritesHeaderAndRows(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "manifest.csv")

	mw, err := NewCSVManifestWriter(csvPath)
	if err != nil {
		t.Fatalf("NewCSVManifestWriter: %v", err)
	}

	rows := []jobs.ManifestRow{
		{
			URL:          "https://example.com/post-1",
			CanonicalURL: "https://example.com/post-1",
			RelativePath: "pages/example.com/post-1.md",
			Title:        "Post 1",
			WordCount:    500,
			ContentHash:  "hash1",
			CrawledAt:    time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC),
			Status:       "ok",
		},
		{
			URL:          "https://example.com/post-2",
			RelativePath: "pages/example.com/post-2.md",
			Title:        "Post 2",
			WordCount:    300,
			ContentHash:  "hash2",
			CrawledAt:    time.Date(2026, 3, 16, 11, 0, 0, 0, time.UTC),
			Status:       "ok",
		},
	}

	ctx := context.Background()
	for _, r := range rows {
		if err := mw.WriteRow(ctx, r); err != nil {
			t.Fatalf("WriteRow: %v", err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Re-read and parse the CSV.
	f, err := os.Open(csvPath)
	if err != nil {
		t.Fatalf("open CSV: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read CSV: %v", err)
	}

	// Header + 2 data rows.
	if len(records) != 3 {
		t.Fatalf("expected 3 records (1 header + 2 rows), got %d", len(records))
	}

	// Verify header.
	header := records[0]
	if header[0] != "url" || header[2] != "relative_path" || header[7] != "status" {
		t.Errorf("unexpected header: %v", header)
	}

	// Verify first data row.
	row1 := records[1]
	if row1[0] != "https://example.com/post-1" {
		t.Errorf("row1 url = %q", row1[0])
	}
	if row1[3] != "Post 1" {
		t.Errorf("row1 title = %q", row1[3])
	}
	if row1[4] != "500" {
		t.Errorf("row1 word_count = %q", row1[4])
	}
}

func TestCSVManifestWriter_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "sub", "dir", "manifest.csv")

	mw, err := NewCSVManifestWriter(csvPath)
	if err != nil {
		t.Fatalf("NewCSVManifestWriter: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		t.Error("expected manifest file to be created in nested directory")
	}
}
