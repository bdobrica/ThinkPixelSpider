package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/bdobrica/ThinkPixelSpider/internal/config"
	"github.com/bdobrica/ThinkPixelSpider/internal/filters"
)

// testCollector creates a Colly collector suitable for httptest servers
// (no AllowedDomains restriction, async, low depth).
func testCollector(maxDepth int) *colly.Collector {
	return colly.NewCollector(
		colly.MaxDepth(maxDepth),
		colly.UserAgent("testbot/1.0"),
		colly.Async(true),
	)
}

// hostFromURL extracts the hostname (without port) from a URL string.
func hostFromURL(raw string) string {
	u, _ := url.Parse(raw)
	return u.Host // includes port for httptest servers, which is what URLFilter needs
}

func TestCrawlServiceRun_LinkDiscovery(t *testing.T) {
	mux := http.NewServeMux()
	var srvURL string

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body>
			<a href="%s/page-1">Page 1</a>
			<a href="%s/page-2">Page 2</a>
			<a href="https://external.com/nope">External</a>
		</body></html>`, srvURL, srvURL)
	})
	mux.HandleFunc("/page-1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Page 1</h1><p>Content.</p></body></html>`)
	})
	mux.HandleFunc("/page-2", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Page 2</h1><p>Content.</p></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL
	host := hostFromURL(srvURL)

	var mu sync.Mutex
	var visitedURLs []string

	svc := &CrawlService{
		Config: config.Config{
			Crawl: config.CrawlConfig{
				MaxPages:              100,
				MaxDepth:              3,
				RequestTimeoutSeconds: 5,
				UserAgent:             "testbot/1.0",
				DelayMS:               0,
				RandomDelayMS:         0,
				Parallelism:           2,
				DiscoveryMode:         "links",
			},
		},
		Collector: testCollector(3),
		SeedURL:   srvURL + "/",
		URLFilter: filters.NewURLFilter([]string{host}),
		PageHandler: func(_ context.Context, url string, status int, body []byte) error {
			mu.Lock()
			visitedURLs = append(visitedURLs, url)
			mu.Unlock()
			return nil
		},
	}

	result, err := svc.Run(context.Background(), host)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.PagesVisited < 1 {
		t.Errorf("PagesVisited = %d, want at least 1", result.PagesVisited)
	}
	if result.Errors != 0 {
		t.Errorf("Errors = %d, want 0", result.Errors)
	}

	mu.Lock()
	count := len(visitedURLs)
	mu.Unlock()

	if count < 1 {
		t.Errorf("PageHandler called %d times, want at least 1", count)
	}

	if result.PagesDiscovered < 1 {
		t.Errorf("PagesDiscovered = %d, want at least 1", result.PagesDiscovered)
	}
}

func TestCrawlServiceRun_MaxPagesLimit(t *testing.T) {
	mux := http.NewServeMux()
	var srvURL string

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		html := "<html><body>"
		for i := 0; i < 20; i++ {
			html += fmt.Sprintf(`<a href="%s/page-%d">Page %d</a>`, srvURL, i, i)
		}
		html += "</body></html>"
		fmt.Fprint(w, html)
	})
	for i := 0; i < 20; i++ {
		i := i
		mux.HandleFunc(fmt.Sprintf("/page-%d", i), func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, `<html><body><h1>Page %d</h1></body></html>`, i)
		})
	}

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL
	host := hostFromURL(srvURL)

	svc := &CrawlService{
		Config: config.Config{
			Crawl: config.CrawlConfig{
				MaxPages:              5,
				MaxDepth:              2,
				RequestTimeoutSeconds: 5,
				UserAgent:             "testbot/1.0",
				Parallelism:           1,
				DiscoveryMode:         "links",
			},
		},
		Collector: testCollector(2),
		SeedURL:   srvURL + "/",
		URLFilter: filters.NewURLFilter([]string{host}),
	}

	result, err := svc.Run(context.Background(), host)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.PagesDiscovered > 5 {
		t.Errorf("PagesDiscovered = %d, want <= 5", result.PagesDiscovered)
	}
}

