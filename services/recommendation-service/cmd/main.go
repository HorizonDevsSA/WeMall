package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	recommendationv1 "github.com/wemall/gen/recommendation/v1"
	"github.com/wemall/pkg/grpcutil"

	"github.com/wemall/recommendation-service/internal/config"
	"github.com/wemall/recommendation-service/internal/db"
	"github.com/wemall/recommendation-service/internal/handler"
	"github.com/wemall/recommendation-service/internal/service"
	"github.com/wemall/recommendation-service/internal/worker"
)

func main() {
	// Initialize logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().Msg("starting recommendation-service")

	// Load config
	cfg := config.Load()

	// Connect to database
	dbPool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer dbPool.Close()

	// Ping database
	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}
	log.Info().Msg("database connected successfully")

	// Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Warn().Err(err).Msg("failed to connect to NATS, order listener disabled")
	} else {
		log.Info().Msg("NATS connected successfully")
		defer nc.Close()
	}

	// Initialize database queries
	queries := db.New(dbPool)

	// Start NATS worker
	orderListener := worker.NewOrderListener(nc, queries)
	orderListener.Start()

	// Initialize services
	recommendationSvc := service.NewRecommendationService(queries, dbPool)

	// Initialize gRPC server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log.Logger)...)

	// Register handlers
	recommendationHandler := handler.NewRecommendationHandler(recommendationSvc)
	recommendationv1.RegisterRecommendationServiceServer(grpcServer, recommendationHandler)

	// Start listening
	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to listen")
	}

	// Start server gracefully
	go func() {
		log.Info().Msgf("gRPC server listening on port %s", cfg.Port)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatal().Err(err).Msg("failed to serve")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("shutting down server...")

	grpcServer.GracefulStop()
	log.Info().Msg("server stopped")
}
