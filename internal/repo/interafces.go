package repo

import (
	"context"
	"errors"
	"time"
)

var ErrLogNotFound = errors.New("log entry not found")

type LogRepo interface {
	Save(ctx context.Context, log LogEntry) error
	SaveBatch(ctx context.Context, logs []LogEntry) error
	FindByID(ctx context.Context, id string) (LogEntry, error)
	Search(ctx context.Context, filter SearchFilter) ([]LogEntry, int, error)
}

type LogEntry struct {
	ID             string
	CreatedAt      time.Time
	ReceivedAt     time.Time
	Level          string
	Source         string
	Host           string
	Environment    string
	Message        string
	UserID         *uint64
	DurationMs     *uint32
	HTTPStatusCode *uint16
	ErrorType      *string
	StackTrace     *string
}

type SearchFilter struct {
	Level       string
	Source      string
	Host        string
	Environment string
	Message     string
	From        time.Time
	To          time.Time
	Limit       int
	Offset      int
}
