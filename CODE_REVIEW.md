# ThinkPixelSpider Code Review Tracker

Date: 2026-03-17
Scope: Phase 0 and Phase 1 implementation
Reviewer: GitHub Copilot (GPT-5.3-Codex)

## How to use this file

1. Keep one checkbox per actionable task.
2. Update Status when work starts or is completed.
3. Add a short note under Validation after tests are run.
4. Keep Evidence links updated if code moves.

## Current status snapshot

- High issues: 2
- Medium issues: 3
- Low issues: 2
- Baseline tests: go test ./... passed, go test -race ./... passed, go vet ./... passed

## Findings Index

| ID | Severity | Status | Title |
|---|---|---|---|
| CR-001 | High | Resolved | Panic risk in crawler OnError when response is nil |
| CR-002 | High | Resolved | Manifest type config accepted but runtime always writes CSV |
| CR-003 | Medium | Resolved | Output path currently duplicates pages segment |
| CR-004 | Medium | Resolved | Discovery mode validated case-insensitively but executed case-sensitively |
| CR-005 | Medium | Resolved | Sitemap recursion can fetch external child URLs |
| CR-006 | Low | Resolved | Max pages is best-effort and can overshoot under concurrency |
| CR-007 | Low | Open | IncludeSubdomains config exists but is not used |

---

## CR-001: Panic risk in crawler OnError when response is nil

Severity: High  
Status: Resolved

### Evidence

