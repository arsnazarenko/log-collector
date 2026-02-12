package syslog

import (
	"context"
	"strings"
	"testing"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSyslogParser(t *testing.T) {
	parser := NewSyslogParser()
	assert.NotNil(t, parser)
}

func TestSyslogParser_Name(t *testing.T) {
	parser := NewSyslogParser()
	assert.Equal(t, "syslog", parser.Name())
}

func TestSyslogParser_ParseLine_Success(t *testing.T) {
	parser := NewSyslogParser()
	line := `<34>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [user@1001 user_id="1001"] Payment processing failed`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, "payment-service", log.Source)
	assert.Equal(t, "prod-server-01.com", log.Host)
	assert.Equal(t, gen.LogEntryInputLevelCritical, log.Level)
	assert.Equal(t, "Payment processing failed", log.Message)
	assert.Equal(t, gen.LogEntryInputEnvironmentProduction, log.Environment)
	assert.NotNil(t, log.Payload)
	userID := uint64(1001)
	assert.Equal(t, &userID, log.Payload.UserId)
}

func TestSyslogParser_ParseLine_EmptyLine(t *testing.T) {
	parser := NewSyslogParser()
	_, err := parser.ParseItem("")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty log")

	_, err = parser.ParseItem("   ")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty log")
}

func TestSyslogParser_ParseLine_MissingHostname(t *testing.T) {
	parser := NewSyslogParser()
	line := `<34>1 2025-01-01T12:00:00.003Z - payment-service - - [user@1001 user_id="1001"] Payment processing failed`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, log.Host, "")
}

func TestSyslogParser_ParseLine_MissingTimestamp(t *testing.T) {
	parser := NewSyslogParser()
	line := `<34>1 - prod-server-01 payment-service - - [user@1001 user_id="1001"] Payment processing failed`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timestamp is required")
}

func TestSyslogParser_ParseLine_MissingAppname(t *testing.T) {
	parser := NewSyslogParser()
	line := `<34>1 2025-01-01T12:00:00.003Z prod-server-01.com - - - [user@1001 user_id="1001"] Payment processing failed`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, log.Source, "")
}

func TestSyslogParser_ParseLine_MissingMessage(t *testing.T) {
	parser := NewSyslogParser()
	line := `<34>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [user@1001 user_id="1001"]`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, log.Message, "")
}

func TestSyslogParser_ParseLine_InvalidFormat(t *testing.T) {
	parser := NewSyslogParser()
	line := `invalid syslog message`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid log format")
}

func TestSyslogParser_ParseLine_AllSeverities(t *testing.T) {
	parser := NewSyslogParser()

	cases := []struct {
		priority string
		level    gen.LogEntryInputLevel
	}{
		{"<0>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [meta@1 event=\"emergency\"] Emergency message", gen.LogEntryInputLevelCritical},
		{"<1>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [meta@1 event=\"alert\"] Alert message", gen.LogEntryInputLevelCritical},
		{"<2>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [meta@1 event=\"critical\"] Critical message", gen.LogEntryInputLevelCritical},
		{"<3>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [meta@1 event=\"error\"] Error message", gen.LogEntryInputLevelError},
		{"<4>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [meta@1 event=\"warning\"] Warning message", gen.LogEntryInputLevelWarning},
		{"<5>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [meta@1 event=\"notice\"] Notice message", gen.LogEntryInputLevelInfo},
		{"<6>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [meta@1 event=\"info\"] Info message", gen.LogEntryInputLevelInfo},
		{"<7>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [meta@1 event=\"debug\"] Debug message", gen.LogEntryInputLevelDebug},
	}

	for _, tc := range cases {
		line := tc.priority

		log, err := parser.ParseItem(line)

		require.NoError(t, err, "Failed to parse line: %s", line)
		assert.Equal(t, tc.level, log.Level)
	}
}

func TestSyslogParser_ParseLine_WithStructuredData(t *testing.T) {
	parser := NewSyslogParser()
	line := `<34>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [http@12345 http_status_code="504" duration_ms="5000"] External API timeout`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.NotNil(t, log.Payload)
	httpStatus := uint16(504)
	assert.Equal(t, &httpStatus, log.Payload.HttpStatusCode)
	durationMs := uint32(5000)
	assert.Equal(t, &durationMs, log.Payload.DurationMs)
}

func TestSyslogParser_ParseLine_WithAllPayloadFields(t *testing.T) {
	parser := NewSyslogParser()
	line := `<34>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [app@1001 user_id="1001" http_status_code="504" duration_ms="5000" error_type="TimeoutException" stack_trace="line1\\nline2"] Payment failed`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.NotNil(t, log.Payload)
	userID := uint64(1001)
	assert.Equal(t, &userID, log.Payload.UserId)
	httpStatus := uint16(504)
	assert.Equal(t, &httpStatus, log.Payload.HttpStatusCode)
	durationMs := uint32(5000)
	assert.Equal(t, &durationMs, log.Payload.DurationMs)
	assert.Equal(t, util.ByPtr("TimeoutException"), log.Payload.ErrorType)
	assert.Equal(t, util.ByPtr("line1\\nline2"), log.Payload.StackTrace)
}

