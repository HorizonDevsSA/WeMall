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


	inventoryv1 "github.com/wemall/gen/inventory/v1"
	productv1 "github.com/wemall/gen/product/v1"
	"github.com/wemall/pkg/grpcutil"
	"github.com/wemall/pkg/logger"
	"github.com/wemall/product-service/internal/config"
	"github.com/wemall/product-service/internal/db"
	"github.com/wemall/product-service/internal/handler"
	"github.com/wemall/product-service/internal/service"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New("product-service", cfg.Environment)
	log.Info().Msg("starting product-service...")

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
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Warn().Err(err).Msg("failed to connect to NATS, event publishing will be disabled")
	} else {
		log.Info().Msg("NATS connected successfully")
		defer nc.Close()
	}

	// Initialize database queries
	queries := db.New(dbPool)

	// Initialize services
	productSvc := service.NewProductService(queries, dbPool, nc)

	// Initialize gRPC server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log)...)
	
	productHandler := handler.NewProductHandler(productSvc)
	productv1.RegisterProductServiceServer(grpcServer, productHandler)

	inventoryHandler := handler.NewInventoryHandler(productSvc)
	inventoryv1.RegisterInventoryServiceServer(grpcServer, inventoryHandler)

	// Listen on port
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to listen on port %s", cfg.GRPCPort)
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Info().Msg("shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Info().Msgf("gRPC server listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
		log.Fatal().Err(err).Msg("gRPC server failed")
	}
}
