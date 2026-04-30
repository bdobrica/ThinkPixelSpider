// Package config defines the central configuration struct and provides
// functions to load configuration from defaults, environment variables,
// and CLI flags with the precedence: flags → env → defaults.
package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for both CLI and daemon execution modes.
type Config struct {
	AppMode  string
	LogLevel string

	// Domain is set via CLI flag only (not an env var).
	Domain string

	Crawl CrawlConfig
	Colly CollyConfig

	Output OutputConfig
	Queue  QueueConfig
}

// CrawlConfig controls crawler behaviour.
type CrawlConfig struct {
	MaxPages              int
	MaxDepth              int
	RequestTimeoutSeconds int
	UserAgent             string
	DelayMS               int
	RandomDelayMS         int
	Parallelism           int
	IncludeSubdomains     bool
	DiscoveryMode         string
	MinWordCount          int
}

// CollyConfig controls the Colly storage backend.
type CollyConfig struct {
	StorageType string // "memory" or "redis"
	RedisAddr   string
	RedisDB     int
	RedisPrefix string
}

// OutputConfig controls CLI-mode output paths and manifest format.
type OutputConfig struct {
	Directory    string
	ManifestType string // "csv" for now; SQLite support is planned.
	SQLitePath   string
}

// QueueConfig controls daemon-mode queue backends.
type QueueConfig struct {
	Backend string // "redis" or "nats"
	Redis   QueueRedisConfig
	NATS    QueueNATSConfig
}

// QueueRedisConfig holds Redis Streams queue settings.
type QueueRedisConfig struct {
	Addr          string
	StreamInput   string
	StreamOutput  string
	ConsumerGroup string
}

// QueueNATSConfig holds NATS JetStream queue settings.
type QueueNATSConfig struct {
	URL           string
	InputSubject  string
	OutputSubject string
	DurableName   string
}

// defaults returns a Config populated with sensible default values.
func defaults() Config {
	return Config{
		AppMode:  "cli",
		LogLevel: "info",
		Crawl: CrawlConfig{
			MaxPages:              500,
			MaxDepth:              4,
			RequestTimeoutSeconds: 15,
			UserAgent:             "thinkpixelspider/1.0",
			DelayMS:               200,
			RandomDelayMS:         300,
			Parallelism:           4,
			IncludeSubdomains:     false,
			DiscoveryMode:         "both",
			MinWordCount:          250,
		},
		Colly: CollyConfig{
			StorageType: "memory",
			RedisAddr:   "redis:6379",
			RedisDB:     0,
			RedisPrefix: "thinkpixelspider",
		},
		Output: OutputConfig{
			Directory:    "./output",
			ManifestType: "csv",
			SQLitePath:   "./output/manifest.sqlite",
		},
		Queue: QueueConfig{
			Backend: "redis",
			Redis: QueueRedisConfig{
				Addr:          "redis:6379",
				StreamInput:   "crawl_jobs",
				StreamOutput:  "crawled_pages",
				ConsumerGroup: "thinkpixelspider",
			},
			NATS: QueueNATSConfig{
				URL:           "nats://nats:4222",
				InputSubject:  "crawl.jobs",
				OutputSubject: "crawl.pages",
				DurableName:   "thinkpixelspider",
			},
		},
	}
}

