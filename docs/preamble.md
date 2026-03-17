# ThinkPixelSpider Preamble

## Purpose

ThinkPixelSpider is a focused web crawling and article extraction project designed to support semantic search, content indexing, and downstream content-processing pipelines.

The project is intended to crawl websites that primarily publish article-like pages, extract their meaningful textual content, normalize that content into Markdown, and emit the resulting data in formats suitable for storage, indexing, or queue-based processing.

Its main target is the class of websites built with **WordPress** and similar CMS platforms, where article pages usually share a common structure and can be discovered and processed systematically.

ThinkPixelSpider is not intended to be a general-purpose browser automation platform or a universal scraper for arbitrary web applications. Its purpose is narrower and more deliberate: to identify and extract **readable, indexable, article-focused content** efficiently and consistently.

---

## Motivation

Modern websites often contain large amounts of non-essential HTML surrounding the actual content of interest.

A typical article page includes:

- navigation menus
- headers and footers
- sidebars
- social widgets
- ads
- subscription prompts
- related content sections
- comments
- tracking and metadata blocks

For semantic indexing and search applications, these surrounding elements create noise. The actual content of interest is usually only a subset of the page.

ThinkPixelSpider exists to solve this problem by creating a repeatable pipeline that:

1. discovers relevant pages
2. extracts the core article content
3. converts that content into a clean textual format
4. outputs it in ways that are useful for indexing or downstream computation

This project is especially motivated by the need to ingest website content into semantic search systems where extraction quality matters as much as crawl coverage.

---

## Core Problem

A crawler that simply downloads HTML is not enough for content indexing.

Raw HTML includes both useful and useless material. If ingested directly, that material reduces the quality of:

- embeddings
- search relevance
- summarization
- clustering
- retrieval
- topic analysis

Therefore, the crawler must do more than collect pages. It must also identify and preserve the page’s main content while removing boilerplate.

The challenge is to do this in a way that is:

- practical
- fast
- explainable
- configurable
- compatible with both local and distributed execution

---

## Project Thesis

The central idea behind ThinkPixelSpider is that article extraction should be approached as a layered problem.

Rather than relying on one single method, the project adopts the following strategy:

1. **Readability first**
2. **Template heuristics second**
3. **DOM differencing third**

This ordering reflects a balance between implementation speed, extraction quality, and future extensibility.

### Readability first

The first extraction layer is a readability-based parser. This provides a strong baseline for CMS-style article pages and solves the majority of the extraction problem without requiring custom site logic.

### Template heuristics second

After readability extraction, additional heuristics can be applied to validate or reject low-quality results. These heuristics can identify common failure cases such as short pages, heavily boilerplate-dominated pages, archive pages, or pages where unrelated content leaks into the extraction output.

### DOM differencing third

For more advanced use cases, ThinkPixelSpider may later support comparing multiple pages from the same site to identify repeating structural elements and isolate page-specific content. This is especially relevant for templated CMS sites such as WordPress.

This more advanced stage is not required for the first version, but it is part of the project’s long-term conceptual direction.

---

## Why WordPress First

ThinkPixelSpider is intentionally optimized first for **WordPress and similar CMS-generated sites**.

This is a good starting point because such sites usually have:

- consistent page templates
- predictable article URL patterns
- sitemap support
- server-rendered HTML
- recognizable metadata
- common article containers and CSS conventions

These characteristics make them highly suitable for:

- focused crawling
- article detection
- readability-based extraction
- future template-aware enhancements

WordPress should not be seen as a limitation, but rather as a practical and strategically useful initial target.

---

## Primary Goals

The primary goals of ThinkPixelSpider are the following.

### 1. Crawl websites safely and predictably

The crawler must remain constrained to the intended site and avoid uncontrolled traversal.

It should:

- respect configured domain boundaries
- avoid revisiting the same page
- support crawl limits such as depth and maximum page count
- optionally respect robots and polite crawling controls

### 2. Discover article-like pages efficiently

The crawler should find useful pages with a bias toward structured discovery methods such as:

