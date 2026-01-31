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

	// "github.com/arsnazarenko/log-collector/pkg/clickhouse"
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

	// ch, err := clickhouse.New(cfg.Clickhouse)
	// if err != nil {
	// 	log.Fatalf("Error connecting to jjkkclickhouse: %s", err)
	// }
	// defer func() {
	// 	_ = ch.Close()
	// }()

	r := chi.NewRouter()
	// create PlayerServer
	serverImpl := v1.NewLogServer(nil)

	server := gen.NewStrictHandler(serverImpl, []gen.StrictMiddlewareFunc{})

	// cors
	corsHandler := cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
	// cors middleware
	r.Use(corsHandler)
	// openapi validation middleware
	r.Use(oapi_middleware.OapiRequestValidator(swagger))
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// metrics middleware
	// r.Use(metrics.HTTPMetricsMiddleware())

	gen.HandlerFromMux(server, r)
	s := &http.Server{
		Handler: r,
		Addr:    net.JoinHostPort(cfg.HTTP.Host, cfg.HTTP.Port),
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go runSwaggerServer(corsHandler)
	go func() {
		<-signals
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		rmq.Close()
		s.Shutdown(ctx)
	}()

	log.Printf("Server started on port %s", cfg.HTTP.Port)
	log.Fatal(s.ListenAndServe())
}

func runSwaggerServer(middleware func(http.Handler) http.Handler) {
	r := chi.NewMux()
	r.Use(middleware)
	r.Handle("/swagger/*", http.StripPrefix("/swagger/", http.FileServer(http.FS(gen.SwaggerUI))))

	swagerServer := &http.Server{
		Handler: r,
		Addr:    net.JoinHostPort("0.0.0.0", "8081"),
	}
	log.Printf("Swagger static server started on port %s", "8081")
	log.Fatal(swagerServer.ListenAndServe())
}

func RunMetricsServer(middleware http.Handler) {
	// mr := chi.NewMux()
	// mr.Use(corsHandler)
	// mr.Use(middleware.Logger)
	// mr.Handle("/metrics", promhttp.Handler())

	// ms := &http.Server{
	// 	Handler: mr,
	// 	Addr:    net.JoinHostPort(cfg.Host, cfg.Port),
	// }
	// log.Printf("Metrics server started on port %s", cfg.Port)
	// log.Fatal(ms.ListenAndServe())
}
