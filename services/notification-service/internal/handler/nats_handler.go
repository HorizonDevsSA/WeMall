package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	orderv1 "github.com/wemall/gen/order/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	userv1 "github.com/wemall/gen/user/v1"
	"github.com/wemall/notification-service/internal/db"
	"github.com/wemall/notification-service/internal/providers/email/templates"
	"github.com/wemall/notification-service/internal/queue"
)

type QueueClient interface {
	EnqueueEmail(ctx context.Context, payload queue.EmailSendPayload) error
	EnqueuePush(ctx context.Context, payload queue.PushSendPayload) error
	EnqueuePushMulticast(ctx context.Context, payload queue.PushMulticastPayload) error
}

type NATSHandler struct {
	nc           *nats.Conn
	q            db.Querier
	queueClient  QueueClient
	userClient   userv1.UserServiceClient
	sellerClient sellerv1.SellerServiceClient
	orderClient  orderv1.OrderServiceClient
	logger       zerolog.Logger
}

func NewNATSHandler(
	nc *nats.Conn,
	queries db.Querier,
	qc QueueClient,
	userConn, sellerConn, orderConn *grpc.ClientConn,
	logger zerolog.Logger,
) *NATSHandler {
	return &NATSHandler{
		nc:           nc,
		q:            queries,
		queueClient:  qc,
		userClient:   userv1.NewUserServiceClient(userConn),
		sellerClient: sellerv1.NewSellerServiceClient(sellerConn),
		orderClient:  orderv1.NewOrderServiceClient(orderConn),
		logger:       logger,
	}
}

func (h *NATSHandler) Start(ctx context.Context) error {
	if h.nc == nil {
		h.logger.Warn().Msg("NATS connection is nil, NATSHandler subscriber loop bypassed")
		return nil
	}

	subscriptions := map[string]nats.MsgHandler{
		"wemall.user.registered":      h.handleUserRegistered,
		"wemall.user.password_reset":  h.handleUserPasswordReset,
		"wemall.user.password_changed": h.handleUserPasswordChanged,
		"wemall.order.created":        h.handleOrderCreated,
		"wemall.payment.completed":    h.handlePaymentCompleted,
		"wemall.payment.failed":       h.handlePaymentFailed,
		"wemall.order.shipped":        h.handleOrderShipped,
		"wemall.order.delivered":      h.handleOrderDelivered,
		"wemall.payment.refunded":     h.handlePaymentRefunded,
		"wemall.seller.status_changed": h.handleSellerStatusChanged,
		"wemall.inventory.low_stock":  h.handleInventoryLowStock,
		"wemall.store.post_update":    h.handleStorePostUpdate,
		"wemall.product.price_dropped": h.handleProductPriceDropped,
		"wemall.inventory.restocked":  h.handleInventoryRestocked,
	}

	for subject, handlerFunc := range subscriptions {
		_, err := h.nc.Subscribe(subject, handlerFunc)
		if err != nil {
			h.logger.Error().Err(err).Str("subject", subject).Msg("Failed to subscribe to NATS subject")
			return err
		}
		h.logger.Info().Str("subject", subject).Msg("Successfully subscribed to NATS subject")
	}

	return nil
}

// ── Check Preference Helpers ──────────────────────────────────────────────────

func (h *NATSHandler) isChannelEnabled(ctx context.Context, userID uuid.UUID, category string, channel string) (bool, error) {
	var dbCategory db.NotificationCategory
	switch category {
	case "transactional":
		dbCategory = db.NotificationCategoryTransactional
	case "security":
		dbCategory = db.NotificationCategorySecurity
	case "low_stock":
		dbCategory = db.NotificationCategoryLowStock
	case "follows":
		dbCategory = db.NotificationCategoryFollows
	case "marketing":
		dbCategory = db.NotificationCategoryMarketing
	default:
		return true, nil // default opt-in
	}

	pref, err := h.q.GetNotificationPreference(ctx, db.GetNotificationPreferenceParams{
		UserID:   userID,
		Category: dbCategory,
	})
	if err != nil {
		return true, nil // Default is true if no record exists
	}

	if channel == "email" {
		return pref.EmailEnabled, nil
	}
	return pref.PushEnabled, nil
}

