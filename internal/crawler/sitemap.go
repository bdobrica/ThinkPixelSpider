package crawler

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// sitemapIndex represents a <sitemapindex> root element.
type sitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	Sitemaps []sitemapEntry `xml:"sitemap"`
}

type sitemapEntry struct {
	Loc string `xml:"loc"`
}

// urlSet represents a <urlset> root element.
type urlSet struct {
	XMLName xml.Name   `xml:"urlset"`
	URLs    []urlEntry `xml:"url"`
}

type urlEntry struct {
	Loc string `xml:"loc"`
}

// sitemapPaths is the ordered list of sitemap locations to try.
var sitemapPaths = []string{
	"/sitemap.xml",
	"/sitemap_index.xml",
}

// DiscoverSitemapURLs fetches sitemap(s) for the given domain and returns
// a deduplicated list of page URLs found.
func DiscoverSitemapURLs(domain string, userAgent string, timeout time.Duration) ([]string, error) {
	client := &http.Client{Timeout: timeout}

	var allURLs []string
	for _, path := range sitemapPaths {
		sitemapURL := "https://" + domain + path
		urls, err := fetchSitemap(client, sitemapURL, userAgent)
		if err != nil {
			continue
		}
		allURLs = append(allURLs, urls...)
		if len(allURLs) > 0 {
			break
		}
	}

	return dedup(allURLs), nil
}

// fetchSitemap retrieves a single sitemap URL and returns all page URLs found.
// It handles both <sitemapindex> (recursive) and <urlset> formats.
func fetchSitemap(client *http.Client, sitemapURL string, userAgent string) ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, sitemapURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitemap %s returned status %d", sitemapURL, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10 MB cap
	if err != nil {
		return nil, err
	}

	// Try sitemap index first.
	var idx sitemapIndex
	if err := xml.Unmarshal(body, &idx); err == nil && len(idx.Sitemaps) > 0 {
		var urls []string
		for _, s := range idx.Sitemaps {
			loc := strings.TrimSpace(s.Loc)
			if loc == "" {
				continue
			}
			childURLs, err := fetchSitemap(client, loc, userAgent)
			if err != nil {
				continue
			}
			urls = append(urls, childURLs...)
		}
		return urls, nil
	}

	// Otherwise parse as urlset.
	var us urlSet
	if err := xml.Unmarshal(body, &us); err != nil {
		return nil, fmt.Errorf("failed to parse sitemap XML: %w", err)
	}

	urls := make([]string, 0, len(us.URLs))
	for _, u := range us.URLs {
		loc := strings.TrimSpace(u.Loc)
		if loc != "" {
			urls = append(urls, loc)
		}
	}
	return urls, nil
}

// dedup returns unique strings preserving order.
func dedup(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, s := range items {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
