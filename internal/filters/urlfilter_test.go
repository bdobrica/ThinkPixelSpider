package filters

import (
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "lowercase scheme and host",
			input: "HTTPS://Example.COM/Path",
			want:  "https://example.com/Path",
		},
		{
			name:  "strip fragment",
			input: "https://example.com/page#section",
			want:  "https://example.com/page",
		},
		{
			name:  "remove utm params",
			input: "https://example.com/page?utm_source=twitter&utm_medium=social&real=yes",
			want:  "https://example.com/page?real=yes",
		},
		{
			name:  "remove fbclid",
			input: "https://example.com/page?fbclid=abc123",
			want:  "https://example.com/page",
		},
		{
			name:  "remove gclid",
			input: "https://example.com/page?gclid=xyz&keep=1",
			want:  "https://example.com/page?keep=1",
		},
		{
			name:  "sort query params",
			input: "https://example.com/page?z=1&a=2&m=3",
			want:  "https://example.com/page?a=2&m=3&z=1",
		},
		{
			name:  "strip trailing slash",
			input: "https://example.com/blog/post/",
			want:  "https://example.com/blog/post",
		},
		{
			name:  "keep root slash",
			input: "https://example.com/",
			want:  "https://example.com/",
		},
		{
			name:  "combined normalization",
			input: "HTTP://WWW.Example.COM/blog/post/?utm_campaign=test&b=2&a=1#top",
			want:  "http://www.example.com/blog/post?a=1&b=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeURL() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeURL_NoQueryString(t *testing.T) {
	got, err := NormalizeURL("https://example.com/simple")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://example.com/simple" {
		t.Errorf("got %q, want unmodified URL", got)
	}
}

func TestNormalizeURL_AllTrackingParamsRemoved(t *testing.T) {
	got, err := NormalizeURL("https://example.com/page?utm_source=x&utm_medium=y&utm_campaign=z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "https://example.com/page" {
		t.Errorf("got %q, want URL with empty query", got)
	}
}

func TestNormalizeURL_InvalidURL(t *testing.T) {
	_, err := NormalizeURL("://")
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestURLFilterAllow(t *testing.T) {
	f := NewURLFilter([]string{"example.com", "www.example.com"})

	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"allowed domain", "https://example.com/blog/post", true},
		{"www variant", "https://www.example.com/blog/post", true},
		{"off-domain rejected", "https://other.com/page", false},
		{"blacklist tag", "https://example.com/tag/golang/", false},
		{"blacklist category", "https://example.com/category/tech/", false},
		{"blacklist wp-admin", "https://example.com/wp-admin/edit.php", false},
		{"blacklist wp-login", "https://example.com/wp-login.php", false},
		{"blacklist feed", "https://example.com/feed", false},
		{"blacklist wp-json", "https://example.com/wp-json/wp/v2/posts", false},
		{"blacklist wp-content", "https://example.com/wp-content/uploads/image.jpg", false},
		{"blacklist author", "https://example.com/author/john/", false},
		{"blacklist search", "https://example.com/search?q=test", false},
		{"blacklist comments", "https://example.com/page/comments/", false},
		{"blacklist trackback", "https://example.com/page/trackback/", false},
		{"blacklist xmlrpc", "https://example.com/xmlrpc.php", false},
		{"blacklist wp-cron", "https://example.com/wp-cron.php", false},
		{"blacklist wp-includes", "https://example.com/wp-includes/js/jquery.js", false},
		{"allowed normal page", "https://example.com/about", true},
		{"allowed article path", "https://example.com/2026/03/hello-world", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := f.Allow(tt.url); got != tt.want {
				t.Errorf("Allow(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestIsLikelyArticle(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://example.com/2024/03/my-post", true},
		{"https://example.com/blog/my-post", true},
		{"https://example.com/news/breaking-story", true},
		{"https://example.com/post/hello", true},
		{"https://example.com/article/deep-dive", true},
		{"https://example.com/articles/deep-dive", true},
		{"https://example.com/about", false},
		{"https://example.com/contact", false},
		{"https://example.com/", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			if got := IsLikelyArticle(tt.url); got != tt.want {
				t.Errorf("IsLikelyArticle(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestAllowedDomainsForDomain(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"example.com", []string{"example.com", "www.example.com"}},
		{"www.example.com", []string{"example.com", "www.example.com"}},
		{"Example.COM", []string{"example.com", "www.example.com"}},
		{"  example.com  ", []string{"example.com", "www.example.com"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := AllowedDomainsForDomain(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("AllowedDomainsForDomain(%q) = %v, want %v", tt.input, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("AllowedDomainsForDomain(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