// loadEnv overlays environment variable values onto cfg.
// Only non-empty env vars override the existing value.
func loadEnv(cfg *Config) {
	if v := os.Getenv("APP_MODE"); v != "" {
		cfg.AppMode = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	// Crawl
	setIntFromEnv(&cfg.Crawl.MaxPages, "CRAWLER_MAX_PAGES")
	setIntFromEnv(&cfg.Crawl.MaxDepth, "CRAWLER_MAX_DEPTH")
	setIntFromEnv(&cfg.Crawl.RequestTimeoutSeconds, "CRAWLER_TIMEOUT_SECONDS")
	if v := os.Getenv("CRAWLER_USER_AGENT"); v != "" {
		cfg.Crawl.UserAgent = v
	}
	setIntFromEnv(&cfg.Crawl.DelayMS, "CRAWLER_DELAY_MS")
	setIntFromEnv(&cfg.Crawl.RandomDelayMS, "CRAWLER_RANDOM_DELAY_MS")
	setIntFromEnv(&cfg.Crawl.Parallelism, "CRAWLER_PARALLELISM")
	setBoolFromEnv(&cfg.Crawl.IncludeSubdomains, "CRAWLER_INCLUDE_SUBDOMAINS")
	if v := os.Getenv("CRAWLER_DISCOVERY_MODE"); v != "" {
		cfg.Crawl.DiscoveryMode = v
	}
	setIntFromEnv(&cfg.Crawl.MinWordCount, "CRAWLER_MIN_WORD_COUNT")

	// Colly storage
	if v := os.Getenv("COLLY_STORAGE"); v != "" {
		cfg.Colly.StorageType = v
	}
	if v := os.Getenv("COLLY_REDIS_ADDR"); v != "" {
		cfg.Colly.RedisAddr = v
	}
	setIntFromEnv(&cfg.Colly.RedisDB, "COLLY_REDIS_DB")
	if v := os.Getenv("COLLY_REDIS_PREFIX"); v != "" {
		cfg.Colly.RedisPrefix = v
	}

	// Output
	if v := os.Getenv("OUTPUT_DIR"); v != "" {
		cfg.Output.Directory = v
	}
	if v := os.Getenv("OUTPUT_MANIFEST_TYPE"); v != "" {
		cfg.Output.ManifestType = v
	}
	if v := os.Getenv("OUTPUT_SQLITE_PATH"); v != "" {
		cfg.Output.SQLitePath = v
	}

	// Queue
	if v := os.Getenv("QUEUE_BACKEND"); v != "" {
		cfg.Queue.Backend = v
	}
	if v := os.Getenv("QUEUE_REDIS_ADDR"); v != "" {
		cfg.Queue.Redis.Addr = v
	}
	if v := os.Getenv("QUEUE_REDIS_INPUT_STREAM"); v != "" {
		cfg.Queue.Redis.StreamInput = v
	}
	if v := os.Getenv("QUEUE_REDIS_OUTPUT_STREAM"); v != "" {
		cfg.Queue.Redis.StreamOutput = v
	}
	if v := os.Getenv("QUEUE_REDIS_CONSUMER_GROUP"); v != "" {
		cfg.Queue.Redis.ConsumerGroup = v
	}
	if v := os.Getenv("QUEUE_NATS_URL"); v != "" {
		cfg.Queue.NATS.URL = v
	}
	if v := os.Getenv("QUEUE_NATS_INPUT_SUBJECT"); v != "" {
		cfg.Queue.NATS.InputSubject = v
	}
	if v := os.Getenv("QUEUE_NATS_OUTPUT_SUBJECT"); v != "" {
		cfg.Queue.NATS.OutputSubject = v
	}
	if v := os.Getenv("QUEUE_NATS_DURABLE_NAME"); v != "" {
		cfg.Queue.NATS.DurableName = v
	}
}

// loadFlags parses CLI flags using the provided FlagSet and overlays them onto
// cfg. Only flags that were explicitly set by the user override the existing
// value.
func loadFlags(cfg *Config, fs *flag.FlagSet, args []string) error {
	domain := fs.String("domain", "", "target domain to crawl")
	output := fs.String("output", "", "output directory for crawled content")
	manifest := fs.String("manifest", "", "manifest format: csv")
	maxPages := fs.Int("max-pages", 0, "maximum number of pages to crawl")
	maxDepth := fs.Int("max-depth", 0, "maximum crawl depth")
	discovery := fs.String("discovery", "", "discovery mode: sitemap, links, or both")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parsing flags: %w", err)
	}

	// Only apply flags that were explicitly provided.
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "domain":
			cfg.Domain = *domain
		case "output":
			cfg.Output.Directory = *output
		case "manifest":
			cfg.Output.ManifestType = *manifest
		case "max-pages":
			cfg.Crawl.MaxPages = *maxPages
		case "max-depth":
			cfg.Crawl.MaxDepth = *maxDepth
		case "discovery":
			cfg.Crawl.DiscoveryMode = *discovery
		}
	})

	return nil
}

// Load builds a Config by applying the three layers in order:
// defaults → environment variables → CLI flags.
// The args slice should typically be os.Args[1:].
func Load(args []string) (Config, error) {
	cfg := defaults()
	loadEnv(&cfg)

	fs := flag.NewFlagSet("thinkpixelspider", flag.ContinueOnError)
	if err := loadFlags(&cfg, fs, args); err != nil {
		return Config{}, err
	}

	if err := validate(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// validate checks that the final Config contains values that make sense.
func validate(cfg *Config) error {
	mode := strings.ToLower(cfg.AppMode)
	if mode != "cli" && mode != "daemon" {
		return fmt.Errorf("invalid APP_MODE %q: must be cli or daemon", cfg.AppMode)
	}

	if cfg.Crawl.MaxPages < 0 {
		return fmt.Errorf("max-pages must be non-negative, got %d", cfg.Crawl.MaxPages)
	}
	if cfg.Crawl.MaxDepth < 0 {
		return fmt.Errorf("max-depth must be non-negative, got %d", cfg.Crawl.MaxDepth)
	}

	dm := strings.ToLower(cfg.Crawl.DiscoveryMode)
	if dm != "sitemap" && dm != "links" && dm != "both" {
		return fmt.Errorf("invalid discovery mode %q: must be sitemap, links, or both", cfg.Crawl.DiscoveryMode)
	}
	cfg.Crawl.DiscoveryMode = dm

	mt := strings.ToLower(cfg.Output.ManifestType)
	if mt == "sqlite" {
		return fmt.Errorf("manifest type %q is not implemented yet: use csv", cfg.Output.ManifestType)
	}
	if mt != "csv" {
		return fmt.Errorf("invalid manifest type %q: must be csv", cfg.Output.ManifestType)
	}
	cfg.Output.ManifestType = mt

	return nil
}

// --- helpers ---

func setIntFromEnv(target *int, envKey string) {
	if v := os.Getenv(envKey); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			*target = n
		}
	}
}

func setBoolFromEnv(target *bool, envKey string) {
	if v := os.Getenv(envKey); v != "" {
		switch strings.ToLower(v) {
		case "true", "1", "yes":
			*target = true
		case "false", "0", "no":
			*target = false
		}
	}
}
