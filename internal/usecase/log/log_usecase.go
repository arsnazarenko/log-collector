package log

import (
	"context"
	"fmt"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/repo"
	"github.com/arsnazarenko/log-collector/internal/repo/persistent"
	"github.com/arsnazarenko/log-collector/internal/usecase"
	"github.com/arsnazarenko/log-collector/internal/util"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

var _ usecase.LogUsecase = (*LogUsecaseImpl)(nil)

type LogUsecaseImpl struct {
	logRepo repo.LogRepo
}

func NewLogUsecase(logRepo repo.LogRepo) *LogUsecaseImpl {
	return &LogUsecaseImpl{
		logRepo: logRepo,
	}
}

// AddLog implements [usecase.LogUsecase].
func (u *LogUsecaseImpl) AddLog(ctx context.Context, log gen.LogEntryInput) error {
	repoLog := persistent.FromAPIInput(log)
	if err := u.logRepo.Save(ctx, *repoLog); err != nil {
		return fmt.Errorf("failed to save log: %w", err)
	}
	return nil
}

func (u *LogUsecaseImpl) AddLogs(ctx context.Context, logs []gen.LogEntryInput) (int, error) {
	if len(logs) == 0 {
		return 0, nil
	}

	repoLogs := make([]repo.LogEntry, 0, len(logs))
	for _, log := range logs {
		repoLog := persistent.FromAPIInput(log)
		repoLogs = append(repoLogs, *repoLog)
	}

	if len(repoLogs) == 1 {
		if err := u.logRepo.Save(ctx, repoLogs[0]); err != nil {
			return 0, fmt.Errorf("failed to save log: %w", err)
		}
	} else {
		if err := u.logRepo.SaveBatch(ctx, repoLogs); err != nil {
			return 0, fmt.Errorf("failed to save logs: %w", err)
		}
	}

	return len(repoLogs), nil
}

func (u *LogUsecaseImpl) SearchLogs(ctx context.Context, params gen.SearchLogsParams) (*gen.SearchResult, error) {
	var level *string
	if params.Level != nil {
		s := string(*params.Level)
		level = &s
	}

	var environment *string
	if params.Environment != nil {
		s := string(*params.Environment)
		environment = &s
	}

	var sortBy *string
	if params.SortBy != nil {
		s := string(*params.SortBy)
		sortBy = &s
	}

	var sortOrder *string
	if params.SortOrder != nil {
		s := string(*params.SortOrder)
		sortOrder = &s
	}

	filter := repo.SearchFilter{
		Level:       level,
		Source:      params.Source,
		Host:        params.Host,
		Environment: environment,
		Message:     params.Message,
		Limit:       util.GetOrDefault(params.Limit, repo.DefaultLimit),
		Offset:      util.GetOrDefault(params.Offset, repo.DefaultOffset),
		From:        params.From,
		To:          params.To,
		SortBy:      sortBy,
		SortOrder:   sortOrder,
	}

	logs, total, err := u.logRepo.Search(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to search logs: %w", err)
	}

	result := &gen.SearchResult{
		Total: total,
		Logs:  make([]gen.LogEntry, 0, len(logs)),
	}

	for _, log := range logs {
		result.Logs = append(result.Logs, u.toGenLogEntry(log))
	}

	return result, nil
}

func (u *LogUsecaseImpl) toGenLogEntry(log repo.LogEntry) gen.LogEntry {
	return gen.LogEntry{
		Id:          types.UUID(uuid.MustParse(log.ID)),
		CreatedAt:   log.CreatedAt,
		ReceivedAt:  log.ReceivedAt,
		Level:       gen.LogEntryLevel(log.Level),
		Source:      log.Source,
		Host:        log.Host,
		Environment: gen.LogEntryEnvironment(log.Environment),
		Message:     log.Message,
		Payload:     u.toGenLogPayload(log),
	}
}

func (u *LogUsecaseImpl) toGenLogPayload(log repo.LogEntry) *gen.LogPayload {
	if log.UserID == nil && log.DurationMs == nil && log.HTTPStatusCode == nil &&
		log.ErrorType == nil && log.StackTrace == nil {
		return nil
	}

	return &gen.LogPayload{
		UserId:         log.UserID,
		DurationMs:     log.DurationMs,
		HttpStatusCode: log.HTTPStatusCode,
		ErrorType:      log.ErrorType,
		StackTrace:     log.StackTrace,
	}
}