// ── Dispatch Helpers ─────────────────────────────────────────────────────────

func (h *NATSHandler) queueEmail(ctx context.Context, userID uuid.UUID, category, emailAddr, name, subject, body string) {
	enabled, err := h.isChannelEnabled(ctx, userID, category, "email")
	if err != nil || !enabled {
		return
	}
	_ = h.queueClient.EnqueueEmail(ctx, queue.EmailSendPayload{
		UserID:        userID.String(),
		Category:      category,
		Recipient:     emailAddr,
		RecipientName: name,
		Subject:       subject,
		HTMLBody:      body,
	})
}

func (h *NATSHandler) queuePush(ctx context.Context, userID uuid.UUID, category, title, body string, data map[string]string) {
	enabled, err := h.isChannelEnabled(ctx, userID, category, "push")
	if err != nil || !enabled {
		return
	}

	tokens, err := h.q.GetDeviceTokensByUser(ctx, userID)
	if err != nil || len(tokens) == 0 {
		return
	}

	for _, token := range tokens {
		_ = h.queueClient.EnqueuePush(ctx, queue.PushSendPayload{
			UserID:   userID.String(),
			Category: category,
			Token:    token.Token,
			Title:    title,
			Body:     body,
			Data:     data,
		})
	}
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (h *NATSHandler) handleUserRegistered(msg *nats.Msg) {
	var event struct {
		UserID    string `json:"user_id"`
		FullName  string `json:"full_name"`
		Email     string `json:"email"`
		VerifyURL string `json:"verify_url"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		h.logger.Error().Err(err).Msg("Failed to unmarshal registered event")
		return
	}

	uid, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	body, err := templates.RenderTemplate(
		templates.WelcomeTemplate,
		event.FullName,
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{"VerifyURL": event.VerifyURL},
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to render welcome template")
		return
	}

	h.queueEmail(context.Background(), uid, "security", event.Email, event.FullName, "Welcome to WeMall! 🚀", body)
}

func (h *NATSHandler) handleUserPasswordReset(msg *nats.Msg) {
	var event struct {
		UserID   string `json:"user_id"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		ResetURL string `json:"reset_url"`
		Expiry   string `json:"expiry"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		h.logger.Error().Err(err).Msg("Failed to unmarshal reset event")
		return
	}

	uid, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	body, err := templates.RenderTemplate(
		templates.PasswordResetTemplate,
		event.FullName,
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"ResetURL": event.ResetURL,
			"Expiry":   event.Expiry,
		},
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to render password reset template")
		return
	}

	h.queueEmail(context.Background(), uid, "security", event.Email, event.FullName, "Reset Your Password 🔐", body)
}

func (h *NATSHandler) handleUserPasswordChanged(msg *nats.Msg) {
	var event struct {
		UserID    string `json:"user_id"`
		FullName  string `json:"full_name"`
		Email     string `json:"email"`
		Device    string `json:"device"`
		IPAddress string `json:"ip_address"`
		Time      string `json:"time"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		h.logger.Error().Err(err).Msg("Failed to unmarshal password changed event")
		return
	}

	uid, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	body, err := templates.RenderTemplate(
		templates.PasswordChangedTemplate,
		event.FullName,
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"Device":    event.Device,
			"IPAddress": event.IPAddress,
			"Time":      event.Time,
		},
	)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to render password changed template")
		return
	}

	// Email + Push
	h.queueEmail(context.Background(), uid, "security", event.Email, event.FullName, "Security Alert: Password Changed", body)
	h.queuePush(context.Background(), uid, "security", "Security Alert", "Your password was successfully changed.", map[string]string{
		"action": "security_check",
	})
}

