package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/config"
	"github.com/arsnazarenko/log-collector/pkg/rabbitmq"
	"github.com/ilyakaznacheev/cleanenv"
)

type GeneratorConfig struct {
	Generator       `yaml:"generator"`
	config.RabbitMQ `yaml:"rabbitmq"`
}

type Generator struct {
	Timeout time.Duration `yaml:"timeout" env:"GENERATOR_TIMEOUT" default:"1s"`
	Mode    string        `yaml:"mode" env:"GENERATOR_MODE" default:"rabbitmq"`
	Format  string        `yaml:"format" env:"GENERATOR_FORMAT" default:"json"`
	Count   int           `yaml:"count" env:"GENERATOR_COUNT" default:"0"`
}

const (
	facility = 4
)

var (
	levels       = []string{"debug", "info", "warning", "error", "critical"}
	environments = []string{"dev", "staging", "production"}
	sources      = []string{"auth-service", "payment-service", "cache-service", "database-service", "api-service", "user-service", "security-service", "analytics-service", "notification-service"}
	hosts        = []string{"prod-server-01.com", "prod-server-02.com", "prod-server-03.com", "prod-server-04.com", "prod-server-05.com", "prod-server-06.com", "prod-server-07.com", "prod-server-08.com", "prod-server-09.com", "-"}
	ips          = []string{"192.168.1.100", "192.168.1.101", "192.168.1.102", "192.168.1.103", "192.168.1.104", "192.168.1.105", "192.168.1.106", "192.168.1.107", "192.168.1.108", "192.168.1.109", "192.168.1.112", "192.168.1.113", "192.168.1.114", "192.168.1.115", "192.168.1.116", "192.168.1.117", "192.168.1.118", "192.168.1.119", "192.168.1.121", "192.168.1.122"}
	httpMethods  = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	paths        = []string{"/api/users", "/api/login", "/api/orders", "/api/products", "/api/health", "/api/nonexistent", "/api/slow", "/api/data", "/api/notifications"}
	statusCodes  = []int{200, 201, 200, 200, 404, 500, 504, 403}
	usernames    = []string{"alice", "bob", "charlie"}
	errorTypes   = []string{"TimeoutException", "ConnectionRefused", "NullPointerException", "UnauthorizedAccess"}

	messages = map[string][]string{
		"debug":    {"Debug info: request received", "Processing data", "Checking cache", "Validating input"},
		"info":     {"User login successful", "Email sent to user@example.com", "User profile fetched", "Report generation complete"},
		"warning":  {"Cache miss for key user:1001", "High memory usage detected", "Slow query execution"},
		"error":    {"Payment processing failed", "Database connection failed", "External api timeout", "Authentication failed"},
		"critical": {"Database connection failed", "Service unavailable", "Critical error in payment module"},
	}
)

type LogData struct {
	Level       string
	Environment string
	Source      string
	Host        string
	Message     string
	Payload     *gen.LogPayload
}

func getConfigPath() string {
	const defaultConfigPath = "/etc/generator/config.yaml"
	const configEnvVariable = "GENERATOR_CONFIG_PATH"

	configPath := flag.String("config", defaultConfigPath, "Path to configuration file")
	flag.Parse()
	if envPath := os.Getenv(configEnvVariable); envPath != "" {
		*configPath = envPath
	}
	return *configPath
}

func loadConfig(path string) (*GeneratorConfig, error) {
	cfg := &GeneratorConfig{}
	err := cleanenv.ReadConfig(path, cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}
	return cfg, nil
}

func getRandomLevel() string {
	return levels[rand.IntN(len(levels))]
}

func getRandomEnvironment() string {
	return environments[rand.IntN(len(environments))]
}

func getRandomSource() string {
	return sources[rand.IntN(len(sources))]
}

func getRandomHost() string {
	return hosts[rand.IntN(len(hosts))]
}

func getRandomIP() string {
	return ips[rand.IntN(len(ips))]
}

func getRandomMessage(level string) string {
	msgs := messages[level]
	return msgs[rand.IntN(len(msgs))]
}

func getRandomUsername() string {
	if rand.IntN(2) == 0 {
		return "-"
	}
	return usernames[rand.IntN(len(usernames))]
}

func getRandomHTTPMethod() string {
	return httpMethods[rand.IntN(len(httpMethods))]
}

func getRandomPath() string {
	return paths[rand.IntN(len(paths))]
}

func getRandomStatusCode() int {
	return statusCodes[rand.IntN(len(statusCodes))]
}

func getRandomByteCount() int {
	return rand.IntN(9000) + 100
}

func getRandomUserID() *uint64 {
	if rand.IntN(2) == 0 {
		return nil
	}
	v := rand.Uint64N(10000)
	return &v
}

func getRandomDurationMs() *uint32 {
	if rand.IntN(2) == 0 {
		return nil
	}
	v := rand.Uint32N(10000)
	return &v
}

func getRandomHTTPStatusCodePtr() *uint16 {
	if rand.IntN(2) == 0 {
		return nil
	}
	v := uint16(getRandomStatusCode())
	return &v
}

func getRandomErrorType() *string {
	if rand.IntN(2) == 0 {
		return nil
	}
	v := errorTypes[rand.IntN(len(errorTypes))]
	return &v
}

func getRandomStackTrace() *string {
	if rand.IntN(3) != 0 {
		return nil
	}
	v := "at securitycheck() at verifytoken()"
	return &v
}

func getRandomProcID() string {
	if rand.IntN(2) == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", rand.IntN(999999))
}

func getRandomMsgID() string {
	if rand.IntN(2) == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", rand.IntN(999999))
}

