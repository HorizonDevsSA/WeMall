package handler

import (
	"context"
	"testing"

	"github.com/google/uuid"
	notificationv1 "github.com/wemall/gen/notification/v1"
	"github.com/wemall/notification-service/internal/db"
)

type mockQuerierGrpc struct {
	db.Querier
	upsertDeviceTokenFunc func(ctx context.Context, arg db.UpsertDeviceTokenParams) (db.UserDeviceToken, error)
}

func (m *mockQuerierGrpc) UpsertDeviceToken(ctx context.Context, arg db.UpsertDeviceTokenParams) (db.UserDeviceToken, error) {
	if m.upsertDeviceTokenFunc != nil {
		return m.upsertDeviceTokenFunc(ctx, arg)
	}
	return db.UserDeviceToken{}, nil
}

func TestGRPCHandler_RegisterDeviceToken(t *testing.T) {
	called := false
	uid := uuid.New()
	queries := &mockQuerierGrpc{
		upsertDeviceTokenFunc: func(ctx context.Context, arg db.UpsertDeviceTokenParams) (db.UserDeviceToken, error) {
			called = true
			if arg.UserID != uid {
				t.Errorf("expected user ID to be %s, got %s", uid, arg.UserID)
			}
			if arg.Token != "test-token" {
				t.Errorf("expected token to be test-token, got %s", arg.Token)
			}
			if arg.Platform != "ios" {
				t.Errorf("expected platform to be ios, got %s", arg.Platform)
			}
			if arg.DeviceName == nil || *arg.DeviceName != "iPhone" {
				t.Errorf("expected device name to be iPhone, got %v", arg.DeviceName)
			}
			return db.UserDeviceToken{}, nil
		},
	}

	h := NewGRPCHandler(queries)

	req := &notificationv1.RegisterDeviceTokenRequest{
		UserId:     uid.String(),
		Token:      "test-token",
		Platform:   "ios",
		DeviceName: "iPhone",
	}

	_, err := h.RegisterDeviceToken(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected UpsertDeviceToken to be called")
	}
}
