package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	deliveryv1 "github.com/wemall/gen/delivery/v1"
	"github.com/wemall/delivery-service/internal/config"
	"github.com/wemall/delivery-service/internal/db"
	deliverygrpc "github.com/wemall/delivery-service/internal/grpc"
	"github.com/wemall/delivery-service/internal/handler"
	"github.com/wemall/delivery-service/internal/service"
	"github.com/wemall/pkg/grpcutil"
	"github.com/wemall/pkg/logger"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logger
	log := logger.New("delivery-service", cfg.Environment)
	log.Info().Msg("starting delivery-service...")

	// 3. Connect to PostgreSQL
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

	// 4. Connect to Redis
	rdbOpts, err := redis.ParseURL(cfg.RedisURL)
	var rdb *redis.Client
	if err != nil {
		// Try standard addr fallback if not full URL scheme
		rdb = redis.NewClient(&redis.Options{
			Addr: cfg.RedisURL,
		})
	} else {
		rdb = redis.NewClient(rdbOpts)
	}

	// Ping Redis
	redisCtx, redisCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer redisCancel()
	if err := rdb.Ping(redisCtx).Err(); err != nil {
		log.Warn().Err(err).Msgf("failed to ping Redis at %s, geo-matching features will fail", cfg.RedisURL)
	} else {
		log.Info().Msg("Redis connected successfully")
	}

	// 5. Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to connect to NATS at %s, proceeding without NATS event messaging", cfg.NatsURL)
	} else {
		defer nc.Close()
		log.Info().Msg("NATS connected successfully")
	}

	// 6. Initialize business logic layers
	deliverySvc := service.NewDeliveryService(queries, dbPool, rdb, nc, log)

	// 7. Start NATS JetStream subscribers
	natsCtx, natsCancel := context.WithCancel(context.Background())
	defer natsCancel()

	natsHandler := handler.NewNatsHandler(nc, deliverySvc, log)
	if err := natsHandler.Start(natsCtx); err != nil {
		log.Fatal().Err(err).Msg("failed to start NATS handlers")
	}

	// 8. Start HTTP webhook server (on port 8086) for 3PL status notifications
	webhookHandler := handler.NewWebhookHandler(queries, log)
	mux := http.NewServeMux()
	mux.Handle("/webhooks/3pl", webhookHandler)
	
	httpSrv := &http.Server{
		Addr:    ":8086",
		Handler: mux,
	}
	
	go func() {
		log.Info().Msg("HTTP Webhook server listening on :8086")
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP webhook server error")
		}
	}()

	// 9. Start gRPC server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log)...)
	deliveryHandler := deliverygrpc.NewDeliveryHandler(deliverySvc, queries)
	deliveryv1.RegisterDeliveryServiceServer(grpcServer, deliveryHandler)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to listen on gRPC port %s", cfg.GRPCPort)
	}

	// Graceful shutdown wiring
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Info().Msg("shutting down delivery-service...")
		
		natsCancel()
		
		ctx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = httpSrv.Shutdown(ctx)
		
		grpcServer.GracefulStop()
	}()

	log.Info().Msgf("gRPC server listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
		log.Fatal().Err(err).Msg("gRPC server failed")
	}
}
