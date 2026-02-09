package log

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockLogRepo struct {
	mock.Mock
}

func (m *MockLogRepo) Save(ctx context.Context, log repo.LogEntry) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *MockLogRepo) SaveBatch(ctx context.Context, logs []repo.LogEntry) error {
	args := m.Called(ctx, logs)
	return args.Error(0)
}

func (m *MockLogRepo) FindByID(ctx context.Context, id string) (repo.LogEntry, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(repo.LogEntry), args.Error(1)
}

func (m *MockLogRepo) Search(ctx context.Context, filter repo.SearchFilter) ([]repo.LogEntry, int, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]repo.LogEntry), args.Int(1), args.Error(2)
}

func TestLogUsecase_AddLogs_Empty(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

	count, err := usecase.AddLogs(ctx, []gen.LogEntryInput{})

	assert.NoError(t, err)
	assert.Equal(t, 0, count)
	mockRepo.AssertNotCalled(t, "Save")
	mockRepo.AssertNotCalled(t, "SaveBatch")
}

func TestLogUsecase_AddLogs_Single(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

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

	mockRepo.On("Save", ctx, mock.AnythingOfType("repo.LogEntry")).Return(nil)

	count, err := usecase.AddLogs(ctx, logs)

	assert.NoError(t, err)
	assert.Equal(t, 1, count)
	mockRepo.AssertExpectations(t)
}

func TestLogUsecase_AddLogs_Multiple(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

	now := time.Now()
	userID := uint64(123)
	logs := []gen.LogEntryInput{
		{
			CreatedAt:   now,
			Level:       gen.LogEntryInputLevelError,
			Source:      "test-service",
			Host:        "localhost",
			Environment: gen.LogEntryInputEnvironmentProduction,
			Message:     "Error message",
			Payload: &gen.LogPayload{
				UserId:    &userID,
				ErrorType: strPtr("TestError"),
			},
		},
		{
			CreatedAt:   now.Add(time.Second),
			Level:       gen.LogEntryInputLevelInfo,
			Source:      "test-service",
			Host:        "localhost",
			Environment: gen.LogEntryInputEnvironmentProduction,
			Message:     "Info message",
		},
	}

	mockRepo.On("SaveBatch", ctx, mock.MatchedBy(func(logs []repo.LogEntry) bool {
		return len(logs) == 2
	})).Return(nil)

	count, err := usecase.AddLogs(ctx, logs)

	assert.NoError(t, err)
	assert.Equal(t, 2, count)
	mockRepo.AssertExpectations(t)
}

func TestLogUsecase_AddLogs_SaveError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

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

	expectedErr := errors.New("database error")
	mockRepo.On("Save", ctx, mock.Anything).Return(expectedErr)

	count, err := usecase.AddLogs(ctx, logs)

	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.ErrorIs(t, err, expectedErr)
	mockRepo.AssertExpectations(t)
}

func TestLogUsecase_AddLogs_SaveBatchError(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

	now := time.Now()
	logs := []gen.LogEntryInput{
		{
			CreatedAt:   now,
			Level:       gen.LogEntryInputLevelError,
			Source:      "test-service",
			Host:        "localhost",
			Environment: gen.LogEntryInputEnvironmentProduction,
			Message:     "Error 1",
		},
		{
			CreatedAt:   now.Add(time.Second),
			Level:       gen.LogEntryInputLevelError,
			Source:      "test-service",
			Host:        "localhost",
			Environment: gen.LogEntryInputEnvironmentProduction,
			Message:     "Error 2",
		},
	}

	expectedErr := errors.New("batch insert error")
	mockRepo.On("SaveBatch", ctx, mock.Anything).Return(expectedErr)

	count, err := usecase.AddLogs(ctx, logs)

	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.ErrorIs(t, err, expectedErr)
	mockRepo.AssertExpectations(t)
}

