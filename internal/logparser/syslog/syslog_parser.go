package syslog

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/logparser"
	"github.com/arsnazarenko/log-collector/internal/util"
	"github.com/leodido/go-syslog/v4"
	"github.com/leodido/go-syslog/v4/rfc5424"
)

var _ logparser.Parser = (*SyslogParser)(nil)

type SyslogParser struct {
	parser syslog.Machine
}

func NewSyslogParser() *SyslogParser {
	parser := rfc5424.NewMachine(rfc5424.WithBestEffort())
	return &SyslogParser{parser: parser}
}

func (s *SyslogParser) Name() string {
	return "syslog"
}

func (s *SyslogParser) severityToLevel(severity uint8) gen.LogEntryInputLevel {
	switch severity {
	case 0, 1, 2:
		return gen.LogEntryInputLevelCritical
	case 3:
		return gen.LogEntryInputLevelError
	case 4:
		return gen.LogEntryInputLevelWarning
	case 5, 6:
		return gen.LogEntryInputLevelInfo
	case 7:
		return gen.LogEntryInputLevelDebug
	default:
		return gen.LogEntryInputLevelInfo
	}
}

func (s *SyslogParser) ParseItem(line string) (gen.LogEntryInput, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return gen.LogEntryInput{}, logparser.ErrEmptyItem
	}

	msg, err := s.parser.Parse([]byte(line))
	if err != nil {
		return gen.LogEntryInput{}, fmt.Errorf("%w: %v", logparser.ErrInvalidLogFormat, err)
	}

	if msg == nil {
		return gen.LogEntryInput{}, fmt.Errorf("%w: empty message", logparser.ErrInvalidLogFormat)
	}

	rfcMsg, ok := msg.(*rfc5424.SyslogMessage)
	if !ok {
		return gen.LogEntryInput{}, fmt.Errorf("%w: unexpected message type", logparser.ErrInvalidLogFormat)
	}

	if rfcMsg.Timestamp == nil || rfcMsg.Timestamp.IsZero() {
		return gen.LogEntryInput{}, fmt.Errorf("%w: timestamp is required", logparser.ErrRequiredFieldMissing)
	}

	severity := util.GetOrDefault(rfcMsg.Severity, uint8(0))

	log := gen.LogEntryInput{
		CreatedAt:   *rfcMsg.Timestamp,
		Host:        util.GetOrDefault(rfcMsg.Hostname, ""),
		Message:     util.GetOrDefault(rfcMsg.Message, ""),
		Source:      util.GetOrDefault(rfcMsg.Appname, ""),
		Environment: gen.LogEntryInputEnvironment(logparser.DefaultEnvironment),
		Level:       s.severityToLevel(severity),
	}

	sd := rfcMsg.StructuredData
	if sd != nil && len(*sd) > 0 {
		var payload gen.LogPayload

		for _, params := range *sd {
			for paramName, paramValue := range params {
				switch paramName {
				case "user_id":
					if val, err := strconv.ParseUint(paramValue, 10, 64); err == nil {
						payload.UserId = &val
					}
				case "duration_ms":
					if val, err := strconv.ParseUint(paramValue, 10, 32); err == nil {
						v := uint32(val)
						payload.DurationMs = &v
					}
				case "http_status_code":
					if val, err := strconv.ParseUint(paramValue, 10, 16); err == nil {
						v := uint16(val)
						payload.HttpStatusCode = &v
					}
				case "error_type":
					payload.ErrorType = &paramValue
				case "stack_trace":
					payload.StackTrace = &paramValue
				}
			}
		}

		if payload.UserId != nil || payload.DurationMs != nil || payload.HttpStatusCode != nil ||
			payload.ErrorType != nil || payload.StackTrace != nil {
			log.Payload = &payload
		}
	}

	return log, nil
}

func (s *SyslogParser) Parse(ctx context.Context, r io.Reader) ([]gen.LogEntryInput, error) {
	scanner := bufio.NewScanner(r)
	var logs []gen.LogEntryInput
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		res, err := s.ParseItem(line)
		if err != nil {
			if errors.Is(err, logparser.ErrEmptyItem) {
				continue
			} else {
				return nil, fmt.Errorf("line %d: %w", lineNum, err)
			}
		}
		logs = append(logs, res)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", logparser.ErrInvalidLogFormat, err)
	}

	return logs, nil
}