func TestCrawlServiceRun_DuplicateLinksDoNotConsumeMaxPages(t *testing.T) {
	mux := http.NewServeMux()
	var srvURL string

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body>
			<a href="%s/duplicate">Duplicate 1</a>
			<a href="%s/duplicate">Duplicate 2</a>
			<a href="%s/duplicate">Duplicate 3</a>
			<a href="%s/unique">Unique</a>
		</body></html>`, srvURL, srvURL, srvURL, srvURL)
	})
	mux.HandleFunc("/duplicate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Duplicate</h1></body></html>`)
	})
	mux.HandleFunc("/unique", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Unique</h1></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL
	hostname := mustHostname(t, srvURL)

	var mu sync.Mutex
	visited := make(map[string]bool)

	svc := &CrawlService{
		Config: config.Config{
			Crawl: config.CrawlConfig{
				MaxPages:              3,
				MaxDepth:              2,
				RequestTimeoutSeconds: 5,
				UserAgent:             "testbot/1.0",
				Parallelism:           1,
				DiscoveryMode:         "links",
			},
		},
		Collector: testCollector(2),
		SeedURL:   srvURL + "/",
		URLFilter: filters.NewURLFilter([]string{hostname}),
		PageHandler: func(_ context.Context, pageURL string, _ int, _ []byte) error {
			mu.Lock()
			visited[pageURL] = true
			mu.Unlock()
			return nil
		},
	}

	result, err := svc.Run(context.Background(), hostname)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.PagesDiscovered != 3 {
		t.Fatalf("PagesDiscovered = %d, want 3 (homepage + duplicate + unique)", result.PagesDiscovered)
	}

	mu.Lock()
	defer mu.Unlock()
	if !visited[srvURL+"/duplicate"] {
		t.Fatal("expected duplicate page to be visited once")
	}
	if !visited[srvURL+"/unique"] {
		t.Fatal("expected unique page to be visited despite duplicate links")
	}
	if len(visited) != 3 {
		t.Fatalf("visited unique URLs = %d, want 3", len(visited))
	}
}

func TestCrawlServiceRun_DuplicateSitemapURLsDoNotConsumeMaxPages(t *testing.T) {
	mux := http.NewServeMux()
	var srvURL string

	mux.HandleFunc("/duplicate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Duplicate</h1></body></html>`)
	})
	mux.HandleFunc("/unique", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><body><h1>Unique</h1></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	srvURL = srv.URL
	hostname := mustHostname(t, srvURL)

	originalDiscover := discoverSitemapURLs
	discoverSitemapURLs = func(domain string, userAgent string, timeout time.Duration) ([]string, error) {
		return []string{
			srvURL + "/duplicate",
			srvURL + "/duplicate",
			srvURL + "/unique",
		}, nil
	}
	defer func() {
		discoverSitemapURLs = originalDiscover
	}()

	var mu sync.Mutex
	visited := make(map[string]bool)

	svc := &CrawlService{
		Config: config.Config{
			Crawl: config.CrawlConfig{
				MaxPages:              2,
				MaxDepth:              1,
				RequestTimeoutSeconds: 5,
				UserAgent:             "testbot/1.0",
				Parallelism:           1,
				DiscoveryMode:         "sitemap",
			},
		},
		Collector: testCollector(1),
		URLFilter: filters.NewURLFilter([]string{hostname}),
		PageHandler: func(_ context.Context, pageURL string, _ int, _ []byte) error {
			mu.Lock()
			visited[pageURL] = true
			mu.Unlock()
			return nil
		},
	}

	result, err := svc.Run(context.Background(), hostname)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if result.PagesDiscovered != 2 {
		t.Fatalf("PagesDiscovered = %d, want 2 (duplicate + unique)", result.PagesDiscovered)
	}

	mu.Lock()
	defer mu.Unlock()
	if !visited[srvURL+"/duplicate"] {
		t.Fatal("expected duplicate sitemap URL to be visited once")
	}
	if !visited[srvURL+"/unique"] {
		t.Fatal("expected unique sitemap URL to be visited despite duplicate entries")
	}
	if len(visited) != 2 {
		t.Fatalf("visited unique URLs = %d, want 2", len(visited))
	}
}

func TestCrawlServiceRun_SkipsNonHTML(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"not": "html"}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()
	host := hostFromURL(srv.URL)

	var handlerCalled bool
	svc := &CrawlService{
		Config: config.Config{
			Crawl: config.CrawlConfig{
				MaxPages:              10,
				MaxDepth:              1,
				RequestTimeoutSeconds: 5,
				UserAgent:             "testbot/1.0",
				Parallelism:           1,
				DiscoveryMode:         "links",
			},
		},
		Collector: testCollector(1),
		SeedURL:   srv.URL + "/",
		URLFilter: filters.NewURLFilter([]string{host}),
		PageHandler: func(_ context.Context, _ string, _ int, _ []byte) error {
			handlerCalled = true
			return nil
		},
	}

	_, err := svc.Run(context.Background(), host)
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}

	if handlerCalled {
		t.Error("PageHandler should not be called for non-HTML responses")
	}
}

