package handler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"

	orderv1 "github.com/wemall/gen/order/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	userv1 "github.com/wemall/gen/user/v1"
	"github.com/wemall/notification-service/internal/db"
	"github.com/wemall/notification-service/internal/queue"
)

type mockQuerier struct {
	db.Querier
	preferenceFunc   func(ctx context.Context, arg db.GetNotificationPreferenceParams) (db.UserNotificationPreference, error)
	deviceTokensFunc func(ctx context.Context, userID uuid.UUID) ([]db.UserDeviceToken, error)
}

func (m *mockQuerier) GetNotificationPreference(ctx context.Context, arg db.GetNotificationPreferenceParams) (db.UserNotificationPreference, error) {
	if m.preferenceFunc != nil {
		return m.preferenceFunc(ctx, arg)
	}
	return db.UserNotificationPreference{EmailEnabled: true, PushEnabled: true}, nil
}

func (m *mockQuerier) GetDeviceTokensByUser(ctx context.Context, userID uuid.UUID) ([]db.UserDeviceToken, error) {
	if m.deviceTokensFunc != nil {
		return m.deviceTokensFunc(ctx, userID)
	}
	return []db.UserDeviceToken{}, nil
}

type mockQueueClient struct {
	emails     []queue.EmailSendPayload
	pushes     []queue.PushSendPayload
	multicasts []queue.PushMulticastPayload
}

func (m *mockQueueClient) EnqueueEmail(ctx context.Context, payload queue.EmailSendPayload) error {
	m.emails = append(m.emails, payload)
	return nil
}

func (m *mockQueueClient) EnqueuePush(ctx context.Context, payload queue.PushSendPayload) error {
	m.pushes = append(m.pushes, payload)
	return nil
}

func (m *mockQueueClient) EnqueuePushMulticast(ctx context.Context, payload queue.PushMulticastPayload) error {
	m.multicasts = append(m.multicasts, payload)
	return nil
}

type mockUserClient struct {
	userv1.UserServiceClient
	getUserFunc func(ctx context.Context, in *userv1.GetUserRequest, opts ...grpc.CallOption) (*userv1.User, error)
}

func (m *mockUserClient) GetUser(ctx context.Context, in *userv1.GetUserRequest, opts ...grpc.CallOption) (*userv1.User, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, in, opts...)
	}
	return &userv1.User{
		Id:       in.Id,
		FullName: "Test User",
		Email:    "test@wemall.co.zw",
	}, nil
}

type mockSellerClient struct {
	sellerv1.SellerServiceClient
	getSellerFunc     func(ctx context.Context, in *sellerv1.GetSellerRequest, opts ...grpc.CallOption) (*sellerv1.Seller, error)
	listFollowersFunc func(ctx context.Context, in *sellerv1.ListStoreFollowersRequest, opts ...grpc.CallOption) (*sellerv1.ListStoreFollowersResponse, error)
}

func (m *mockSellerClient) GetSeller(ctx context.Context, in *sellerv1.GetSellerRequest, opts ...grpc.CallOption) (*sellerv1.Seller, error) {
	if m.getSellerFunc != nil {
		return m.getSellerFunc(ctx, in, opts...)
	}
	return &sellerv1.Seller{
		Id:        in.Id,
		UserId:    uuid.New().String(),
		StoreName: "Test Store",
	}, nil
}

func (m *mockSellerClient) ListStoreFollowers(ctx context.Context, in *sellerv1.ListStoreFollowersRequest, opts ...grpc.CallOption) (*sellerv1.ListStoreFollowersResponse, error) {
	if m.listFollowersFunc != nil {
		return m.listFollowersFunc(ctx, in, opts...)
	}
	return &sellerv1.ListStoreFollowersResponse{
		UserIds: []string{},
	}, nil
}

type mockOrderClient struct {
	orderv1.OrderServiceClient
	getOrderFunc func(ctx context.Context, in *orderv1.GetOrderRequest, opts ...grpc.CallOption) (*orderv1.Order, error)
}

