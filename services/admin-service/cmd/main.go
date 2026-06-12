package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/wemall/admin-service/internal/config"
	"github.com/wemall/admin-service/internal/db"
	"github.com/wemall/admin-service/internal/handler"
	"github.com/wemall/admin-service/internal/service"
	adminv1 "github.com/wemall/gen/admin/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	"github.com/wemall/pkg/grpcutil"
	"github.com/wemall/pkg/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New("admin-service", cfg.Environment)
	log.Info().Msg("starting admin-service...")

	// 1. Connect to admin database
	adminPool, err := pgxpool.New(context.Background(), cfg.DB_URL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to admin database")
	}
	defer adminPool.Close()

	if err := adminPool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to ping admin database")
	}
	log.Info().Msg("admin database connected")

	// 2. Connect to users database
	usersPool, err := pgxpool.New(context.Background(), cfg.UsersDBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to users database")
	}
	defer usersPool.Close()

	// 3. Connect to sellers database
	sellersPool, err := pgxpool.New(context.Background(), cfg.SellersDBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to sellers database")
	}
	defer sellersPool.Close()

	// 4. Connect to disputes database
	disputesPool, err := pgxpool.New(context.Background(), cfg.DisputesDBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to disputes database")
	}
	defer disputesPool.Close()

	// 5. Connect to orders database
	ordersPool, err := pgxpool.New(context.Background(), cfg.OrdersDBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to orders database")
	}
	defer ordersPool.Close()

	// 6. Connect to Seller Service via gRPC
	sellerConn, err := grpc.Dial(cfg.SellerServiceAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to dial seller-service at %s", cfg.SellerServiceAddr)
	}
	defer sellerConn.Close()
	sellerClient := sellerv1.NewSellerServiceClient(sellerConn)

	// 7. Instantiate Service & Handler
	queries := db.New(adminPool)
	adminSvc := service.NewAdminService(
		queries,
		adminPool,
		usersPool,
		sellersPool,
		disputesPool,
		ordersPool,
		sellerClient,
	)

	// 8. Start gRPC server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log)...)
	adminv1.RegisterAdminServiceServer(grpcServer, handler.NewAdminHandler(adminSvc))

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to listen on port %s", cfg.GRPCPort)
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Info().Msg("shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Info().Msgf("gRPC server listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal().Err(err).Msg("gRPC server failed")
	}
}
