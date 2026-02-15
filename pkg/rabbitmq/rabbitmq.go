package rabbitmq

import (
	"fmt"
	"log"
	"sync"
	"time"

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
	inner    *rabbitmqInner
	pub      *rabbitmq.Publisher
	mu       sync.RWMutex
	healthy  bool
	healthMu sync.RWMutex
	config   config.RabbitMQ
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
		inner:   inner,
		pub:     nil,
		healthy: false,
		config:  cfg,
	}, nil
}

func (p *RabbitPublisher) DeclareExchange() error {
	return p.declareExchangeWithRetry(5, time.Second)
}

func (p *RabbitPublisher) declareExchangeWithRetry(maxRetries int, initialDelay time.Duration) error {
	var lastErr error
	delay := initialDelay

	for i := range maxRetries {
		pub, err := rabbitmq.NewPublisher(
			p.inner.conn,
			rabbitmq.WithPublisherOptionsExchangeName(p.inner.config.ExchangeName),
			rabbitmq.WithPublisherOptionsExchangeDeclare,
			rabbitmq.WithPublisherOptionsExchangeDurable,
			rabbitmq.WithPublisherOptionsExchangeKind("direct"),
		)
		if err == nil {
			p.mu.Lock()
			p.pub = pub
			p.setHealthy(true)
			p.mu.Unlock()
			log.Printf("Declared exchange: %s", p.inner.config.ExchangeName)
			return nil
		}
		lastErr = err
		log.Printf("Failed to declare exchange (attempt %d/%d): %v", i+1, maxRetries, err)

		if i < maxRetries-1 {
			log.Printf("Waiting %v before retry...", delay)
			time.Sleep(delay)
			delay *= 2
		}
	}

	p.setHealthy(false)
	return fmt.Errorf("failed to declare exchange after %d attempts: %w", maxRetries, lastErr)
}

func (p *RabbitPublisher) isHealthy() bool {
	p.healthMu.RLock()
	defer p.healthMu.RUnlock()
	return p.healthy
}

func (p *RabbitPublisher) setHealthy(healthy bool) {
	p.healthMu.Lock()
	defer p.healthMu.Unlock()
	p.healthy = healthy
}

func (p *RabbitPublisher) Reconnect() error {
	log.Printf("Attempting to reconnect and redeclare exchange...")

	p.mu.Lock()
	if p.pub != nil {
		p.pub.Close()
		p.pub = nil
	}
	p.mu.Unlock()

	p.setHealthy(false)

	err := p.declareExchangeWithRetry(5, time.Second)
	if err != nil {
		return fmt.Errorf("failed to reconnect: %w", err)
	}

	log.Printf("Successfully reconnected and redeclared exchange")
	return nil
}

func (p *RabbitPublisher) Publish(message string) error {
	p.mu.RLock()
	pub := p.pub
	p.mu.RUnlock()

	if pub == nil || !p.isHealthy() {
		log.Printf("Publisher not healthy, attempting to reconnect...")
		err := p.Reconnect()
		if err != nil {
			return fmt.Errorf("failed to reconnect before publish: %w", err)
		}

		p.mu.RLock()
		pub = p.pub
		p.mu.RUnlock()
	}

	err := pub.Publish(
		[]byte(message),
		[]string{"logs.entry"},
		rabbitmq.WithPublishOptionsExchange(p.inner.config.ExchangeName),
		rabbitmq.WithPublishOptionsPersistentDelivery,
	)
	if err != nil {
		log.Printf("Publish failed with error: %v, marking publisher as unhealthy", err)
		p.setHealthy(false)
		return fmt.Errorf("publish failed: %w", err)
	}

	return nil
}

func (p *RabbitPublisher) Close() error {
	p.mu.Lock()
	if p.pub != nil {
		p.pub.Close()
		p.pub = nil
	}
	p.mu.Unlock()

	p.setHealthy(false)

	if p.inner.conn != nil {
		return p.inner.conn.Close()
	}
	return nil
}
