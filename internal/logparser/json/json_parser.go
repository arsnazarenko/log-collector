package json

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/logparser"
)

var _ logparser.Parser = (*JSONParser)(nil)

type JSONParser struct{}

func NewJSONParser() *JSONParser {
	return &JSONParser{}
}

func (j *JSONParser) Name() string {
	return "json"
}

func validateJSONLogEntry(log gen.LogEntryInput) error {
	if log.CreatedAt.IsZero() {
		return fmt.Errorf("%w: created_at is required", logparser.ErrRequiredFieldMissing)
	}
	if log.Host == "" {
		return fmt.Errorf("%w: host is required", logparser.ErrRequiredFieldMissing)
	}
	if log.Message == "" {
		return fmt.Errorf("%w: message is required", logparser.ErrRequiredFieldMissing)
	}
	if log.Source == "" {
		return fmt.Errorf("%w: source is required", logparser.ErrRequiredFieldMissing)
	}

	validLevels := map[gen.LogEntryInputLevel]bool{
		gen.LogEntryInputLevelCritical: true,
		gen.LogEntryInputLevelDebug:    true,
		gen.LogEntryInputLevelError:    true,
		gen.LogEntryInputLevelInfo:     true,
		gen.LogEntryInputLevelWarning:  true,
	}
	if !validLevels[log.Level] {
		return fmt.Errorf("%w: invalid level", logparser.ErrInvalidLogFormat)
	}

	validEnvs := map[gen.LogEntryInputEnvironment]bool{
		gen.LogEntryInputEnvironmentDev:        true,
		gen.LogEntryInputEnvironmentProduction: true,
		gen.LogEntryInputEnvironmentStaging:    true,
	}
	if !validEnvs[log.Environment] {
		return fmt.Errorf("%w: invalid environment", logparser.ErrInvalidLogFormat)
	}
	return nil
}

func (j *JSONParser) ParseItem(item string) (gen.LogEntryInput, error) {
	item = strings.TrimSpace(item)
	if item == "" {
		return gen.LogEntryInput{}, logparser.ErrEmptyItem
	}

	var log gen.LogEntryInput
	if err := json.Unmarshal([]byte(item), &log); err != nil {
		return gen.LogEntryInput{}, fmt.Errorf("%w: %v", logparser.ErrInvalidLogFormat, err)
	}
	if err := validateJSONLogEntry(log); err != nil {
		return gen.LogEntryInput{}, err
	}
	return log, nil
}

func (j *JSONParser) Parse(ctx context.Context, r io.Reader) ([]gen.LogEntryInput, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read input: %v", logparser.ErrInvalidLogFormat, err)
	}

	var entries []gen.LogEntryInput
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("%w: failed to parse JSON array: %v", logparser.ErrInvalidLogFormat, err)
	}

	for i, entry := range entries {
		if err := validateJSONLogEntry(entry); err != nil {
			return nil, fmt.Errorf("invalid json object [%d]: %w", i, err)
		}
	}

	return entries, nil
}