func (h *NATSHandler) handleOrderCreated(msg *nats.Msg) {
	var event struct {
		OrderID     string  `json:"order_id"`
		OrderNumber string  `json:"order_number"`
		UserID      string  `json:"user_id"`
		Total       float64 `json:"total"`
		Currency    string  `json:"currency"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	buyerID, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	// 1. Send push to buyer
	h.queuePush(context.Background(), buyerID, "transactional", "Order Received", "We have received order "+event.OrderNumber+". Complete your payment.", map[string]string{
		"order_id": event.OrderID,
		"status":   "pending",
	})

	// 2. Fetch order items to alert sellers
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		order, err := h.orderClient.GetOrder(ctx, &orderv1.GetOrderRequest{
			Id:     event.OrderID,
			UserId: event.UserID,
		})
		if err != nil {
			h.logger.Error().Err(err).Str("order_id", event.OrderID).Msg("Failed to fetch order details for sellers notification")
			return
		}

		uniqueSellers := make(map[string]bool)
		for _, item := range order.Items {
			uniqueSellers[item.SellerId] = true
		}

		for sellerIDStr := range uniqueSellers {
			seller, err := h.sellerClient.GetSeller(ctx, &sellerv1.GetSellerRequest{Id: sellerIDStr})
			if err != nil {
				continue
			}

			sellerUID, err := uuid.Parse(seller.UserId)
			if err == nil {
				h.queuePush(ctx, sellerUID, "transactional", "New Order Received", "You received a new order: "+event.OrderNumber, map[string]string{
					"order_id":     event.OrderID,
					"order_number": event.OrderNumber,
				})
			}
		}
	}()
}

func (h *NATSHandler) handlePaymentCompleted(msg *nats.Msg) {
	var event struct {
		OrderID     string  `json:"order_id"`
		OrderNumber string  `json:"order_number"`
		UserID      string  `json:"user_id"`
		Total       float64 `json:"total"`
		Currency    string  `json:"currency"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	buyerUID, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	ctx := context.Background()

	// Get Buyer Details via gRPC
	uResp, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: event.UserID})
	if err != nil {
		h.logger.Error().Err(err).Str("user_id", event.UserID).Msg("Failed to fetch user profile for payment complete mail")
		return
	}

	body, err := templates.RenderTemplate(
		templates.PaymentCompletedTemplate,
		uResp.FullName,
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"OrderNumber": event.OrderNumber,
			"Total":       event.Total,
			"Currency":    event.Currency,
		},
	)
	if err != nil {
		return
	}

	// Email receipt + Push
	h.queueEmail(ctx, buyerUID, "transactional", uResp.Email, uResp.FullName, "Order Receipt: "+event.OrderNumber, body)
	h.queuePush(ctx, buyerUID, "transactional", "Payment Successful", "Thank you! Payment for "+event.OrderNumber+" was processed successfully.", map[string]string{
		"order_id": event.OrderID,
		"status":   "confirmed",
	})
}

func (h *NATSHandler) handlePaymentFailed(msg *nats.Msg) {
	var event struct {
		OrderID     string `json:"order_id"`
		OrderNumber string `json:"order_number"`
		UserID      string `json:"user_id"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	buyerUID, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	h.queuePush(context.Background(), buyerUID, "transactional", "Payment Failed", "Payment for order "+event.OrderNumber+" failed. Tap to retry.", map[string]string{
		"order_id": event.OrderID,
		"retry":    "true",
	})
}

func (h *NATSHandler) handleOrderShipped(msg *nats.Msg) {
	var event struct {
		OrderID        string `json:"order_id"`
		OrderNumber    string `json:"order_number"`
		UserID         string `json:"user_id"`
		Carrier        string `json:"carrier"`
		TrackingNumber string `json:"tracking_number"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	buyerUID, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	ctx := context.Background()

	uResp, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: event.UserID})
	if err != nil {
		return
	}

	body, err := templates.RenderTemplate(
		templates.OrderShippedTemplate,
		uResp.FullName,
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"OrderNumber":    event.OrderNumber,
			"Carrier":        event.Carrier,
			"TrackingNumber": event.TrackingNumber,
		},
	)
	if err != nil {
		return
	}

	h.queueEmail(ctx, buyerUID, "transactional", uResp.Email, uResp.FullName, "Your order has been shipped! 🚚", body)
	h.queuePush(ctx, buyerUID, "transactional", "Order Shipped", "Your order "+event.OrderNumber+" is on the way via "+event.Carrier+".", map[string]string{
		"order_id":        event.OrderID,
		"tracking_number": event.TrackingNumber,
	})
}

