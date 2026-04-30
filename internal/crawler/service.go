package crawler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/bdobrica/ThinkPixelSpider/internal/config"
	"github.com/bdobrica/ThinkPixelSpider/internal/filters"
)

// PageHandler is called for every successfully fetched HTML page.
// The implementation decides what to do with the response (extract, convert, save).
type PageHandler func(ctx context.Context, url string, statusCode int, body []byte) error

// CrawlService orchestrates a single domain crawl using Colly.
// It is used by both CLI and daemon modes — only the PageHandler differs.
type CrawlService struct {
	Config      config.Config
	URLFilter   *filters.URLFilter
	PageHandler PageHandler

	// Collector, if non-nil, is used instead of creating one via NewCollector.
	// This is useful for testing with custom transport/settings.
	Collector *colly.Collector

	// SeedURL, if set, overrides the default https://<domain>/ seed for link
	// discovery. Useful for testing against httptest servers.
	SeedURL string
}

// CrawlResult holds aggregate stats returned after a crawl completes.
type CrawlResult struct {
	PagesDiscovered int
	PagesVisited    int
	Errors          int
	StartedAt       time.Time
	FinishedAt      time.Time
}

var discoverSitemapURLs = DiscoverSitemapURLs

// Run executes a full crawl for the given domain.
func (s *CrawlService) Run(ctx context.Context, domain string) (*CrawlResult, error) {
	allowedDomains := filters.AllowedDomainsForDomain(domain)
	urlFilter := s.URLFilter
	if urlFilter == nil {
		urlFilter = filters.NewURLFilter(allowedDomains)
	}

	c := s.Collector
	if c == nil {
		c = NewCollector(s.Config.Crawl, allowedDomains)
	}

	result := &CrawlResult{StartedAt: time.Now()}
	var discovered int64
	var visited int64
	var errCount int64
	maxPages := int64(s.Config.Crawl.MaxPages)
	scheduler := newVisitScheduler(&discovered, maxPages)

	// --- Callbacks ---

	// Normalise URLs before every request so Colly's visited-set works on
	// canonical forms.
	c.OnRequest(func(r *colly.Request) {
		normalized, err := filters.NormalizeURL(r.URL.String())
		if err == nil && normalized != r.URL.String() {
			r.URL, _ = r.URL.Parse(normalized)
		}
	})

	// Link discovery: OnHTML("a[href]") — normalise, filter, then visit.
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if link == "" {
			return
		}

		normalized, err := filters.NormalizeURL(link)
		if err != nil {
			return
		}

		if !urlFilter.Allow(normalized) {
			return
		}

		scheduler.Schedule(normalized, e.Request.Visit)
	})

	// Page handler callback.
	c.OnResponse(func(r *colly.Response) {
		ct := r.Headers.Get("Content-Type")
		if ct != "" && !isHTML(ct) {
			return
		}

		atomic.AddInt64(&visited, 1)

		if s.PageHandler != nil {
			if err := s.PageHandler(ctx, r.Request.URL.String(), r.StatusCode, r.Body); err != nil {
				log.Printf("page handler error for %s: %v", r.Request.URL, err)
				atomic.AddInt64(&errCount, 1)
			}
		}
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Printf("crawl error %s: %v", safeResponseURL(r), err)
		atomic.AddInt64(&errCount, 1)
	})

	// --- Discovery ---

	discoveryMode := strings.ToLower(s.Config.Crawl.DiscoveryMode)

	// Sitemap discovery.
	if discoveryMode == "sitemap" || discoveryMode == "both" {
		timeout := time.Duration(s.Config.Crawl.RequestTimeoutSeconds) * time.Second
		sitemapURLs, err := discoverSitemapURLs(domain, s.Config.Crawl.UserAgent, timeout)
		if err != nil {
			log.Printf("sitemap discovery error: %v", err)
		}
		for _, u := range sitemapURLs {
			normalized, err := filters.NormalizeURL(u)
			if err != nil {
				continue
			}
			if !urlFilter.Allow(normalized) {
				continue
			}
			if !scheduler.Schedule(normalized, c.Visit) {
				if atomic.LoadInt64(&discovered) >= maxPages && maxPages > 0 {
					break
				}
				continue
			}
		}
	}

	// Link discovery seed: visit the homepage.
	if discoveryMode == "links" || discoveryMode == "both" {
		homepage := s.SeedURL
		if homepage == "" {
			homepage = fmt.Sprintf("https://%s/", domain)
		}
		if normalized, err := filters.NormalizeURL(homepage); err == nil {
			homepage = normalized
		}
		scheduler.Schedule(homepage, c.Visit)
	}

	c.Wait()

	result.PagesDiscovered = int(atomic.LoadInt64(&discovered))
	result.PagesVisited = int(atomic.LoadInt64(&visited))
	result.Errors = int(atomic.LoadInt64(&errCount))
	result.FinishedAt = time.Now()

	return result, nil
}

// isHTML checks if the Content-Type header indicates an HTML response.
func isHTML(ct string) bool {
	return strings.Contains(strings.ToLower(ct), "text/html") ||
		strings.Contains(strings.ToLower(ct), "application/xhtml+xml")
}

func safeResponseURL(r *colly.Response) string {
	if r == nil || r.Request == nil || r.Request.URL == nil {
		return "<unknown>"
	}

	return r.Request.URL.String()
}

func reservePageSlot(discovered *int64, maxPages int64) bool {
	if maxPages <= 0 {
		atomic.AddInt64(discovered, 1)
		return true
	}

	for {
		current := atomic.LoadInt64(discovered)
		if current >= maxPages {
			return false
		}
		if atomic.CompareAndSwapInt64(discovered, current, current+1) {
			return true
		}
	}
}

func releasePageSlot(discovered *int64, maxPages int64) {
	if maxPages <= 0 {
		atomic.AddInt64(discovered, -1)
		return
	}

	atomic.AddInt64(discovered, -1)
}

type visitScheduler struct {
	discovered *int64
	maxPages   int64

	mu        sync.Mutex
	scheduled map[string]struct{}
}

func newVisitScheduler(discovered *int64, maxPages int64) *visitScheduler {
	return &visitScheduler{
		discovered: discovered,
		maxPages:   maxPages,
		scheduled:  make(map[string]struct{}),
	}
}

func (s *visitScheduler) Schedule(url string, visit func(string) error) bool {
	s.mu.Lock()
	if _, exists := s.scheduled[url]; exists {
		s.mu.Unlock()
		return false
	}
	if !reservePageSlot(s.discovered, s.maxPages) {
		s.mu.Unlock()
		return false
	}
	s.scheduled[url] = struct{}{}
	s.mu.Unlock()

	if err := visit(url); err != nil {
		s.mu.Lock()
		delete(s.scheduled, url)
		s.mu.Unlock()
		releasePageSlot(s.discovered, s.maxPages)
		return false
	}

	return true
}
