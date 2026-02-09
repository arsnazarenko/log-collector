package persistent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/repo"
	"github.com/arsnazarenko/log-collector/pkg/clickhouse"
)

var _ repo.LogRepo = (*LogClickhouseRepo)(nil)

type LogClickhouseRepo struct {
	ch *clickhouse.Clickhouse
}

func NewLogClickhouseRepo(ch *clickhouse.Clickhouse) *LogClickhouseRepo {
	return &LogClickhouseRepo{ch: ch}
}

func FromAPIInput(input gen.LogEntryInput) *repo.LogEntry {
	receivedAt := time.Now()

	logEntry := &repo.LogEntry{
		CreatedAt:   input.CreatedAt,
		ReceivedAt:  receivedAt,
		Level:       string(input.Level),
		Source:      input.Source,
		Host:        input.Host,
		Environment: string(input.Environment),
		Message:     input.Message,
	}

	if input.Payload != nil {
		logEntry.UserID = input.Payload.UserId
		logEntry.DurationMs = input.Payload.DurationMs
		logEntry.HTTPStatusCode = input.Payload.HttpStatusCode
		logEntry.ErrorType = input.Payload.ErrorType
		logEntry.StackTrace = input.Payload.StackTrace
	}

	return logEntry
}

func (r *LogClickhouseRepo) CreateTable(ctx context.Context) error {
	const createTableQuery = `
CREATE TABLE IF NOT EXISTS logs ON CLUSTER '{cluster}' (
	id UUID DEFAULT generateUUIDv4(),
	created_at DateTime,
	received_at DateTime,
	level LowCardinality(String),
	source String,
	host String,
	environment LowCardinality(String),
	message String,
	user_id Nullable(UInt64),
	duration_ms Nullable(UInt32),
	http_status_code Nullable(UInt16),
	error_type Nullable(String),
	stack_trace Nullable(String)
) ENGINE = ReplicatedMergeTree(
	'/clickhouse/tables/logs',
	'{replica}'
)
ORDER BY (created_at)
PARTITION BY toYYYYMM(created_at);
	`
	if err := r.ch.Conn.Exec(ctx, createTableQuery); err != nil {
		return fmt.Errorf("failed to create logs table: %w", err)
	}
	return nil
}

