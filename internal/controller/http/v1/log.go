package v1

import (
	"context"
	"fmt"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/usecase"
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

// AddLogs implements [gen.StrictServerInterface].
func (l *LogServerImpl) AddLogs(ctx context.Context, request gen.AddLogsRequestObject) (gen.AddLogsResponseObject, error) {
	return nil, fmt.Errorf("unimplemented")
}

// SearchLogs implements [gen.StrictServerInterface].
func (l *LogServerImpl) SearchLogs(ctx context.Context, request gen.SearchLogsRequestObject) (gen.SearchLogsResponseObject, error) {
	return nil, fmt.Errorf("unimplemented")
}

// UploadLogs implements [gen.StrictServerInterface].
func (l *LogServerImpl) UploadLogs(ctx context.Context, request gen.UploadLogsRequestObject) (gen.UploadLogsResponseObject, error) {
	return nil, fmt.Errorf("unimplemented")
}
