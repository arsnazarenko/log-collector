package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/arsnazarenko/log-collector/internal/config"
)

type Clickhouse struct {
	Conn driver.Conn
}

func New(cfg config.Clickhouse) (*Clickhouse, error) {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: cfg.Hosts,
			Auth: clickhouse.Auth{
				Database: cfg.Database,
				Username: cfg.User,
				Password: cfg.Password,
			},
			ClientInfo: clickhouse.ClientInfo{
				Products: []struct {
					Name    string
					Version string
				}{
					{Name: "log-collector", Version: "1.0"},
				},
			},
			ConnOpenStrategy: clickhouse.ConnOpenRoundRobin,
			// TLS: &tls.Config{
			// 	InsecureSkipVerify: true,
			// },
		})
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to clickhouse: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			return nil, fmt.Errorf("clickhouse exception [%d] %s \n%s", exception.Code, exception.Message, exception.StackTrace)
		}
		return nil, fmt.Errorf("failed to ping clickhouse: %w", err)
	}

	return &Clickhouse{Conn: conn}, nil
}

func (ch *Clickhouse) Close() error {
	return ch.Conn.Close()
}
