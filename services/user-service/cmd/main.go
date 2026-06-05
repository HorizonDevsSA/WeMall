package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	userv1 "github.com/wemall/gen/user/v1"
	"github.com/wemall/pkg/grpcutil"
	"github.com/wemall/pkg/logger"
	"github.com/wemall/user-service/internal/auth"
	"github.com/wemall/user-service/internal/config"
	"github.com/wemall/user-service/internal/db"
	"github.com/wemall/user-service/internal/handler"
	"github.com/wemall/user-service/internal/service"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New("user-service", cfg.Environment)
	log.Info().Msg("starting user-service...")

	// Connect to database
	dbPool, err := pgxpool.New(context.Background(), cfg.DBUrl)
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

	// Initialize JWT manager
	jwtManager := auth.New(cfg.JWTSecret, cfg.JWTRefreshSecret)

	// Initialize services
	authSvc := service.NewAuthService(queries, cfg, jwtManager)
	userSvc := service.NewUserService(queries, dbPool)

	// Initialize gRPC server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log)...)
	userHandler := handler.NewUserHandler(authSvc, userSvc)
	userv1.RegisterUserServiceServer(grpcServer, userHandler)

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
