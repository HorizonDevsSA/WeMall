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
	"google.golang.org/grpc"

	mediav1 "github.com/wemall/gen/media/v1"
	"github.com/wemall/pkg/auth"
	"github.com/wemall/pkg/grpcutil"
	"github.com/wemall/pkg/logger"
	"github.com/wemall/media-service/internal/config"
	"github.com/wemall/media-service/internal/db"
	"github.com/wemall/media-service/internal/handler"
	"github.com/wemall/media-service/internal/service"
)

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logger
	log := logger.New("media-service", cfg.Environment)
	log.Info().Msg("starting WeMall media-service...")

	// 3. Connect to database
	dbPool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}
	log.Info().Msg("database connected successfully")

	// 4. Connect to NATS (Optional)
	var nc *nats.Conn
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	nc, err = nats.Connect(natsURL)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to connect to NATS at %s, proceeding without events", natsURL)
	} else {
		log.Info().Msg("NATS connected successfully")
		defer nc.Close()
	}

	// 5. Initialize JWT Auth Manager
	accessSecret := os.Getenv("JWT_SECRET")
	if accessSecret == "" {
		accessSecret = os.Getenv("ACCESS_SECRET")
	}
	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		refreshSecret = os.Getenv("REFRESH_SECRET")
	}
	var authMgr *auth.Manager
	if accessSecret != "" {
		authMgr = auth.New(auth.Config{
			AccessSecret:  accessSecret,
			RefreshSecret: refreshSecret,
		})
		log.Info().Msg("auth manager loaded successfully")
	} else {
		log.Warn().Msg("JWT_SECRET not found in environment, auth verification will be bypassed")
	}

	// 6. Initialize database queries
	queries := db.New(dbPool)

	// 7. Initialize business services
	mediaSvc := service.NewMediaService(cfg, log, queries, dbPool, nc)

	// 8. Initialize REST HTTP Server
	restMux := http.NewServeMux()
	restHandler := handler.NewMediaRestHandler(mediaSvc, authMgr)
	restHandler.RegisterRoutes(restMux)

	httpServer := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: restMux,
	}

	go func() {
		log.Info().Msgf("REST HTTP server listening on port %s", cfg.HTTPPort)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("REST HTTP server failed")
		}
	}()

	// 9. Initialize gRPC Server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log)...)
	grpcHandler := handler.NewMediaGrpcHandler(mediaSvc)
	mediav1.RegisterMediaServiceServer(grpcServer, grpcHandler)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to listen on port %s", cfg.GRPCPort)
	}

	// 10. Graceful shutdown coordination
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan
		log.Info().Msg("shutting down servers gracefully...")

		// Shutdown gRPC
		grpcServer.GracefulStop()

		// Shutdown HTTP
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		httpServer.Shutdown(ctx)

		log.Info().Msg("all servers stopped.")
	}()

	log.Info().Msgf("gRPC server listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
		log.Fatal().Err(err).Msg("gRPC server failed")
	}
}
