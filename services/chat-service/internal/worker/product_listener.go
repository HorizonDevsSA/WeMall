package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
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
	sub *nats.Subscription
}

func NewProductListener(natsURL string, svc *service.ChatService) (*ProductListener, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}

	return &ProductListener{
		nc:  nc,
		svc: svc,
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

	// 1. Fetch the broadcast group for this seller
	thread, err := l.svc.GetBroadcastThreadForSeller(ctx, event.SellerID)
	if err != nil {
		// No broadcast group exists yet for this seller. We could auto-create it:
		log.Printf("No broadcast group for seller %s, creating one automatically...", event.SellerID)
		thread, err = l.svc.CreateBroadcastGroup(ctx, event.SellerID, "Store Updates")
		if err != nil {
			log.Printf("Failed to create broadcast group for seller %s: %v", event.SellerID, err)
			return
		}
	}

	// 2. Send the broadcast message
	content := fmt.Sprintf("New product available: %s!", event.Title)
	msgType := "MESSAGE_TYPE_PRODUCT"

	_, err = l.svc.SendMessage(
		ctx,
		thread.ID,
		event.SellerID, // Sender is the seller
		msgType,
		content,
		event.ImageURL,
		event.ProductID, // reference_id
	)

	if err != nil {
		log.Printf("Failed to send broadcast message for product %s: %v", event.ProductID, err)
		return
	}

	log.Printf("Broadcasted new product %s to followers of seller %s", event.ProductID, event.SellerID)
}

func (l *ProductListener) Close() {
	if l.sub != nil {
		_ = l.sub.Unsubscribe()
	}
	if l.nc != nil {
		l.nc.Close()
	}
}
