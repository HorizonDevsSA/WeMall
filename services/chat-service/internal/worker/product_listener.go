package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	fb "github.com/wemall/chat-service/internal/firebase"
	"github.com/wemall/chat-service/internal/service"
)

type ProductCreatedEvent struct {
	ProductID string `json:"product_id"`
	SellerID  string `json:"seller_id"`
	Title     string `json:"title"`
	ImageURL  string `json:"image_url"`
}

type ProductListener struct {
	nc  *nats.Conn
	svc *service.ChatService
	fc  *fb.FirestoreClient
	sub *nats.Subscription
}

func NewProductListener(natsURL string, svc *service.ChatService, fc *fb.FirestoreClient) (*ProductListener, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	return &ProductListener{
		nc:  nc,
		svc: svc,
		fc:  fc,
	}, nil
}

func (l *ProductListener) Start() error {
	sub, err := l.nc.Subscribe("wemall.product.created", func(msg *nats.Msg) {
		var event ProductCreatedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal wemall.product.created event: %v", err)
			return
		}

		l.handleProductCreated(event)
	})
	if err != nil {
		return err
	}
	l.sub = sub
	log.Println("Subscribed to wemall.product.created events")
	return nil
}

func (l *ProductListener) handleProductCreated(event ProductCreatedEvent) {
	ctx := context.Background()

	// 1. Fetch or auto-create the broadcast group for this seller
	thread, err := l.svc.GetBroadcastThreadForSeller(ctx, event.SellerID)
	if err != nil {
		log.Printf("No broadcast group for seller %s, creating one automatically...", event.SellerID)
		thread, err = l.svc.CreateBroadcastGroup(ctx, event.SellerID, "Store Updates", "")
		if err != nil {
			log.Printf("Failed to create broadcast group for seller %s: %v", event.SellerID, err)
			return
		}
	}

	// 2. Build rich metadata JSON
	meta := map[string]interface{}{
		"product_id": event.ProductID,
		"title":      event.Title,
		"image_url":  event.ImageURL,
	}
	metaBytes, _ := json.Marshal(meta)

	content := fmt.Sprintf("New product available: %s!", event.Title)

	_, err = l.svc.SendMessage(
		ctx,
		thread.ID,
		event.SellerID,      // sender is the seller
		"MESSAGE_TYPE_PRODUCT",
		content,
		event.ImageURL,      // media_url
		event.ProductID,     // reference_id
		metaBytes,           // metadata JSON
	)
	if err != nil {
		log.Printf("Failed to send broadcast message for product %s: %v", event.ProductID, err)
		return
	}

	log.Printf("Broadcasted new product %s to followers of seller %s", event.ProductID, event.SellerID)

	// 3. Write to Firestore for push notification trigger
	if l.fc != nil {
		firestoreThreadID := fmt.Sprintf("group_%s", event.SellerID)

		// Ensure thread document exists in Firestore
		_ = l.fc.EnsureThreadExists(ctx, firestoreThreadID, "GROUP", "Store Updates", event.SellerID, []string{event.SellerID})

		msgData := map[string]interface{}{
			"senderId":    event.SellerID,
			"type":        "PRODUCT",
			"content":     content,
			"mediaUrl":    event.ImageURL,
			"referenceId": event.ProductID,
			"metadata": map[string]string{
				"product_id": event.ProductID,
				"title":      event.Title,
				"image_url":  event.ImageURL,
			},
		}

		docID, writeErr := l.fc.WriteMessage(ctx, firestoreThreadID, msgData)
		if writeErr != nil {
			log.Printf("[firestore] Failed to write product announcement: %v", writeErr)
		} else {
			log.Printf("[firestore] Wrote product announcement %s to thread %s", docID, firestoreThreadID)
		}
	}
}


func (l *ProductListener) Close() {
	if l.sub != nil {
		_ = l.sub.Unsubscribe()
	}
	if l.nc != nil {
		l.nc.Close()
	}
}