- sitemap parsing
- internal link traversal
- optional feed-based discovery

It should prioritize article-rich discovery paths and avoid wasting crawl budget on low-value pages where possible.

### 3. Extract readable content

The crawler must produce content that is suitable for indexing and analysis.

That means it should attempt to isolate:

- title
- byline when available
- canonical URL
- main body content
- site metadata when useful

while minimizing contamination from repetitive page furniture.

### 4. Convert extracted content to Markdown

Markdown is chosen as a clean and portable representation of extracted content.

It is easier to inspect, version, store, index, and transform than raw HTML, while still preserving document structure such as:

- headings
- paragraphs
- lists
- links
- block quotes
- basic formatting

### 5. Support both local and distributed execution

ThinkPixelSpider should function in two major operational modes:

- as a **CLI tool** for direct crawling to local output
- as a **daemon/worker** for queue-based distributed processing

This ensures the project is useful both as a development tool and as part of larger ingestion systems.

### 6. Remain composable and infrastructure-friendly

The project should integrate cleanly with modern infrastructure patterns, including:

- environment-based configuration
- containers
- Kubernetes
- Redis
- NATS
- downstream indexing pipelines

---

## Secondary Goals

In addition to the core functionality, ThinkPixelSpider should aim to provide:

- deterministic URL normalization
- deduplication support
- canonical URL handling
- output manifest generation
- storage flexibility
- queue abstraction
- future extensibility for smarter extraction methods

These are important because real-world crawling often becomes difficult not at the fetching layer, but at the coordination and data-quality layer.

---

## Non-Goals

To keep the project focused, several things are deliberately out of scope for the early versions.

### 1. Full browser rendering

ThinkPixelSpider is not intended, initially, to behave like a headless browser platform for heavily JavaScript-dependent websites.

The first versions target sites whose content is accessible through ordinary HTML responses.

### 2. Arbitrary scraping for all page types

The project is not trying to scrape every type of structured web data. It is focused on article-like content rather than product catalogs, dashboards, web apps, or highly custom interactive pages.

### 3. Internet-scale general crawling

ThinkPixelSpider is a focused crawler, not a search-engine-scale web frontier manager.

It should be able to scale horizontally for many domain jobs, but that is different from attempting unrestricted crawling of the public web.

### 4. Anti-bot evasion

The project is not intended to compete with stealth browser tools or anti-bot bypass systems.

### 5. Perfect extraction for every site

No single extraction method will work perfectly on every site. The goal is high-quality extraction for the intended target class, not universal perfection.

---

## Execution Model

ThinkPixelSpider is designed around a single core crawling and extraction pipeline with multiple runtime modes.

### CLI mode

In CLI mode, a user provides:

- a domain
- an output directory
- an output manifest format

The crawler then:

- discovers pages
- extracts readable content
- converts content to Markdown
- writes Markdown files locally
- writes a manifest mapping URLs to relative file paths and metadata

This mode is intended for:

- local testing
- dataset generation
- content export
- search demos
- offline processing

### Daemon mode

In daemon mode, the application runs as a worker that receives crawl jobs from an input queue and emits extracted page results to an output queue.

This mode is intended for:

- distributed crawling
- event-driven ingestion
- containerized deployments
- Kubernetes-based worker pools

The same extraction pipeline should be shared between both modes.

---

## Design Principles

The implementation of ThinkPixelSpider should follow several design principles.

### Focus over generality

It is better to be very good at article-oriented crawling than superficially flexible across every possible website type.

### Simplicity first

The first implementation should use well-understood components and a clean architecture. Advanced behaviors should be added only where they clearly improve the system.

### Shared core, multiple outputs

CLI and daemon modes should share the same crawl and extraction logic. The difference between them should mostly be the way jobs enter the system and the way results leave it.

### Configurability without fragmentation

The project should be configurable through environment variables so that it can run cleanly in local environments, Docker, and Kubernetes.

### Deterministic outputs