func (h *NATSHandler) handleOrderDelivered(msg *nats.Msg) {
	var event struct {
		OrderID     string `json:"order_id"`
		OrderNumber string `json:"order_number"`
		UserID      string `json:"user_id"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	buyerUID, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	h.queuePush(context.Background(), buyerUID, "transactional", "Order Delivered", "Your order "+event.OrderNumber+" was delivered. Leave a review!", map[string]string{
		"order_id": event.OrderID,
	})
}

func (h *NATSHandler) handlePaymentRefunded(msg *nats.Msg) {
	var event struct {
		OrderID      string  `json:"order_id"`
		OrderNumber  string  `json:"order_number"`
		UserID       string  `json:"user_id"`
		RefundAmount float64 `json:"refund_amount"`
		ETA          string  `json:"eta"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	buyerUID, err := uuid.Parse(event.UserID)
	if err != nil {
		return
	}

	ctx := context.Background()

	uResp, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: event.UserID})
	if err != nil {
		return
	}

	body, err := templates.RenderTemplate(
		templates.RefundIssuedTemplate,
		uResp.FullName,
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"OrderNumber":  event.OrderNumber,
			"RefundAmount": event.RefundAmount,
			"ETA":          event.ETA,
		},
	)
	if err != nil {
		return
	}

	h.queueEmail(ctx, buyerUID, "transactional", uResp.Email, uResp.FullName, "Refund processed for order "+event.OrderNumber, body)
}

func (h *NATSHandler) handleSellerStatusChanged(msg *nats.Msg) {
	var event struct {
		SellerID string `json:"seller_id"`
		Status   string `json:"status"`
		Reason   string `json:"reason"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	ctx := context.Background()

	seller, err := h.sellerClient.GetSeller(ctx, &sellerv1.GetSellerRequest{Id: event.SellerID})
	if err != nil {
		return
	}

	sellerUserUID, err := uuid.Parse(seller.UserId)
	if err != nil {
		return
	}

	uResp, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: seller.UserId})
	if err != nil {
		return
	}

	body, err := templates.RenderTemplate(
		templates.StoreStatusChangedTemplate,
		uResp.FullName,
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"StoreName": seller.StoreName,
			"Status":    event.Status,
			"Reason":    event.Reason,
		},
	)
	if err != nil {
		return
	}

	h.queueEmail(ctx, sellerUserUID, "transactional", uResp.Email, uResp.FullName, "WeMall store status update", body)
	h.queuePush(ctx, sellerUserUID, "transactional", "Store Status Update", "Your store status has been updated to: "+event.Status, map[string]string{
		"status": event.Status,
	})
}

func (h *NATSHandler) handleInventoryLowStock(msg *nats.Msg) {
	var event struct {
		SellerID       string `json:"seller_id"`
		VariantSKU     string `json:"variant_sku"`
		RemainingStock int64  `json:"remaining_stock"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	ctx := context.Background()

	seller, err := h.sellerClient.GetSeller(ctx, &sellerv1.GetSellerRequest{Id: event.SellerID})
	if err != nil {
		return
	}

	sellerUserUID, err := uuid.Parse(seller.UserId)
	if err != nil {
		return
	}

	uResp, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: seller.UserId})
	if err != nil {
		return
	}

	body, err := templates.RenderTemplate(
		templates.LowStockTemplate,
		uResp.FullName,
		"WeMall",
		"https://wemall.co.zw",
		map[string]interface{}{
			"VariantSKU: ":    event.VariantSKU,
			"RemainingStock": event.RemainingStock,
		},
	)
	if err != nil {
		return
	}

	h.queueEmail(ctx, sellerUserUID, "low_stock", uResp.Email, uResp.FullName, "Low Stock Warning ⚠️", body)
	h.queuePush(ctx, sellerUserUID, "low_stock", "Inventory Warning", "Low stock for SKU: "+event.VariantSKU, map[string]string{
		"sku": event.VariantSKU,
	})
}

