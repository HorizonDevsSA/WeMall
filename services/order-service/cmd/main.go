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

	orderv1 "github.com/wemall/gen/order/v1"
	productv1 "github.com/wemall/gen/product/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	"github.com/wemall/order-service/internal/config"
	"github.com/wemall/order-service/internal/db"
	"github.com/wemall/order-service/internal/handler"
	"github.com/wemall/order-service/internal/service"
	"github.com/wemall/pkg/grpcutil"
	"github.com/wemall/pkg/logger"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New("order-service", cfg.Environment)
	log.Info().Msg("starting order-service...")

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

	// Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to connect to NATS at %s, proceeding without NATS events", cfg.NatsURL)
	} else {
		defer nc.Close()
		log.Info().Msg("NATS connected successfully")
	}

	// Connect to product-service gRPC
	productConn, err := grpc.Dial(cfg.ProductServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to dial product-service at %s", cfg.ProductServiceAddr)
	}
	defer productConn.Close()
	productClient := productv1.NewProductServiceClient(productConn)
	log.Info().Msgf("connected to product-service at %s", cfg.ProductServiceAddr)

	// Connect to seller-service gRPC
	sellerConn, err := grpc.Dial(cfg.SellerServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to dial seller-service at %s", cfg.SellerServiceAddr)
	}
	defer sellerConn.Close()
	sellerClient := sellerv1.NewSellerServiceClient(sellerConn)
	log.Info().Msgf("connected to seller-service at %s", cfg.SellerServiceAddr)

	// Initialize services
	cartSvc := service.NewCartService(queries, productClient, sellerClient)
	orderSvc := service.NewOrderService(queries, dbPool, productClient, sellerClient, nc)

	// Initialize gRPC server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log)...)
	orderHandler := handler.NewOrderHandler(cartSvc, orderSvc)
	orderv1.RegisterOrderServiceServer(grpcServer, orderHandler)

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
