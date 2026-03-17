package markdown

import (
	"fmt"
	"strings"
	"time"
)

// FrontMatter holds the metadata fields written as YAML front matter at the
// top of each Markdown output file.
type FrontMatter struct {
	URL          string
	CanonicalURL string
	Title        string
	Byline       string
	SiteName     string
	CrawledAt    time.Time
	WordCount    int
	ContentHash  string
}

// Render returns the YAML front matter block followed by two newlines.
// Values are quoted to avoid YAML special-character issues.
func (fm *FrontMatter) Render() string {
	var b strings.Builder
	b.WriteString("---\n")
	writeField(&b, "url", fm.URL)
	if fm.CanonicalURL != "" {
		writeField(&b, "canonical_url", fm.CanonicalURL)
	}
	writeField(&b, "title", fm.Title)
	if fm.Byline != "" {
		writeField(&b, "byline", fm.Byline)
	}
	if fm.SiteName != "" {
		writeField(&b, "site_name", fm.SiteName)
	}
	writeField(&b, "crawled_at", fm.CrawledAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&b, "word_count: %d\n", fm.WordCount)
	writeField(&b, "content_hash", fm.ContentHash)
	b.WriteString("---\n")
	return b.String()
}

// writeField writes a single YAML key-value pair with proper quoting.
func writeField(b *strings.Builder, key, value string) {
	// Quote the value to handle colons, special chars, etc.
	fmt.Fprintf(b, "%s: %q\n", key, value)
}
