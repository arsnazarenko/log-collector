package clf

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/logparser"
)

var _ logparser.Parser = (*CLFParser)(nil)

const (
	clfPattern    = `^(\S+) \S+ \S+ \[([^\]]+)\] "(\S+ \S+ \S+)" (\d{3}) (\d+|-)$`
	defaultSource = ""
)

var clfRegex = regexp.MustCompile(clfPattern)

type CLFParser struct{}

func NewCLFParser() *CLFParser {
	return &CLFParser{}
}

func (c *CLFParser) Name() string {
	return "clf"
}

func (c *CLFParser) statusToLevel(statusCode uint16) gen.LogEntryInputLevel {
	switch {
	case statusCode >= 500:
		return gen.LogEntryInputLevelError
	case statusCode >= 400:
		return gen.LogEntryInputLevelWarning
	default:
		return gen.LogEntryInputLevelInfo
	}
}

func (c *CLFParser) parseCLFTimestamp(ts string) (time.Time, error) {
	layout := "02/Jan/2006:15:04:05 -0700"
	return time.Parse(layout, ts)
}

func (c *CLFParser) ParseItem(line string) (gen.LogEntryInput, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return gen.LogEntryInput{}, logparser.ErrEmptyItem
	}

	matches := clfRegex.FindStringSubmatch(line)
	if len(matches) != 6 {
		return gen.LogEntryInput{}, fmt.Errorf("%w: invalid CLF format", logparser.ErrInvalidLogFormat)
	}

	timestamp, err := c.parseCLFTimestamp(matches[2])
	if err != nil {
		return gen.LogEntryInput{}, fmt.Errorf("%w: invalid timestamp: %v", logparser.ErrInvalidLogFormat, err)
	}

	statusCode, err := strconv.ParseUint(matches[4], 10, 16)
	if err != nil {
		return gen.LogEntryInput{}, fmt.Errorf("%w: invalid status code: %v", logparser.ErrInvalidLogFormat, err)
	}
	host, message := matches[1], matches[3]

	if host == "-" {
		host = ""
	}
	if message == "-" {
		message = ""
	}

	statusCodePtr := uint16(statusCode)
	log := gen.LogEntryInput{
		CreatedAt:   timestamp,
		Host:        host,
		Message:     message,
		Source:      defaultSource,
		Environment: gen.LogEntryInputEnvironment(logparser.DefaultEnvironment),
		Level:       c.statusToLevel(statusCodePtr),
		Payload: &gen.LogPayload{
			HttpStatusCode: &statusCodePtr,
		},
	}

	return log, nil
}

func (c *CLFParser) Parse(ctx context.Context, r io.Reader) ([]gen.LogEntryInput, error) {
	scanner := bufio.NewScanner(r)
	var logs []gen.LogEntryInput
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		log, err := c.ParseItem(line)
		if err != nil {
			if errors.Is(err, logparser.ErrEmptyItem) {
				continue
			} else {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
		}
		logs = append(logs, log)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", logparser.ErrInvalidLogFormat, err)
	}

	return logs, nil
}
