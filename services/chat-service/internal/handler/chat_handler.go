package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	chatv1 "github.com/wemall/gen/chat/v1"
	"github.com/wemall/chat-service/internal/db"
	"github.com/wemall/chat-service/internal/service"
	werr "github.com/wemall/pkg/errors"
)

type ChatHandler struct {
	chatv1.UnimplementedChatServiceServer
	svc *service.ChatService
}

func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

func (h *ChatHandler) CreateThread(ctx context.Context, req *chatv1.CreateThreadRequest) (*chatv1.Thread, error) {
	thread, err := h.svc.CreateThread(ctx, req.BuyerId, req.SellerId, req.OrderId)
	if err != nil {
		return nil, err
	}
	return mapToPbThread(thread), nil
}

func (h *ChatHandler) CreateBroadcastGroup(ctx context.Context, req *chatv1.CreateBroadcastGroupRequest) (*chatv1.Thread, error) {
	thread, err := h.svc.CreateBroadcastGroup(ctx, req.SellerId, req.Title)
	if err != nil {
		return nil, err
	}
	return mapToPbThread(thread), nil
}

func (h *ChatHandler) SendMessage(ctx context.Context, req *chatv1.SendMessageRequest) (*chatv1.Message, error) {
	uid, err := uuid.Parse(req.ThreadId)
	if err != nil {
		return nil, werr.InvalidArgument("invalid thread_id")
	}

	msgType := "MESSAGE_TYPE_TEXT"
	switch req.Type {
	case chatv1.MessageType_MESSAGE_TYPE_IMAGE:
		msgType = "MESSAGE_TYPE_IMAGE"
	case chatv1.MessageType_MESSAGE_TYPE_VIDEO:
		msgType = "MESSAGE_TYPE_VIDEO"
	case chatv1.MessageType_MESSAGE_TYPE_DOCUMENT:
		msgType = "MESSAGE_TYPE_DOCUMENT"
	case chatv1.MessageType_MESSAGE_TYPE_AUDIO:
		msgType = "MESSAGE_TYPE_AUDIO"
	case chatv1.MessageType_MESSAGE_TYPE_PRODUCT:
		msgType = "MESSAGE_TYPE_PRODUCT"
	case chatv1.MessageType_MESSAGE_TYPE_ORDER:
		msgType = "MESSAGE_TYPE_ORDER"
	case chatv1.MessageType_MESSAGE_TYPE_PROMOTION:
		msgType = "MESSAGE_TYPE_PROMOTION"
	}

	msg, err := h.svc.SendMessage(ctx, uid, req.SenderId, msgType, req.Content, req.MediaUrl, req.ReferenceId)
	if err != nil {
		return nil, err
	}

	return mapToPbMessage(msg), nil
}

func (h *ChatHandler) ListThreads(ctx context.Context, req *chatv1.ListThreadsRequest) (*chatv1.ListThreadsResponse, error) {
	var threads []db.Thread
	var err error

	if req.Role == "BUYER" {
		threads, err = h.svc.ListThreadsForBuyer(ctx, req.UserId)
	} else if req.Role == "SELLER" {
		threads, err = h.svc.ListThreadsForSeller(ctx, req.UserId)
	} else {
		return nil, werr.InvalidArgument("invalid role")
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

func mapToPbThread(t *db.Thread) *chatv1.Thread {
	thType := chatv1.ThreadType_THREAD_TYPE_UNSPECIFIED
	if t.Type == "THREAD_TYPE_DIRECT" {
		thType = chatv1.ThreadType_THREAD_TYPE_DIRECT
	} else if t.Type == "THREAD_TYPE_BROADCAST" {
		thType = chatv1.ThreadType_THREAD_TYPE_BROADCAST
	}

	return &chatv1.Thread{
		Id:        t.ID.String(),
		Type:      thType,
		Title:     t.Title.String,
		BuyerId:   t.BuyerID.String,
		SellerId:  t.SellerID,
		OrderId:   t.OrderID.String,
		CreatedAt: timestamppb.New(t.CreatedAt.Time),
		UpdatedAt: timestamppb.New(t.UpdatedAt.Time),
	}
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
	}

	return &chatv1.Message{
		Id:          m.ID.String(),
		ThreadId:    m.ThreadID.String(),
		SenderId:    m.SenderID,
		Type:        msgType,
		Content:     m.Content,
		MediaUrl:    m.MediaUrl.String,
		ReferenceId: m.ReferenceID.String,
		IsRead:      m.IsRead,
		CreatedAt:   timestamppb.New(m.CreatedAt.Time),
	}
}