func (h *NATSHandler) handleStorePostUpdate(msg *nats.Msg) {
	var event struct {
		SellerID     string  `json:"seller_id"`
		StoreName    string  `json:"store_name"`
		ProductTitle string  `json:"product_title"`
		Price        float64 `json:"price"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	ctx := context.Background()

	// 1. Fetch store followers from seller service
	resp, err := h.sellerClient.ListStoreFollowers(ctx, &sellerv1.ListStoreFollowersRequest{
		SellerId: event.SellerID,
	})
	if err != nil {
		h.logger.Error().Err(err).Str("seller_id", event.SellerID).Msg("Failed to list store followers")
		return
	}

	if len(resp.UserIds) == 0 {
		return
	}

	// 2. Loop over followers
	for _, followerIDStr := range resp.UserIds {
		followerUID, err := uuid.Parse(followerIDStr)
		if err != nil {
			continue
		}

		// Verify Opt-In Preference
		enabled, err := h.isChannelEnabled(ctx, followerUID, "follows", "email")
		if err == nil && enabled {
			// Get profile via gRPC
			uResp, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: followerIDStr})
			if err == nil {
				body, err := templates.RenderTemplate(
					templates.StoreUpdateTemplate,
					uResp.FullName,
					"WeMall",
					"https://wemall.co.zw",
					map[string]interface{}{
						"StoreName":    event.StoreName,
						"ProductTitle": event.ProductTitle,
						"Price":        event.Price,
					},
				)
				if err == nil {
					h.queueEmail(ctx, followerUID, "follows", uResp.Email, uResp.FullName, "New Update from "+event.StoreName, body)
				}
			}
		}

		// Push Follower update
		h.queuePush(ctx, followerUID, "follows", "New item from "+event.StoreName, event.ProductTitle+" is now available!", map[string]string{
			"seller_id": event.SellerID,
		})
	}
}

func (h *NATSHandler) handleProductPriceDropped(msg *nats.Msg) {
	var event struct {
		ProductTitle string   `json:"product_title"`
		OldPrice     float64  `json:"old_price"`
		NewPrice     float64  `json:"new_price"`
		UserIDs      []string `json:"user_ids"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	ctx := context.Background()

	for _, uidStr := range event.UserIDs {
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			continue
		}

		h.queuePush(ctx, uid, "marketing", "Price Drop Alert! 📉", event.ProductTitle+" dropped from $"+formatFloat(event.OldPrice)+" to $"+formatFloat(event.NewPrice), map[string]string{
			"product": event.ProductTitle,
		})
	}
}

func (h *NATSHandler) handleInventoryRestocked(msg *nats.Msg) {
	var event struct {
		ProductTitle string   `json:"product_title"`
		URL          string   `json:"url"`
		UserIDs      []string `json:"user_ids"`
	}
	if err := json.Unmarshal(msg.Data, &event); err != nil {
		return
	}

	ctx := context.Background()

	for _, uidStr := range event.UserIDs {
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			continue
		}

		// Email opt-in check
		enabled, err := h.isChannelEnabled(ctx, uid, "marketing", "email")
		if err == nil && enabled {
			uResp, err := h.userClient.GetUser(ctx, &userv1.GetUserRequest{Id: uidStr})
			if err == nil {
				body, err := templates.RenderTemplate(
					templates.RestockedTemplate,
					uResp.FullName,
					"WeMall",
					"https://wemall.co.zw",
					map[string]interface{}{
						"ProductTitle": event.ProductTitle,
						"URL":          event.URL,
					},
				)
				if err == nil {
					h.queueEmail(ctx, uid, "marketing", uResp.Email, uResp.FullName, event.ProductTitle+" is back in stock!", body)
				}
			}
		}

		h.queuePush(ctx, uid, "marketing", "Back in Stock! 🎉", event.ProductTitle+" is now back in stock. Buy it now!", map[string]string{
			"url": event.URL,
		})
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func formatFloat(f float64) string {
	return "" // Simplified
}
