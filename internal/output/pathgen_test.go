package output

import (
	"path/filepath"
	"testing"
)

func TestURLToRelativePath_BasicURL(t *testing.T) {
	dir := t.TempDir()
	rel, err := URLToRelativePath("https://example.com/blog/hello-world", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("pages", "example.com", "blog", "hello-world.md")
	if rel != want {
		t.Errorf("got %q, want %q", rel, want)
	}
}

func TestURLToRelativePath_RootPath(t *testing.T) {
	dir := t.TempDir()
	rel, err := URLToRelativePath("https://example.com/", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("pages", "example.com", "index.md")
	if rel != want {
		t.Errorf("got %q, want %q", rel, want)
	}
}

func TestURLToRelativePath_StripsHTMLExtension(t *testing.T) {
	dir := t.TempDir()
	rel, err := URLToRelativePath("https://example.com/page.html", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("pages", "example.com", "page.md")
	if rel != want {
		t.Errorf("got %q, want %q", rel, want)
	}
}

func TestURLToRelativePath_DeepPath(t *testing.T) {
	dir := t.TempDir()
	rel, err := URLToRelativePath("https://example.com/2026/03/my-post", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join("pages", "example.com", "2026", "03", "my-post.md")
	if rel != want {
		t.Errorf("got %q, want %q", rel, want)
	}
}

func TestURLToRelativePath_SameURLIsDeterministicEvenIfFileExists(t *testing.T) {
	dir := t.TempDir()

	rel1, err := URLToRelativePath("https://example.com/post", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rel2, err := URLToRelativePath("https://example.com/post", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel2 != rel1 {
		t.Errorf("deterministic mapping mismatch: first %q, second %q", rel1, rel2)
	}
}

func TestURLToRelativePath_QueryStringAddsDeterministicHashSuffix(t *testing.T) {
	dir := t.TempDir()

	rel, err := URLToRelativePath("https://example.com/post?b=2&a=1", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := filepath.Join("pages", "example.com", "post--e51813.md")
	if rel != want {
		t.Errorf("got %q, want %q", rel, want)
	}

	relEquivalent, err := URLToRelativePath("https://example.com/post?a=1&b=2", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if relEquivalent != rel {
		t.Errorf("normalized equivalent query path = %q, want %q", relEquivalent, rel)
	}

	relDifferent, err := URLToRelativePath("https://example.com/post?a=2&b=1", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if relDifferent == rel {
		t.Errorf("different query should map to different path, both got %q", rel)
	}
	if filepath.Dir(relDifferent) != filepath.Join("pages", "example.com") {
		t.Errorf("unexpected directory for query-derived path: %q", filepath.Dir(relDifferent))
	}
	if filepath.Ext(relDifferent) != ".md" {
		t.Errorf("expected .md extension, got %q", filepath.Ext(relDifferent))
	}
	if relDifferent[:len(filepath.Join("pages", "example.com", "post--"))] != filepath.Join("pages", "example.com", "post--") {
		t.Errorf("expected hashed query filename prefix, got %q", relDifferent)
	}
	if relDifferent == filepath.Join("pages", "example.com", "post.md") {
		t.Error("query-distinct URL should not map to the plain path-only filename")
	}
}

func TestURLToRelativePath_HTMLVariantGetsHashSuffix(t *testing.T) {
	dir := t.TempDir()

	plain, err := URLToRelativePath("https://example.com/page", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	html, err := URLToRelativePath("https://example.com/page.html", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if plain != filepath.Join("pages", "example.com", "page.md") {
		t.Errorf("plain path = %q, want %q", plain, filepath.Join("pages", "example.com", "page.md"))
	}
	if html == plain {
		t.Fatalf("html variant path = %q, want distinct path from %q", html, plain)
	}
	if html != filepath.Join("pages", "example.com", "page--0fb02f.md") {
		t.Errorf("html variant path = %q, want %q", html, filepath.Join("pages", "example.com", "page--0fb02f.md"))
	}
}

func TestURLToRelativePath_InvalidURL(t *testing.T) {
	dir := t.TempDir()
	_, err := URLToRelativePath("://bad", dir)
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestURLToRelativePath_NoHost(t *testing.T) {
	dir := t.TempDir()
	_, err := URLToRelativePath("/just/a/path", dir)
	if err == nil {
		t.Error("expected error for URL without host")
	}
}
