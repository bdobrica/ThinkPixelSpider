package extractor

import (
	"testing"
)

// sampleArticleHTML is a minimal HTML page with enough readable content.
const sampleArticleHTML = `<!DOCTYPE html>
<html>
<head><title>Test Article</title></head>
<body>
<article>
<h1>Test Article</h1>
<p>This is the first paragraph of the test article with enough words to pass a
basic word count check. We need to make sure that the readability extraction
returns meaningful content from the page.</p>
<p>A second paragraph adds more words to exceed the minimum word count threshold
used in validation. The extractor should capture this as the main content of the
page and strip any surrounding navigation or boilerplate.</p>
<p>Third paragraph continues the article content. It discusses various topics
related to the extraction pipeline, including readability, word counting, and
markdown conversion. This helps ensure a realistic word count.</p>
</article>
</body>
</html>`

// sampleThinHTML is a page with very little readable content.
const sampleThinHTML = `<!DOCTYPE html>
<html>
<head><title>Thin Page</title></head>
<body>
<nav><a href="/">Home</a></nav>
<main><p>Short.</p></main>
</body>
</html>`

func TestExtract_BasicArticle(t *testing.T) {
	result, err := Extract([]byte(sampleArticleHTML), "https://example.com/blog/test")
	if err != nil {
		t.Fatalf("Extract returned error: %v", err)
	}

	if result.Title == "" {
		t.Error("expected non-empty title")
	}
	if result.TextContent == "" {
		t.Error("expected non-empty text content")
	}
	if result.Content == "" {
		t.Error("expected non-empty HTML content")
	}
	if result.WordCount == 0 {
		t.Error("expected non-zero word count")
	}
}

func TestExtract_InvalidURL(t *testing.T) {
	_, err := Extract([]byte("<html></html>"), "://bad-url")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestValidate_PassesAboveThreshold(t *testing.T) {
	r := &Result{WordCount: 300}
	if err := Validate(r, 250); err != nil {
		t.Errorf("expected validation to pass, got: %v", err)
	}
}

func TestValidate_FailsBelowThreshold(t *testing.T) {
	r := &Result{WordCount: 10}
	err := Validate(r, 250)
	if err == nil {
		t.Error("expected validation to fail for low word count")
	}
}

func TestValidate_ExactThreshold(t *testing.T) {
	r := &Result{WordCount: 250}
	if err := Validate(r, 250); err != nil {
		t.Errorf("expected validation to pass at exact threshold, got: %v", err)
	}
}

func TestCountWords(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"hello", 1},
		{"hello world", 2},
		{"  spaced   out  ", 2},
		{"one\ntwo\tthree", 3},
	}
	for _, tt := range tests {
		got := countWords(tt.input)
		if got != tt.want {
			t.Errorf("countWords(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
