package v1

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/util"
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

func (m *MockLogUsecase) AddLog(ctx context.Context, log gen.LogEntryInput) error {
	args := m.Called(ctx, log)
	return args.Error(0)
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
				ErrorType:      util.ByPtr("TestError"),
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
		Limit:       util.ByPtr(10),
		Offset:      util.ByPtr(0),
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
	mockUC.AssertNotCalled(t, "AddLogs")
}

func TestLogServerImpl_UploadLogs_NoFile(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())

	request := gen.UploadLogsRequestObject{
		Body: reader,
	}

	response, err := server.UploadLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp400, ok := response.(gen.UploadLogs400JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "file is required", resp400.Error)
	mockUC.AssertNotCalled(t, "AddLogs")
}

func TestLogServerImpl_UploadLogs_InvalidFormat(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "test.log")
	part.Write([]byte("invalid log format"))
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())

	request := gen.UploadLogsRequestObject{
		Body: reader,
	}

	response, err := server.UploadLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp500, ok := response.(gen.UploadLogs500JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "failed to parse log file", resp500.Error)
	mockUC.AssertNotCalled(t, "AddLogs")
}

func TestLogServerImpl_UploadLogs_JSON_Success(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	jsonContent := `[{"created_at":"2025-01-01T12:00:00Z","level":"error","source":"test","host":"localhost","environment":"production","message":"test"}]`

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "logs.json")
	part.Write([]byte(jsonContent))
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())

	mockUC.On("AddLogs", ctx, mock.MatchedBy(func(logs []gen.LogEntryInput) bool {
		return len(logs) == 1 && logs[0].Message == "test"
	})).Return(1, nil)

	request := gen.UploadLogsRequestObject{
		Body: reader,
	}

	response, err := server.UploadLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp201, ok := response.(gen.UploadLogs201JSONResponse)
	require.True(t, ok)
	assert.NotNil(t, resp201.Message)
	assert.Contains(t, *resp201.Message, "successfully")
	assert.NotNil(t, resp201.InsertedCount)
	assert.Equal(t, 1, *resp201.InsertedCount)
	mockUC.AssertExpectations(t)
}

func TestLogServerImpl_UploadLogs_CLF_Success(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	clfContent := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234`

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "logs.txt")
	part.Write([]byte(clfContent))
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())

	mockUC.On("AddLogs", ctx, mock.MatchedBy(func(logs []gen.LogEntryInput) bool {
		return len(logs) == 1 && logs[0].Source == ""
	})).Return(1, nil)

	request := gen.UploadLogsRequestObject{
		Body: reader,
	}

	response, err := server.UploadLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp201, ok := response.(gen.UploadLogs201JSONResponse)
	require.True(t, ok)
	assert.NotNil(t, resp201.Message)
	assert.Contains(t, *resp201.Message, "successfully")
	assert.NotNil(t, resp201.InsertedCount)
	assert.Equal(t, 1, *resp201.InsertedCount)
	mockUC.AssertExpectations(t)
}

func TestLogServerImpl_UploadLogs_Fallback_Success(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	clfContent := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234`

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "logs.unknown")
	part.Write([]byte(clfContent))
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())

	mockUC.On("AddLogs", ctx, mock.MatchedBy(func(logs []gen.LogEntryInput) bool {
		return len(logs) == 1
	})).Return(1, nil)

	request := gen.UploadLogsRequestObject{
		Body: reader,
	}

	response, err := server.UploadLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp201, ok := response.(gen.UploadLogs201JSONResponse)
	require.True(t, ok)
	assert.NotNil(t, resp201.Message)
	assert.Contains(t, *resp201.Message, "successfully")
	assert.NotNil(t, resp201.InsertedCount)
	assert.Equal(t, 1, *resp201.InsertedCount)
	mockUC.AssertExpectations(t)
}

func TestLogServerImpl_UploadLogs_AddLogsError(t *testing.T) {
	ctx := context.Background()
	mockUC := new(MockLogUsecase)
	server := NewLogServer(mockUC)

	jsonContent := `[{"created_at":"2025-01-01T12:00:00Z","level":"error","source":"test","host":"localhost","environment":"production","message":"test"}]`

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "logs.json")
	part.Write([]byte(jsonContent))
	writer.Close()

	reader := multipart.NewReader(&buf, writer.Boundary())

	mockUC.On("AddLogs", ctx, mock.Anything).Return(0, errors.New("database error"))

	request := gen.UploadLogsRequestObject{
		Body: reader,
	}

	response, err := server.UploadLogs(ctx, request)

	require.NoError(t, err)
	assert.NotNil(t, response)

	resp500, ok := response.(gen.UploadLogs500JSONResponse)
	require.True(t, ok)
	assert.Equal(t, "failed to save logs", resp500.Error)
	mockUC.AssertExpectations(t)
}
