package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"

	paymentv1 "github.com/wemall/gen/payment/v1"
	"github.com/wemall/pkg/grpcutil"
	"github.com/wemall/pkg/logger"
	"github.com/wemall/payment-service/internal/config"
	"github.com/wemall/payment-service/internal/db"
	"github.com/wemall/payment-service/internal/handler"
	"github.com/wemall/payment-service/internal/service"
	"github.com/wemall/payment-service/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New("payment-service", cfg.Environment)
	log.Info().Msg("starting payment-service...")

	// 1. Connect to database
	dbPool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}
	log.Info().Msg("database connected successfully")

	// 2. Connect to NATS
	var nc *nats.Conn
	if cfg.NatsURL != "" {
		nc, err = nats.Connect(cfg.NatsURL)
		if err != nil {
			log.Warn().Err(err).Msgf("failed to connect to NATS at %s, proceeding without NATS", cfg.NatsURL)
		} else {
			log.Info().Msg("NATS connected successfully")
			defer nc.Close()
		}
	}

	// 3. Instantiate Services
	queries := db.New(dbPool)
	paymentSvc := service.NewPaymentService(queries, dbPool, nc, cfg.StripeSecretKey, cfg.GooglePayMerchantID)

	// 4. Start Background Workers & NATS Subscribers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bgWorker := worker.NewWorker(nc, queries, paymentSvc, log)
	if err := bgWorker.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start background workers")
	}

	// 5. Setup gRPC Server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log)...)
	paymentv1.RegisterPaymentServiceServer(grpcServer, handler.NewPaymentHandler(paymentSvc))

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to listen on port %s", cfg.GRPCPort)
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Info().Msg("shutting down gRPC server...")
		cancel() // Stop background workers
		grpcServer.GracefulStop()
	}()

	log.Info().Msgf("gRPC server listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal().Err(err).Msg("gRPC server failed")
	}
}
