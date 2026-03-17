// Package markdown provides HTML-to-Markdown conversion, Markdown
// normalization, and YAML front matter generation.
package markdown

import (
	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
)

// Converter converts cleaned HTML content to Markdown.
type Converter interface {
	Convert(html string) (string, error)
}

// DefaultConverter uses the html-to-markdown library.
type DefaultConverter struct{}

// NewConverter returns a DefaultConverter.
func NewConverter() *DefaultConverter {
	return &DefaultConverter{}
}

// Convert transforms an HTML string into Markdown using html-to-markdown.
func (c *DefaultConverter) Convert(html string) (string, error) {
	return htmltomarkdown.ConvertString(html)
}
