package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/wemall/chat-service/internal/db"
	werr "github.com/wemall/pkg/errors"
)

type ChatService struct {
	q *db.Queries
	// In a real implementation we would have sellerv1.SellerServiceClient here
}

func NewChatService(q *db.Queries) *ChatService {
	return &ChatService{
		q: q,
	}
}

func (s *ChatService) CreateThread(ctx context.Context, buyerID, sellerID, orderID string) (*db.Thread, error) {
	thread, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:     "THREAD_TYPE_DIRECT",
		Title:    pgtype.Text{Valid: false},
		BuyerID:  pgtype.Text{String: buyerID, Valid: buyerID != ""},
		SellerID: sellerID,
		OrderID:  pgtype.Text{String: orderID, Valid: orderID != ""},
	})
	if err != nil {
		return nil, werr.Internal(err)
	}
	return &thread, nil
}

func (s *ChatService) CreateBroadcastGroup(ctx context.Context, sellerID, title string) (*db.Thread, error) {
	thread, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:     "THREAD_TYPE_BROADCAST",
		Title:    pgtype.Text{String: title, Valid: title != ""},
		BuyerID:  pgtype.Text{Valid: false}, // Broadcast has no single buyer
		SellerID: sellerID,
		OrderID:  pgtype.Text{Valid: false},
	})
	if err != nil {
		return nil, werr.Internal(err)
	}
	return &thread, nil
}

func (s *ChatService) SendMessage(ctx context.Context, threadID uuid.UUID, senderID, msgType, content, mediaUrl, referenceId string) (*db.Message, error) {
	msg, err := s.q.CreateMessage(ctx, db.CreateMessageParams{
		ThreadID:    threadID,
		SenderID:    senderID,
		Type:        msgType,
		Content:     content,
		MediaUrl:    pgtype.Text{String: mediaUrl, Valid: mediaUrl != ""},
		ReferenceID: pgtype.Text{String: referenceId, Valid: referenceId != ""},
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	// Update the thread's updated_at timestamp so it bubbles up in lists
	_ = s.q.UpdateThreadTimestamp(ctx, threadID)

	return &msg, nil
}

func (s *ChatService) ListThreadsForBuyer(ctx context.Context, buyerID string) ([]db.Thread, error) {
	// 1. Get all direct threads for the buyer
	directThreads, err := s.q.ListDirectThreadsForBuyer(ctx, pgtype.Text{String: buyerID, Valid: true})
	if err != nil {
		return nil, werr.Internal(err)
	}

	// 2. Mocking seller subscriptions: In a real app we'd call SellerService.ListFollowedStores(buyerID)
	mockFollowedSellerIDs := []string{"seller_123", "seller_456"}

	// 3. Fetch broadcast threads for these sellers
	broadcastThreads, err := s.q.ListBroadcastThreadsForSellers(ctx, mockFollowedSellerIDs)
	if err != nil {
		return nil, werr.Internal(err)
	}

	// Combine them
	allThreads := append(directThreads, broadcastThreads...)
	return allThreads, nil
}

func (s *ChatService) ListThreadsForSeller(ctx context.Context, sellerID string) ([]db.Thread, error) {
	threads, err := s.q.ListThreadsForSeller(ctx, sellerID)
	if err != nil {
		return nil, werr.Internal(err)
	}
	return threads, nil
}

func (s *ChatService) ListMessages(ctx context.Context, threadID uuid.UUID) ([]db.Message, error) {
	msgs, err := s.q.ListMessages(ctx, threadID)
	if err != nil {
		return nil, werr.Internal(err)
	}
	return msgs, nil
}

func (s *ChatService) GetBroadcastThreadForSeller(ctx context.Context, sellerID string) (*db.Thread, error) {
	thread, err := s.q.GetBroadcastThreadForSeller(ctx, sellerID)
	if err != nil {
		return nil, err // Let caller handle not found
	}
	return &thread, nil
}
