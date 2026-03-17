package markdown

import (
	"regexp"
	"strings"
)

// reBlankLines matches two or more consecutive blank lines.
var reBlankLines = regexp.MustCompile(`\n{3,}`)

// Normalize cleans up Markdown text:
//   - trims leading/trailing whitespace
//   - collapses three or more consecutive newlines to two (one blank line)
//   - normalises line endings to \n
func Normalize(md string) string {
	// Normalize line endings.
	md = strings.ReplaceAll(md, "\r\n", "\n")
	md = strings.ReplaceAll(md, "\r", "\n")

	// Collapse repeated blank lines.
	md = reBlankLines.ReplaceAllString(md, "\n\n")

	// Trim leading/trailing whitespace.
	md = strings.TrimSpace(md)

	return md
}
