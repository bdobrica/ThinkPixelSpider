package crawler

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestDiscoverSitemapURLs_URLSet(t *testing.T) {
	sitemapXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://testserver/blog/post-1</loc></url>
  <url><loc>https://testserver/blog/post-2</loc></url>
  <url><loc>https://testserver/blog/post-1</loc></url>
</urlset>`

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sitemap.xml" {
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(sitemapXML))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	// We need to make DiscoverSitemapURLs work with our test server.
	// Since it constructs URLs from domain, we'll test fetchSitemap directly.
	urls, err := fetchSitemap(srv.Client(), srv.URL+"/sitemap.xml", "testbot", map[string]bool{"testserver": true})
	if err != nil {
		t.Fatalf("fetchSitemap error: %v", err)
	}

	if len(urls) != 3 {
		t.Fatalf("got %d URLs before dedup, want 3", len(urls))
	}

	deduped := dedup(urls)
	if len(deduped) != 2 {
		t.Fatalf("got %d URLs after dedup, want 2", len(deduped))
	}
}

func TestDiscoverSitemapURLs_SitemapIndex(t *testing.T) {
	indexXML := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>CHILD_URL</loc></sitemap>
</sitemapindex>`

	childXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://testserver/blog/from-child</loc></url>
</urlset>`

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap_index.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(indexXML))
		case "/child-sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(childXML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Patch the index XML to use the test server URL.
	patchedIndex := strings.Replace(indexXML, "CHILD_URL", srv.URL+"/child-sitemap.xml", 1)
	srv2 := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap_index.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(patchedIndex))
		case "/child-sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(childXML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv2.Close()

	u, err := url.Parse(srv2.URL)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}

	urls, err := fetchSitemap(srv2.Client(), srv2.URL+"/sitemap_index.xml", "testbot", map[string]bool{u.Hostname(): true})
	if err != nil {
		t.Fatalf("fetchSitemap error: %v", err)
	}

	if len(urls) != 1 {
		t.Fatalf("got %d URLs, want 1", len(urls))
	}
	if urls[0] != "https://testserver/blog/from-child" {
		t.Errorf("URL = %q, want %q", urls[0], "https://testserver/blog/from-child")
	}
}

func TestDiscoverSitemapURLsFromBaseURL_HTTPServer(t *testing.T) {
	var srvURL string
	sitemapXMLTemplate := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>SERVER_URL/blog/post-1</loc></url>
  <url><loc>SERVER_URL/blog/post-2</loc></url>
</urlset>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sitemap.xml" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte(strings.ReplaceAll(sitemapXMLTemplate, "SERVER_URL", srvURL)))
	}))
	defer srv.Close()
	srvURL = srv.URL

	urls, err := DiscoverSitemapURLsFromBaseURL(srvURL, "testbot", 5*time.Second)
	if err != nil {
		t.Fatalf("DiscoverSitemapURLsFromBaseURL() error: %v", err)
	}

	if len(urls) != 2 {
		t.Fatalf("got %d URLs, want 2", len(urls))
	}
	if urls[0] != srvURL+"/blog/post-1" {
		t.Errorf("urls[0] = %q, want %q", urls[0], srvURL+"/blog/post-1")
	}
	if urls[1] != srvURL+"/blog/post-2" {
		t.Errorf("urls[1] = %q, want %q", urls[1], srvURL+"/blog/post-2")
	}
}

func TestFetchSitemap_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	host, parseErr := url.Parse(srv.URL)
	if parseErr != nil {
		t.Fatalf("url.Parse: %v", parseErr)
	}
	_, err := fetchSitemap(client, srv.URL+"/sitemap.xml", "testbot", map[string]bool{host.Hostname(): true})
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestFetchSitemap_InvalidXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("this is not xml"))
	}))
	defer srv.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	host, parseErr := url.Parse(srv.URL)
	if parseErr != nil {
		t.Fatalf("url.Parse: %v", parseErr)
	}
	_, err := fetchSitemap(client, srv.URL+"/sitemap.xml", "testbot", map[string]bool{host.Hostname(): true})
	if err == nil {
		t.Fatal("expected error for invalid XML, got nil")
	}
}

func TestFetchSitemap_IgnoresExternalChildSitemaps(t *testing.T) {
	var srvURL string
	indexXMLTemplate := `<?xml version="1.0" encoding="UTF-8"?>
<sitemapindex xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <sitemap><loc>EXTERNAL_CHILD</loc></sitemap>
  <sitemap><loc>INTERNAL_CHILD</loc></sitemap>
</sitemapindex>`

	childXML := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url><loc>https://example.com/from-internal-child</loc></url>
</urlset>`

	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sitemap_index.xml":
			indexXML := strings.Replace(indexXMLTemplate, "EXTERNAL_CHILD", "https://external.example.com/child.xml", 1)
			indexXML = strings.Replace(indexXML, "INTERNAL_CHILD", srvURL+"/child-sitemap.xml", 1)
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(indexXML))
		case "/child-sitemap.xml":
			w.Header().Set("Content-Type", "application/xml")
			_, _ = w.Write([]byte(childXML))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	srvURL = srv.URL

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}

	urls, err := fetchSitemap(srv.Client(), srv.URL+"/sitemap_index.xml", "testbot", map[string]bool{u.Hostname(): true})
	if err != nil {
		t.Fatalf("fetchSitemap error: %v", err)
	}

	if len(urls) != 1 {
		t.Fatalf("got %d URLs, want 1", len(urls))
	}
	if urls[0] != "https://example.com/from-internal-child" {
		t.Errorf("URL = %q, want %q", urls[0], "https://example.com/from-internal-child")
	}
}

func TestIsHTML(t *testing.T) {
	tests := []struct {
		ct   string
		want bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"application/xhtml+xml", true},
		{"application/json", false},
		{"image/png", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ct, func(t *testing.T) {
			if got := isHTML(tt.ct); got != tt.want {
				t.Errorf("isHTML(%q) = %v, want %v", tt.ct, got, tt.want)
			}
		})
	}
}

func TestDedup(t *testing.T) {
	input := []string{"a", "b", "a", "c", "b", "d"}
	got := dedup(input)
	want := []string{"a", "b", "c", "d"}
	if len(got) != len(want) {
		t.Fatalf("dedup len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("dedup[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
