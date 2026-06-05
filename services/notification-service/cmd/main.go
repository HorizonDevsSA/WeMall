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

	notificationv1 "github.com/wemall/gen/notification/v1"
	"github.com/wemall/pkg/grpcutil"
	"github.com/wemall/pkg/logger"
	"github.com/wemall/notification-service/internal/config"
	"github.com/wemall/notification-service/internal/db"
	"github.com/wemall/notification-service/internal/handler"
	"github.com/wemall/notification-service/internal/providers/email"
	"github.com/wemall/notification-service/internal/providers/push"
	"github.com/wemall/notification-service/internal/queue"
)

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. Initialize logger
	log := logger.New("notification-service", cfg.Environment)
	log.Info().Msg("starting notification-service...")

	// 3. Connect to PostgreSQL
	dbPool, err := pgxpool.New(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer dbPool.Close()

	if err := dbPool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}
	log.Info().Msg("Database connected successfully")

	queries := db.New(dbPool)

	// 4. Connect to NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to connect to NATS at %s, proceeding without NATS event handler", cfg.NatsURL)
	} else {
		defer nc.Close()
		log.Info().Msg("NATS connected successfully")
	}

	// 5. Connect to Downstream Services (gRPC)
	userAddr := getEnv("USER_SERVICE_ADDR", "user-service:9001")
	sellerAddr := cfg.SellerServiceAddr
	orderAddr := getEnv("ORDER_SERVICE_ADDR", "order-service:9005")

	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	userConn, err := grpc.Dial(userAddr, dialOpts...)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to dial user-service at %s", userAddr)
	}
	defer userConn.Close()

	sellerConn, err := grpc.Dial(sellerAddr, dialOpts...)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to dial seller-service at %s", sellerAddr)
	}
	defer sellerConn.Close()

	orderConn, err := grpc.Dial(orderAddr, dialOpts...)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to dial order-service at %s", orderAddr)
	}
	defer orderConn.Close()

	log.Info().Msg("gRPC downstream clients dialled successfully")

	// 6. Initialize providers
	smtpProvider := email.NewSMTPProvider(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, "WeMall", "https://wemall.co.zw", log)
	fcmProvider, err := push.NewFCMProvider(cfg.FirebaseCredJSON, cfg.FirebaseCredPath, log)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize FCM Provider")
	}

	// 7. Initialize Asynq task queue client and worker
	queueClient := queue.NewClient(cfg.RedisURL)
	defer queueClient.Close()

	worker := queue.NewWorker(cfg.RedisURL, 10, dbPool, smtpProvider, fcmProvider, log)

	// Start worker in a separate goroutine
	go func() {
		if err := worker.Start(); err != nil {
			log.Fatal().Err(err).Msg("Background worker execution failed")
		}
	}()

	// 8. Initialize NATS handlers & Subscribe
	natsHandler := handler.NewNATSHandler(nc, queries, queueClient, userConn, sellerConn, orderConn, log)
	if err := natsHandler.Start(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to start NATS subscriber loop")
	}

	// 9. Initialize gRPC Server
	grpcServer := grpc.NewServer(grpcutil.UnaryServerOptions(log)...)
	grpcHandler := handler.NewGRPCHandler(queries)
	notificationv1.RegisterNotificationServiceServer(grpcServer, grpcHandler)

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatal().Err(err).Msgf("Failed to listen on port %s", cfg.GRPCPort)
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Info().Msg("Shutting down servers...")
		grpcServer.GracefulStop()
		worker.Shutdown()
		log.Info().Msg("Shutdown complete.")
	}()

	log.Info().Msgf("gRPC server listening on port %s", cfg.GRPCPort)
	if err := grpcServer.Serve(lis); err != nil && err != grpc.ErrServerStopped {
		log.Fatal().Err(err).Msg("gRPC server failed to serve")
	}
}

func getEnv(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
