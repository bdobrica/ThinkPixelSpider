// Package output defines the PageSink and ManifestWriter interfaces and
// provides filesystem and CSV implementations for CLI-mode output.
package output

import (
	"context"

	"github.com/bdobrica/ThinkPixelSpider/internal/jobs"
)

// PageSink persists a crawled page. Implementations include FilesystemPageSink
// (CLI mode) and QueuePageSink (daemon mode).
type PageSink interface {
	SavePage(ctx context.Context, page *jobs.CrawledPage) error
}

// ManifestWriter records per-page metadata for later querying or export.
type ManifestWriter interface {
	WriteRow(ctx context.Context, row jobs.ManifestRow) error
	Close() error
}
