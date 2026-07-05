package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"github.com/wemall/chat-service/internal/db"
	werr "github.com/wemall/pkg/errors"
)

// ChatService handles chat business logic.
type ChatService struct {
	q *db.Queries
}

func NewChatService(q *db.Queries) *ChatService {
	return &ChatService{q: q}
}

// ChatServiceInterface is the public contract of ChatService, used for compile-time
// assertion in tests and as a seam for mocking in higher-level tests.
type ChatServiceInterface interface {
	CreateThread(ctx context.Context, buyerID, sellerID, orderID, participantAvatar string) (*db.Thread, error)
	CreateDeliveryThread(ctx context.Context, buyerID, deliveryBoyID, orderID, participantAvatar string) (*db.Thread, error)
	CreateCourierThread(ctx context.Context, buyerID, courierStationID, orderID, participantAvatar string) (*db.Thread, error)
	CreateSupportThread(ctx context.Context, buyerID, participantAvatar string) (*db.Thread, error)
	CreateBroadcastGroup(ctx context.Context, sellerID, title, participantAvatar string) (*db.Thread, error)
	GetOrCreateSystemThread(ctx context.Context) (*db.Thread, error)
	EnsureUserInSystemThread(ctx context.Context, userID string) error
	JoinBroadcastGroup(ctx context.Context, userID, sellerID string) error
	LeaveBroadcastGroup(ctx context.Context, userID, sellerID string) error
	SendMessage(ctx context.Context, threadID uuid.UUID, senderID, msgType, content, mediaUrl, referenceId string, metadataJSON []byte) (*db.Message, error)
	ListThreadsForUser(ctx context.Context, userID string) ([]db.Thread, error)
	ListThreadsForBuyer(ctx context.Context, buyerID string) ([]db.Thread, error)
	ListThreadsForSeller(ctx context.Context, sellerID string) ([]db.Thread, error)
	ListMessages(ctx context.Context, threadID uuid.UUID) ([]db.Message, error)
	GetBroadcastThreadForSeller(ctx context.Context, sellerID string) (*db.Thread, error)
}

// ───────────────────── Thread creation ─────────────────────────────────────

func (s *ChatService) CreateThread(ctx context.Context, buyerID, sellerID, orderID, participantAvatar string) (*db.Thread, error) {
	thread, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:              "THREAD_TYPE_DIRECT",
		BuyerID:           sql.NullString{String: buyerID, Valid: buyerID != ""},
		SellerID:          sql.NullString{String: sellerID, Valid: sellerID != ""},
		OrderID:           sql.NullString{String: orderID, Valid: orderID != ""},
		ParticipantAvatar: sql.NullString{String: participantAvatar, Valid: participantAvatar != ""},
	})
	if err != nil {
		return nil, werr.Internal(err)
	}
	return &thread, nil
}

// CreateDeliveryThread creates a chat thread between a buyer and a delivery boy.
func (s *ChatService) CreateDeliveryThread(ctx context.Context, buyerID, deliveryBoyID, orderID, participantAvatar string) (*db.Thread, error) {
	thread, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:              "THREAD_TYPE_DELIVERY",
		BuyerID:           sql.NullString{String: buyerID, Valid: buyerID != ""},
		DeliveryBoyID:     sql.NullString{String: deliveryBoyID, Valid: deliveryBoyID != ""},
		OrderID:           sql.NullString{String: orderID, Valid: orderID != ""},
		ParticipantAvatar: sql.NullString{String: participantAvatar, Valid: participantAvatar != ""},
	})
	if err != nil {
		return nil, werr.Internal(err)
	}
	return &thread, nil
}

// CreateCourierThread creates a chat thread between a buyer and a courier station.
func (s *ChatService) CreateCourierThread(ctx context.Context, buyerID, courierStationID, orderID, participantAvatar string) (*db.Thread, error) {
	thread, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:              "THREAD_TYPE_COURIER",
		BuyerID:           sql.NullString{String: buyerID, Valid: buyerID != ""},
		CourierStationID:  sql.NullString{String: courierStationID, Valid: courierStationID != ""},
		OrderID:           sql.NullString{String: orderID, Valid: orderID != ""},
		ParticipantAvatar: sql.NullString{String: participantAvatar, Valid: participantAvatar != ""},
	})
	if err != nil {
		return nil, werr.Internal(err)
	}
	return &thread, nil
}

// CreateSupportThread creates a support thread for a buyer.
func (s *ChatService) CreateSupportThread(ctx context.Context, buyerID, participantAvatar string) (*db.Thread, error) {
	thread, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:              "THREAD_TYPE_SUPPORT",
		BuyerID:           sql.NullString{String: buyerID, Valid: buyerID != ""},
		Title:             sql.NullString{String: "Support", Valid: true},
		ParticipantAvatar: sql.NullString{String: participantAvatar, Valid: participantAvatar != ""},
	})
	if err != nil {
		return nil, werr.Internal(err)
	}
	return &thread, nil
}