func TestSyslogParser_ParseLine_WithoutStructuredData(t *testing.T) {
	parser := NewSyslogParser()
	line := `<14>1 2025-01-01T12:00:05.003Z prod-server-02.com auth-service - - [meta@1 event="login"] User login successful`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Nil(t, log.Payload)
	assert.Equal(t, gen.LogEntryInputLevelInfo, log.Level)
}

func TestSyslogParser_Parse_Success(t *testing.T) {
	parser := NewSyslogParser()
	input := `<34>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [user@1001 user_id="1001"] Payment processing failed
<14>1 2025-01-01T12:00:05.003Z prod-server-02.com auth-service - - [meta@1 event="login"] User login successful`

	ctx := context.Background()
	logs, err := parser.Parse(ctx, strings.NewReader(input))

	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, "payment-service", logs[0].Source)
	assert.Equal(t, "auth-service", logs[1].Source)
}

func TestSyslogParser_Parse_WithEmptyLines(t *testing.T) {
	parser := NewSyslogParser()
	input := `<34>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [user@1001 user_id="1001"] Payment processing failed

<14>1 2025-01-01T12:00:05.003Z prod-server-02.com auth-service - - [meta@1 event="login"] User login successful`

	ctx := context.Background()
	logs, err := parser.Parse(ctx, strings.NewReader(input))

	require.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestSyslogParser_Parse_FirstError(t *testing.T) {
	parser := NewSyslogParser()
	input := `<34>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [user@1001 user_id="1001"] Payment processing failed
invalid syslog line
<14>1 2025-01-01T12:00:05.003Z prod-server-02.com auth-service - - [meta@1 event="login"] User login successful`

	ctx := context.Background()
	_, err := parser.Parse(ctx, strings.NewReader(input))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2")
	assert.Contains(t, err.Error(), "invalid log format")
}

func TestSyslogParser_Parse_MissingRequiredField(t *testing.T) {
	parser := NewSyslogParser()
	input := `<34>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [user@1001 user_id="1001"] Payment processing failed
<34>1 - prod-server-01 payment-service - - [user@1001 user_id="1001"] Missing time`

	ctx := context.Background()
	_, err := parser.Parse(ctx, strings.NewReader(input))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2")
	assert.Contains(t, err.Error(), "timestamp is required")
}

func TestSyslogParser_Parse_ValidMultipleValuesWithStructInfo(t *testing.T) {
	parser := NewSyslogParser()
	input := `
<34>1 2025-01-01T12:00:00.003Z prod-server-01.com payment-service - - [user@1001 user_id="1001"] payment processing failed
<46>1 2025-01-01T12:02:00.003Z prod-server-04.com database-service 128 987 [error@32473 error_type="connectionrefused"] database connection failed
<34>1 2025-01-01T12:04:00.003Z prod-server-01.com payment-service - 123 [http@12345 http_status_code="504" duration_ms="5000"] external api timeout
<30>1 2025-01-01T12:05:00.003Z prod-server-06.com user-service - - [user@1001 user_id="1002"] user profile fetched
<11>1 2025-01-01T12:06:00.003Z prod-server-07.com security-service - - [security@9999 user_id="9999" error_type="unauthorizedaccess" stack_trace="at securitycheck()\\nat verifytoken()" duration_ms="10000"] authentication failed
<31>1 2025-01-01T12:08:00.003Z prod-server-09.com analytics-service 123 - [analytics@555 duration_ms="1500"] report generation complete
`
	ctx := context.Background()
	logs, err := parser.Parse(ctx, strings.NewReader(input))

	require.NoError(t, err)
	assert.Equal(t, 6, len(logs))
}

func TestSyslogParser_Parse_ValidMultipleValuesWithoutStructInfo(t *testing.T) {
	parser := NewSyslogParser()
	input := `
<14>1 2025-01-01T12:00:05.003Z prod-server-02.com auth-service 123 321 - user login successful
<22>1 2025-01-01T12:01:00.003Z prod-server-03.com cache-service - - - cache miss for key user:1001
<23>1 2025-01-01T12:03:00.003Z prod-server-05.com api-service 123123 666 - debug info: request received
<20>1 2025-01-01T12:07:00.003Z prod-server-08.com notification-service - 4890 - email sent to user@example.com
<14>1 2025-01-01T12:00:05.003Z prod-server-02.com auth-service_1 - 321 - user login successful
<22>1 2025-01-01T12:01:00.003Z prod-server-03.com cache-service_2 111 - - cache miss for key user:1001
<23>1 2025-01-01T12:03:00.003Z prod-server-05.com api-service_3 - 123123 - debug info: request received
<20>1 2025-01-01T12:07:00.003Z prod-server-08.com notification-service_4 12312 4890 - email sent to user@example.com
`
	ctx := context.Background()
	logs, err := parser.Parse(ctx, strings.NewReader(input))

	require.NoError(t, err)
	assert.Equal(t, 8, len(logs))
}
