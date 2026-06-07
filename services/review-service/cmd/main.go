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
	"google.golang.org/grpc/credentials/insecure"

	reviewv1 "github.com/wemall/gen/review/v1"
	"github.com/wemall/pkg/grpcutil"
	"github.com/wemall/pkg/logger"
	"github.com/wemall/review-service/internal/config"
	"github.com/wemall/review-service/internal/db"
	"github.com/wemall/review-service/internal/handler"
	"github.com/wemall/review-service/internal/service"
	"github.com/wemall/review-service/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New("review-service", cfg.Environment)
	log.Info().Msg("starting review-service...")

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

	// 3. Connect to Order Service via gRPC
	orderConn, err := grpc.Dial(cfg.OrderServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to dial order-service at %s", cfg.OrderServiceAddr)
	}
	defer orderConn.Close()
	log.Info().Msgf("order-service connection dialed at %s", cfg.OrderServiceAddr)

	// 4. Instantiate Services
	queries := db.New(dbPool)
	reviewSvc := service.NewReviewService(queries, dbPool, nc)

	// 5. Start Background Workers & NATS Subscribers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	bgWorker := worker.NewWorker(nc, queries, dbPool, reviewSvc, orderConn, log)
	if err := bgWorker.Start(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to start background workers")
	}

	// 6. Setup gRPC Server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log)...)
	reviewv1.RegisterReviewServiceServer(grpcServer, handler.NewReviewHandler(reviewSvc))

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