The same page should produce stable identifiers, stable normalized URLs, and predictable file paths wherever possible.

### Extensibility

The architecture should support future enhancements such as:

- stronger site-specific heuristics
- better article detection
- template-aware extraction
- distributed deduplication
- alternative output sinks

---

## Planned Technical Direction

The current implementation direction is based on the following stack:

- **Go** for implementation language
- **Colly** for crawling and request orchestration
- **go-readability** for primary readable-content extraction
- **HTML-to-Markdown conversion** for normalized output
- **CSV or SQLite** for local manifests
- **Redis or NATS** for queue-based distributed workflows
- **in-memory or Redis-backed Colly storage** for crawl-state tracking

This stack was chosen because it provides a practical balance of:

- performance
- implementation speed
- operational simplicity
- deployment friendliness
- ecosystem maturity

---

## Planned Extraction Flow

The intended extraction flow is:

1. discover a candidate URL
2. normalize and validate the URL
3. verify that it belongs to the allowed crawl scope
4. fetch the HTML page
5. verify that the response is processable HTML
6. run readability-based extraction
7. apply post-extraction quality heuristics
8. convert extracted HTML to Markdown
9. normalize the Markdown
10. emit the result to a local sink or queue sink

This flow is designed so that extraction remains modular and improvements can be inserted between stages without changing the entire application.

---

## Discovery Strategy

ThinkPixelSpider should prefer structured and high-value page discovery strategies.

### Sitemap-first approach

Sitemaps are often the cleanest and most efficient source of URLs on WordPress and similar sites. They frequently contain direct links to articles and reduce the amount of noisy traversal needed.

### Link discovery second

When crawling pages directly, internal links should be inspected and filtered carefully. Not every internal link should be visited automatically; article-oriented URL heuristics should help guide traversal.

### Feed support as optional

RSS or feed discovery can be useful for recent-content workflows, but it is not a substitute for full site discovery.

---

## Output Philosophy

The project’s output should be designed for downstream use, not only for raw archival.

This means the output should be:

- structured
- clean
- traceable back to the source URL
- easy to inspect manually
- easy to store and reprocess automatically

Markdown is the preferred portable representation of extracted content, while manifest data provides the mapping between:

- source URL
- canonical URL
- local file path or sink identity
- metadata
- extraction status

In daemon mode, queue messages should provide equivalent information in structured event form.

---

## Distributed Operation Philosophy

In distributed environments, ThinkPixelSpider should behave as a stateless worker wherever possible.

This means:

- jobs arrive through a queue
- workers process one job or crawl scope at a time
- results are emitted downstream
- persistent coordination is delegated to queue systems or external stores where needed

This design is well suited to Kubernetes because it enables:

- horizontal scaling
- failure recovery
- independent worker replication
- infrastructure-level orchestration

---

## Why Markdown

Markdown is used not because it is the only valid output format, but because it offers a strong balance between structure and simplicity.

Compared with raw HTML, Markdown is:

- cleaner
- easier to diff
- easier to inspect
- easier to feed into text-processing pipelines
- easier to store in versioned repositories
- better suited to human debugging

For semantic search use cases, Markdown also gives a high-quality textual representation while preserving useful document structure.

---

## Long-Term Direction

The first versions of ThinkPixelSpider should focus on being reliable, understandable, and useful.

Longer-term evolution may include:

- stronger extraction heuristics
- template comparison techniques
- richer metadata extraction
- content scoring
- better duplicate-content detection
- multi-backend storage and sink support
- crawl observability and metrics
- more advanced scheduling and crawl orchestration

The project should evolve from a practical crawler into a robust ingestion component for content intelligence workflows, while preserving the simplicity of its original purpose.

---

## Summary

ThinkPixelSpider is a focused article-crawling and extraction system intended to transform website content into clean, Markdown-based outputs that are ready for semantic indexing and search.

Its core idea is straightforward:

- crawl only what matters
- extract only what is readable
- output only what is useful

It is designed to start simple, operate efficiently, and remain extensible as the project grows in sophistication.