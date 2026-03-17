package markdown

import (
	"strings"
	"testing"
)

func TestDefaultConverter_BasicHTML(t *testing.T) {
	c := NewConverter()
	md, err := c.Convert("<h1>Hello</h1><p>World</p>")
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if !strings.Contains(md, "Hello") {
		t.Error("expected markdown to contain 'Hello'")
	}
	if !strings.Contains(md, "World") {
		t.Error("expected markdown to contain 'World'")
	}
}

func TestDefaultConverter_Links(t *testing.T) {
	c := NewConverter()
	md, err := c.Convert(`<a href="https://example.com">Link</a>`)
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if !strings.Contains(md, "https://example.com") {
		t.Error("expected markdown to preserve link URL")
	}
}

func TestDefaultConverter_EmptyInput(t *testing.T) {
	c := NewConverter()
	md, err := c.Convert("")
	if err != nil {
		t.Fatalf("Convert returned error: %v", err)
	}
	if strings.TrimSpace(md) != "" {
		t.Errorf("expected empty output, got %q", md)
	}
}
