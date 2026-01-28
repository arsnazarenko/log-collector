package service

import (
	"context"

	v1 "github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
)

var _ v1.StrictServerInterface = (*LogService)(nil)

type LogService struct{}

// AddLogs implements [gen.StrictServerInterface].
func (l *LogService) AddLogs(ctx context.Context, request v1.AddLogsRequestObject) (v1.AddLogsResponseObject, error) {
	panic("unimplemented")
}

// SearchLogs implements [gen.StrictServerInterface].
func (l *LogService) SearchLogs(ctx context.Context, request v1.SearchLogsRequestObject) (v1.SearchLogsResponseObject, error) {
	panic("unimplemented")
}

// UploadLogs implements [gen.StrictServerInterface].
func (l *LogService) UploadLogs(ctx context.Context, request v1.UploadLogsRequestObject) (v1.UploadLogsResponseObject, error) {
	panic("unimplemented")
}
