package v1

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"testing"
	"time"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockLogUsecase struct {
	mock.Mock
}

func (m *MockLogUsecase) AddLogs(ctx context.Context, logs []gen.LogEntryInput) (int, error) {
	args := m.Called(ctx, logs)
	return args.Int(0), args.Error(1)
}

func (m *MockLogUsecase) SearchLogs(ctx context.Context, params gen.SearchLogsParams) (*gen.SearchResult, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*gen.SearchResult), args.Error(1)
}

func (m *MockLogUsecase) UploadLogs(ctx context.Context, file io.Reader, filename string) (int, error) {
	args := m.Called(ctx, file, filename)
	return args.Int(0), args.Error(1)
}

func TestNewLogServer(t *testing.T) {
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	assert.NotNil(t, server)
	assert.Equal(t, mockUC, server.logUC)
}

func TestLogServerImpl_AddLogs_Success(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	now := time.Now()
	userID := uint64(123)
	httpStatus := uint16(200)
	logs := []gen.LogEntryInput{
		{
			CreatedAt:   now,
			Level:       gen.LogEntryInputLevelError,
			Source:      "test-service",
			Host:        "localhost",
			Environment: gen.LogEntryInputEnvironmentProduction,
			Message:     "Test message",
			Payload: &gen.LogPayload{
				UserId:         &userID,
				HttpStatusCode: &httpStatus,
				ErrorType:      strPtr("TestError"),
			},
		},
	}

	request := gen.AddLogsRequestObject{
		Body: &logs,
	}

	mockUC.On("AddLogs", ctx, logs).Return(1, nil)

	response, err := server.AddLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp201, ok := response.(gen.AddLogs201JSONResponse)
	require.True(t, ok)
	assert.NotNil(t, resp201.Message)
	assert.Contains(t, *resp201.Message, "successfully")
	assert.NotNil(t, resp201.InsertedCount)
	assert.Equal(t, 1, *resp201.InsertedCount)
	mockUC.AssertExpectations(t)
}

func TestLogServerImpl_AddLogs_NilBody(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	request := gen.AddLogsRequestObject{
		Body: nil,
	}

	response, err := server.AddLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp400, ok := response.(gen.AddLogs400JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "request body is required", resp400.Error)
	mockUC.AssertNotCalled(t, "AddLogs")
}

func TestLogServerImpl_AddLogs_UsecaseError(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	now := time.Now()
	logs := []gen.LogEntryInput{
		{
			CreatedAt:   now,
			Level:       gen.LogEntryInputLevelError,
			Source:      "test-service",
			Host:        "localhost",
			Environment: gen.LogEntryInputEnvironmentProduction,
			Message:     "Test message",
		},
	}

	request := gen.AddLogsRequestObject{
		Body: &logs,
	}

	expectedErr := errors.New("database error")
	mockUC.On("AddLogs", ctx, logs).Return(0, expectedErr)

	response, err := server.AddLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp500, ok := response.(gen.AddLogs500JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "failed to add logs", resp500.Error)
	assert.NotNil(t, resp500.Details)
	assert.NotNil(t, resp500.Timestamp)
	mockUC.AssertExpectations(t)
}

func TestLogServerImpl_AddLogs_EmptyLogs(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	logs := []gen.LogEntryInput{}

	request := gen.AddLogsRequestObject{
		Body: &logs,
	}

	mockUC.On("AddLogs", ctx, logs).Return(0, nil)

	response, err := server.AddLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp201, ok := response.(gen.AddLogs201JSONResponse)
	require.True(t, ok)
	assert.Equal(t, 0, *resp201.InsertedCount)
	mockUC.AssertExpectations(t)
}

func TestLogServerImpl_SearchLogs_Success(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	now := time.Now()
	level := gen.SearchLogsParamsLevelError
	env := gen.Production

	params := gen.SearchLogsParams{
		Level:       &level,
		Environment: &env,
		Limit:       intPtr(10),
		Offset:      intPtr(0),
	}

	request := gen.SearchLogsRequestObject{
		Params: params,
	}

	expectedResult := &gen.SearchResult{
		Total: 1,
		Logs: []gen.LogEntry{
			{
				Id:          openapi_types.UUID(uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")),
				CreatedAt:   now,
				ReceivedAt:  now.Add(time.Second),
				Level:       gen.LogEntryLevelError,
				Source:      "test-service",
				Host:        "localhost",
				Environment: gen.LogEntryEnvironmentProduction,
				Message:     "Test message",
			},
		},
	}

	mockUC.On("SearchLogs", ctx, params).Return(expectedResult, nil)

	response, err := server.SearchLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp200, ok := response.(gen.SearchLogs200JSONResponse)
	require.True(t, ok)
	assert.Equal(t, 1, resp200.Total)
	assert.Len(t, resp200.Logs, 1)
	assert.Equal(t, "test-service", resp200.Logs[0].Source)
	mockUC.AssertExpectations(t)
}

func TestLogServerImpl_SearchLogs_UsecaseError(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	params := gen.SearchLogsParams{}
	request := gen.SearchLogsRequestObject{
		Params: params,
	}

	expectedErr := errors.New("search error")
	mockUC.On("SearchLogs", ctx, params).Return(nil, expectedErr)

	response, err := server.SearchLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp500, ok := response.(gen.SearchLogs500JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "failed to search logs", resp500.Error)
	assert.NotNil(t, resp500.Details)
	assert.NotNil(t, resp500.Timestamp)
	mockUC.AssertExpectations(t)
}

func TestLogServerImpl_UploadLogs_NilBody(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	request := gen.UploadLogsRequestObject{
		Body: nil,
	}

	response, err := server.UploadLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp400, ok := response.(gen.UploadLogs400JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "request body is required", resp400.Error)
	mockUC.AssertNotCalled(t, "UploadLogs")
}

func TestLogServerImpl_UploadLogs_NotImplemented(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	body := strings.NewReader("--boundary\r\nContent-Disposition: form-data; name=\"file\"; filename=\"test.log\"\r\nContent-Type: text/plain\r\n\r\ntest log\r\n--boundary--\r\n")
	reader := multipart.NewReader(body, "boundary")

	request := gen.UploadLogsRequestObject{
		Body: reader,
	}

	response, err := server.UploadLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp201, ok := response.(gen.UploadLogs201JSONResponse)
	require.True(t, ok)
	assert.NotNil(t, resp201.Message)
	assert.Contains(t, *resp201.Message, "not implemented")
	assert.NotNil(t, resp201.InsertedCount)
	assert.Equal(t, 0, *resp201.InsertedCount)
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
