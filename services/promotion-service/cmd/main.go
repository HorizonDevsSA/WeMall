package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	promotionv1 "github.com/wemall/gen/promotion/v1"
	"github.com/wemall/pkg/grpcutil"

	"github.com/wemall/promotion-service/internal/config"
	"github.com/wemall/promotion-service/internal/db"
	"github.com/wemall/promotion-service/internal/handler"
	"github.com/wemall/promotion-service/internal/service"
)

func main() {
	// Initialize logger
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	log.Info().Msg("starting promotion-service")

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

	// Initialize database queries
	queries := db.New(dbPool)

	// Initialize services
	promotionSvc := service.NewPromotionService(queries, dbPool)

	// Initialize gRPC server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log.Logger)...)

	// Register handlers
	promotionHandler := handler.NewPromotionHandler(promotionSvc)
	promotionv1.RegisterPromotionServiceServer(grpcServer, promotionHandler)

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
