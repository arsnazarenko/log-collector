package rabbitmq

import (
	"fmt"
	"log"

	"github.com/arsnazarenko/log-collector/internal/config"
	"github.com/wagslane/go-rabbitmq"
)

type RabbitConsumer struct {
	conn        *rabbitmq.Conn
	consumer    *rabbitmq.Consumer
	config      config.RabbitMQ
	handlerFunc rabbitmq.Handler
}

type MessageHandler func(rabbitmq.Delivery) rabbitmq.Action

func NewRabbitConsumer(cfg config.RabbitMQ) *RabbitConsumer {
	return &RabbitConsumer{
		config: cfg,
	}
}

func (r *RabbitConsumer) Connect() error {
	uris := make([]string, len(r.config.Brokers))
	for i, broker := range r.config.Brokers {
		uris[i] = fmt.Sprintf("amqp://%s:%s@%s", r.config.Username, r.config.Password, broker)
	}

	resolver := rabbitmq.NewStaticResolver(uris, false)

	conn, err := rabbitmq.NewClusterConn(resolver, rabbitmq.WithConnectionOptionsLogging)
	if err != nil {
		return fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}
	r.conn = conn
	log.Printf("Connected to RabbitMQ cluster")
	return nil
}

func (r *RabbitConsumer) DeclareExchange() error {
	_, err := rabbitmq.NewPublisher(
		r.conn,
		rabbitmq.WithPublisherOptionsExchangeName("logs"),
		rabbitmq.WithPublisherOptionsExchangeDeclare,
		rabbitmq.WithPublisherOptionsExchangeKind("direct"),
		rabbitmq.WithPublisherOptionsExchangeDurable,
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}
	log.Printf("Declared exchange: logs")
	return nil
}

func (r *RabbitConsumer) DeclareQueue() error {
	consumer, err := rabbitmq.NewConsumer(
		r.conn,
		r.config.QueueName,
		rabbitmq.WithConsumerOptionsExchangeName("logs"),
		rabbitmq.WithConsumerOptionsExchangeKind("direct"),
		rabbitmq.WithConsumerOptionsExchangeDurable,
		rabbitmq.WithConsumerOptionsExchangeDeclare,
		rabbitmq.WithConsumerOptionsQueueDurable,
		rabbitmq.WithConsumerOptionsQueueQuorum,
		rabbitmq.WithConsumerOptionsRoutingKey("logs.entry"),
		rabbitmq.WithConsumerOptionsQOSPrefetch(r.config.PrefetchCount),
		rabbitmq.WithConsumerOptionsLogging,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	r.consumer = consumer
	log.Printf("Declared queue: %s", r.config.QueueName)
	return nil
}

func (r *RabbitConsumer) Consume(handler MessageHandler) error {
	r.handlerFunc = func(d rabbitmq.Delivery) rabbitmq.Action {
		return handler(d)
	}

	go func() {
		if err := r.consumer.Run(r.handlerFunc); err != nil {
			log.Printf("consumer error: %v\n", err)
		}
	}()

	log.Printf("Started consuming messages from queue: %s", r.config.QueueName)
	return nil
}

func (r *RabbitConsumer) Close() error {
	if r.consumer != nil {
		r.consumer.Close()
	}
	if r.conn != nil {
		if err := r.conn.Close(); err != nil {
			return fmt.Errorf("failed to close rabbitmq connection: %w", err)
		}
	}
	log.Printf("RabbitMQ connection closed")
	return nil
}
