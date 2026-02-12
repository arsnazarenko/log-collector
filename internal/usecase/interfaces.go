package usecase

import (
	"context"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
)

type LogUsecase interface {
	AddLogs(ctx context.Context, logs []gen.LogEntryInput) (int, error)
	AddLog(ctx context.Context, log gen.LogEntryInput) error
	SearchLogs(ctx context.Context, params gen.SearchLogsParams) (*gen.SearchResult, error)
}
