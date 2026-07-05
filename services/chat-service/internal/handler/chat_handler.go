package handler

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/wemall/gen/chat/v1"
	"github.com/wemall/chat-service/internal/db"
	"github.com/wemall/chat-service/internal/service"
	werr "github.com/wemall/pkg/errors"
)

// ChatHandler implements the gRPC ChatService server.
type ChatHandler struct {
	chatv1.UnimplementedChatServiceServer
	svc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

// ───────────────────── Thread Creation ─────────────────────────────────────

func (h *ChatHandler) CreateThread(ctx context.Context, req *chatv1.CreateThreadRequest) (*chatv1.Thread, error) {
	var thread *db.Thread
	var err error

	switch req.Type {
	case chatv1.ThreadType_THREAD_TYPE_DELIVERY:
		thread, err = h.svc.CreateDeliveryThread(ctx, req.BuyerId, req.DeliveryBoyId, req.OrderId, req.ParticipantAvatar)
	case chatv1.ThreadType_THREAD_TYPE_COURIER:
		thread, err = h.svc.CreateCourierThread(ctx, req.BuyerId, req.CourierStationId, req.OrderId, req.ParticipantAvatar)
	case chatv1.ThreadType_THREAD_TYPE_SUPPORT:
		thread, err = h.svc.CreateSupportThread(ctx, req.BuyerId, req.ParticipantAvatar)
	default:
		thread, err = h.svc.CreateThread(ctx, req.BuyerId, req.SellerId, req.OrderId, req.ParticipantAvatar)
	}
	if err != nil {
		return nil, err
	}
	return mapToPbThread(thread), nil
}

func (h *ChatHandler) CreateBroadcastGroup(ctx context.Context, req *chatv1.CreateBroadcastGroupRequest) (*chatv1.Thread, error) {
	thread, err := h.svc.CreateBroadcastGroup(ctx, req.SellerId, req.Title, req.ParticipantAvatar)
	if err != nil {
		return nil, err
	}
	return mapToPbThread(thread), nil
}

// ───────────────────── Messaging ──────────────────────────────────────────

func (h *ChatHandler) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.Message, error) {
	uid, err := uuid.Parse(req.ThreadId)
	if err != nil {
		return nil, werr.InvalidArgument("invalid thread_id")
	}

	msgType := protoMsgTypeToString(req.Type)

	// Serialise optional google.protobuf.Struct metadata to raw JSON bytes
	var metaBytes []byte
	if req.Metadata != nil {
		metaBytes, err = protojson.Marshal(req.Metadata)
		if err != nil {
			return nil, werr.InvalidArgument("invalid metadata")
		}
	}

	msg, err := h.svc.SendMessage(ctx, uid, req.SenderId, msgType, req.Content, req.MediaUrl, req.ReferenceId, metaBytes)
	if err != nil {
		return nil, err
	}
	return mapToPbMessage(msg), nil
}

// ───────────────────── Listing ────────────────────────────────────────────

func (h *ChatHandler) ListThreads(ctx context.Context, req *chatv1.ListThreadsRequest) (*chatv1.ListThreadsResponse, error) {
	var threads []db.Thread
	var err error

	switch req.Role {
	case "SELLER":
		threads, err = h.svc.ListThreadsForSeller(ctx, req.UserId)
	default:
		// buyer, delivery, courier, support, system → unified query
		threads, err = h.svc.ListThreadsForUser(ctx, req.UserId)
	}

	if err != nil {
		return nil, err
	}

	pbThreads := make([]*chatv1.Thread, 0, len(threads))
	for _, t := range threads {
		pbThreads = append(pbThreads, mapToPbThread(&t))
	}
	return &chatv1.ListThreadsResponse{Threads: pbThreads}, nil
}

func (h *ChatHandler) ListMessages(ctx context.Context, req *chatv1.ListMessagesRequest) (*chatv1.ListMessagesResponse, error) {
	uid, err := uuid.Parse(req.ThreadId)
	if err != nil {
		return nil, werr.InvalidArgument("invalid thread_id")
	}

	msgs, err := h.svc.ListMessages(ctx, uid)
	if err != nil {
		return nil, err
	}

	pbMsgs := make([]*chatv1.Message, 0, len(msgs))
	for _, m := range msgs {
		pbMsgs = append(pbMsgs, mapToPbMessage(&m))
	}
	return &chatv1.ListMessagesResponse{Messages: pbMsgs}, nil
}

// ───────────────────── Mappers ────────────────────────────────────────────

