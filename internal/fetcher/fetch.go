// Package fetcher provides HTTP retrieval for remote ontology resources.
// It handles content negotiation, format detection from HTTP response headers,
// request timeouts, and response body size limiting.
package fetcher

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/IndependentImpact/ttl2d3/internal/config"
	"github.com/IndependentImpact/ttl2d3/internal/parser"
)

// DefaultTimeout is applied to the full HTTP round-trip (connect + headers +
// body) when no deadline is present on the supplied context.
const DefaultTimeout = 30 * time.Second

// MaxBodyBytes is the maximum number of bytes read from a single HTTP response
// body.  Responses larger than this limit are rejected with an error.
const MaxBodyBytes = 50 * 1024 * 1024 // 50 MiB

// defaultClient is used for all outgoing HTTP requests.  Its Timeout field
// covers the full request lifecycle including reading the response body up to
// MaxBodyBytes, so the caller's context is not canceled prematurely.
var defaultClient = &http.Client{
	Timeout: DefaultTimeout,
}

// IsURL reports whether s is an HTTP or HTTPS URL.
func IsURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// acceptHeader returns the HTTP Accept header value appropriate for the given
// format hint.  When hint is config.InputAuto a preference-ordered list of all
// supported RDF MIME types is returned so that servers with content negotiation
// return a machine-readable format rather than an HTML page.
func acceptHeader(hint config.InputFormat) string {
	switch hint {
	case config.InputTurtle:
		return "text/turtle, application/x-turtle;q=0.9, */*;q=0.1"
	case config.InputRDFXML:
		return "application/rdf+xml, application/xml;q=0.9, text/xml;q=0.8, */*;q=0.1"
	case config.InputJSONLD:
		return "application/ld+json, application/json;q=0.9, */*;q=0.1"
	default: // InputAuto
		return "text/turtle, application/ld+json;q=0.9, application/rdf+xml;q=0.8, application/xml;q=0.7, */*;q=0.1"
	}
}

// Fetch retrieves the resource at url and returns its content as a
// [io.ReadCloser], the resolved [config.InputFormat], and any error.
//
// The caller is responsible for closing the returned ReadCloser when
// non-nil.
//
// Format resolution order:
//  1. If hint is not [config.InputAuto] it is returned as-is (the Accept
//     header is still sent so the server returns an appropriate format).
//  2. The HTTP Content-Type response header is inspected via
//     [parser.FormatFromContentType].
//  3. If the Content-Type is absent or unrecognised, the URL path extension
//     is tried via [parser.DetectFormat].
//  4. If all of the above fail, [config.InputAuto] is returned; the caller
//     may then apply byte-sniffing via [parser.Parse].
//
// A [DefaultTimeout] is applied to the full HTTP round-trip when ctx has no
// deadline.  Responses larger than [MaxBodyBytes] are rejected with an error
// returned from the body reader, not from Fetch itself.
func Fetch(ctx context.Context, url string, hint config.InputFormat) (io.ReadCloser, config.InputFormat, error) {
	if !IsURL(url) {
		// Guard against accidental non-HTTP schemes.
		scheme := url
		if idx := strings.Index(url, "://"); idx >= 0 {
			scheme = url[:idx]
		}
		return nil, config.InputAuto, fmt.Errorf("fetcher: unsupported URL scheme %q", scheme)
	}

	// Choose the HTTP client: use defaultClient (which has DefaultTimeout) when
	// the caller's context has no deadline; otherwise build a short-lived client
	// matching the context deadline so the caller's cancellation is honoured.
	client := defaultClient
	if _, ok := ctx.Deadline(); ok {
		client = &http.Client{} // no Timeout; the context carries the deadline
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, config.InputAuto, fmt.Errorf("fetcher: building request for %s: %w", url, err)
	}
	req.Header.Set("Accept", acceptHeader(hint))
	req.Header.Set("User-Agent", "ttl2d3/1 (https://github.com/IndependentImpact/ttl2d3)")

	resp, err := client.Do(req) //nolint:bodyclose // body is returned to the caller
	if err != nil {
		return nil, config.InputAuto, fmt.Errorf("fetcher: HTTP GET %s: %w", url, err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, config.InputAuto, fmt.Errorf("fetcher: HTTP GET %s: unexpected status %s", url, resp.Status)
	}

	// Resolve output format.
	resolved := hint
	if resolved == config.InputAuto {
		resolved = parser.FormatFromContentType(resp.Header.Get("Content-Type"))
		if resolved == config.InputAuto {
			// Fall back to URL path extension; ignore errors from DetectFormat.
			if detected, detectErr := parser.DetectFormat(url); detectErr == nil {
				resolved = detected
			}
		}
	}

	// Wrap the body with a size limiter to prevent OOM on huge responses.
	limited := &limitedReadCloser{
		r:     io.LimitReader(resp.Body, MaxBodyBytes+1),
		c:     resp.Body,
		limit: MaxBodyBytes,
		read:  0,
	}
	return limited, resolved, nil
}

// limitedReadCloser wraps an io.ReadCloser and returns an error when the
// total number of bytes read would exceed limit.
type limitedReadCloser struct {
	r     io.Reader // LimitReader wrapping the real body
	c     io.Closer // real response body (for Close)
	limit int64     // maximum bytes allowed
	read  int64     // bytes read so far
}

func (l *limitedReadCloser) Read(p []byte) (int, error) {
	n, err := l.r.Read(p)
	l.read += int64(n)
	if l.read > l.limit {
		_ = l.c.Close()
		return n, fmt.Errorf("fetcher: response body exceeds %d MiB limit", MaxBodyBytes/(1024*1024))
	}
	return n, err
}

func (l *limitedReadCloser) Close() error {
	return l.c.Close()
}
