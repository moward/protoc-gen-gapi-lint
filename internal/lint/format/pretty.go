package format

import (
	"fmt"
	"io"

	"github.com/protoc-extensions/protoc-gen-gapi-lint/internal/lint"
)

// PrettyEncoder implements the Encoder interface for pretty printing lint responses.
type PrettyEncoder struct {
	writer io.Writer
}

// NewPrettyEncoder creates a new PrettyEncoder.
func NewPrettyEncoder(writer io.Writer) *PrettyEncoder {
	return &PrettyEncoder{writer: writer}
}

// Encode encodes the given []lint.Response in a pretty format.
func (p *PrettyEncoder) Encode(v interface{}) error {
	responses, ok := v.([]lint.Response)
	if !ok {
		return fmt.Errorf("unsupported type: expected []lint.Response, got %T", v)
	}

	for _, response := range responses {
		for _, problem := range response.Problems {
			loc := problem.Location
			if loc == nil && problem.Descriptor != nil {
				loc = problem.Descriptor.GetSourceInfo()
			}

			// Format: file:line:column - message (ruleID|ruleURI)
			if _, err := fmt.Fprintf(p.writer,
				"%s:%d:%d - %s (%s%s%s%s%s)\n",
				response.FilePath,
				loc.Span[0]+1, // start line
				loc.Span[1]+1, // start column
				problem.Message,
				"\u001b]8;;", // OSC 8 hyperlinks might not work in all terminals; consider alternatives if needed.
				problem.GetRuleURI(),
				"\u0007",
				problem.RuleID,
				"\u001b]8;;\u0007", // Closing OSC 8 sequence.
			); err != nil {
				return err
			}
		}
	}

	return nil
}
