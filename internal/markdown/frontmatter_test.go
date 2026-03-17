package markdown

import (
	"strings"
	"testing"
	"time"
)

func TestFrontMatter_Render_AllFields(t *testing.T) {
	fm := &FrontMatter{
		URL:          "https://example.com/blog/post",
		CanonicalURL: "https://example.com/blog/post",
		Title:        "Test Post",
		Byline:       "Jane Doe",
		SiteName:     "Example",
		CrawledAt:    time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC),
		WordCount:    812,
		ContentHash:  "abc123",
	}

	got := fm.Render()

	expectations := []string{
		"---",
		`url: "https://example.com/blog/post"`,
		`canonical_url: "https://example.com/blog/post"`,
		`title: "Test Post"`,
		`byline: "Jane Doe"`,
		`site_name: "Example"`,
		`crawled_at: "2026-03-16T10:00:00Z"`,
		"word_count: 812",
		`content_hash: "abc123"`,
	}

	for _, exp := range expectations {
		if !strings.Contains(got, exp) {
			t.Errorf("expected front matter to contain %q\ngot:\n%s", exp, got)
		}
	}
}

func TestFrontMatter_Render_OptionalFieldsOmitted(t *testing.T) {
	fm := &FrontMatter{
		URL:         "https://example.com/page",
		Title:       "Page Title",
		CrawledAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		WordCount:   100,
		ContentHash: "def456",
	}

	got := fm.Render()

	if strings.Contains(got, "canonical_url") {
		t.Error("expected no canonical_url when empty")
	}
	if strings.Contains(got, "byline") {
		t.Error("expected no byline when empty")
	}
	if strings.Contains(got, "site_name") {
		t.Error("expected no site_name when empty")
	}

	// Required fields should still be present.
	if !strings.Contains(got, `url: "https://example.com/page"`) {
		t.Error("expected url field")
	}
	if !strings.Contains(got, `title: "Page Title"`) {
		t.Error("expected title field")
	}
}

func TestFrontMatter_Render_SpecialCharsInTitle(t *testing.T) {
	fm := &FrontMatter{
		URL:         "https://example.com/post",
		Title:       `Title with "quotes" and: colons`,
		CrawledAt:   time.Now(),
		WordCount:   50,
		ContentHash: "xyz",
	}

	got := fm.Render()
	// The title should be properly quoted.
	if !strings.Contains(got, `title: "Title with \"quotes\" and: colons"`) {
		t.Errorf("expected properly quoted title, got:\n%s", got)
	}
}