func protoMsgTypeToString(t chatv1.MessageType) string {
	switch t {
	case chatv1.MessageType_MESSAGE_TYPE_IMAGE:
		return "MESSAGE_TYPE_IMAGE"
	case chatv1.MessageType_MESSAGE_TYPE_VIDEO:
		return "MESSAGE_TYPE_VIDEO"
	case chatv1.MessageType_MESSAGE_TYPE_DOCUMENT:
		return "MESSAGE_TYPE_DOCUMENT"
	case chatv1.MessageType_MESSAGE_TYPE_AUDIO:
		return "MESSAGE_TYPE_AUDIO"
	case chatv1.MessageType_MESSAGE_TYPE_PRODUCT:
		return "MESSAGE_TYPE_PRODUCT"
	case chatv1.MessageType_MESSAGE_TYPE_ORDER:
		return "MESSAGE_TYPE_ORDER"
	case chatv1.MessageType_MESSAGE_TYPE_PROMOTION:
		return "MESSAGE_TYPE_PROMOTION"
	case chatv1.MessageType_MESSAGE_TYPE_COUPON:
		return "MESSAGE_TYPE_COUPON"
	default:
		return "MESSAGE_TYPE_TEXT"
	}
}

func mapToPbThread(t *db.Thread) *chatv1.Thread {
	thType := chatv1.ThreadType_THREAD_TYPE_UNSPECIFIED
	switch t.Type {
	case "THREAD_TYPE_DIRECT":
		thType = chatv1.ThreadType_THREAD_TYPE_DIRECT
	case "THREAD_TYPE_BROADCAST":
		thType = chatv1.ThreadType_THREAD_TYPE_BROADCAST
	case "THREAD_TYPE_DELIVERY":
		thType = chatv1.ThreadType_THREAD_TYPE_DELIVERY
	case "THREAD_TYPE_COURIER":
		thType = chatv1.ThreadType_THREAD_TYPE_COURIER
	case "THREAD_TYPE_SUPPORT":
		thType = chatv1.ThreadType_THREAD_TYPE_SUPPORT
	case "THREAD_TYPE_SYSTEM":
		thType = chatv1.ThreadType_THREAD_TYPE_SYSTEM
	}

	pb := &chatv1.Thread{
		Id:                t.ID.String(),
		Type:              thType,
		Title:             t.Title.String,
		BuyerId:           t.BuyerID.String,
		SellerId:          t.SellerID.String,
		OrderId:           t.OrderID.String,
		DeliveryBoyId:     t.DeliveryBoyID.String,
		CourierStationId:  t.CourierStationID.String,
		SupportAgentId:    t.SupportAgentID.String,
		ParticipantAvatar: t.ParticipantAvatar.String,
	}
	if t.CreatedAt.Valid {
		pb.CreatedAt = timestamppb.New(t.CreatedAt.Time)
	}
	if t.UpdatedAt.Valid {
		pb.UpdatedAt = timestamppb.New(t.UpdatedAt.Time)
	}
	return pb
}

func mapToPbMessage(m *db.Message) *chatv1.Message {
	msgType := chatv1.MessageType_MESSAGE_TYPE_TEXT
	switch m.Type {
	case "MESSAGE_TYPE_IMAGE":
		msgType = chatv1.MessageType_MESSAGE_TYPE_IMAGE
	case "MESSAGE_TYPE_VIDEO":
		msgType = chatv1.MessageType_MESSAGE_TYPE_VIDEO
	case "MESSAGE_TYPE_DOCUMENT":
		msgType = chatv1.MessageType_MESSAGE_TYPE_DOCUMENT
	case "MESSAGE_TYPE_AUDIO":
		msgType = chatv1.MessageType_MESSAGE_TYPE_AUDIO
	case "MESSAGE_TYPE_PRODUCT":
		msgType = chatv1.MessageType_MESSAGE_TYPE_PRODUCT
	case "MESSAGE_TYPE_ORDER":
		msgType = chatv1.MessageType_MESSAGE_TYPE_ORDER
	case "MESSAGE_TYPE_PROMOTION":
		msgType = chatv1.MessageType_MESSAGE_TYPE_PROMOTION
	case "MESSAGE_TYPE_COUPON":
		msgType = chatv1.MessageType_MESSAGE_TYPE_COUPON
	}

	pb := &chatv1.Message{
		Id:          m.ID.String(),
		ThreadId:    m.ThreadID.String(),
		SenderId:    m.SenderID,
		Type:        msgType,
		Content:     m.Content,
		MediaUrl:    m.MediaUrl.String,
		ReferenceId: m.ReferenceID.String,
		IsRead:      m.IsRead,
	}
	if m.CreatedAt.Valid {
		pb.CreatedAt = timestamppb.New(m.CreatedAt.Time)
	}

	// Deserialise JSONB metadata into google.protobuf.Struct
	if m.Metadata.Valid && len(m.Metadata.RawMessage) > 0 {
		var raw map[string]interface{}
		if err := json.Unmarshal(m.Metadata.RawMessage, &raw); err == nil {
			if s, err2 := structpb.NewStruct(raw); err2 == nil {
				pb.Metadata = s
			}
		}
	}

	return pb
}