func getRandomPayload() *gen.LogPayload {
	payload := &gen.LogPayload{}
	payload.UserId = getRandomUserID()
	payload.DurationMs = getRandomDurationMs()
	payload.HttpStatusCode = getRandomHTTPStatusCodePtr()
	payload.ErrorType = getRandomErrorType()
	payload.StackTrace = getRandomStackTrace()

	if payload.UserId == nil && payload.DurationMs == nil && payload.HttpStatusCode == nil &&
		payload.ErrorType == nil && payload.StackTrace == nil {
		return nil
	}
	return payload
}

func getRandomLogData() LogData {
	level := getRandomLevel()
	return LogData{
		Level:       level,
		Environment: getRandomEnvironment(),
		Source:      getRandomSource(),
		Host:        getRandomHost(),
		Message:     getRandomMessage(level),
		Payload:     getRandomPayload(),
	}
}

func getRandomTimestamp() time.Time {
	return time.Now().UTC().Add(-time.Duration(rand.IntN(3600)) * time.Second)
}

func generateJSON() string {
	data := getRandomLogData()

	entry := gen.LogEntryInput{
		CreatedAt:   getRandomTimestamp(),
		Level:       gen.LogEntryInputLevel(data.Level),
		Source:      data.Source,
		Host:        data.Host,
		Environment: gen.LogEntryInputEnvironment(data.Environment),
		Message:     data.Message,
		Payload:     data.Payload,
	}

	b, _ := json.Marshal(entry)
	return string(b)
}

func calculatePRIVAL(level string) int {
	severity := 0
	switch level {
	case "critical":
		severity = 2
	case "error":
		severity = 3
	case "warning":
		severity = 4
	case "info":
		severity = 6
	case "debug":
		severity = 7
	}
	return facility*8 + severity
}

func generateSyslog() string {
	data := getRandomLogData()

	prival := calculatePRIVAL(data.Level)
	timestamp := getRandomTimestamp().Format("2006-01-02T15:04:05.000Z")
	hostname := data.Host
	appname := data.Source
	procid := getRandomProcID()
	msgid := getRandomMsgID()

	structuredData := buildStructuredData(data.Payload)
	if structuredData == "" {
		structuredData = "-"
	}

	message := data.Message
	if message == "" {
		message = "-"
	}

	return fmt.Sprintf("<%d>1 %s %s %s %s %s %s %s",
		prival, timestamp, hostname, appname, procid, msgid, structuredData, message)
}

func buildStructuredData(payload *gen.LogPayload) string {
	if payload == nil {
		return "-"
	}

	var parts []string
	sdID := fmt.Sprintf("user@%d", rand.IntN(999999))

	if payload.UserId != nil {
		parts = append(parts, fmt.Sprintf(`user_id="%d"`, *payload.UserId))
	}
	if payload.DurationMs != nil {
		parts = append(parts, fmt.Sprintf(`duration_ms="%d"`, *payload.DurationMs))
	}
	if payload.HttpStatusCode != nil {
		parts = append(parts, fmt.Sprintf(`http_status_code="%d"`, *payload.HttpStatusCode))
	}
	if payload.ErrorType != nil {
		parts = append(parts, fmt.Sprintf(`error_type="%s"`, *payload.ErrorType))
	}
	if payload.StackTrace != nil {
		parts = append(parts, fmt.Sprintf(`stack_trace="%s"`, strings.ReplaceAll(*payload.StackTrace, `"`, `\"`)))
	}

	if len(parts) == 0 {
		return "-"
	}

	return fmt.Sprintf("[%s %s]", sdID, strings.Join(parts, " "))
}

func generateCLF() string {
	ip := getRandomIP()
	ident := "-"
	authuser := getRandomUsername()
	timestamp := getRandomTimestamp().Format("02/Jan/2006:15:04:05 -0700")

	method := getRandomHTTPMethod()
	path := getRandomPath()
	request := fmt.Sprintf(`"%s %s HTTP/1.1"`, method, path)

	statusCode := getRandomStatusCode()
	bytes := getRandomByteCount()

	return fmt.Sprintf(`%s %s %s [%s] %s %d %d`, ip, ident, authuser, timestamp, request, statusCode, bytes)
}

func main() {
	configPath := getConfigPath()
	log.Printf("Load config from: %s", configPath)
	cfg, err := loadConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	var publisher *rabbitmq.RabbitPublisher
	if cfg.Mode == "rabbitmq" {
		publisher, err = rabbitmq.NewPublisher(cfg.RabbitMQ)
		if err != nil {
			log.Fatalf("Error connecting to rabbitmq: %s", err)
		}
		defer publisher.Close()
		if err := publisher.DeclareExchange(); err != nil {
			log.Fatalf("Error declaring new exchange: %s", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		cancel()
	}()

	count := 0
	for {
		if cfg.Count > 0 && count >= cfg.Count {
			log.Printf("Generated %d logs. Stopping.", count)
			break
		}

		select {
		case <-ctx.Done():
			log.Printf("Stopped after generating %d logs", count)
			return
		default:
			var logEntry string

			switch cfg.Format {
			case "json":
				logEntry = generateJSON()
			case "syslog":
				logEntry = generateSyslog()
			case "clf":
				logEntry = generateCLF()
			default:
				log.Fatalf("Unknown format: %s", cfg.Format)
			}

			switch cfg.Mode {
			case "rabbitmq":
				if err := publisher.Publish(logEntry); err != nil {
					log.Printf("Failed to publish: %v", err)
				}
			case "stdout":
				fmt.Println(logEntry)
			}
			count++
			time.Sleep(cfg.Timeout)
		}
	}
}
