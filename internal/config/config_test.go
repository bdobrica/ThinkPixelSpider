package config
package config

import (
	"os"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := defaults()

	if cfg.AppMode != "cli" {
		t.Errorf("AppMode = %q, want %q", cfg.AppMode, "cli")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.Crawl.MaxPages != 500 {
		t.Errorf("Crawl.MaxPages = %d, want 500", cfg.Crawl.MaxPages)
	}
	if cfg.Crawl.MaxDepth != 4 {
		t.Errorf("Crawl.MaxDepth = %d, want 4", cfg.Crawl.MaxDepth)
	}
	if cfg.Crawl.UserAgent != "thinkpixelspider/1.0" {
		t.Errorf("Crawl.UserAgent = %q, want %q", cfg.Crawl.UserAgent, "thinkpixelspider/1.0")
	}
	if cfg.Crawl.DiscoveryMode != "both" {
		t.Errorf("Crawl.DiscoveryMode = %q, want %q", cfg.Crawl.DiscoveryMode, "both")
	}
	if cfg.Crawl.MinWordCount != 250 {
		t.Errorf("Crawl.MinWordCount = %d, want 250", cfg.Crawl.MinWordCount)
	}
	if cfg.Colly.StorageType != "memory" {
		t.Errorf("Colly.StorageType = %q, want %q", cfg.Colly.StorageType, "memory")
	}
	if cfg.Output.ManifestType != "csv" {
		t.Errorf("Output.ManifestType = %q, want %q", cfg.Output.ManifestType, "csv")
	}
}

func TestLoadEnvOverridesDefaults(t *testing.T) {
	envVars := map[string]string{
		"APP_MODE":                  "daemon",
		"LOG_LEVEL":                 "debug",
		"CRAWLER_MAX_PAGES":         "100",
		"CRAWLER_MAX_DEPTH":         "2",
		"CRAWLER_TIMEOUT_SECONDS":   "30",
		"CRAWLER_USER_AGENT":        "testbot/2.0",
		"CRAWLER_DELAY_MS":          "500",
		"CRAWLER_RANDOM_DELAY_MS":   "100",
		"CRAWLER_PARALLELISM":       "8",
		"CRAWLER_INCLUDE_SUBDOMAINS": "true",
		"CRAWLER_DISCOVERY_MODE":    "sitemap",
		"CRAWLER_MIN_WORD_COUNT":    "50",
		"COLLY_STORAGE":             "redis",
		"COLLY_REDIS_ADDR":          "localhost:6380",
		"COLLY_REDIS_DB":            "3",
		"COLLY_REDIS_PREFIX":        "test",
		"OUTPUT_DIR":                "/tmp/out",
		"OUTPUT_MANIFEST_TYPE":      "sqlite",
		"OUTPUT_SQLITE_PATH":        "/tmp/out/db.sqlite",
	}

	for k, v := range envVars {
		t.Setenv(k, v)
	}

	cfg := defaults()
	loadEnv(&cfg)

	if cfg.AppMode != "daemon" {
		t.Errorf("AppMode = %q, want %q", cfg.AppMode, "daemon")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.Crawl.MaxPages != 100 {
		t.Errorf("Crawl.MaxPages = %d, want 100", cfg.Crawl.MaxPages)
	}
	if cfg.Crawl.MaxDepth != 2 {
		t.Errorf("Crawl.MaxDepth = %d, want 2", cfg.Crawl.MaxDepth)
	}
	if cfg.Crawl.RequestTimeoutSeconds != 30 {
		t.Errorf("Crawl.RequestTimeoutSeconds = %d, want 30", cfg.Crawl.RequestTimeoutSeconds)
	}
	if cfg.Crawl.UserAgent != "testbot/2.0" {
		t.Errorf("Crawl.UserAgent = %q, want %q", cfg.Crawl.UserAgent, "testbot/2.0")
	}
	if cfg.Crawl.DelayMS != 500 {
		t.Errorf("Crawl.DelayMS = %d, want 500", cfg.Crawl.DelayMS)
	}
	if cfg.Crawl.RandomDelayMS != 100 {
		t.Errorf("Crawl.RandomDelayMS = %d, want 100", cfg.Crawl.RandomDelayMS)
	}
	if cfg.Crawl.Parallelism != 8 {
		t.Errorf("Crawl.Parallelism = %d, want 8", cfg.Crawl.Parallelism)
	}
	if !cfg.Crawl.IncludeSubdomains {
		t.Error("Crawl.IncludeSubdomains = false, want true")
	}
	if cfg.Crawl.DiscoveryMode != "sitemap" {
		t.Errorf("Crawl.DiscoveryMode = %q, want %q", cfg.Crawl.DiscoveryMode, "sitemap")
	}
	if cfg.Crawl.MinWordCount != 50 {
		t.Errorf("Crawl.MinWordCount = %d, want 50", cfg.Crawl.MinWordCount)
	}
	if cfg.Colly.StorageType != "redis" {
		t.Errorf("Colly.StorageType = %q, want %q", cfg.Colly.StorageType, "redis")
	}
	if cfg.Colly.RedisAddr != "localhost:6380" {
		t.Errorf("Colly.RedisAddr = %q, want %q", cfg.Colly.RedisAddr, "localhost:6380")
	}
	if cfg.Colly.RedisDB != 3 {
		t.Errorf("Colly.RedisDB = %d, want 3", cfg.Colly.RedisDB)
	}
	if cfg.Colly.RedisPrefix != "test" {
		t.Errorf("Colly.RedisPrefix = %q, want %q", cfg.Colly.RedisPrefix, "test")
	}
	if cfg.Output.Directory != "/tmp/out" {
		t.Errorf("Output.Directory = %q, want %q", cfg.Output.Directory, "/tmp/out")
	}
	if cfg.Output.ManifestType != "sqlite" {
		t.Errorf("Output.ManifestType = %q, want %q", cfg.Output.ManifestType, "sqlite")
	}
	if cfg.Output.SQLitePath != "/tmp/out/db.sqlite" {
		t.Errorf("Output.SQLitePath = %q, want %q", cfg.Output.SQLitePath, "/tmp/out/db.sqlite")
	}
}

func TestLoadFlagsOverrideEnv(t *testing.T) {
	// Set env vars first.
	t.Setenv("OUTPUT_DIR", "/env/output")
	t.Setenv("OUTPUT_MANIFEST_TYPE", "sqlite")
	t.Setenv("CRAWLER_MAX_PAGES", "100")

	// Load with flags that override some values.
	cfg, err := Load([]string{
		"--domain", "example.com",
		"--output", "/flag/output",
		"--manifest", "csv",
	})
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.Domain != "example.com" {
		t.Errorf("Domain = %q, want %q", cfg.Domain, "example.com")
	}
	// Flag should win over env.
	if cfg.Output.Directory != "/flag/output" {
		t.Errorf("Output.Directory = %q, want %q", cfg.Output.Directory, "/flag/output")
	}
	if cfg.Output.ManifestType != "csv" {
		t.Errorf("Output.ManifestType = %q, want %q", cfg.Output.ManifestType, "csv")
	}
	// Env should still apply when no flag was set.
	if cfg.Crawl.MaxPages != 100 {
		t.Errorf("Crawl.MaxPages = %d, want 100 (from env)", cfg.Crawl.MaxPages)
	}
}

func TestLoadPrecedenceDefaultsOnly(t *testing.T) {
	// Ensure no env vars leak in.
	for _, key := range []string{
		"APP_MODE", "CRAWLER_MAX_PAGES", "OUTPUT_DIR", "OUTPUT_MANIFEST_TYPE",
	} {
		os.Unsetenv(key)
	}

	cfg, err := Load([]string{})
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if cfg.AppMode != "cli" {
		t.Errorf("AppMode = %q, want default %q", cfg.AppMode, "cli")
	}
	if cfg.Crawl.MaxPages != 500 {
		t.Errorf("Crawl.MaxPages = %d, want default 500", cfg.Crawl.MaxPages)
	}
	if cfg.Output.Directory != "./output" {
		t.Errorf("Output.Directory = %q, want default %q", cfg.Output.Directory, "./output")
	}
}

func TestValidateRejectsInvalidAppMode(t *testing.T) {
	_, err := Load([]string{})
	if err != nil {
		t.Fatalf("unexpected error with defaults: %v", err)
	}

	t.Setenv("APP_MODE", "invalid")
	_, err = Load([]string{})
	if err == nil {
		t.Fatal("expected error for invalid APP_MODE, got nil")
	}
}

func TestValidateRejectsInvalidDiscoveryMode(t *testing.T) {
	t.Setenv("CRAWLER_DISCOVERY_MODE", "magic")
	_, err := Load([]string{})
	if err == nil {
		t.Fatal("expected error for invalid discovery mode, got nil")
	}
}

func TestValidateRejectsInvalidManifestType(t *testing.T) {
	_, err := Load([]string{"--manifest", "json"})
	if err == nil {
		t.Fatal("expected error for invalid manifest type, got nil")
	}
}

func TestValidateRejectsNegativeMaxPages(t *testing.T) {
	_, err := Load([]string{"--max-pages", "-1"})
	if err == nil {
		t.Fatal("expected error for negative max-pages, got nil")
	}
}

func TestSetBoolFromEnvValues(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"TRUE", true},
		{"FALSE", false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got := false
			t.Setenv("TEST_BOOL", tt.value)
			setBoolFromEnv(&got, "TEST_BOOL")
			if got != tt.want {
				t.Errorf("setBoolFromEnv(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestInvalidIntEnvIgnored(t *testing.T) {
	t.Setenv("CRAWLER_MAX_PAGES", "notanumber")

	cfg := defaults()
	loadEnv(&cfg)

	// Should keep the default when the env var is not a valid int.
	if cfg.Crawl.MaxPages != 500 {
		t.Errorf("Crawl.MaxPages = %d, want default 500 (invalid env should be ignored)", cfg.Crawl.MaxPages)
	}
}
