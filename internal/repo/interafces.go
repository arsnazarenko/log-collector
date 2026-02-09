package repo

import (
	"context"
	"errors"
	"time"
)

var ErrLogNotFound = errors.New("log entry not found")

const (
	DefaultSortBy    = "created_at"
	DefaultSortOrder = "DESC"
	DefaultOffset    = 0
	DefaultLimit     = 100
)

type LogRepo interface {
	Save(ctx context.Context, log LogEntry) error
	SaveBatch(ctx context.Context, logs []LogEntry) error
	FindByID(ctx context.Context, id string) (LogEntry, error)
	Search(ctx context.Context, filter SearchFilter) ([]LogEntry, int, error)
}

type LogEntry struct {
	ID             string    `db:"id"`
	CreatedAt      time.Time `db:"created_at"`
	ReceivedAt     time.Time `db:"received_at"`
	Level          string    `db:"level"`
	Source         string    `db:"source"`
	Host           string    `db:"host"`
	Environment    string    `db:"environment"`
	Message        string    `db:"message"`
	UserID         *uint64   `db:"user_id"`
	DurationMs     *uint32   `db:"duration_ms"`
	HTTPStatusCode *uint16   `db:"http_status_code"`
	ErrorType      *string   `db:"error_type"`
	StackTrace     *string   `db:"stack_trace"`
}

type SearchFilter struct {
	Level       *string
	Source      *string
	Host        *string
	Environment *string
	Message     *string
	From        *time.Time
	To          *time.Time
	Limit       int
	Offset      int
	SortBy      *string
	SortOrder   *string
}
