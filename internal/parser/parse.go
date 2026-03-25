package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/IndependentImpact/ttl2d3/internal/config"
)

// Parse reads from r and parses it as the RDF serialisation identified by
// format.  When format is [config.InputAuto] the format is resolved
// automatically:
//
//  1. If filename is not "-" (stdin), the file extension is tried first via
//     [DetectFormat].
//  2. If the extension is unrecognised (or filename is "-"), up to
//     [sniffSize] bytes are read from r and [SniffFormat] is called.
//     The consumed bytes are prepended back to the reader before the
//     underlying parser is called.
//
// Parse returns a non-nil *[Graph] on success and a descriptive error
// otherwise.
func Parse(r io.Reader, filename, baseIRI string, format config.InputFormat) (*Graph, error) {
	if format == config.InputAuto {
		if filename != "-" {
			if detected, err := DetectFormat(filename); err == nil {
				format = detected
			}
		}

		// Fall back to content sniffing when the format is still unknown.
		if format == config.InputAuto {
			var buf bytes.Buffer
			if _, err := io.CopyN(&buf, r, sniffSize); err != nil && !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("parser: reading input for format sniffing: %w", err)
			}
			detected, err := SniffFormat(buf.Bytes())
			if err != nil {
				return nil, err
			}
			format = detected
			// Restore the bytes that were consumed during sniffing.
			r = io.MultiReader(&buf, r)
		}
	}

	switch format {
	case config.InputTurtle:
		return ParseTurtle(r, baseIRI)
	case config.InputRDFXML:
		return ParseRDFXML(r, baseIRI)
	case config.InputJSONLD:
		return ParseJSONLD(r, baseIRI)
	default:
		return nil, fmt.Errorf("parser: unknown format %q", format)
	}
}
