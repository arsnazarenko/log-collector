package v1

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/logparser"
	"github.com/arsnazarenko/log-collector/internal/logparser/clf"
	"github.com/arsnazarenko/log-collector/internal/logparser/json"
	"github.com/arsnazarenko/log-collector/internal/logparser/syslog"
	"github.com/arsnazarenko/log-collector/internal/metrics"
	"github.com/arsnazarenko/log-collector/internal/usecase"
	"github.com/arsnazarenko/log-collector/internal/util"
)

var _ gen.StrictServerInterface = (*LogServerImpl)(nil)

type LogServerImpl struct {
	logUC usecase.LogUsecase
}

func NewLogServer(uc usecase.LogUsecase) *LogServerImpl {
	return &LogServerImpl{
		logUC: uc,
	}
}

func (l *LogServerImpl) AddLogs(ctx context.Context, request gen.AddLogsRequestObject) (gen.AddLogsResponseObject, error) {
	if request.Body == nil {
		return gen.AddLogs400JSONResponse{
			BadRequestJSONResponse: gen.BadRequestJSONResponse{
				Error: "request body is required",
			},
		}, nil
	}

	count, err := l.logUC.AddLogs(ctx, *request.Body)
	if err != nil {
		metrics.RecordLogsProcessed("http", "error", len(*request.Body))
		details := map[string]any{"error": err.Error()}
		return gen.AddLogs500JSONResponse{
			InternalServerErrorJSONResponse: gen.InternalServerErrorJSONResponse{
				Error:     "failed to add logs",
				Details:   &details,
				Timestamp: util.ByPtr(time.Now()),
			},
		}, nil
	}

	metrics.RecordLogsProcessed("http", "success", len(*request.Body))
	message := "logs added successfully"
	return gen.AddLogs201JSONResponse{
		Message:       &message,
		InsertedCount: &count,
	}, nil
}

func errTo500Response(err error, fmtStr string, v ...any) gen.InternalServerErrorJSONResponse {
	details := map[string]any{"error": err.Error()}
	return gen.InternalServerErrorJSONResponse{
		Error:     fmt.Sprintf(fmtStr, v...),
		Details:   &details,
		Timestamp: util.ByPtr(time.Now()),
	}
}

func (l *LogServerImpl) SearchLogs(ctx context.Context, request gen.SearchLogsRequestObject) (gen.SearchLogsResponseObject, error) {
	result, err := l.logUC.SearchLogs(ctx, request.Params)
	if err != nil {
		return gen.SearchLogs500JSONResponse{
			InternalServerErrorJSONResponse: errTo500Response(err, "failed to search logs"),
		}, nil
	}

	return gen.SearchLogs200JSONResponse(*result), nil
}

func getParserByExtension(filename string) logparser.Parser {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		return json.NewJSONParser()
	case ".log":
		return syslog.NewSyslogParser()
	case ".txt":
		return clf.NewCLFParser()
	default:
		return nil
	}
}

func tryParseWithFallback(ctx context.Context, file io.Reader) ([]gen.LogEntryInput, error) {
	parsers := []struct {
		name   string
		parser logparser.Parser
	}{
		{"json", json.NewJSONParser()},
		{"syslog", syslog.NewSyslogParser()},
		{"clf", clf.NewCLFParser()},
	}

	for _, p := range parsers {
		if seeker, ok := file.(io.ReadSeeker); ok {
			seeker.Seek(0, io.SeekStart)
		}

		observe := metrics.ObserveLogParsing(p.name, "http_upload", 0)
		logs, err := p.parser.Parse(ctx, file)
		observe()

		if err == nil && len(logs) > 0 {
			return logs, nil
		}
	}

	return nil, logparser.ErrUnsupportedFormat
}

func (l *LogServerImpl) UploadLogs(ctx context.Context, request gen.UploadLogsRequestObject) (gen.UploadLogsResponseObject, error) {
	if request.Body == nil {
		return gen.UploadLogs400JSONResponse{
			BadRequestJSONResponse: gen.BadRequestJSONResponse{
				Error: "request body is required",
			},
		}, nil
	}

	part, err := request.Body.NextPart()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return gen.UploadLogs400JSONResponse{
				BadRequestJSONResponse: gen.BadRequestJSONResponse{
					Error: "file is required",
				},
			}, nil
		}
		return gen.UploadLogs500JSONResponse{
			InternalServerErrorJSONResponse: errTo500Response(err, "failed to read multipart body"),
		}, nil
	}

	filename := part.FileName()

	var content bytes.Buffer
	if _, err = io.Copy(&content, part); err != nil {
		return gen.UploadLogs500JSONResponse{
			InternalServerErrorJSONResponse: errTo500Response(err, "failed to read file content"),
		}, nil
	}

	var logs []gen.LogEntryInput
	if filename != "" {
		parser := getParserByExtension(filename)
		if parser != nil {
			observe := metrics.ObserveLogParsing(parser.Name(), "http_upload", 0)
			logs, err = parser.Parse(ctx, bytes.NewReader(content.Bytes()))
			observe()

			if err != nil {
				return gen.UploadLogs500JSONResponse{
					InternalServerErrorJSONResponse: errTo500Response(err, "failed to parse log file"),
				}, nil
			} else if len(logs) > 0 {
				count, ucErr := l.logUC.AddLogs(ctx, logs)
				if ucErr != nil {
					metrics.RecordLogsProcessed("http_upload", "error", len(logs))
					return gen.UploadLogs500JSONResponse{
						InternalServerErrorJSONResponse: errTo500Response(ucErr, "failed to save logs"),
					}, nil
				}
				metrics.RecordLogsProcessed("http_upload", "success", len(logs))
				message := "logs uploaded successfully"
				return gen.UploadLogs201JSONResponse{
					Message:       &message,
					InsertedCount: &count,
				}, nil
			}
		}
	}

	logs, err = tryParseWithFallback(ctx, bytes.NewReader(content.Bytes()))
	if err != nil {
		return gen.UploadLogs500JSONResponse{
			InternalServerErrorJSONResponse: errTo500Response(err, "failed to parse log file"),
		}, nil
	}

	count, err := l.logUC.AddLogs(ctx, logs)
	if err != nil {
		metrics.RecordLogsProcessed("http_upload", "error", len(logs))
		return gen.UploadLogs500JSONResponse{
			InternalServerErrorJSONResponse: errTo500Response(err, "failed to save logs"),
		}, nil
	}

	metrics.RecordLogsProcessed("http_upload", "success", len(logs))
	message := "logs uploaded successfully"
	return gen.UploadLogs201JSONResponse{
		Message:       &message,
		InsertedCount: &count,
	}, nil
}
