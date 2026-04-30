package output

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bdobrica/ThinkPixelSpider/internal/jobs"
)

func TestFilesystemPageSink_SavePage(t *testing.T) {
	dir := t.TempDir()
	sink := NewFilesystemPageSink(dir, nil)

	page := &jobs.CrawledPage{
		URL:             "https://example.com/blog/test-post",
		Title:           "Test Post",
		MarkdownContent: "# Test Post\n\nHello world.\n",
		CrawledAt:       time.Now(),
	}

	err := sink.SavePage(context.Background(), page)
	if err != nil {
		t.Fatalf("SavePage returned error: %v", err)
	}

	// Verify the file was written.
	expectedRel := filepath.Join("pages", "example.com", "blog", "test-post.md")
	absPath := filepath.Join(dir, expectedRel)
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("expected file at %s: %v", absPath, err)
	}
	if string(data) != page.MarkdownContent {
		t.Errorf("file content = %q, want %q", string(data), page.MarkdownContent)
	}

	// Verify RelativePath was set on the page.
	if page.RelativePath != expectedRel {
		t.Errorf("RelativePath = %q, want %q", page.RelativePath, expectedRel)
	}
}

func TestFilesystemPageSink_WithManifest(t *testing.T) {
	dir := t.TempDir()
	csvPath := filepath.Join(dir, "manifest.csv")
	mw, err := NewCSVManifestWriter(csvPath)
	if err != nil {
		t.Fatalf("NewCSVManifestWriter: %v", err)
	}

	sink := NewFilesystemPageSink(dir, mw)

	page := &jobs.CrawledPage{
		URL:             "https://example.com/article",
		Title:           "Article",
		MarkdownContent: "# Article\n",
		WordCount:       300,
		ContentHash:     "abc123",
		CrawledAt:       time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC),
	}

	if err := sink.SavePage(context.Background(), page); err != nil {
		t.Fatalf("SavePage: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify manifest CSV was written.
	data, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	content := string(data)
	if !contains(content, "https://example.com/article") {
		t.Error("manifest should contain the page URL")
	}
	if !contains(content, "Article") {
		t.Error("manifest should contain the title")
	}
}

func TestFilesystemPageSink_SavePage_QueryDistinctURLsDoNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	sink := NewFilesystemPageSink(dir, nil)

	first := &jobs.CrawledPage{
		URL:             "https://example.com/post?a=1",
		Title:           "First",
		MarkdownContent: "# First\n",
		CrawledAt:       time.Now(),
	}
	second := &jobs.CrawledPage{
		URL:             "https://example.com/post?a=2",
		Title:           "Second",
		MarkdownContent: "# Second\n",
		CrawledAt:       time.Now(),
	}

	if err := sink.SavePage(context.Background(), first); err != nil {
		t.Fatalf("SavePage(first): %v", err)
	}
	if err := sink.SavePage(context.Background(), second); err != nil {
		t.Fatalf("SavePage(second): %v", err)
	}

	if first.RelativePath == second.RelativePath {
		t.Fatalf("query-distinct URLs mapped to same relative path %q", first.RelativePath)
	}

	firstContent, err := os.ReadFile(filepath.Join(dir, first.RelativePath))
	if err != nil {
		t.Fatalf("read first output: %v", err)
	}
	if string(firstContent) != first.MarkdownContent {
		t.Errorf("first file content = %q, want %q", string(firstContent), first.MarkdownContent)
	}

	secondContent, err := os.ReadFile(filepath.Join(dir, second.RelativePath))
	if err != nil {
		t.Fatalf("read second output: %v", err)
	}
	if string(secondContent) != second.MarkdownContent {
		t.Errorf("second file content = %q, want %q", string(secondContent), second.MarkdownContent)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsString(s, substr)
}

func containsString(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
