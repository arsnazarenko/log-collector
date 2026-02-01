package app

import (
	"context"
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
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	oapi_middleware "github.com/oapi-codegen/nethttp-middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	rmq_consumer "github.com/arsnazarenko/log-collector/pkg/rabbitmq"
	"github.com/wagslane/go-rabbitmq"
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
		log.Printf("Received new msg from RabbitMQ: %v", d)
		return rabbitmq.Ack
	})

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

	// FIXME: create UseCase
	serverImpl := v1.NewLogServer(nil)
	server := gen.NewStrictHandler(serverImpl, []gen.StrictMiddlewareFunc{})

	r.Route("/", func(r chi.Router) {
		r.Use(oapi_middleware.OapiRequestValidator(swagger))
		r.Use(middleware.Logger)
		r.Use(middleware.Recoverer)
		gen.HandlerFromMux(server, r)
	})

	s := &http.Server{
		Handler: r,
		Addr:    net.JoinHostPort(cfg.HTTP.Host, cfg.HTTP.Port),
	}

	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		<-sigs
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3) // if shutdown execure more than timeout => interrupt by context cancel()
		defer cancel()
		rmq.Close()
		s.Shutdown(ctx)
	}()

	log.Printf("Server started on port %s", cfg.HTTP.Port)
	log.Fatal(s.ListenAndServe())
}
