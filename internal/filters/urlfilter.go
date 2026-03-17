// Package filters implements URL normalization, blacklist/allowlist rules,
// and article-candidate heuristics for crawl URL filtering.
package filters

import (
	"net/url"
	"regexp"
	"sort"
	"strings"
)

// trackingParams is the set of query parameter prefixes/names stripped during
// URL normalization.
var trackingParams = map[string]bool{
	"utm_source":   true,
	"utm_medium":   true,
	"utm_campaign": true,
	"utm_term":     true,
	"utm_content":  true,
	"fbclid":       true,
	"gclid":        true,
}

// NormalizeURL canonicalises a URL string by lowercasing the scheme and host,
// stripping fragments, removing tracking query parameters, sorting remaining
// query parameters, and removing default ports.
func NormalizeURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	// Remove fragment.
	u.Fragment = ""
	u.RawFragment = ""

	// Strip tracking query params and sort the rest.
	if u.RawQuery != "" {
		params := u.Query()
		for key := range params {
			if trackingParams[strings.ToLower(key)] {
				delete(params, key)
			}
		}
		if len(params) == 0 {
			u.RawQuery = ""
		} else {
			u.RawQuery = sortedQueryEncode(params)
		}
	}

	// Remove trailing slash for non-root paths.
	if len(u.Path) > 1 && strings.HasSuffix(u.Path, "/") {
		u.Path = strings.TrimRight(u.Path, "/")
	}

	return u.String(), nil
}

// sortedQueryEncode encodes url.Values with keys in sorted order for deterministic output.
func sortedQueryEncode(v url.Values) string {
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		vals := v[k]
		sort.Strings(vals)
		for j, val := range vals {
			if i > 0 || j > 0 {
				b.WriteByte('&')
			}
			b.WriteString(url.QueryEscape(k))
			b.WriteByte('=')
			b.WriteString(url.QueryEscape(val))
		}
	}
	return b.String()
}

// --- Blacklist / allowlist rules ---

// blacklistSegments are URL path segments that indicate non-article pages.
var blacklistSegments = []string{
	"/tag/",
	"/category/",
	"/author/",
	"/search",
	"/wp-login",
	"/wp-admin",
	"/wp-json",
	"/wp-content/",
	"/wp-includes/",
	"/feed",
	"/comments",
	"/attachment/",
	"/trackback/",
	"/replytocom",
	"/xmlrpc.php",
	"/wp-cron.php",
}

// articlePatterns match URL paths that are likely articles.
var articlePatterns = []*regexp.Regexp{
	regexp.MustCompile(`/\d{4}/\d{2}/[^/]+`), // /YYYY/MM/slug
	regexp.MustCompile(`/blog/[^/]+`),        // /blog/slug
	regexp.MustCompile(`/news/[^/]+`),        // /news/slug
	regexp.MustCompile(`/post/[^/]+`),        // /post/slug
	regexp.MustCompile(`/articles?/[^/]+`),   // /article/slug or /articles/slug
}

// URLFilter decides whether a URL should be visited.
type URLFilter struct {
	AllowedDomains map[string]bool
}

// NewURLFilter creates a URLFilter that permits the given domains.
func NewURLFilter(domains []string) *URLFilter {
	allowed := make(map[string]bool, len(domains))
	for _, d := range domains {
		allowed[strings.ToLower(d)] = true
	}
	return &URLFilter{AllowedDomains: allowed}
}

// Allow returns true if the URL should be visited by the crawler.
// It normalizes the URL, checks domain membership, and applies blacklist rules.
func (f *URLFilter) Allow(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Hostname())
	if !f.AllowedDomains[host] {
		return false
	}

	return !isBlacklisted(u.Path)
}

// IsLikelyArticle returns true if the URL path matches common article patterns.
func IsLikelyArticle(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	p := strings.ToLower(u.Path)
	for _, re := range articlePatterns {
		if re.MatchString(p) {
			return true
		}
	}
	return false
}

// isBlacklisted checks a URL path against the blacklist segments.
func isBlacklisted(path string) bool {
	lower := strings.ToLower(path)
	for _, seg := range blacklistSegments {
		if strings.Contains(lower, seg) {
			return true
		}
	}
	return false
}

// AllowedDomainsForDomain returns the domain and its www variant.
func AllowedDomainsForDomain(domain string) []string {
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "www.")

	return []string{domain, "www." + domain}
}
