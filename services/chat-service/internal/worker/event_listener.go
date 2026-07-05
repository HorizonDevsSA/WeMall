package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/wemall/chat-service/internal/service"
)

// ──────────────────────── Event payload types ─────────────────────────────

type StoreFollowedEvent struct {
	FollowerID string `json:"follower_id"`
	SellerID   string `json:"seller_id"`
}

type StoreUnfollowedEvent struct {
	FollowerID string `json:"follower_id"`
	SellerID   string `json:"seller_id"`
}

type CouponCreatedEvent struct {
	CouponID   string  `json:"coupon_id"`
	SellerID   string  `json:"seller_id"`
	Code       string  `json:"code"`
	Discount   float64 `json:"discount"`
	ExpiresAt  string  `json:"expires_at"`
}

type PromotionCreatedEvent struct {
	PromotionID string  `json:"promotion_id"`
	SellerID    string  `json:"seller_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Discount    float64 `json:"discount"`
	BannerURL   string  `json:"banner_url"`
}

// ──────────────────────── EventListener ──────────────────────────────────

// EventListener subscribes to store-follow/unfollow, coupon, and promotion events
// and dispatches them into the chat system.
type EventListener struct {
	nc   *nats.Conn
	svc  *service.ChatService
	subs []*nats.Subscription
}

func NewEventListener(natsURL string, svc *service.ChatService) (*EventListener, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, err
	}
	return &EventListener{nc: nc, svc: svc}, nil
}

func (l *EventListener) Start() error {
	subscriptions := []struct {
		subject string
		handler func([]byte)
	}{
		{"wemall.store.followed", l.handleStoreFollowed},
		{"wemall.store.unfollowed", l.handleStoreUnfollowed},
		{"wemall.coupon.created", l.handleCouponCreated},
		{"wemall.promotion.created", l.handlePromotionCreated},
	}

	for _, s := range subscriptions {
		s := s // capture range var
		sub, err := l.nc.Subscribe(s.subject, func(msg *nats.Msg) {
			s.handler(msg.Data)
		})
		if err != nil {
			return fmt.Errorf("failed to subscribe to %s: %w", s.subject, err)
		}
		l.subs = append(l.subs, sub)
		log.Printf("Subscribed to %s", s.subject)
	}
	return nil
}

func (l *EventListener) Close() {
	for _, sub := range l.subs {
		_ = sub.Unsubscribe()
	}
	if l.nc != nil {
		l.nc.Close()
	}
}

// ──────────────────────── Handlers ───────────────────────────────────────

// handleStoreFollowed: adds the buyer to the seller's broadcast group.
func (l *EventListener) handleStoreFollowed(data []byte) {
	var event StoreFollowedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("[event_listener] Failed to parse store.followed: %v", err)
		return
	}
	ctx := context.Background()
	if err := l.svc.JoinBroadcastGroup(ctx, event.FollowerID, event.SellerID); err != nil {
		log.Printf("[event_listener] Failed to join broadcast group (follower=%s seller=%s): %v",
			event.FollowerID, event.SellerID, err)
		return
	}
	log.Printf("[event_listener] User %s joined broadcast group for seller %s", event.FollowerID, event.SellerID)
}

// handleStoreUnfollowed: removes the buyer from the seller's broadcast group.
func (l *EventListener) handleStoreUnfollowed(data []byte) {
	var event StoreUnfollowedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("[event_listener] Failed to parse store.unfollowed: %v", err)
		return
	}
	ctx := context.Background()
	if err := l.svc.LeaveBroadcastGroup(ctx, event.FollowerID, event.SellerID); err != nil {
		log.Printf("[event_listener] Failed to leave broadcast group (follower=%s seller=%s): %v",
			event.FollowerID, event.SellerID, err)
		return
	}
	log.Printf("[event_listener] User %s left broadcast group for seller %s", event.FollowerID, event.SellerID)
}

// handleCouponCreated: broadcasts a coupon message to the seller's followers.
func (l *EventListener) handleCouponCreated(data []byte) {
	var event CouponCreatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("[event_listener] Failed to parse coupon.created: %v", err)
		return
	}
	ctx := context.Background()

	thread, err := l.svc.GetBroadcastThreadForSeller(ctx, event.SellerID)
	if err != nil {
		thread, err = l.svc.CreateBroadcastGroup(ctx, event.SellerID, "Store Updates", "")
		if err != nil {
			log.Printf("[event_listener] Cannot get/create broadcast group for coupon: %v", err)
			return
		}
	}

	meta := map[string]interface{}{
		"coupon_id":  event.CouponID,
		"code":       event.Code,
		"discount":   event.Discount,
		"expires_at": event.ExpiresAt,
	}
	metaBytes, _ := json.Marshal(meta)

	content := fmt.Sprintf("🎟️ New Coupon! Use code %s to get %.0f%% off!", event.Code, event.Discount)
	_, err = l.svc.SendMessage(ctx, thread.ID, event.SellerID,
		"MESSAGE_TYPE_COUPON", content, "", event.CouponID, metaBytes)
	if err != nil {
		log.Printf("[event_listener] Failed to broadcast coupon %s: %v", event.CouponID, err)
		return
	}
	log.Printf("[event_listener] Broadcasted coupon %s to followers of seller %s", event.CouponID, event.SellerID)
}

// handlePromotionCreated: broadcasts a promotion message to the seller's followers.
func (l *EventListener) handlePromotionCreated(data []byte) {
	var event PromotionCreatedEvent
	if err := json.Unmarshal(data, &event); err != nil {
		log.Printf("[event_listener] Failed to parse promotion.created: %v", err)
		return
	}
	ctx := context.Background()

	thread, err := l.svc.GetBroadcastThreadForSeller(ctx, event.SellerID)
	if err != nil {
		thread, err = l.svc.CreateBroadcastGroup(ctx, event.SellerID, "Store Updates", "")
		if err != nil {
			log.Printf("[event_listener] Cannot get/create broadcast group for promotion: %v", err)
			return
		}
	}

	meta := map[string]interface{}{
		"promotion_id": event.PromotionID,
		"title":        event.Title,
		"description":  event.Description,
		"discount":     event.Discount,
		"banner_url":   event.BannerURL,
	}
	metaBytes, _ := json.Marshal(meta)

	content := fmt.Sprintf("🔥 %s — Get %.0f%% off! %s", event.Title, event.Discount, event.Description)
	_, err = l.svc.SendMessage(ctx, thread.ID, event.SellerID,
		"MESSAGE_TYPE_PROMOTION", content, event.BannerURL, event.PromotionID, metaBytes)
	if err != nil {
		log.Printf("[event_listener] Failed to broadcast promotion %s: %v", event.PromotionID, err)
		return
	}
	log.Printf("[event_listener] Broadcasted promotion %s to followers of seller %s", event.PromotionID, event.SellerID)
}