// CreateBroadcastGroup creates or retrieves the broadcast group for a seller.
func (s *ChatService) CreateBroadcastGroup(ctx context.Context, sellerID, title, participantAvatar string) (*db.Thread, error) {
	thread, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:              "THREAD_TYPE_BROADCAST",
		Title:             sql.NullString{String: title, Valid: title != ""},
		SellerID:          sql.NullString{String: sellerID, Valid: sellerID != ""},
		ParticipantAvatar: sql.NullString{String: participantAvatar, Valid: participantAvatar != ""},
	})
	if err != nil {
		return nil, werr.Internal(err)
	}
	return &thread, nil
}

// GetOrCreateSystemThread ensures the global WeMall Updates thread exists.
func (s *ChatService) GetOrCreateSystemThread(ctx context.Context) (*db.Thread, error) {
	thread, err := s.q.GetSystemThread(ctx)
	if err == nil {
		return &thread, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, werr.Internal(err)
	}
	// Create the global system thread
	newThread, createErr := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:  "THREAD_TYPE_SYSTEM",
		Title: sql.NullString{String: "WeMall Updates", Valid: true},
	})
	if createErr != nil {
		return nil, werr.Internal(createErr)
	}
	return &newThread, nil
}

// ───────────────────── Group membership ────────────────────────────────────

// EnsureUserInSystemThread adds a user to the global app system thread if not already a member.
func (s *ChatService) EnsureUserInSystemThread(ctx context.Context, userID string) error {
	thread, err := s.GetOrCreateSystemThread(ctx)
	if err != nil {
		return err
	}
	return s.q.AddThreadMember(ctx, db.AddThreadMemberParams{
		ThreadID: thread.ID,
		UserID:   userID,
		Role:     "MEMBER",
	})
}

// JoinBroadcastGroup adds a follower to a seller's broadcast group.
func (s *ChatService) JoinBroadcastGroup(ctx context.Context, userID, sellerID string) error {
	thread, err := s.GetBroadcastThreadForSeller(ctx, sellerID)
	if err != nil {
		// Auto-create broadcast group for this seller
		newThread, createErr := s.CreateBroadcastGroup(ctx, sellerID, "Store Updates", "")
		if createErr != nil {
			return createErr
		}
		thread = newThread
	}
	return s.q.AddThreadMember(ctx, db.AddThreadMemberParams{
		ThreadID: thread.ID,
		UserID:   userID,
		Role:     "MEMBER",
	})
}

// LeaveBroadcastGroup removes a user from a seller's broadcast group.
func (s *ChatService) LeaveBroadcastGroup(ctx context.Context, userID, sellerID string) error {
	thread, err := s.GetBroadcastThreadForSeller(ctx, sellerID)
	if err != nil {
		return nil // Already not a member
	}
	return s.q.RemoveThreadMember(ctx, db.RemoveThreadMemberParams{
		ThreadID: thread.ID,
		UserID:   userID,
	})
}

// ───────────────────── Messaging ──────────────────────────────────────────

// SendMessage creates a message on a thread, with optional JSONB metadata for rich content.
func (s *ChatService) SendMessage(ctx context.Context, threadID uuid.UUID, senderID, msgType, content, mediaUrl, referenceId string, metadataJSON []byte) (*db.Message, error) {
	var metaRaw pqtype.NullRawMessage
	if len(metadataJSON) > 0 {
		metaRaw = pqtype.NullRawMessage{RawMessage: json.RawMessage(metadataJSON), Valid: true}
	}
	msg, err := s.q.CreateMessage(ctx, db.CreateMessageParams{
		ThreadID:    threadID,
		SenderID:    senderID,
		Type:        msgType,
		Content:     content,
		MediaUrl:    sql.NullString{String: mediaUrl, Valid: mediaUrl != ""},
		ReferenceID: sql.NullString{String: referenceId, Valid: referenceId != ""},
		Metadata:    metaRaw,
	})
	if err != nil {
		return nil, werr.Internal(err)
	}
	_ = s.q.UpdateThreadTimestamp(ctx, threadID)
	return &msg, nil
}

// ───────────────────── Thread listing ──────────────────────────────────────

// ListThreadsForUser retrieves all threads where the given user is a participant,
// including broadcast/system groups they belong to via thread_members.
func (s *ChatService) ListThreadsForUser(ctx context.Context, userID string) ([]db.Thread, error) {
	// Ensure the user is in the system update thread
	_ = s.EnsureUserInSystemThread(ctx, userID)

	threads, err := s.q.ListThreadsForUser(ctx, sql.NullString{String: userID, Valid: true})
	if err != nil {
		return nil, werr.Internal(err)
	}
	return threads, nil
}

// ListThreadsForBuyer retrieves all direct and group threads for a buyer.
func (s *ChatService) ListThreadsForBuyer(ctx context.Context, buyerID string) ([]db.Thread, error) {
	return s.ListThreadsForUser(ctx, buyerID)
}

// ListThreadsForSeller retrieves all threads for a seller.
func (s *ChatService) ListThreadsForSeller(ctx context.Context, sellerID string) ([]db.Thread, error) {
	threads, err := s.q.ListThreadsForSeller(ctx, sql.NullString{String: sellerID, Valid: true})
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
	thread, err := s.q.GetBroadcastThreadForSeller(ctx, sql.NullString{String: sellerID, Valid: true})
	if err != nil {
		return nil, err
	}
	return &thread, nil
}