func (r *LogClickhouseRepo) Save(ctx context.Context, log repo.LogEntry) error {
	const query = `
		INSERT INTO logs (
			created_at, received_at, level, source, host, environment,
			message, user_id, duration_ms, http_status_code, error_type, stack_trace
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	if err := r.ch.Conn.Exec(ctx, query,
		log.CreatedAt, log.ReceivedAt, log.Level, log.Source,
		log.Host, log.Environment, log.Message, log.UserID, log.DurationMs,
		log.HTTPStatusCode, log.ErrorType, log.StackTrace,
	); err != nil {
		return fmt.Errorf("failed to save log entry: %w", err)
	}

	return nil
}

func (r *LogClickhouseRepo) SaveBatch(ctx context.Context, logs []repo.LogEntry) error {
	if len(logs) == 0 {
		return nil
	}

	const query = `
		INSERT INTO logs (
			created_at, received_at, level, source, host, environment,
			message, user_id, duration_ms, http_status_code, error_type, stack_trace
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	batch, err := r.ch.Conn.PrepareBatch(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}
	for _, log := range logs {
		if err := batch.Append(
			log.CreatedAt, log.ReceivedAt, log.Level, log.Source,
			log.Host, log.Environment, log.Message, log.UserID, log.DurationMs,
			log.HTTPStatusCode, log.ErrorType, log.StackTrace,
		); err != nil {
			return fmt.Errorf("failed to append log to batch: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	return nil
}

func (r *LogClickhouseRepo) FindByID(ctx context.Context, id string) (repo.LogEntry, error) {
	const query = `
		SELECT
			id, created_at, received_at, level, source, host, environment,
			message, user_id, duration_ms, http_status_code, error_type, stack_trace
		FROM logs
		WHERE id = ?
		LIMIT 1
	`

	var log repo.LogEntry
	if err := r.ch.Conn.QueryRow(ctx, query, id).Scan(
		&log.ID, &log.CreatedAt, &log.ReceivedAt, &log.Level, &log.Source,
		&log.Host, &log.Environment, &log.Message, &log.UserID, &log.DurationMs,
		&log.HTTPStatusCode, &log.ErrorType, &log.StackTrace,
	); err != nil {
		if err.Error() == "clickhouse: rows: []" {
			return repo.LogEntry{}, repo.ErrLogNotFound
		}
		return repo.LogEntry{}, fmt.Errorf("failed to find log by id: %w", err)
	}

	return log, nil
}

func (r *LogClickhouseRepo) Search(ctx context.Context, filter repo.SearchFilter) ([]repo.LogEntry, int, error) {
	whereClause, args := r.buildWhereClause(filter)
	orderByClause := r.buildOrderByClause(filter)

	const queryFormat = `
		SELECT
			id, created_at, received_at, level, source, host, environment,
			message, user_id, duration_ms, http_status_code, error_type, stack_trace
		FROM logs
		%s
		%s
		LIMIT ?
		OFFSET ?
	`

	query := fmt.Sprintf(queryFormat, whereClause, orderByClause)
	args = append(args, filter.Limit, filter.Offset)

	rows, err := r.ch.Conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search logs: %w", err)
	}
	defer rows.Close()

	var logs []repo.LogEntry
	for rows.Next() {
		var rawValue repo.LogEntry
		if err := rows.Scan(
			&rawValue.ID, &rawValue.CreatedAt, &rawValue.ReceivedAt, &rawValue.Level, &rawValue.Source, &rawValue.Host, &rawValue.Environment,
			&rawValue.Message, &rawValue.UserID, &rawValue.DurationMs, &rawValue.HTTPStatusCode, &rawValue.ErrorType, &rawValue.StackTrace,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan log: %w", err)
		}
		logs = append(logs, rawValue)

		// var (
		// 	id             string
		// 	createdAt      time.Time
		// 	receivedAt     time.Time
		// 	level          string
		// 	source         string
		// 	host           string
		// 	environment    string
		// 	message        string
		// 	userID         *uint64
		// 	durationMs     *uint32
		// 	httpStatusCode *uint16
		// 	errorType      *string
		// 	stackTrace     *string
		// )
		// if err := rows.Scan(
		// 	&id, &createdAt, &receivedAt, &level, &source, &host, &environment,
		// 	&message, &userID, &durationMs, &httpStatusCode, &errorType, &stackTrace,
		// ); err != nil {
		// 	return nil, 0, fmt.Errorf("failed to scan log: %w", err)
		// }
		// logs = append(logs, repo.LogEntry{
		// 	ID:             id,
		// 	CreatedAt:      createdAt,
		// 	ReceivedAt:     receivedAt,
		// 	Level:          level,
		// 	Source:         source,
		// 	Host:           host,
		// 	Environment:    environment,
		// 	Message:        message,
		// 	UserID:         userID,
		// 	DurationMs:     durationMs,
		// 	HTTPStatusCode: httpStatusCode,
		// 	ErrorType:      errorType,
		// 	StackTrace:     stackTrace,
		// })
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate rows: %w", err)
	}

	countQuery := fmt.Sprintf("SELECT count() FROM logs %s", whereClause)
	var total uint64
	if err := r.ch.Conn.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&total); err != nil {
		return logs, 0, fmt.Errorf("failed to count logs: %w", err)
	}

	return logs, int(total), nil
}

func (r *LogClickhouseRepo) buildWhereClause(filter repo.SearchFilter) (string, []any) {
	conditions := []string{}
	args := []any{}

	if filter.Level != nil {
		conditions = append(conditions, "level = ?")
		args = append(args, *filter.Level)
	}

	if filter.Source != nil {
		conditions = append(conditions, "source = ?")
		args = append(args, *filter.Source)
	}

	if filter.Host != nil {
		conditions = append(conditions, "host = ?")
		args = append(args, *filter.Host)
	}

	if filter.Environment != nil {
		conditions = append(conditions, "environment = ?")
		args = append(args, *filter.Environment)
	}

	if filter.Message != nil {
		conditions = append(conditions, "message LIKE ?")
		args = append(args, "%"+*filter.Message+"%")
	}

	if filter.From != nil && !filter.From.IsZero() {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, *filter.From)
	}

	if filter.To != nil && !filter.To.IsZero() {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, *filter.To)
	}

	if len(conditions) == 0 {
		return "", args
	}

	var whereClause strings.Builder
	whereClause.WriteString("WHERE " + conditions[0])
	for i := 1; i < len(conditions); i++ {
		whereClause.WriteString(" AND " + conditions[i])
	}

	return whereClause.String(), args
}

func (r *LogClickhouseRepo) buildOrderByClause(filter repo.SearchFilter) string {
	var sortBy string
	if filter.SortBy != nil {
		sortBy = *filter.SortBy
	} else {
		sortBy = repo.DefaultSortBy
	}

	var sortOrder string
	if filter.SortOrder != nil {
		sortOrder = *filter.SortOrder
	} else {
		sortOrder = repo.DefaultSortOrder
	}

	return fmt.Sprintf("ORDER BY %s %s", sortBy, sortOrder)
}
