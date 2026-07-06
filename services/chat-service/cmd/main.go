package main

import (
	"context"
	"database/sql"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	chatv1 "github.com/wemall/gen/chat/v1"
	"github.com/wemall/chat-service/internal/config"
	"github.com/wemall/chat-service/internal/db"
	fb "github.com/wemall/chat-service/internal/firebase"
	"github.com/wemall/chat-service/internal/handler"
	"github.com/wemall/chat-service/internal/service"
	"github.com/wemall/chat-service/internal/worker"
)

func main() {
	cfg := config.Load()

	// Connect to Postgres using database/sql + lib/pq
	sqlDB, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to open database connection: %v\n", err)
	}
	defer sqlDB.Close()
	if err := sqlDB.PingContext(context.Background()); err != nil {
		log.Fatalf("Unable to reach database: %v\n", err)
	}

	queries := db.New(sqlDB)

	// Initialize Service and Handler
	chatService := service.NewChatService(queries)
	chatHandler := handler.NewChatHandler(chatService)

	// Initialize Firestore client for writing announcements
	var firestoreClient *fb.FirestoreClient
	firestoreClient, err = fb.NewFirestoreClient(context.Background(), cfg.FirebaseCredentialsFile)
	if err != nil {
		log.Printf("Warning: Failed to initialize Firestore client: %v", err)
	} else {
		defer firestoreClient.Close()
	}

	// Initialize product broadcast NATS worker
	productListener, err := worker.NewProductListener(cfg.NatsURL, chatService, firestoreClient)
	if err != nil {
		log.Printf("Warning: Failed to initialize product NATS listener: %v", err)
	} else {
		if err := productListener.Start(); err != nil {
			log.Printf("Warning: Failed to start product NATS listener: %v", err)
		} else {
			defer productListener.Close()
		}
	}

	// Initialize store-follow / coupon / promotion NATS worker
	eventListener, err := worker.NewEventListener(cfg.NatsURL, chatService, firestoreClient)
	if err != nil {
		log.Printf("Warning: Failed to initialize event NATS listener: %v", err)
	} else {
		if err := eventListener.Start(); err != nil {
			log.Printf("Warning: Failed to start event NATS listener: %v", err)
		} else {
			defer eventListener.Close()
		}
	}

	// Set up gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	chatv1.RegisterChatServiceServer(grpcServer, chatHandler)

	// Graceful shutdown
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

