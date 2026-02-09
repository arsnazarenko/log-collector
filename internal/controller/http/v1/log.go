package v1

import (
	"context"
	"errors"
	"io"
	"time"

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
		details := map[string]any{"error": err.Error()}
		return gen.AddLogs500JSONResponse{
			InternalServerErrorJSONResponse: gen.InternalServerErrorJSONResponse{
				Error:     "failed to add logs",
				Details:   &details,
				Timestamp: timePtr(time.Now()),
			},
		}, nil
	}

	message := "logs added successfully"
	return gen.AddLogs201JSONResponse{
		Message:       &message,
		InsertedCount: &count,
	}, nil
}

func (l *LogServerImpl) SearchLogs(ctx context.Context, request gen.SearchLogsRequestObject) (gen.SearchLogsResponseObject, error) {
	result, err := l.logUC.SearchLogs(ctx, request.Params)
	if err != nil {
		details := map[string]any{"error": err.Error()}
		return gen.SearchLogs500JSONResponse{
			InternalServerErrorJSONResponse: gen.InternalServerErrorJSONResponse{
				Error:     "failed to search logs",
				Details:   &details,
				Timestamp: timePtr(time.Now()),
			},
		}, nil
	}

	return gen.SearchLogs200JSONResponse(*result), nil
}

func (l *LogServerImpl) UploadLogs(ctx context.Context, request gen.UploadLogsRequestObject) (gen.UploadLogsResponseObject, error) {
	if request.Body == nil {
		return gen.UploadLogs400JSONResponse{
			BadRequestJSONResponse: gen.BadRequestJSONResponse{
				Error: "request body is required",
			},
		}, nil
	}

	_, err := request.Body.NextPart()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return gen.UploadLogs400JSONResponse{
				BadRequestJSONResponse: gen.BadRequestJSONResponse{
					Error: "file is required",
				},
			}, nil
		}
		details := map[string]interface{}{"error": err.Error()}
		return gen.UploadLogs500JSONResponse{
			InternalServerErrorJSONResponse: gen.InternalServerErrorJSONResponse{
				Error:     "failed to read multipart body",
				Details:   &details,
				Timestamp: timePtr(time.Now()),
			},
		}, nil
	}

	count := 0
	message := "file upload not implemented yet"
	return gen.UploadLogs201JSONResponse{
		Message:       &message,
		InsertedCount: &count,
	}, nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}
