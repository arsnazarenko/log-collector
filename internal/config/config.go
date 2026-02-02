package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type (
	Config struct {
		HTTP       `yaml:"http"`
		Clickhouse `yaml:"clickhouse"`
		RabbitMQ   `yaml:"rabbitmq"`
	}

	HTTP struct {
		Host string `yaml:"host" env:"HTTP_HOST"`
		Port string `yaml:"port" env:"HTTP_PORT"`
	}

	Clickhouse struct {
		Database string   `yaml:"database" env:"CLICKHOUSE_DATABASE"`
		User     string   `yaml:"user" env:"CLICKHOUSE_USER"`
		Password string   `yaml:"password" env:"CLICKHOUSE_PASSWORD"`
		Hosts    []string `yaml:"hosts" env:"CLICKHOUSE_HOSTS"`
	}

	RabbitMQ struct {
		Brokers       []string `yaml:"brokers" env:"RABBITMQ_BROKERS"`
		Username      string   `yaml:"username" env:"RABBITMQ_USERNAME"`
		Password      string   `yaml:"password" env:"RABBITMQ_PASSWORD"`
		QueueName     string   `yaml:"queue_name" env:"RABBITMQ_QUEUE_NAME"`
		PrefetchCount int      `yaml:"prefetch_count" env:"RABBITMQ_PREFETCH_COUNT"`
	}
)

func NewConfig(path string) (*Config, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig(path, cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}
	return cfg, nil
}
