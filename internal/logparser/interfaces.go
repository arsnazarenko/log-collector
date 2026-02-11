package logparser

import (
	"context"
	"errors"
	"io"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
)

// parse errors
var (
	ErrUnsupportedFormat    = errors.New("unsupported log format")
	ErrInvalidLogFormat     = errors.New("invalid log format")
	ErrRequiredFieldMissing = errors.New("required field is missing")
	ErrEmptyItem            = errors.New("empty log line/object")
)

// default field values
const (
	DefaultEnvironment = "production"
)

type Parser interface {
	Parse(ctx context.Context, r io.Reader) ([]gen.LogEntryInput, error)

	// Parse one log Entry
	// json: log entry json object
	// clf/syslog: one line of log
	ParseItem(line string) (gen.LogEntryInput, error)

	// json, syslog, CLF
	Name() string
}