func TestLogUsecase_SearchLogs_Success(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

	now := time.Now()
	level := gen.SearchLogsParamsLevelError
	env := gen.Production
	message := "test"

	params := gen.SearchLogsParams{
		Level:       &level,
		Environment: &env,
		Message:     &message,
		Limit:       intPtr(10),
		Offset:      intPtr(0),
	}

	expectedLogs := []repo.LogEntry{
		{
			ID:          "550e8400-e29b-41d4-a716-446655440000",
			CreatedAt:   now,
			ReceivedAt:  now,
			Level:       "error",
			Source:      "test-service",
			Host:        "localhost",
			Environment: "production",
			Message:     "test message",
		},
	}

	mockRepo.On("Search", ctx, mock.MatchedBy(func(filter repo.SearchFilter) bool {
		return filter.Level != nil && *filter.Level == "error" &&
			filter.Environment != nil && *filter.Environment == "production" &&
			filter.Message != nil && *filter.Message == "test" &&
			filter.Limit == 10 &&
			filter.Offset == 0
	})).Return(expectedLogs, 1, nil)

	result, err := usecase.SearchLogs(ctx, params)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Logs, 1)
	assert.Equal(t, "error", string(result.Logs[0].Level))
	assert.Equal(t, "test-service", result.Logs[0].Source)
	mockRepo.AssertExpectations(t)
}

func TestLogUsecase_SearchLogs_AllParams(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

	now := time.Now()
	level := gen.SearchLogsParamsLevelError
	env := gen.Production
	source := "auth-service"
	message := "failed"
	from := now.Add(-24 * time.Hour)
	to := now

	params := gen.SearchLogsParams{
		Level:       &level,
		Source:      &source,
		Environment: &env,
		Message:     &message,
		From:        &from,
		To:          &to,
		Limit:       intPtr(50),
		Offset:      intPtr(10),
	}

	mockRepo.On("Search", ctx, mock.MatchedBy(func(filter repo.SearchFilter) bool {
		return filter.Level != nil && *filter.Level == "error" &&
			filter.Source != nil && *filter.Source == "auth-service" &&
			filter.Environment != nil && *filter.Environment == "production" &&
			filter.Message != nil && *filter.Message == "failed" &&
			filter.From != nil && !filter.From.IsZero() &&
			filter.To != nil && !filter.To.IsZero() &&
			filter.Limit == 50 &&
			filter.Offset == 10
	})).Return([]repo.LogEntry{}, 0, nil)

	result, err := usecase.SearchLogs(ctx, params)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	assert.Len(t, result.Logs, 0)
	mockRepo.AssertExpectations(t)
}

func TestLogUsecase_SearchLogs_NoFilters(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

	params := gen.SearchLogsParams{}

	mockRepo.On("Search", ctx, mock.MatchedBy(func(filter repo.SearchFilter) bool {
		return filter.Level == nil &&
			filter.Source == nil &&
			filter.Environment == nil &&
			filter.Message == nil &&
			filter.Limit == 100 &&
			filter.Offset == 0
	})).Return([]repo.LogEntry{}, 0, nil)

	result, err := usecase.SearchLogs(ctx, params)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	mockRepo.AssertExpectations(t)
}

func TestLogUsecase_SearchLogs_Error(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

	params := gen.SearchLogsParams{}

	expectedErr := errors.New("search error")
	mockRepo.On("Search", ctx, mock.Anything).Return(nil, 0, expectedErr)

	result, err := usecase.SearchLogs(ctx, params)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, expectedErr)
	mockRepo.AssertExpectations(t)
}

func TestLogUsecase_SearchLogs_WithHost(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

	host := "prod-server-01.com"

	params := gen.SearchLogsParams{
		Host:   &host,
		Limit:  intPtr(10),
		Offset: intPtr(0),
	}

	mockRepo.On("Search", ctx, mock.MatchedBy(func(filter repo.SearchFilter) bool {
		return filter.Host != nil && *filter.Host == "prod-server-01.com" &&
			filter.Limit == 10 &&
			filter.Offset == 0
	})).Return([]repo.LogEntry{}, 0, nil)

	result, err := usecase.SearchLogs(ctx, params)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
	mockRepo.AssertExpectations(t)
}

func TestLogUsecase_UploadLogs_NotImplemented(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockLogRepo)
	usecase := NewLogUsecase(mockRepo)

	count, err := usecase.UploadLogs(ctx, nil, "test.log")

	assert.Error(t, err)
	assert.Equal(t, 0, count)
	assert.Contains(t, err.Error(), "not implemented")
}

func strPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
