# ThinkPixelSpider — TODO

> Actionable task list organized by implementation phase.
> For architecture details, domain models, interface definitions, and design
> rationale, see [docs/architecture.md](docs/architecture.md).

---

## Phase 0 — Project bootstrap

- [x] Initialize Go module (`go mod init`)
- [x] Create directory layout: `cmd/thinkpixelspider/`, `cmd/thinkpixelspiderd/`, `internal/`, `pkg/`
- [x] Add `.gitignore` (Go defaults, output dirs, IDE files)
- [x] Add `Makefile` with targets: `build`, `test`, `lint`, `run-cli`
- [x] Add `Dockerfile` (multi-stage build)
- [x] Pick and pin dependency versions: Colly, go-readability, HTML-to-Markdown lib
- [x] Define core domain types in `internal/jobs/models.go`: `CrawlJob`, `CrawledPage`, `ManifestRow`, `CrawlSummary`, `CrawlError` — [arch §4](docs/architecture.md#4-core-domain-models)

---

## Phase 1 — Local CLI MVP

> Goal: crawl one WordPress domain, produce Markdown files + CSV manifest.
> Success: run on a real WP site, get 20–100 useful Markdown files.
> Ref: [arch §26 Phase 1](docs/architecture.md#phase-1--local-cli-mvp), [arch §28](docs/architecture.md#28-minimal-first-milestone-i-would-build)

### Config

- [ ] Implement `Config` struct — [arch §16](docs/architecture.md#16-configuration-model)
- [ ] Load config from env vars — [arch §17](docs/architecture.md#17-suggested-env-vars)
- [ ] Load config from CLI flags (`--domain`, `--output`, `--manifest`)
- [ ] Apply precedence: flags → env → defaults

### Crawler core

- [ ] Implement Colly collector factory with in-memory storage — [arch §7](docs/architecture.md#7-colly-storage-plan), [arch §20](docs/architecture.md#20-colly-factory-details)
- [ ] Set up `AllowedDomains` (domain + www variant) — [arch §8](docs/architecture.md#8-domain-restrictions-and-duplicate-prevention)
- [ ] Configure rate limiting, user agent, max depth, request timeout
- [ ] Implement sitemap discovery (`/sitemap.xml`, `/sitemap_index.xml`) — [arch §9](docs/architecture.md#9-crawl-discovery-strategy)
- [ ] Implement link discovery via `OnHTML("a[href]")` with normalization — [arch §21](docs/architecture.md#21-what-should-happen-on-onhtmlahref)
- [ ] Implement `CrawlService` orchestrator — [arch §19](docs/architecture.md#19-crawl-service-design)

### URL filtering

- [ ] Implement URL normalization (lowercase, strip fragments, remove tracking params, sort query params) — [arch §8](docs/architecture.md#8-domain-restrictions-and-duplicate-prevention)
- [ ] Implement URL blacklist rules (reject `/tag/`, `/category/`, `/wp-admin/`, etc.) — [arch §10 Tier 1](docs/architecture.md#10-article-candidate-filtering)
- [ ] Implement article-likely URL boost heuristics (`/YYYY/MM/slug`, `/blog/`, etc.)

### Extraction

- [ ] Integrate go-readability: extract title, byline, site name, HTML content, text, excerpt — [arch §11 Step 3](docs/architecture.md#step-3-go-readability-extraction)
- [ ] Implement basic post-extraction validation (minimum word count threshold) — [arch §11 Step 4](docs/architecture.md#step-4-post-extraction-heuristics)
- [ ] Implement HTML-to-Markdown conversion behind `MarkdownConverter` interface — [arch §11 Step 5](docs/architecture.md#step-5-html-to-markdown)
- [ ] Implement Markdown normalization (trim whitespace, collapse blank lines) — [arch §11 Step 6](docs/architecture.md#step-6-normalize-markdown)
- [ ] Add YAML front matter to Markdown output (url, title, byline, crawled_at, word_count, content_hash) — [arch §12](docs/architecture.md#12-markdown-output-format)

### Output

- [ ] Implement `PageSink` interface — [arch §19](docs/architecture.md#19-crawl-service-design)
- [ ] Implement `FilesystemPageSink`: deterministic path from URL, hash fallback for collisions — [arch §5A](docs/architecture.md#file-naming-rule)
- [ ] Implement `ManifestWriter` interface — [arch §13](docs/architecture.md#13-cli-output-persistence)
- [ ] Implement CSV manifest writer

### CLI entry point

- [ ] Wire up `cmd/thinkpixelspider/main.go`: config → collector → crawl service → filesystem sink → CSV manifest
- [ ] Print crawl summary on completion (pages discovered, visited, extracted, errors)

### Tests (Phase 1)

- [ ] Unit tests: URL normalization
- [ ] Unit tests: URL filtering / blacklist rules
- [ ] Unit tests: path generation from URL
- [ ] Unit tests: Markdown normalization
- [ ] Unit tests: config loading
- [ ] Integration test: local HTTP test server with fake WP pages; validate output files — [arch §27](docs/architecture.md#27-testing-plan)
- [ ] Manual validation: run against 3–5 real WordPress sites

---

## Phase 2 — Stronger CLI

> Goal: production-quality CLI with SQLite, better dedup, better filtering.
> Ref: [arch §26 Phase 2](docs/architecture.md#phase-2--stronger-cli)

### Manifest & dedup

- [ ] Implement SQLite manifest writer (`pages` table + indexes) — [arch §13](docs/architecture.md#13-cli-output-persistence)
- [ ] Make SQLite the default manifest, CSV optional via `--manifest csv`
- [ ] Implement content hashing (SHA-256 of extracted text)
- [ ] Skip pages with duplicate content hashes

### URL & canonical handling

- [ ] Extract and use canonical URL from page metadata — [arch §8](docs/architecture.md#canonical-handling)
- [ ] Use canonical URL as primary identity when it differs from fetched URL

### Filtering improvements

- [ ] Add HTML metadata signals: `<article>`, `og:type=article`, schema.org — [arch §10 Tier 2](docs/architecture.md#tier-2-html-metadata-rules)
- [ ] Add post-extraction heuristics: link density, boilerplate phrase rejection — [arch §11 Step 4](docs/architecture.md#step-4-post-extraction-heuristics)
- [ ] Make minimum word count configurable via `CRAWLER_MIN_WORD_COUNT`

### CLI polish

- [ ] Print richer crawl summary stats
- [ ] Error log to SQLite or `errors.csv` — [arch §22](docs/architecture.md#22-error-handling-strategy)
- [ ] Content-type check: only process `text/html` responses — [arch §25](docs/architecture.md#25-security-and-politeness)
- [ ] Max response body size limit

### Tests (Phase 2)

- [ ] Unit tests: canonical URL extraction
- [ ] Unit tests: content hashing and dedup logic
- [ ] Unit tests: SQLite manifest read/write
- [ ] Integration test: verify dedup across pages with same content

---

## Phase 3 — Daemon MVP

> Goal: queue-driven worker that crawls domains from Redis Streams.
> Ref: [arch §26 Phase 3](docs/architecture.md#phase-3--daemon-mvp), [arch §6](docs/architecture.md#6-queue-abstraction)

### Queue abstraction

- [ ] Define `CrawlJobConsumer` interface — [arch §6](docs/architecture.md#input-consumer)
- [ ] Define `PageResultProducer` interface — [arch §6](docs/architecture.md#output-producer)
- [ ] Implement Redis Streams consumer (consumer groups) — [arch §6](docs/architecture.md#redis-option)
- [ ] Implement Redis Streams producer
- [ ] Define JSON message formats for input/output — [arch §14](docs/architecture.md#14-daemon-queue-message-formats)

### Daemon wiring

- [ ] Implement `QueuePageSink` (wraps `PageResultProducer`)
- [ ] Wire up `cmd/thinkpixelspiderd/main.go`: queue consumer → crawl service → queue sink
- [ ] Emit `CrawlSummary` event on job completion
- [ ] Emit `CrawlError` events for per-page failures

### Logging

- [ ] Structured logging with fields: job_id, domain, url, status, elapsed_ms — [arch §23](docs/architecture.md#23-observability)

### Tests (Phase 3)

- [ ] Unit tests: JSON codec for queue messages
- [ ] Integration test: enqueue job → worker consumes → emits page events + summary — [arch §27](docs/architecture.md#queue-integration-tests)

---

## Phase 4 — Kubernetes readiness

> Goal: container-ready daemon with health checks and metrics.
> Ref: [arch §26 Phase 4](docs/architecture.md#phase-4--k8s-readiness), [arch §15](docs/architecture.md#15-kubernetes-compatibility-design)

### Health & observability

- [ ] Expose `/healthz` and `/readyz` endpoints
- [ ] Expose `/metrics` endpoint (Prometheus format) — [arch §23](docs/architecture.md#metrics)
- [ ] Instrument key counters: jobs_started, jobs_completed, pages_visited, pages_extracted, extraction_failures, crawl_duration_seconds

### Deployment

- [ ] Finalize multi-stage `Dockerfile`
- [ ] Create Helm chart or Kustomize manifests
- [ ] Graceful shutdown on SIGTERM (drain in-flight crawl, close queue connections)
- [ ] Verify all config comes from env vars in daemon mode — [arch §15](docs/architecture.md#k8s-friendly-configuration)

### Tests (Phase 4)

- [ ] Smoke test: deploy one worker + Redis in a dev cluster
- [ ] Verify health endpoints return correct status
- [ ] Verify metrics are populated during a crawl

---

## Phase 5 — NATS JetStream support

> Goal: alternative queue backend using NATS.
> Ref: [arch §26 Phase 5](docs/architecture.md#phase-5--nats-support), [arch §6](docs/architecture.md#nats-option)

- [ ] Implement NATS JetStream consumer (same `CrawlJobConsumer` interface)
- [ ] Implement NATS JetStream producer (same `PageResultProducer` interface)
- [ ] Add `QUEUE_BACKEND=nats` config path
- [ ] Integration test: same enqueue → consume → emit flow over NATS

---

## Phase 6 — Distributed improvements

> Goal: multi-worker robustness and coordination.
> Ref: [arch §26 Phase 6](docs/architecture.md#phase-6--distributed-improvements)

- [ ] Implement Redis-backed Colly storage for cross-worker visited-URL tracking — [arch §7](docs/architecture.md#option-2-redis)
- [ ] Add `COLLY_STORAGE=redis` config path
- [ ] Implement job leases / retry semantics in queue consumer
- [ ] Implement dead-letter handling for failed jobs
- [ ] Stronger cross-worker content dedup (Redis-backed content hash set)

---

## Future / backlog

> Not planned for initial phases. Tracked here for reference.
> See also: [docs/preamble.md — Long-Term Direction](docs/preamble.md#long-term-direction)

- [ ] DOM differencing extraction (compare multiple pages to isolate templates)
- [ ] RSS/feed-based discovery mode
- [ ] Resumable CLI crawls (persist crawl frontier to SQLite)
- [ ] Subdomain crawling support (`CRAWLER_INCLUDE_SUBDOMAINS`)
- [ ] Distributed tracing (OpenTelemetry)
- [ ] Content scoring / quality ranking
- [ ] Multi-backend output sinks (S3, GCS, etc.)
- [ ] Crawl scheduling and orchestration layer
