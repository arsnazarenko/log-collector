package rabbitmq

import (
	"fmt"
	"log"

	"github.com/arsnazarenko/log-collector/internal/config"
	"github.com/wagslane/go-rabbitmq"
)

type MessageHandler func(rabbitmq.Delivery) rabbitmq.Action

type rabbitmqInner struct {
	conn   *rabbitmq.Conn
	config config.RabbitMQ
}

type RabbitConsumer struct {
	inner       *rabbitmqInner
	consumer    *rabbitmq.Consumer
	handlerFunc rabbitmq.Handler
}

type RabbitPublisher struct {
	inner *rabbitmqInner
	pub   *rabbitmq.Publisher
}

func connect(cfg config.RabbitMQ) (*rabbitmqInner, error) {
	uris := make([]string, len(cfg.Brokers))
	for i, broker := range cfg.Brokers {
		uris[i] = fmt.Sprintf("amqp://%s:%s@%s", cfg.Username, cfg.Password, broker)
	}

	resolver := rabbitmq.NewStaticResolver(uris, false)

	conn, err := rabbitmq.NewClusterConn(resolver, rabbitmq.WithConnectionOptionsLogging)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	return &rabbitmqInner{
		conn:   conn,
		config: cfg,
	}, err
}

func NewConsumer(cfg config.RabbitMQ) (*RabbitConsumer, error) {
	inner, err := connect(cfg)
	if err != nil {
		return nil, err
	}
	return &RabbitConsumer{
		inner:       inner,
		consumer:    nil,
		handlerFunc: nil,
	}, nil
}

func (c *RabbitConsumer) DeclareQueue() error {
	consumer, err := rabbitmq.NewConsumer(
		c.inner.conn,
		c.inner.config.QueueName,
		rabbitmq.WithConsumerOptionsExchangeName(c.inner.config.ExchangeName),
		rabbitmq.WithConsumerOptionsExchangeKind("direct"),
		rabbitmq.WithConsumerOptionsExchangeDurable,
		rabbitmq.WithConsumerOptionsExchangeDeclare,
		rabbitmq.WithConsumerOptionsQueueDurable,
		rabbitmq.WithConsumerOptionsQueueQuorum,
		rabbitmq.WithConsumerOptionsRoutingKey("logs.entry"),
		rabbitmq.WithConsumerOptionsQOSPrefetch(c.inner.config.PrefetchCount),
		rabbitmq.WithConsumerOptionsLogging,
	)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}
	c.consumer = consumer
	log.Printf("Declared queue: %s", c.inner.config.QueueName)
	return nil
}

func (c *RabbitConsumer) Consume(handler MessageHandler) error {
	c.handlerFunc = func(d rabbitmq.Delivery) rabbitmq.Action {
		return handler(d)
	}

	go func() {
		if err := c.consumer.Run(c.handlerFunc); err != nil {
			log.Printf("consumer error: %v\n", err)
		}
	}()

	log.Printf("Started consuming messages from queue: %s", c.inner.config.QueueName)
	return nil
}

func (c *RabbitConsumer) Close() error {
	if c.consumer != nil {
		c.consumer.Close()
	}
	if c.inner.conn != nil {
		return c.inner.conn.Close()
	}
	return nil
}

func NewPublisher(cfg config.RabbitMQ) (*RabbitPublisher, error) {
	inner, err := connect(cfg)
	if err != nil {
		return nil, err
	}

	return &RabbitPublisher{
		inner: inner,
		pub:   nil,
	}, nil
}

func (p *RabbitPublisher) DeclareExchange() error {
	pub, err := rabbitmq.NewPublisher(
		p.inner.conn,
		rabbitmq.WithPublisherOptionsExchangeName(p.inner.config.ExchangeName),
		rabbitmq.WithPublisherOptionsExchangeDeclare,
		rabbitmq.WithPublisherOptionsExchangeKind("direct"),
		rabbitmq.WithPublisherOptionsExchangeDurable,
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %w", err)
	}
	p.pub = pub
	log.Printf("Declared exchange: %s", p.inner.config.ExchangeName)
	return nil
}

func (p *RabbitPublisher) Publish(message string) error {
	return p.pub.Publish(
		[]byte(message),
		[]string{"logs.entry"},
		rabbitmq.WithPublishOptionsExchange(p.inner.config.ExchangeName),
	)
}

func (p *RabbitPublisher) Close() error {
	if p.pub != nil {
		p.pub.Close()
	}
	if p.inner.conn != nil {
		return p.inner.conn.Close()
	}
	return nil
}
