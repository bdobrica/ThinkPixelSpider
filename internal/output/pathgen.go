package output

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/bdobrica/ThinkPixelSpider/internal/filters"
)

// URLToRelativePath derives a deterministic file path from a page URL.
//
// Strategy (per arch §5A):
//   - Normalize the URL first so equivalent URLs share the same mapping.
//   - Use the URL host + path to build a directory/file structure.
//   - Strip leading/trailing slashes, replace remaining slashes with os separators.
//   - Append ".md" extension.
//   - If the mapping would otherwise be lossy (for example due to query
//     parameters or stripping .html/.htm), append a short hash suffix:
//     "slug--<hash>.md".
func URLToRelativePath(rawURL string, baseDir string) (string, error) {
	_ = baseDir

	normalizedURL, err := filters.NormalizeURL(rawURL)
	if err != nil {
		return "", fmt.Errorf("normalize URL %q: %w", rawURL, err)
	}

	u, err := url.Parse(normalizedURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", normalizedURL, err)
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", fmt.Errorf("URL %q has no host", normalizedURL)
	}

	// Clean the path: trim slashes, default to "index" for root.
	originalPath := strings.Trim(u.Path, "/")
	p := originalPath
	if p == "" {
		p = "index"
	}
	// Remove .html / .htm extensions so we get clean .md names.
	cleanPath := strings.TrimSuffix(strings.TrimSuffix(p, ".html"), ".htm")
	if cleanPath == "" {
		cleanPath = "index"
	}

	base := filepath.Join("pages", host, filepath.FromSlash(cleanPath))
	rel := base + ".md"

	pathWasStripped := originalPath != "" && cleanPath != originalPath
	if u.RawQuery != "" || pathWasStripped {
		rel = base + "--" + shortHash(normalizedURL) + ".md"
	}

	return rel, nil
}

// shortHash returns the first 6 hex chars of the SHA-256 of s.
func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:3])
}
