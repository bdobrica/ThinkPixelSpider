package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

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

	if result.PagesDiscovered > 10 {
		t.Errorf("PagesDiscovered = %d, expected it to be limited near MaxPages=5", result.PagesDiscovered)
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
