package output

import (
	"os"
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

func TestURLToRelativePath_CollisionAddsHash(t *testing.T) {
	dir := t.TempDir()

	rel1, _ := URLToRelativePath("https://example.com/post", dir)
	absPath := filepath.Join(dir, rel1)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(absPath, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	rel2, err := URLToRelativePath("https://example.com/post", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel2 == rel1 {
		t.Error("expected collision to produce different path")
	}
	if filepath.Ext(rel2) != ".md" {
		t.Errorf("expected .md extension, got %q", filepath.Ext(rel2))
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
