package main

import (
	"strings"
	"testing"

	"github.com/bdobrica/ThinkPixelSpider/internal/config"
)

func TestNewManifestWriter_CSV(t *testing.T) {
	outDir := t.TempDir()
	cfg := config.Config{
		Output: config.OutputConfig{
			Directory:    outDir,
			ManifestType: "CSV",
		},
	}

	w, manifestPath, err := newManifestWriter(cfg)
	if err != nil {
		t.Fatalf("newManifestWriter(csv): %v", err)
	}
	if w == nil {
		t.Fatal("newManifestWriter(csv) returned nil writer")
	}
	if !strings.HasSuffix(manifestPath, "manifest.csv") {
		t.Fatalf("manifest path = %q, want suffix %q", manifestPath, "manifest.csv")
	}

	if err := w.Close(); err != nil {
		t.Fatalf("Close(): %v", err)
	}
}

func TestNewManifestWriter_SQLiteNotImplemented(t *testing.T) {
	outDir := t.TempDir()
	cfg := config.Config{
		Output: config.OutputConfig{
			Directory:    outDir,
			ManifestType: "sqlite",
			SQLitePath:   outDir + "/manifest.sqlite",
		},
	}

	w, manifestPath, err := newManifestWriter(cfg)
	if err == nil {
		t.Fatal("newManifestWriter(sqlite) error = nil, want explicit not implemented error")
	}
	if w != nil {
		t.Fatal("newManifestWriter(sqlite) writer should be nil")
	}
	if manifestPath != "" {
		t.Fatalf("newManifestWriter(sqlite) manifest path = %q, want empty", manifestPath)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "not implemented") {
		t.Fatalf("sqlite error = %q, want to mention not implemented", err.Error())
	}
}
