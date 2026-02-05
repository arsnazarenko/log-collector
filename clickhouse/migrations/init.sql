CREATE TABLE IF NOT EXISTS logs ON CLUSTER '{cluster}' (
	id UUID DEFAULT generateUUIDv4(),
	created_at DateTime,
	received_at DateTime,
	level LowCardinality(String),
	source String,
	host String,
	environment LowCardinality(String),
	message String,
	user_id Nullable(UInt64),
	duration_ms Nullable(UInt32),
	http_status_code Nullable(UInt16),
	error_type Nullable(String),
	stack_trace Nullable(String)
) ENGINE = ReplicatedMergeTree(
	'/clickhouse/tables/logs',
	'{replica}'
)
ORDER BY (created_at)
PARTITION BY toYYYYMM(created_at);

