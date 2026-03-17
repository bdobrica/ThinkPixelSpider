// Package extractor integrates go-readability to extract main content from
// HTML pages and provides post-extraction validation.
package extractor

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"time"

	readability "github.com/go-shiori/go-readability"
)

// Result holds the extracted content and metadata from a single HTML page.
type Result struct {
	Title         string
	Byline        string
	SiteName      string
	Language      string
	PublishedTime *time.Time
	Content       string // cleaned HTML from readability
	TextContent   string
	Excerpt       string
	WordCount     int
}

// Extract runs go-readability on raw HTML and returns the extracted content.
// pageURL is the URL the HTML was fetched from, used by readability for
// resolving relative links.
func Extract(html []byte, pageURL string) (*Result, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("invalid page URL %q: %w", pageURL, err)
	}

	article, err := readability.FromReader(bytes.NewReader(html), u)
	if err != nil {
		return nil, fmt.Errorf("readability extraction failed: %w", err)
	}

	wordCount := countWords(article.TextContent)

	return &Result{
		Title:         article.Title,
		Byline:        article.Byline,
		SiteName:      article.SiteName,
		Language:      article.Language,
		PublishedTime: article.PublishedTime,
		Content:       article.Content,
		TextContent:   article.TextContent,
		Excerpt:       article.Excerpt,
		WordCount:     wordCount,
	}, nil
}

// Validate checks whether an extraction result meets the minimum quality
// threshold. It returns an error describing why the result was rejected, or
// nil if it passes.
func Validate(r *Result, minWordCount int) error {
	if r.WordCount < minWordCount {
		return fmt.Errorf("word count %d below minimum %d", r.WordCount, minWordCount)
	}
	return nil
}

// countWords counts words by splitting on whitespace.
func countWords(text string) int {
	return len(strings.Fields(text))
}