func (m *mockOrderClient) GetOrder(ctx context.Context, in *orderv1.GetOrderRequest, opts ...grpc.CallOption) (*orderv1.Order, error) {
	if m.getOrderFunc != nil {
		return m.getOrderFunc(ctx, in, opts...)
	}
	return &orderv1.Order{
		Id:          in.Id,
		OrderNumber: "WM-TEST-001",
		UserId:      in.UserId,
	}, nil
}

func TestNATSHandler_UserRegistered(t *testing.T) {
	qc := &mockQueueClient{}
	queries := &mockQuerier{}

	logger := zerolog.Nop()

	h := &NATSHandler{
		q:           queries,
		queueClient: qc,
		logger:      logger,
	}

	event := struct {
		UserID    string `json:"user_id"`
		FullName  string `json:"full_name"`
		Email     string `json:"email"`
		VerifyURL string `json:"verify_url"`
	}{
		UserID:    uuid.New().String(),
		FullName:  "John Doe",
		Email:     "john@example.com",
		VerifyURL: "https://wemall.co.zw/verify?token=123",
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	msg := &nats.Msg{
		Subject: "wemall.user.registered",
		Data:    data,
	}

	h.handleUserRegistered(msg)

	if len(qc.emails) != 1 {
		t.Fatalf("expected 1 email to be queued, got %d", len(qc.emails))
	}

	email := qc.emails[0]
	if email.Recipient != "john@example.com" {
		t.Errorf("expected recipient to be john@example.com, got %s", email.Recipient)
	}
	if email.RecipientName != "John Doe" {
		t.Errorf("expected recipient name to be John Doe, got %s", email.RecipientName)
	}
	if email.Category != "security" {
		t.Errorf("expected category to be security, got %s", email.Category)
	}
}

func TestNATSHandler_StorePostUpdate(t *testing.T) {
	qc := &mockQueueClient{}
	queries := &mockQuerier{
		deviceTokensFunc: func(ctx context.Context, userID uuid.UUID) ([]db.UserDeviceToken, error) {
			return []db.UserDeviceToken{
				{
					ID:        uuid.New(),
					UserID:    userID,
					Token:     "mock-fcm-token",
					Platform:  "ios",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
			}, nil
		},
	}
	logger := zerolog.Nop()

	followerID := uuid.New().String()
	sellerClient := &mockSellerClient{
		listFollowersFunc: func(ctx context.Context, in *sellerv1.ListStoreFollowersRequest, opts ...grpc.CallOption) (*sellerv1.ListStoreFollowersResponse, error) {
			return &sellerv1.ListStoreFollowersResponse{
				UserIds: []string{followerID},
			}, nil
		},
	}

	userClient := &mockUserClient{
		getUserFunc: func(ctx context.Context, in *userv1.GetUserRequest, opts ...grpc.CallOption) (*userv1.User, error) {
			return &userv1.User{
				Id:        in.Id,
				FullName:  "Follower User",
				Email:     "follower@example.com",
				AvatarUrl: "",
			}, nil
		},
	}

	h := &NATSHandler{
		q:            queries,
		queueClient:  qc,
		sellerClient: sellerClient,
		userClient:   userClient,
		logger:       logger,
	}

	event := struct {
		SellerID     string  `json:"seller_id"`
		StoreName    string  `json:"store_name"`
		ProductTitle string  `json:"product_title"`
		Price        float64 `json:"price"`
	}{
		SellerID:     uuid.New().String(),
		StoreName:    "Amazing store",
		ProductTitle: "Cool Gadget",
		Price:        99.99,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal event: %v", err)
	}

	msg := &nats.Msg{
		Subject: "wemall.store.post_update",
		Data:    data,
	}

	h.handleStorePostUpdate(msg)

	if len(qc.emails) != 1 {
		t.Errorf("expected 1 email, got %d", len(qc.emails))
	} else {
		email := qc.emails[0]
		if email.Recipient != "follower@example.com" {
			t.Errorf("expected email recipient to be follower@example.com, got %s", email.Recipient)
		}
	}

	if len(qc.pushes) != 1 {
		t.Errorf("expected 1 push, got %d", len(qc.pushes))
	} else {
		push := qc.pushes[0]
		if push.UserID != followerID {
			t.Errorf("expected push user ID to be %s, got %s", followerID, push.UserID)
		}
	}
}