- [internal/crawler/service.go](internal/crawler/service.go#L117)
- [internal/crawler/service.go](internal/crawler/service.go#L118)

### Why this matters

Transport-level failures can trigger OnError without a response object. Dereferencing response fields without a nil check can panic and terminate the crawl.

### Remediation plan

- [x] Guard response and request pointers before accessing URL.
- [x] Log a safe fallback message when URL is unavailable.
- [x] Add a unit test that simulates OnError with nil response.

### Validation

- [x] go test ./internal/crawler
- [ ] Manual run against an unreachable host should not panic.

Notes:

- Implemented safe URL logging via nil-safe helper in [internal/crawler/service.go](internal/crawler/service.go).
- Added regression tests for nil response, nil request, and normal URL handling in [internal/crawler/service_test.go](internal/crawler/service_test.go).

---

## CR-002: Manifest type config accepted but runtime always writes CSV

Severity: High  
Status: Resolved

### Evidence

- [internal/config/config.go](internal/config/config.go#L280)
- [internal/config/config.go](internal/config/config.go#L281)
- [cmd/thinkpixelspider/main.go](cmd/thinkpixelspider/main.go#L39)
- [cmd/thinkpixelspider/main.go](cmd/thinkpixelspider/main.go#L40)

### Why this matters

Users can request sqlite via config/flags, but runtime behavior silently ignores it and writes CSV, causing contract drift and surprising behavior.

### Remediation plan

- [x] Implement manifest writer selection by config value.
- [x] If sqlite is not implemented yet, fail fast with clear error.
- [x] Add CLI test covering manifest selection behavior.

### Validation

- [x] go test ./cmd/thinkpixelspider
- [ ] Verify --manifest csv creates manifest.csv.
- [x] Verify --manifest sqlite either creates sqlite output or returns explicit not implemented error.

Notes:

- Added manifest writer factory and selection logic in [cmd/thinkpixelspider/main.go](cmd/thinkpixelspider/main.go).
- Added tests for CSV selection and explicit sqlite-not-implemented error in [cmd/thinkpixelspider/main_test.go](cmd/thinkpixelspider/main_test.go).

---

## CR-003: Output path currently duplicates pages segment

Severity: Medium  
Status: Resolved

### Evidence

- [cmd/thinkpixelspider/main.go](cmd/thinkpixelspider/main.go#L47)
- [internal/output/pathgen.go](internal/output/pathgen.go#L40)
- [internal/output/filesystem.go](internal/output/filesystem.go#L36)

### Why this matters

Current joining behavior results in output/pages/pages/... which does not match expected output layout and may break downstream consumers.

### Remediation plan

- [x] Decide canonical path contract (recommended: output/pages/domain/... ).
- [x] Make path generator and sink agree on one relative-root strategy.
- [ ] Update tests and README examples to the same contract.

### Validation

- [x] go test ./internal/output ./cmd/thinkpixelspider
- [x] Integration test asserts final paths exactly match documented layout.

Notes:

- CLI sink base directory now uses output root in [cmd/thinkpixelspider/main.go](cmd/thinkpixelspider/main.go), preventing output/pages/pages/... duplication.
- Integration assertions now validate output/pages/<domain>/... paths in [cmd/thinkpixelspider/integration_test.go](cmd/thinkpixelspider/integration_test.go).

---

## CR-004: Discovery mode validated case-insensitively but executed case-sensitively

Severity: Medium  
Status: Resolved

### Evidence

- [internal/config/config.go](internal/config/config.go#L275)
- [internal/config/config.go](internal/config/config.go#L276)
- [internal/crawler/service.go](internal/crawler/service.go#L124)
- [internal/crawler/service.go](internal/crawler/service.go#L127)
- [internal/crawler/service.go](internal/crawler/service.go#L150)

### Why this matters

Values like BOTH may pass validation but skip discovery branches at runtime due to strict lowercase comparisons.

### Remediation plan

- [x] Normalize discovery mode to lowercase during config load.
- [x] Keep runtime comparisons against normalized value.
- [x] Add tests for case-variant values from env and flags.

### Validation

- [x] go test ./internal/config ./internal/crawler
- [ ] Run with CRAWLER_DISCOVERY_MODE=BOTH and confirm both paths execute.

Notes:

- Discovery mode is now normalized during validation in [internal/config/config.go](internal/config/config.go).
- Crawler runtime now lowercases discovery mode before branching in [internal/crawler/service.go](internal/crawler/service.go).
- Regression tests for mixed-case env and flag values were added in [internal/config/config_test.go](internal/config/config_test.go).

---

## CR-005: Sitemap recursion can fetch external child URLs

Severity: Medium  
Status: Resolved

### Evidence

- [internal/crawler/sitemap.go](internal/crawler/sitemap.go#L87)
- [internal/crawler/sitemap.go](internal/crawler/sitemap.go#L92)

### Why this matters

A sitemap index can point to off-domain locations. Following them blindly can trigger unexpected outbound requests and security concerns.

### Remediation plan

- [x] Restrict child sitemap fetches to allowed host(s) and expected schemes.
- [ ] Optionally add an explicit allowlist override for advanced cases.
- [x] Add tests where sitemap index includes external loc values.

### Validation

- [x] go test ./internal/crawler
- [x] Confirm external loc entries are skipped.

Notes:

- Recursive sitemap child URLs are now resolved and validated against allowed hosts and HTTP(S) schemes in [internal/crawler/sitemap.go](internal/crawler/sitemap.go).
- Added external-child skip regression coverage in [internal/crawler/sitemap_test.go](internal/crawler/sitemap_test.go).

---

## CR-006: Max pages is best-effort and can overshoot under concurrency

Severity: Low  
Status: Resolved

### Evidence

- [internal/crawler/service.go](internal/crawler/service.go#L92)
- [internal/crawler/service.go](internal/crawler/service.go#L96)
- [internal/crawler/service.go](internal/crawler/service.go#L141)
- [internal/crawler/service.go](internal/crawler/service.go#L144)
- [internal/crawler/service.go](internal/crawler/service.go#L155)

### Why this matters

Non-atomic check-then-increment allows multiple goroutines to pass the limit simultaneously. Current behavior is approximate, not strict.

### Remediation plan

- [x] Decide if strict limit is required for Phase 1.
- [x] If strict, use atomic reservation pattern before visit.
- [x] Add deterministic test for no overshoot.

### Validation

- [x] go test ./internal/crawler
- [x] Stress run with concurrent reservation attempts and low max pages.

Notes:

- Added atomic slot reservation helper and integrated it across link discovery, sitemap discovery, and seed scheduling in [internal/crawler/service.go](internal/crawler/service.go).
- Added deterministic concurrency test and stricter max-pages assertion in [internal/crawler/service_test.go](internal/crawler/service_test.go).

---

## CR-007: IncludeSubdomains config exists but is not used

Severity: Low  
Status: Open

### Evidence

- [internal/config/config.go](internal/config/config.go#L38)
- [internal/filters/urlfilter.go](internal/filters/urlfilter.go#L174)

### Why this matters

Config suggests behavior that is not implemented, which can confuse operators and lead to incorrect assumptions.

### Remediation plan

- [ ] Implement include-subdomain expansion in allowed domain logic.
- [ ] Or remove/defer config surface until implementation exists.
- [ ] Add tests for enabled and disabled subdomain behavior.

### Validation

- [ ] go test ./internal/filters ./internal/config
- [ ] Manual crawl test with subdomain links.

---

## Cross-cutting test additions

- [ ] Add a focused regression suite for all closed CR IDs.
- [ ] Add at least one end-to-end CLI scenario that checks config, paths, manifest output, and failure safety.

## Decision log

- 2026-03-17: Initial Phase 0/1 review created with 7 tracked issues.
