package main

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/bdobrica/ThinkPixelSpider/internal/config"
	"github.com/bdobrica/ThinkPixelSpider/internal/crawler"
	"github.com/bdobrica/ThinkPixelSpider/internal/extractor"
	"github.com/bdobrica/ThinkPixelSpider/internal/filters"
	"github.com/bdobrica/ThinkPixelSpider/internal/jobs"
	"github.com/bdobrica/ThinkPixelSpider/internal/markdown"
	"github.com/bdobrica/ThinkPixelSpider/internal/output"
)

func TestIntegration_FullCrawlPipeline(t *testing.T) {
	// --- Set up fake WP site ---
	mux := http.NewServeMux()
	var srvURL string

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html><head><title>Test WordPress Site</title></head>
<body>
<nav>
  <a href="%[1]s/2026/03/first-article">First Article</a>
  <a href="%[1]s/2026/03/second-article">Second Article</a>
  <a href="%[1]s/about">About</a>
  <a href="%[1]s/category/tech/">Tech Category</a>
  <a href="%[1]s/tag/golang/">Golang Tag</a>
  <a href="%[1]s/wp-admin/edit.php">Admin</a>
  <a href="https://external-site.example.com/">External</a>
</nav>
<main><h1>Welcome</h1><p>Homepage content.</p></main>
</body></html>`, srvURL)
	})

	mux.HandleFunc("/2026/03/first-article", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>First Article - Test Site</title></head>
<body><article>
<h1>First Article</h1>
<p>This is the first article with enough content to pass the word count threshold.
It contains multiple paragraphs discussing various topics related to web crawling
and content extraction. The readability library should have no trouble extracting
the main content from this page because it is wrapped in a proper article tag.</p>
<p>The second paragraph adds more words and discusses the importance of testing
web crawlers against realistic page structures. WordPress sites commonly use
article tags, heading elements, and paragraph elements to structure their content.
This makes them excellent candidates for readability-based extraction.</p>
<p>A third paragraph ensures we comfortably exceed any reasonable minimum word
count threshold. We talk about Colly, Go, sitemaps, and markdown conversion.
These are all components of the ThinkPixelSpider architecture that work together
to produce clean, readable output from crawled web pages.</p>
</article></body></html>`)
	})

	mux.HandleFunc("/2026/03/second-article", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Second Article - Test Site</title></head>
<body><article>
<h1>Second Article</h1>
<p>The second article covers a different topic but still has enough content to
be considered a valid extraction target. It discusses the architecture of modern
web spiders and how they handle rate limiting, URL normalization, and duplicate
detection across large crawl jobs spanning thousands of pages.</p>
<p>Web crawlers must be polite to the servers they visit. This means respecting
robots.txt directives, implementing request delays, and limiting the number of
concurrent connections. The ThinkPixelSpider project addresses all of these
concerns through its CrawlConfig settings and Colly LimitRules.</p>
<p>Finally, the output pipeline converts extracted HTML into clean Markdown with
YAML front matter. This makes the content easily searchable and usable by
downstream systems such as language models, search indexes, and documentation
generators that consume structured Markdown files.</p>
</article></body></html>`)
	})

	mux.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>About</title></head>
<body><main><p>Short about page.</p></main></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL

	u, _ := url.Parse(srvURL)
	hostname := u.Hostname() // "127.0.0.1" without port — matches URLFilter.Allow() and URLToRelativePath

	// --- Set up output directory ---
	outDir := t.TempDir()
	manifestPath := filepath.Join(outDir, "manifest.csv")

	manifest, err := output.NewCSVManifestWriter(manifestPath)
	if err != nil {
		t.Fatalf("NewCSVManifestWriter: %v", err)
	}
	defer manifest.Close()

	sink := output.NewFilesystemPageSink(outDir, manifest)
	converter := markdown.NewConverter()

	// --- Build page handler (same as main.go) ---
	var extracted int64
	minWords := 20 // low threshold so articles pass, thin pages don't

	pageHandler := func(ctx context.Context, pageURL string, statusCode int, body []byte) error {
		result, err := extractor.Extract(body, pageURL)
		if err != nil {
			return fmt.Errorf("extraction: %w", err)
		}

		if err := extractor.Validate(result, minWords); err != nil {
			return nil // skip thin pages
		}

		md, err := converter.Convert(result.Content)
		if err != nil {
			return fmt.Errorf("markdown conversion: %w", err)
		}
		md = markdown.Normalize(md)

		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(result.TextContent)))

		fm := &markdown.FrontMatter{
			URL:         pageURL,
			Title:       result.Title,
			Byline:      result.Byline,
			SiteName:    result.SiteName,
			CrawledAt:   time.Now().UTC(),
			WordCount:   result.WordCount,
			ContentHash: hash,
		}

		fullMD := fm.Render() + "\n" + md + "\n"

		page := &jobs.CrawledPage{
			URL:              pageURL,
			Title:            result.Title,
			Byline:           result.Byline,
			SiteName:         result.SiteName,
			Language:         result.Language,
			PublishedTime:    result.PublishedTime,
			MarkdownContent:  fullMD,
			TextContent:      result.TextContent,
			Excerpt:          result.Excerpt,
			WordCount:        result.WordCount,
			ContentHash:      hash,
			HTTPStatus:       statusCode,
			CrawledAt:        time.Now().UTC(),
			ExtractionMethod: "readability",
		}

		if err := sink.SavePage(ctx, page); err != nil {
			return fmt.Errorf("saving page: %w", err)
		}

		atomic.AddInt64(&extracted, 1)
		return nil
	}

	// --- Create Colly collector for httptest (no AllowedDomains) ---
	c := colly.NewCollector(
		colly.MaxDepth(3),
		colly.UserAgent("testbot/1.0"),
		colly.Async(true),
	)

	svc := &crawler.CrawlService{
		Config: config.Config{
			Crawl: config.CrawlConfig{
				MaxPages:              100,
				MaxDepth:              3,
				RequestTimeoutSeconds: 10,
				UserAgent:             "testbot/1.0",
				Parallelism:           2,
				DiscoveryMode:         "links",
				MinWordCount:          20,
			},
		},
		Collector:   c,
		SeedURL:     srvURL + "/",
		URLFilter:   filters.NewURLFilter([]string{hostname}),
		PageHandler: pageHandler,
	}

	// --- Run crawl ---
	result, err := svc.Run(context.Background(), hostname)
	if err != nil {
		t.Fatalf("CrawlService.Run() error: %v", err)
	}

	// Close manifest so CSV is flushed.
	if err := manifest.Close(); err != nil {
		t.Fatalf("manifest.Close(): %v", err)
	}

	// --- Validate crawl result stats ---
	if result.PagesDiscovered < 1 {
		t.Errorf("PagesDiscovered = %d, want >= 1", result.PagesDiscovered)
	}
	if result.PagesVisited < 2 {
		t.Errorf("PagesVisited = %d, want >= 2 (homepage + at least one article)", result.PagesVisited)
	}
	extractedCount := int(atomic.LoadInt64(&extracted))
	if extractedCount < 2 {
		t.Errorf("extracted = %d, want >= 2 (two articles)", extractedCount)
	}

	// --- Validate output files exist ---
	article1Path := filepath.Join(outDir, "pages", hostname, "2026", "03", "first-article.md")
	article2Path := filepath.Join(outDir, "pages", hostname, "2026", "03", "second-article.md")

	assertFileExists(t, article1Path)
	assertFileExists(t, article2Path)

	// Verify article content has YAML front matter.
	content1, err := os.ReadFile(article1Path)
	if err != nil {
		t.Fatalf("read article 1: %v", err)
	}
	assertContains(t, string(content1), "---", "article 1 should have front matter delimiters")
	assertContains(t, string(content1), "url:", "article 1 should have url in front matter")
	assertContains(t, string(content1), "title:", "article 1 should have title in front matter")
	assertContains(t, string(content1), "word_count:", "article 1 should have word_count in front matter")
	assertContains(t, string(content1), "content_hash:", "article 1 should have content_hash in front matter")
	// Verify the body contains some article text.
	assertContains(t, string(content1), "first article", "article 1 should contain article text")

	content2, err := os.ReadFile(article2Path)
	if err != nil {
		t.Fatalf("read article 2: %v", err)
	}
	assertContains(t, string(content2), "---", "article 2 should have front matter delimiters")
	assertContains(t, string(content2), "second article", "article 2 should contain article text")

	// --- Validate filtered URLs were NOT extracted ---
	categoryDir := filepath.Join(outDir, "pages", hostname, "category")
	if _, err := os.Stat(categoryDir); err == nil {
		t.Error("category directory should not exist (URL was filtered)")
	}

	tagDir := filepath.Join(outDir, "pages", hostname, "tag")
	if _, err := os.Stat(tagDir); err == nil {
		t.Error("tag directory should not exist (URL was filtered)")
	}

	wpAdminDir := filepath.Join(outDir, "pages", hostname, "wp-admin")
	if _, err := os.Stat(wpAdminDir); err == nil {
		t.Error("wp-admin directory should not exist (URL was filtered)")
	}

	// --- Validate manifest CSV ---
	f, err := os.Open(manifestPath)
	if err != nil {
		t.Fatalf("open manifest: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("read manifest CSV: %v", err)
	}

	// At least header + 2 article rows.
	if len(records) < 3 {
		t.Fatalf("manifest has %d records, want at least 3 (header + 2 articles)", len(records))
	}

	// Verify header row.
	header := records[0]
	if header[0] != "url" {
		t.Errorf("manifest header[0] = %q, want %q", header[0], "url")
	}
	if header[7] != "status" {
		t.Errorf("manifest header[7] = %q, want %q", header[7], "status")
	}

	// Verify data rows reference our article URLs.
	var foundArticle1, foundArticle2 bool
	for _, row := range records[1:] {
		if strings.Contains(row[0], "first-article") {
			foundArticle1 = true
			if row[7] != "ok" {
				t.Errorf("article 1 status = %q, want %q", row[7], "ok")
			}
		}
		if strings.Contains(row[0], "second-article") {
			foundArticle2 = true
			if row[7] != "ok" {
				t.Errorf("article 2 status = %q, want %q", row[7], "ok")
			}
		}
	}
	if !foundArticle1 {
		t.Error("manifest should contain first-article URL")
	}
	if !foundArticle2 {
		t.Error("manifest should contain second-article URL")
	}

	// Ensure no blacklisted URLs appear in manifest.
	for _, row := range records[1:] {
		pageURL := row[0]
		if strings.Contains(pageURL, "/category/") {
			t.Errorf("manifest should not contain category URL: %s", pageURL)
		}
		if strings.Contains(pageURL, "/tag/") {
			t.Errorf("manifest should not contain tag URL: %s", pageURL)
		}
		if strings.Contains(pageURL, "/wp-admin/") {
			t.Errorf("manifest should not contain wp-admin URL: %s", pageURL)
		}
	}
}

// --- Test helpers ---

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", path)
	}
}

func assertContains(t *testing.T, s, substr, msg string) {
	t.Helper()
	if !strings.Contains(strings.ToLower(s), strings.ToLower(substr)) {
		t.Errorf("%s: %q not found in output (len=%d)", msg, substr, len(s))
	}
}
