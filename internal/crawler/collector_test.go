package crawler

import (
	"testing"

	"github.com/bdobrica/ThinkPixelSpider/internal/config"
)

func TestNewCollector(t *testing.T) {
	cfg := config.CrawlConfig{
		MaxDepth:              3,
		UserAgent:             "testbot/1.0",
		RequestTimeoutSeconds: 10,
		Parallelism:           2,
		DelayMS:               100,
		RandomDelayMS:         50,
	}
	domains := []string{"example.com", "www.example.com"}

	c := NewCollector(cfg, domains)
	if c == nil {
		t.Fatal("NewCollector returned nil")
	}

	// Verify user agent was set.
	if c.UserAgent != "testbot/1.0" {
		t.Errorf("UserAgent = %q, want %q", c.UserAgent, "testbot/1.0")
	}

	// Verify max depth was set.
	if c.MaxDepth != 3 {
		t.Errorf("MaxDepth = %d, want 3", c.MaxDepth)
	}
}
