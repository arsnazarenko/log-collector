package v1

import (
	"context"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/usecase"
)

var _ gen.StrictServerInterface = (*LogServerImpl)(nil)

type LogServerImpl struct {
	logUC usecase.LogUsecase
}

// AddLogs implements [gen.StrictServerInterface].
func (l *LogServerImpl) AddLogs(ctx context.Context, request gen.AddLogsRequestObject) (gen.AddLogsResponseObject, error) {
	panic("unimplemented")
}

// SearchLogs implements [gen.StrictServerInterface].
func (l *LogServerImpl) SearchLogs(ctx context.Context, request gen.SearchLogsRequestObject) (gen.SearchLogsResponseObject, error) {
	panic("unimplemented")
}

// UploadLogs implements [gen.StrictServerInterface].
func (l *LogServerImpl) UploadLogs(ctx context.Context, request gen.UploadLogsRequestObject) (gen.UploadLogsResponseObject, error) {
	panic("unimplemented")
}
