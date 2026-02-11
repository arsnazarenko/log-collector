package app

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/arsnazarenko/log-collector/api/openapi/v1/gen"
	"github.com/arsnazarenko/log-collector/internal/config"
	v1 "github.com/arsnazarenko/log-collector/internal/controller/http/v1"
	"github.com/arsnazarenko/log-collector/internal/repo/persistent"
	log_usecase "github.com/arsnazarenko/log-collector/internal/usecase/log"
	"github.com/arsnazarenko/log-collector/pkg/clickhouse"
	rmq_consumer "github.com/arsnazarenko/log-collector/pkg/rabbitmq"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	oapi_middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/wagslane/go-rabbitmq"
)

func getConfigPath() string {
	const defaultConfigPath = "/etc/collector/config.yaml"
	const configEnvVariable = "COLLECTOR_CONFIG_PATH"

	configPath := flag.String("config", defaultConfigPath, "Path to configuration file")
	flag.Parse()
	if envPath := os.Getenv(configEnvVariable); envPath != "" {
		*configPath = envPath
	}
	return *configPath
}

func Run() {
	configPath := getConfigPath()
	log.Printf("Load config from: %s", configPath)
	cfg, err := config.NewConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	swagger, err := gen.GetSwagger()
	if err != nil {
		log.Fatalf("Error loading swagger spec: %s", err)
	}
	swagger.Servers = nil

	ctx := context.Background()
	ch, err := clickhouse.New(cfg.Clickhouse)
	if err != nil {
		log.Fatalf("Error connecting to clickhouse: %s", err)
	}
	defer ch.Close()

	logRepo := persistent.NewLogClickhouseRepo(ch)
	if err = logRepo.CreateTable(ctx); err != nil {
		log.Fatalf("Clickhouse error: %s", err)
	}

	logUsecase := log_usecase.NewLogUsecase(logRepo)
	serverImpl := v1.NewLogServer(logUsecase)
	server := gen.NewStrictHandler(serverImpl, []gen.StrictMiddlewareFunc{})

	r := chi.NewRouter()

	corsHandler := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS", "HEAD"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	})

	r.Use(corsHandler)

	r.Handle("/swagger/*", http.StripPrefix("/swagger/", http.FileServer(http.FS(gen.SwaggerUI))))
	r.Handle("/metrics", promhttp.Handler())

	r.Route("/", func(r chi.Router) {
		r.Use(oapi_middleware.OapiRequestValidator(swagger))
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)
		gen.HandlerFromMux(server, r)
	})

	s := &http.Server{
		Handler: r,
		Addr:    net.JoinHostPort(cfg.Host, cfg.Port),
	}

	rmq, err := rmq_consumer.New(cfg.RabbitMQ)
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
	rmq.Consume(func(d rabbitmq.Delivery) rabbitmq.Action {
		var logInput gen.LogEntryInput
		if err = json.Unmarshal(d.Body, &logInput); err != nil {
			log.Printf("Failed to parse log entry from RabbitMQ: %v", err)
			return rabbitmq.NackDiscard
		}

		if _, err = logUsecase.AddLogs(context.Background(), []gen.LogEntryInput{logInput}); err != nil {
			log.Printf("Failed to save log from RabbitMQ: %v", err)
			return rabbitmq.NackDiscard
		}

		log.Printf("Saved log from RabbitMQ: %s", logInput.Message)
		return rabbitmq.Ack
	})

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3) // if shutdown execure more than timeout => interrupt by context cancel()
		defer cancel()
		rmq.Close()
		s.Shutdown(ctx)
	}()

	log.Printf("Server started on port %s", cfg.Port)
	log.Fatal(s.ListenAndServe())
}
