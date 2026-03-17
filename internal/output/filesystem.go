package output

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bdobrica/ThinkPixelSpider/internal/jobs"
)

// FilesystemPageSink writes crawled pages as Markdown files under a base
// output directory and optionally records each page in a ManifestWriter.
type FilesystemPageSink struct {
	BaseDir  string
	Manifest ManifestWriter
}

// NewFilesystemPageSink creates a sink that writes into baseDir.
// If manifest is non-nil, a manifest row is written for every saved page.
func NewFilesystemPageSink(baseDir string, manifest ManifestWriter) *FilesystemPageSink {
	return &FilesystemPageSink{
		BaseDir:  baseDir,
		Manifest: manifest,
	}
}

// SavePage writes the page's Markdown content to a deterministic file path
// and appends a row to the manifest.
func (s *FilesystemPageSink) SavePage(ctx context.Context, page *jobs.CrawledPage) error {
	rel, err := URLToRelativePath(page.URL, s.BaseDir)
	if err != nil {
		return fmt.Errorf("path generation: %w", err)
	}

	absPath := filepath.Join(s.BaseDir, rel)

	// Ensure the parent directory exists.
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(absPath), err)
	}

	if err := os.WriteFile(absPath, []byte(page.MarkdownContent), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", absPath, err)
	}

	// Store the relative path back on the page for downstream use.
	page.RelativePath = rel

	if s.Manifest != nil {
		row := jobs.ManifestRow{
			URL:          page.URL,
			CanonicalURL: page.CanonicalURL,
			RelativePath: rel,
			Title:        page.Title,
			WordCount:    page.WordCount,
			ContentHash:  page.ContentHash,
			CrawledAt:    page.CrawledAt,
			Status:       "ok",
		}
		if err := s.Manifest.WriteRow(ctx, row); err != nil {
			return fmt.Errorf("manifest write: %w", err)
		}
	}

	return nil
}
