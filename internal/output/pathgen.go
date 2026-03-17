package output

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// URLToRelativePath derives a deterministic file path from a page URL.
//
// Strategy (per arch §5A):
//   - Use the URL host + path to build a directory/file structure.
//   - Strip leading/trailing slashes, replace remaining slashes with os separators.
//   - Append ".md" extension.
//   - If the resulting path collides with an existing file in baseDir, append
//     a short hash suffix: "slug--<hash>.md".
func URLToRelativePath(rawURL string, baseDir string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	host := strings.ToLower(u.Hostname())
	if host == "" {
		return "", fmt.Errorf("URL %q has no host", rawURL)
	}

	// Clean the path: trim slashes, default to "index" for root.
	p := strings.Trim(u.Path, "/")
	if p == "" {
		p = "index"
	}
	// Remove .html / .htm extensions so we get clean .md names.
	p = strings.TrimSuffix(p, ".html")
	p = strings.TrimSuffix(p, ".htm")

	rel := filepath.Join("pages", host, filepath.FromSlash(p)) + ".md"

	// If the path already exists, append a hash suffix to avoid collisions.
	candidate := filepath.Join(baseDir, rel)
	if _, err := os.Stat(candidate); err == nil {
		hash := shortHash(rawURL)
		ext := filepath.Ext(rel)
		rel = strings.TrimSuffix(rel, ext) + "--" + hash + ext
	}

	return rel, nil
}

// shortHash returns the first 6 hex chars of the SHA-256 of s.
func shortHash(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h[:3])
}
