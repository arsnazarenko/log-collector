package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	logParsingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "log_parsing_duration_seconds",
			Help:    "Log parsing duration in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
		},
		[]string{"parser", "source", "log_count"},
	)

	logParsingSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "log_parsing_summary_seconds",
			Help:       "Log parsing summary in seconds",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.95: 0.005, 0.99: 0.001},
		},
		[]string{"parser", "source"},
	)

	clickhouseConnectionsStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "clickhouse_connection_status",
			Help: "Clickhouse connection status (1=connected, 0=disconnected)",
		},
		[]string{"host"},
	)

	clickhouseQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "clickhouse_query_duration_seconds",
			Help:    "Clickhouse query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "host"},
	)

	rabbitmqConnectionsStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "rabbitmq_connection_status",
			Help: "RabbitMQ connection status (1=connected, 0=disconnected)",
		},
		[]string{"host"},
	)

	rabbitmqMessagesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rabbitmq_messages_total",
			Help: "Total number of RabbitMQ messages processed",
		},
		[]string{"status", "queue"},
	)

	rabbitmqProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "rabbitmq_processing_duration_seconds",
			Help:    "RabbitMQ message processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"queue"},
	)

	logsProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "logs_processed_total",
			Help: "Total number of logs processed",
		},
		[]string{"source", "status"},
	)

	appStartTime = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "app_start_time_seconds",
			Help: "Application start timestamp in seconds since epoch",
		},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		logParsingDuration,
		logParsingSummary,
		clickhouseConnectionsStatus,
		clickhouseQueryDuration,
		rabbitmqConnectionsStatus,
		rabbitmqMessagesTotal,
		rabbitmqProcessingDuration,
		logsProcessedTotal,
		appStartTime,
	)
	appStartTime.SetToCurrentTime()
}

func RecordHTTPRequest(method, endpoint string, status int, duration time.Duration) {
	httpRequestsTotal.WithLabelValues(method, endpoint, string(rune(status))).Inc()
	httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

func ObserveHTTPRequestDuration(method, endpoint string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		httpRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
	}
}

func RecordLogParsing(parser, source string, logCount int, duration time.Duration) {
	logParsingDuration.WithLabelValues(parser, source, formatLogCount(logCount)).Observe(duration.Seconds())
	logParsingSummary.WithLabelValues(parser, source).Observe(duration.Seconds())
}

func ObserveLogParsing(parser, source string, logCount int) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		RecordLogParsing(parser, source, logCount, duration)
	}
}

func SetClickhouseStatus(host string, connected bool) {
	status := float64(0)
	if connected {
		status = 1
	}
	clickhouseConnectionsStatus.WithLabelValues(host).Set(status)
}

func ObserveClickhouseQuery(operation, host string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		clickhouseQueryDuration.WithLabelValues(operation, host).Observe(duration.Seconds())
	}
}

func SetRabbitmqStatus(host string, connected bool) {
	status := float64(0)
	if connected {
		status = 1
	}
	rabbitmqConnectionsStatus.WithLabelValues(host).Set(status)
}

func RecordRabbitmqMessage(status, queue string, duration time.Duration) {
	rabbitmqMessagesTotal.WithLabelValues(status, queue).Inc()
	rabbitmqProcessingDuration.WithLabelValues(queue).Observe(duration.Seconds())
}

func ObserveRabbitmqProcessing(queue string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		RecordRabbitmqMessage("processed", queue, duration)
	}
}

func RecordRabbitmqError(queue string) {
	rabbitmqMessagesTotal.WithLabelValues("error", queue).Inc()
}

func RecordLogsProcessed(source, status string, count int) {
	logsProcessedTotal.WithLabelValues(source, status).Add(float64(count))
}

func formatLogCount(count int) string {
	if count <= 1 {
		return "1"
	} else if count <= 10 {
		return "2-10"
	} else if count <= 100 {
		return "11-100"
	} else if count <= 1000 {
		return "101-1000"
	}
	return "1000+"
}
