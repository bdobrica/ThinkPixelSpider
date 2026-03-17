package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/bdobrica/ThinkPixelSpider/internal/config"
	"github.com/bdobrica/ThinkPixelSpider/internal/crawler"
	"github.com/bdobrica/ThinkPixelSpider/internal/extractor"
	"github.com/bdobrica/ThinkPixelSpider/internal/jobs"
	"github.com/bdobrica/ThinkPixelSpider/internal/markdown"
	"github.com/bdobrica/ThinkPixelSpider/internal/output"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(os.Args[1:])
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.Domain == "" {
		return fmt.Errorf("--domain flag is required")
	}

	// --- Manifest ---
	manifestPath := filepath.Join(cfg.Output.Directory, "manifest.csv")
	manifest, err := output.NewCSVManifestWriter(manifestPath)
	if err != nil {
		return fmt.Errorf("creating manifest: %w", err)
	}
	defer manifest.Close()

	// --- Page sink ---
	pagesDir := filepath.Join(cfg.Output.Directory, "pages")
	sink := output.NewFilesystemPageSink(pagesDir, manifest)

	// --- Markdown converter ---
	converter := markdown.NewConverter()

	// --- Page handler ---
	var extracted int64
	minWords := cfg.Crawl.MinWordCount

	pageHandler := func(ctx context.Context, pageURL string, statusCode int, body []byte) error {
		// Step 1: Extract content with go-readability.
		result, err := extractor.Extract(body, pageURL)
		if err != nil {
			return fmt.Errorf("extraction: %w", err)
		}

		// Step 2: Validate extraction result.
		if err := extractor.Validate(result, minWords); err != nil {
			// Below threshold — skip silently.
			return nil
		}

		// Step 3: Convert HTML to Markdown.
		md, err := converter.Convert(result.Content)
		if err != nil {
			return fmt.Errorf("markdown conversion: %w", err)
		}

		// Step 4: Normalize Markdown.
		md = markdown.Normalize(md)

		// Step 5: Compute content hash.
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(result.TextContent)))

		// Step 6: Build front matter.
		fm := &markdown.FrontMatter{
			URL:         pageURL,
			Title:       result.Title,
			Byline:      result.Byline,
			SiteName:    result.SiteName,
			CrawledAt:   time.Now().UTC(),
			WordCount:   result.WordCount,
			ContentHash: hash,
		}

		// Step 7: Assemble final Markdown with front matter.
		fullMD := fm.Render() + "\n" + md + "\n"

		// Step 8: Build CrawledPage and save via sink.
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

	// --- Crawl service ---
	svc := &crawler.CrawlService{
		Config:      cfg,
		PageHandler: pageHandler,
	}

	log.Printf("Starting crawl of %s (max %d pages, depth %d, discovery %s)",
		cfg.Domain, cfg.Crawl.MaxPages, cfg.Crawl.MaxDepth, cfg.Crawl.DiscoveryMode)

	ctx := context.Background()
	result, err := svc.Run(ctx, cfg.Domain)
	if err != nil {
		return fmt.Errorf("crawl failed: %w", err)
	}

	// --- Summary ---
	elapsed := result.FinishedAt.Sub(result.StartedAt).Round(time.Millisecond)
	pagesExtracted := atomic.LoadInt64(&extracted)

	fmt.Println()
	fmt.Println("=== Crawl Summary ===")
	fmt.Printf("Domain:          %s\n", cfg.Domain)
	fmt.Printf("Pages discovered: %d\n", result.PagesDiscovered)
	fmt.Printf("Pages visited:    %d\n", result.PagesVisited)
	fmt.Printf("Pages extracted:  %d\n", pagesExtracted)
	fmt.Printf("Errors:           %d\n", result.Errors)
	fmt.Printf("Duration:         %s\n", elapsed)
	fmt.Printf("Output:           %s\n", cfg.Output.Directory)
	fmt.Printf("Manifest:         %s\n", manifestPath)

	return nil
}
