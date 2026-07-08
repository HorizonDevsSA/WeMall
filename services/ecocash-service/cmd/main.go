package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"

	ecocashv1 "github.com/wemall/gen/ecocash/v1"
	"github.com/wemall/pkg/grpcutil"
	"github.com/wemall/pkg/logger"
	"github.com/wemall/ecocash-service/internal/config"
	"github.com/wemall/ecocash-service/internal/db"
	"github.com/wemall/ecocash-service/internal/gateway"
	"github.com/wemall/ecocash-service/internal/handler"
	"github.com/wemall/ecocash-service/internal/service"
	"github.com/wemall/ecocash-service/internal/worker"
)

func main() {
	// 1. Config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Logger (zerolog — pretty in dev, JSON in prod)
	log := logger.New("ecocash-service", cfg.Environment)
	log.Info().Msg("starting ecocash-service...")

	// 3. Database
	dbPool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}
	log.Info().Msg("database connected successfully")

	queries := db.New(dbPool)

	// 4. NATS (optional — service degrades gracefully without it)
	var nc *nats.Conn
	if cfg.NatsURL != "" {
		nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to connect to NATS at %s — proceeding without NATS", cfg.NatsURL)
		} else {
			log.Info().Msg("NATS connected successfully")
			defer nc.Close()
		}
	}

	// 5. EcoCash Gateway client
	// NotifyURL is the publicly reachable webhook endpoint; it is embedded in
	// every charge request so EcoCash knows where to POST status updates.
	webhookPath := getEnv("WEBHOOK_SECRET_PATH", "/webhooks/ecocash/callback")
	notifyURL := cfg.WebhookBaseURL + webhookPath

	gwClient := gateway.NewClient(gateway.GatewayConfig{
		BaseURL:        cfg.EcoCashBaseURL,
		Username:       cfg.EcoCashUsername,
		Password:       cfg.EcoCashPassword,   // not logged
		MerchantCode:   cfg.EcoCashMerchantCode,
		MerchantPin:    cfg.EcoCashMerchantPin, // not logged
		MerchantNumber: cfg.EcoCashMerchantNumber,
		TerminalID:     cfg.EcoCashTerminalID,
		CountryCode:    cfg.EcoCashCountryCode,
		MerchantName:   cfg.EcoCashMerchantName,
		SuperMerchant:  cfg.EcoCashSuperMerchant,
		NotifyURL:      notifyURL,
		ProxySecret:    cfg.EcoCashProxySecret,
	}, log)

	// 6. Service (use-case layer)
	svc := service.NewEcoCashService(queries, dbPool, gwClient, nc, notifyURL, log)

	// 7. Background workers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	reconciler := worker.NewReconciler(queries, svc, log)
	outboxRelay := worker.NewOutboxRelay(queries, nc, log)

	go reconciler.Start(ctx)
	go outboxRelay.Start(ctx)

	// 8. HTTP server for EcoCash webhook endpoint
	webhookHandler := handler.NewWebhookHandler(svc, log)
	mux := http.NewServeMux()
	mux.Handle(webhookPath, webhookHandler)

	httpPort := getEnv("HTTP_PORT", "8018")
	httpServer := &http.Server{
		Addr:    ":" + httpPort,
		Handler: mux,
	}
	go func() {
		log.Info().Msgf("webhook HTTP server listening on port %s", httpPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("webhook HTTP server failed")
		}
	}()

	// 9. gRPC server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log)...)
	ecocashHandler := handler.NewEcoCashHandler(svc)
	ecocashv1.RegisterEcoCashServiceServer(grpcServer, ecocashHandler)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to listen on port %s", cfg.GRPCPort)
	}

	// 10. Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Info().Msg("shutting down ecocash-service...")
		cancel() // stop reconciler and outbox relay
		grpcServer.GracefulStop()
		_ = httpServer.Shutdown(context.Background())
	}()

	log.Info().Msgf("gRPC server listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
		log.Fatal().Err(err).Msg("gRPC server failed")
	}
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
