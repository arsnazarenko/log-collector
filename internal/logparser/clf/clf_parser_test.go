package clf

import (
	"context"
	"strings"
	"testing"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCLFParser(t *testing.T) {
	parser := NewCLFParser()
	assert.NotNil(t, parser)
}

func TestCLFParser_Name(t *testing.T) {
	parser := NewCLFParser()
	assert.Equal(t, "clf", parser.Name())
}

func TestCLFParser_ParseLine_Success(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, "192.168.1.100", log.Host)
	assert.Equal(t, "", log.Source)
	assert.Equal(t, gen.LogEntryInputLevelInfo, log.Level)
	assert.Equal(t, "GET /api/users HTTP/1.1", log.Message)
	assert.Equal(t, gen.LogEntryInputEnvironmentProduction, log.Environment)
	assert.NotNil(t, log.Payload)
	statusCode := uint16(200)
	assert.Equal(t, &statusCode, log.Payload.HttpStatusCode)
}

func TestCLFParser_ParseLine_EmptyLine(t *testing.T) {
	parser := NewCLFParser()
	_, err := parser.ParseItem("")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty log")

	_, err = parser.ParseItem("   ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty log")

	_, err = parser.ParseItem("   ")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty log")
}

func TestCLFParser_ParseLine_2xxStatus(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, gen.LogEntryInputLevelInfo, log.Level)
}

func TestCLFParser_ParseLine_3xxStatus(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /redirect HTTP/1.1" 301 1234`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, gen.LogEntryInputLevelInfo, log.Level)
}

func TestCLFParser_ParseLine_4xxStatus(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/orders HTTP/1.1" 404 123`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, gen.LogEntryInputLevelWarning, log.Level)
}

func TestCLFParser_ParseLine_5xxStatus(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/products HTTP/1.1" 500 890`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, gen.LogEntryInputLevelError, log.Level)
}

func TestCLFParser_ParseLine_InvalidFormat(t *testing.T) {
	parser := NewCLFParser()
	line := `invalid clf format`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CLF format")
}

func TestCLFParser_ParseLine_MissingTimestamp(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.100 - - ] "GET /api/users HTTP/1.1" 200 1234`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CLF format")
}

func TestCLFParser_ParseLine_InvalidTimestamp(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.100 - - [invalid-timestamp] "GET /api/users HTTP/1.1" 200 1234`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp")
}

func TestCLFParser_ParseLine_InvalidStatusCode(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/users HTTP/1.1" abc 1234`

	_, err := parser.ParseItem(line)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CLF format")
}

func TestCLFParser_ParseLine_ZeroSize(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "HEAD /api/health HTTP/1.1" 200 0`

	_, err := parser.ParseItem(line)

	require.NoError(t, err)
}

func TestCLFParser_ParseLine_DashSize(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 -`

	_, err := parser.ParseItem(line)

	require.NoError(t, err)
}

func TestCLFParser_ParseLine_PostRequest(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.101 - - [01/Jan/2025:12:00:05 +0000] "POST /api/login HTTP/1.1" 200 567`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, "POST /api/login HTTP/1.1", log.Message)
}

func TestCLFParser_ParseLine_DeleteRequest(t *testing.T) {
	parser := NewCLFParser()
	line := `192.168.1.104 - - [01/Jan/2025:12:00:20 +0000] "DELETE /api/users/123 HTTP/1.1" 403 456`

	log, err := parser.ParseItem(line)

	require.NoError(t, err)
	assert.Equal(t, gen.LogEntryInputLevelWarning, log.Level)
	assert.Equal(t, "DELETE /api/users/123 HTTP/1.1", log.Message)
}

func TestCLFParser_Parse_Success(t *testing.T) {
	parser := NewCLFParser()
	input := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234
192.168.1.101 - - [01/Jan/2025:12:00:05 +0000] "POST /api/login HTTP/1.1" 200 567`

	ctx := context.Background()
	logs, err := parser.Parse(ctx, strings.NewReader(input))

	require.NoError(t, err)
	assert.Len(t, logs, 2)
	assert.Equal(t, "GET /api/users HTTP/1.1", logs[0].Message)
	assert.Equal(t, "POST /api/login HTTP/1.1", logs[1].Message)
}

func TestCLFParser_Parse_WithEmptyLines(t *testing.T) {
	parser := NewCLFParser()
	input := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234

192.168.1.101 - - [01/Jan/2025:12:00:05 +0000] "POST /api/login HTTP/1.1" 200 567`

	ctx := context.Background()
	logs, err := parser.Parse(ctx, strings.NewReader(input))

	require.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestCLFParser_Parse_FirstError(t *testing.T) {
	parser := NewCLFParser()
	input := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234
invalid clf line
192.168.1.101 - - [01/Jan/2025:12:00:05 +0000] "POST /api/login HTTP/1.1" 200 567`

	ctx := context.Background()
	_, err := parser.Parse(ctx, strings.NewReader(input))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2")
	assert.Contains(t, err.Error(), "invalid CLF format")
}

func TestCLFParser_Parse_InvalidTimestamp(t *testing.T) {
	parser := NewCLFParser()
	input := `192.168.1.100 - - [01/Jan/2025:12:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234
192.168.1.101 - - [invalid-date] "POST /api/login HTTP/1.1" 200 567`

	ctx := context.Background()
	_, err := parser.Parse(ctx, strings.NewReader(input))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "line 2")
	assert.Contains(t, err.Error(), "invalid timestamp")
}
