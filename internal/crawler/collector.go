// Package crawler provides a Colly collector factory and the CrawlService
// orchestrator that drives crawl discovery, URL filtering, and page callbacks.
package crawler

import (
	"time"

	"github.com/gocolly/colly/v2"

	"github.com/bdobrica/ThinkPixelSpider/internal/config"
)

// NewCollector creates a configured Colly collector from the application config
// and the list of allowed domains. For Phase 1 it always uses in-memory storage.
func NewCollector(cfg config.CrawlConfig, allowedDomains []string) *colly.Collector {
	c := colly.NewCollector(
		colly.AllowedDomains(allowedDomains...),
		colly.MaxDepth(cfg.MaxDepth),
		colly.UserAgent(cfg.UserAgent),
		colly.Async(true),
	)

	c.SetRequestTimeout(time.Duration(cfg.RequestTimeoutSeconds) * time.Second)

	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: cfg.Parallelism,
		Delay:       time.Duration(cfg.DelayMS) * time.Millisecond,
		RandomDelay: time.Duration(cfg.RandomDelayMS) * time.Millisecond,
	})

	return c
}
