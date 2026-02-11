package json

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJSONParser(t *testing.T) {
	parser := NewJSONParser()
	assert.NotNil(t, parser)
}

func TestJSONParser_Name(t *testing.T) {
	parser := NewJSONParser()
	assert.Equal(t, "json", parser.Name())
}

func TestJSONParser_ParseLine_Success(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	userID := uint64(1001)
	httpStatus := uint16(504)
	line := `{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"error","source":"payment-service","host":"prod-server-01.com","environment":"production","message":"Payment processing failed","payload":{"user_id":1001,"http_status_code":504,"error_type":"TimeoutException"}}`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.NotNil(t, log)
	assert.Equal(t, "payment-service", log.Source)
	assert.Equal(t, "prod-server-01.com", log.Host)
	assert.Equal(t, gen.LogEntryInputLevelError, log.Level)
	assert.Equal(t, "Payment processing failed", log.Message)
	assert.Equal(t, gen.LogEntryInputEnvironmentProduction, log.Environment)
	assert.NotNil(t, log.Payload)
	assert.Equal(t, &userID, log.Payload.UserId)
	assert.Equal(t, &httpStatus, log.Payload.HttpStatusCode)
	assert.Equal(t, util.ByPtr("TimeoutException"), log.Payload.ErrorType)
}

func TestJSONParser_ParseLine_WithoutPayload(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	line := `{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"info","source":"auth-service","host":"prod-server-02.com","environment":"production","message":"User login successful"}`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.NotNil(t, log)
	assert.Nil(t, log.Payload)
}

func TestJSONParser_ParseLine_EmptyLine(t *testing.T) {
	parser := NewJSONParser()
	_, err := parser.ParseItem("")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty log")

	_, err = parser.ParseItem("   ")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty log")
}

func TestJSONParser_ParseLine_MissingCreatedAt(t *testing.T) {
	parser := NewJSONParser()
	line := `{"level":"error","source":"payment-service","host":"prod-server-01.com","environment":"production","message":"Payment processing failed"}`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "created_at is required")
}

func TestJSONParser_ParseLine_MissingHost(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	line := `{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"error","source":"payment-service","environment":"production","message":"Payment processing failed"}`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")
}

func TestJSONParser_ParseLine_MissingMessage(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	line := `{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"error","source":"payment-service","host":"prod-server-01.com","environment":"production"}`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestJSONParser_ParseLine_MissingSource(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	line := `{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"error","host":"prod-server-01.com","environment":"production","message":"Payment processing failed"}`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "source is required")
}

func TestJSONParser_ParseLine_InvalidLevel(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	line := `{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"invalid","source":"payment-service","host":"prod-server-01.com","environment":"production","message":"Payment processing failed"}`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid level")
}

func TestJSONParser_ParseLine_InvalidEnvironment(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	line := `{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"error","source":"payment-service","host":"prod-server-01.com","environment":"invalid","message":"Payment processing failed"}`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid environment")
}

func TestJSONParser_ParseLine_AllLevels(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()

	levels := []struct {
		level    gen.LogEntryInputLevel
		levelStr string
	}{
		{gen.LogEntryInputLevelCritical, "critical"},
		{gen.LogEntryInputLevelError, "error"},
		{gen.LogEntryInputLevelWarning, "warning"},
		{gen.LogEntryInputLevelInfo, "info"},
		{gen.LogEntryInputLevelDebug, "debug"},
	}

	for _, tc := range levels {
		line := `{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"` + tc.levelStr + `","source":"test-service","host":"test-host.com","environment":"production","message":"Test message"}`

		log, err := parser.ParseItem(line)

		require.NoError(t, err)
		assert.Equal(t, tc.level, log.Level)
	}
}

func TestJSONParser_ParseLine_AllEnvironments(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()

	envs := []struct {
		env    gen.LogEntryInputEnvironment
		envStr string
	}{
		{gen.LogEntryInputEnvironmentDev, "dev"},
		{gen.LogEntryInputEnvironmentStaging, "staging"},
		{gen.LogEntryInputEnvironmentProduction, "production"},
	}

	for _, tc := range envs {
		line := `{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"info","source":"test-service","host":"test-host.com","environment":"` + tc.envStr + `","message":"Test message"}`

		log, err := parser.ParseItem(line)

		require.NoError(t, err)
		assert.Equal(t, tc.env, log.Environment)
	}
}

func TestJSONParser_ParseLine_InvalidJSON(t *testing.T) {
	parser := NewJSONParser()
	line := `{"invalid json`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log format")
}

func TestJSONParser_Parse_Success(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	input := `[{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"error","source":"payment-service","host":"prod-server-01.com","environment":"production","message":"Payment processing failed"},
{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"info","source":"auth-service","host":"prod-server-02.com","environment":"production","message":"User login successful"}]`

	ctx := context.Background()
	logs, err := parser.Parse(ctx, strings.NewReader(input))

	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, "payment-service", logs[0].Source)
	assert.Equal(t, "auth-service", logs[1].Source)
}

func TestJSONParser_Parse_WithEmptyLines(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	input := `[{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"error","source":"payment-service","host":"prod-server-01.com","environment":"production","message":"Payment processing failed"},
{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"info","source":"auth-service","host":"prod-server-02.com","environment":"production","message":"User login successful"}]`

	ctx := context.Background()
	logs, err := parser.Parse(ctx, strings.NewReader(input))

	require.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestJSONParser_Parse_FirstError(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	input := `[{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"error","source":"payment-service","host":"prod-server-01.com","environment":"production","message":"Payment processing failed"},
{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"invalid","source":"auth-service","host":"prod-server-02.com","environment":"production","message":"User login successful"},
{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"info","source":"auth-service","host":"prod-server-02.com","environment":"production","message":"User login successful"}]`

	ctx := context.Background()
	_, err := parser.Parse(ctx, strings.NewReader(input))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid json object [1]")
	assert.Contains(t, err.Error(), "invalid log format")
}

func TestJSONParser_Parse_MissingRequiredField(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	input := `[{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"error","source":"payment-service","host":"prod-server-01.com","environment":"production","message":"Payment processing failed"},
{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"info","source":"auth-service","environment":"production","message":"User login successful"}]`

	ctx := context.Background()
	_, err := parser.Parse(ctx, strings.NewReader(input))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid json object [1]")
	assert.Contains(t, err.Error(), "host is required")
}

func TestJSONParser_Parse_InvalidEnum(t *testing.T) {
	parser := NewJSONParser()
	now := time.Now()
	input := `[{"created_at":"` + now.UTC().Format(time.RFC3339Nano) + `","level":"invalid","source":"payment-service","host":"prod-server-01.com","environment":"production","message":"Payment processing failed"}]`

	ctx := context.Background()
	_, err := parser.Parse(ctx, strings.NewReader(input))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid json object [0]")
	assert.Contains(t, err.Error(), "invalid level")
}
