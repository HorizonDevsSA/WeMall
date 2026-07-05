package handler

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"github.com/wemall/chat-service/internal/db"
	chatv1 "github.com/wemall/gen/chat/v1"
)

func TestMapToPbThread(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	dbThread := db.Thread{
		ID:                id,
		Type:              "THREAD_TYPE_DELIVERY",
		Title:             sql.NullString{String: "Delivery Updates", Valid: true},
		BuyerID:           sql.NullString{String: "buyer-1", Valid: true},
		SellerID:          sql.NullString{String: "seller-2", Valid: true},
		OrderID:           sql.NullString{String: "order-3", Valid: true},
		DeliveryBoyID:     sql.NullString{String: "boy-4", Valid: true},
		CourierStationID:  sql.NullString{String: "station-5", Valid: true},
		SupportAgentID:    sql.NullString{String: "agent-6", Valid: true},
		ParticipantAvatar: sql.NullString{String: "https://logo.com", Valid: true},
		CreatedAt:         sql.NullTime{Time: now, Valid: true},
		UpdatedAt:         sql.NullTime{Time: now, Valid: true},
	}

	pb := mapToPbThread(&dbThread)

	if pb.Id != id.String() {
		t.Errorf("expected ID %s, got %s", id.String(), pb.Id)
	}
	if pb.Type != chatv1.ThreadType_THREAD_TYPE_DELIVERY {
		t.Errorf("expected ThreadType_THREAD_TYPE_DELIVERY, got %v", pb.Type)
	}
	if pb.Title != "Delivery Updates" {
		t.Errorf("expected title 'Delivery Updates', got %s", pb.Title)
	}
	if pb.BuyerId != "buyer-1" || pb.SellerId != "seller-2" || pb.OrderId != "order-3" {
		t.Errorf("mismatched IDs: buyer=%s, seller=%s, order=%s", pb.BuyerId, pb.SellerId, pb.OrderId)
	}
	if pb.DeliveryBoyId != "boy-4" || pb.CourierStationId != "station-5" || pb.SupportAgentId != "agent-6" {
		t.Errorf("mismatched agent/station/boy IDs")
	}
	if pb.ParticipantAvatar != "https://logo.com" {
		t.Errorf("expected participant avatar 'https://logo.com', got %s", pb.ParticipantAvatar)
	}
	if pb.CreatedAt.AsTime().Unix() != now.Unix() || pb.UpdatedAt.AsTime().Unix() != now.Unix() {
		t.Errorf("timestamps mismatched")
	}
}

func TestMapToPbMessage_ValidMetadata(t *testing.T) {
	id := uuid.New()
	threadID := uuid.New()
	now := time.Now()

	meta := map[string]interface{}{
		"product_id": "prod-123",
		"price":      99.9,
	}
	metaBytes, _ := json.Marshal(meta)

	dbMsg := db.Message{
		ID:          id,
		ThreadID:    threadID,
		SenderID:    "user-1",
		Type:        "MESSAGE_TYPE_PRODUCT",
		Content:     "Look at this!",
		MediaUrl:    sql.NullString{String: "https://img.jpg", Valid: true},
		ReferenceID: sql.NullString{String: "prod-123", Valid: true},
		IsRead:      true,
		CreatedAt:   sql.NullTime{Time: now, Valid: true},
		Metadata:    pqtype.NullRawMessage{RawMessage: metaBytes, Valid: true},
	}

	pb := mapToPbMessage(&dbMsg)

	if pb.Id != id.String() {
		t.Errorf("expected ID %s, got %s", id.String(), pb.Id)
	}
	if pb.Type != chatv1.MessageType_MESSAGE_TYPE_PRODUCT {
		t.Errorf("expected MessageType_MESSAGE_TYPE_PRODUCT, got %v", pb.Type)
	}
	if pb.Content != "Look at this!" {
		t.Errorf("expected content 'Look at this!', got %s", pb.Content)
	}
	if pb.MediaUrl != "https://img.jpg" {
		t.Errorf("expected media url, got %s", pb.MediaUrl)
	}
	if pb.ReferenceId != "prod-123" {
		t.Errorf("expected reference ID 'prod-123', got %s", pb.ReferenceId)
	}
	if !pb.IsRead {
		t.Error("expected IsRead to be true")
	}

	if pb.Metadata == nil {
		t.Fatal("expected metadata to be parsed into protobuf Struct, got nil")
	}

	fields := pb.Metadata.GetFields()
	if fields["product_id"].GetStringValue() != "prod-123" {
		t.Errorf("expected metadata product_id to be 'prod-123', got %v", fields["product_id"])
	}
	if fields["price"].GetNumberValue() != 99.9 {
		t.Errorf("expected metadata price to be 99.9, got %v", fields["price"])
	}
}

func TestMapToPbMessage_InvalidMetadata(t *testing.T) {
	dbMsg := db.Message{
		ID:        uuid.New(),
		ThreadID:  uuid.New(),
		SenderID:  "user-1",
		Type:      "MESSAGE_TYPE_TEXT",
		Content:   "hello",
		Metadata:  pqtype.NullRawMessage{RawMessage: []byte("{invalid-json"), Valid: true},
	}

	pb := mapToPbMessage(&dbMsg)

	if pb.Metadata != nil {
		t.Errorf("expected pb.Metadata to be nil for invalid JSON, got %v", pb.Metadata)
	}
}

func TestProtoMsgTypeToString(t *testing.T) {
	tests := []struct {
		pbType chatv1.MessageType
		want   string
	}{
		{chatv1.MessageType_MESSAGE_TYPE_IMAGE, "MESSAGE_TYPE_IMAGE"},
		{chatv1.MessageType_MESSAGE_TYPE_VIDEO, "MESSAGE_TYPE_VIDEO"},
		{chatv1.MessageType_MESSAGE_TYPE_DOCUMENT, "MESSAGE_TYPE_DOCUMENT"},
		{chatv1.MessageType_MESSAGE_TYPE_AUDIO, "MESSAGE_TYPE_AUDIO"},
		{chatv1.MessageType_MESSAGE_TYPE_PRODUCT, "MESSAGE_TYPE_PRODUCT"},
		{chatv1.MessageType_MESSAGE_TYPE_ORDER, "MESSAGE_TYPE_ORDER"},
		{chatv1.MessageType_MESSAGE_TYPE_PROMOTION, "MESSAGE_TYPE_PROMOTION"},
		{chatv1.MessageType_MESSAGE_TYPE_COUPON, "MESSAGE_TYPE_COUPON"},
		{chatv1.MessageType_MESSAGE_TYPE_TEXT, "MESSAGE_TYPE_TEXT"},
	}

	for _, tc := range tests {
		got := protoMsgTypeToString(tc.pbType)
		if got != tc.want {
			t.Errorf("protoMsgTypeToString(%v) = %q, want %q", tc.pbType, got, tc.want)
		}
	}
}
