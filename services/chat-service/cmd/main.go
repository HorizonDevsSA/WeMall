package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"

	chatv1 "github.com/wemall/gen/chat/v1"
	"github.com/wemall/chat-service/internal/config"
	"github.com/wemall/chat-service/internal/db"
	"github.com/wemall/chat-service/internal/handler"
	"github.com/wemall/chat-service/internal/service"
	"github.com/wemall/chat-service/internal/worker"
)

func main() {
	cfg := config.Load()

	// Connect to Database
	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	queries := db.New(dbPool)

	// Initialize Service and Handler
	chatService := service.NewChatService(queries)
	chatHandler := handler.NewChatHandler(chatService)

	// Initialize NATS worker
	productListener, err := worker.NewProductListener(cfg.NatsURL, chatService)
	if err != nil {
		log.Printf("Failed to initialize NATS listener: %v", err)
	} else {
		if err := productListener.Start(); err != nil {
			log.Printf("Failed to start NATS listener: %v", err)
		} else {
			defer productListener.Close()
		}
	}

	// Set up gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	chatv1.RegisterChatServiceServer(grpcServer, chatHandler)

	// Handle graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Chat Service listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