func TestSafeResponseURL_NilResponse(t *testing.T) {
	if got := safeResponseURL(nil); got != "<unknown>" {
		t.Errorf("safeResponseURL(nil) = %q, want %q", got, "<unknown>")
	}
}

func TestSafeResponseURL_NilRequest(t *testing.T) {
	r := &colly.Response{}
	if got := safeResponseURL(r); got != "<unknown>" {
		t.Errorf("safeResponseURL(response with nil request) = %q, want %q", got, "<unknown>")
	}
}

func TestSafeResponseURL_WithURL(t *testing.T) {
	u, err := url.Parse("https://example.com/path")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}

	r := &colly.Response{Request: &colly.Request{URL: u}}
	if got := safeResponseURL(r); got != "https://example.com/path" {
		t.Errorf("safeResponseURL(response with URL) = %q, want %q", got, "https://example.com/path")
	}
}

func TestVisitScheduler_DuplicateURLDoesNotConsumeBudgetTwice(t *testing.T) {
	var discovered int64
	scheduler := newVisitScheduler(&discovered, 2)

	var visitCalls int64
	visit := func(string) error {
		atomic.AddInt64(&visitCalls, 1)
		return nil
	}

	if ok := scheduler.Schedule("https://example.com/duplicate", visit); !ok {
		t.Fatal("first Schedule() = false, want true")
	}
	if ok := scheduler.Schedule("https://example.com/duplicate", visit); ok {
		t.Fatal("second Schedule() on duplicate URL = true, want false")
	}
	if ok := scheduler.Schedule("https://example.com/unique", visit); !ok {
		t.Fatal("Schedule() for unique URL after duplicate = false, want true")
	}

	if got := atomic.LoadInt64(&visitCalls); got != 2 {
		t.Fatalf("visit calls = %d, want 2", got)
	}
	if got := atomic.LoadInt64(&discovered); got != 2 {
		t.Fatalf("discovered = %d, want 2", got)
	}
	if ok := scheduler.Schedule("https://example.com/overflow", visit); ok {
		t.Fatal("Schedule() beyond maxPages = true, want false")
	}
}

func TestVisitScheduler_FailedVisitRefundsBudget(t *testing.T) {
	var discovered int64
	scheduler := newVisitScheduler(&discovered, 1)

	failingVisit := func(string) error {
		return fmt.Errorf("visit failed")
	}

	if ok := scheduler.Schedule("https://example.com/fail", failingVisit); ok {
		t.Fatal("Schedule() with failing visit = true, want false")
	}
	if got := atomic.LoadInt64(&discovered); got != 0 {
		t.Fatalf("discovered after failed visit = %d, want 0", got)
	}

	var accepted string
	workingVisit := func(url string) error {
		accepted = url
		return nil
	}

	if ok := scheduler.Schedule("https://example.com/success", workingVisit); !ok {
		t.Fatal("Schedule() after refund = false, want true")
	}
	if accepted != "https://example.com/success" {
		t.Fatalf("accepted URL = %q, want %q", accepted, "https://example.com/success")
	}
	if got := atomic.LoadInt64(&discovered); got != 1 {
		t.Fatalf("discovered after successful retry = %d, want 1", got)
	}
}

func TestReservePageSlot_StrictUnderConcurrency(t *testing.T) {
	const maxPages int64 = 5
	const workers = 100

	var discovered int64
	var reserved int64

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if reservePageSlot(&discovered, maxPages) {
				atomic.AddInt64(&reserved, 1)
			}
		}()
	}
	wg.Wait()

	if reserved != maxPages {
		t.Errorf("reserved slots = %d, want %d", reserved, maxPages)
	}
	if got := atomic.LoadInt64(&discovered); got != maxPages {
		t.Errorf("discovered = %d, want %d", got, maxPages)
	}
}

func mustHostname(t *testing.T, raw string) string {
	t.Helper()

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q): %v", raw, err)
	}

	return u.Hostname()
}
