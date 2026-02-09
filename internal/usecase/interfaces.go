package usecase

import (
	"context"
	"io"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
)

type LogUsecase interface {
	AddLogs(ctx context.Context, logs []gen.LogEntryInput) (int, error)
	SearchLogs(ctx context.Context, params gen.SearchLogsParams) (*gen.SearchResult, error)
	UploadLogs(ctx context.Context, file io.Reader, filename string) (int, error)
}
