package app

import (
	"log"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/config"
	"github.com/arsnazarenko/log-collector/pkg/clickhouse"
	"github.com/arsnazarenko/log-collector/pkg/rabbitmq"
)

func Run() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	swagger, err := gen.GetSwagger()
	if err != nil {
		log.Fatalf("Error loading swagger spec: %s", err)
	}
	swagger.Servers = nil

	ch, err := clickhouse.New(cfg.Clickhouse)
	defer func() {
		_ = ch.Close()
	}()

	if err != nil {
		log.Fatalf("Error connecting to clickhouse: %s", err)
	}

	rmq := rabbitmq.NewRabbitConsumer(cfg.RabbitMQ)
	err = rmq.Connect()
	if err != nil {
		log.Fatalf("Error connecting to rabbitmq: %s", err)
	}
	defer rmq.Close()

	err = rmq.DeclareExchange()
	if err != nil {
		log.Fatalf("Error declaring exchange: %s", err)
	}

	err = rmq.DeclareQueue()
	if err != nil {
		log.Fatalf("Error declaring queue: %s", err)
	}

	_ = rmq

	log.Printf("Server started on port %s", cfg.HTTP.Port)
}
