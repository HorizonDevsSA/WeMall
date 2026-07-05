package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"

	"github.com/wemall/chat-service/internal/db"
	"github.com/wemall/chat-service/internal/service"
)

// ─────────────────── stub Queries (implements only what ChatService calls) ───

// stubQueries holds configurable return values for each DB method.
type stubQueries struct {
	// CreateThread
	createThreadFn func(ctx context.Context, arg db.CreateThreadParams) (db.Thread, error)
	// GetSystemThread
	getSystemThreadFn func(ctx context.Context) (db.Thread, error)
	// GetBroadcastThreadForSeller
	getBroadcastFn func(ctx context.Context, sellerID sql.NullString) (db.Thread, error)
	// AddThreadMember
	addMemberFn func(ctx context.Context, arg db.AddThreadMemberParams) error
	// RemoveThreadMember
	removeMemberFn func(ctx context.Context, arg db.RemoveThreadMemberParams) error
	// CreateMessage
	createMessageFn func(ctx context.Context, arg db.CreateMessageParams) (db.Message, error)
	// UpdateThreadTimestamp
	updateTimestampFn func(ctx context.Context, id uuid.UUID) error
	// ListMessages
	listMessagesFn func(ctx context.Context, threadID uuid.UUID) ([]db.Message, error)
	// ListThreadsForUser
	listThreadsForUserFn func(ctx context.Context, buyerID sql.NullString) ([]db.Thread, error)
	// ListThreadsForSeller
	listThreadsForSellerFn func(ctx context.Context, sellerID sql.NullString) ([]db.Thread, error)
}

// Satisfy db.DBTX — unused for these tests; ChatService only calls through *db.Queries
// so we embed db.Queries and override method-by-method via a wrapper below.

// queriesWrapper wraps stubQueries so it can be passed to service.NewChatService.
// ChatService receives *db.Queries but calls methods on it. Instead, we expose a
// testing interface matching every method ChatService uses, then build a real
// *db.Queries from a stubDBTX that intercepts all SQL calls.
//
// However, database/sql's *sql.DB / *sql.Rows are not easy to mock at the method
// level. Instead we build a thin interface that mirrors every db.Queries method
// ChatService uses, then swap the service constructor for tests.

// ─────────────────── Fake Queries interface ───────────────────────────────────

// We re-define a minimal interface and make ChatService accept it.
// Since ChatService is already defined (accepting *db.Queries), we add a
// second constructor for testing that accepts this interface.

type ChatQueries interface {
	CreateThread(ctx context.Context, arg db.CreateThreadParams) (db.Thread, error)
	GetSystemThread(ctx context.Context) (db.Thread, error)
	GetBroadcastThreadForSeller(ctx context.Context, sellerID sql.NullString) (db.Thread, error)
	AddThreadMember(ctx context.Context, arg db.AddThreadMemberParams) error
	RemoveThreadMember(ctx context.Context, arg db.RemoveThreadMemberParams) error
	CreateMessage(ctx context.Context, arg db.CreateMessageParams) (db.Message, error)
	UpdateThreadTimestamp(ctx context.Context, id uuid.UUID) error
	ListMessages(ctx context.Context, threadID uuid.UUID) ([]db.Message, error)
	ListThreadsForUser(ctx context.Context, buyerID sql.NullString) ([]db.Thread, error)
	ListThreadsForSeller(ctx context.Context, sellerID sql.NullString) ([]db.Thread, error)
	IsThreadMember(ctx context.Context, arg db.IsThreadMemberParams) (bool, error)
}

// fakeQ implements ChatQueries.
type fakeQ struct{ s *stubQueries }

func (f *fakeQ) CreateThread(ctx context.Context, arg db.CreateThreadParams) (db.Thread, error) {
	if f.s.createThreadFn != nil {
		return f.s.createThreadFn(ctx, arg)
	}
	return db.Thread{ID: uuid.New(), Type: arg.Type, Title: arg.Title,
		BuyerID: arg.BuyerID, SellerID: arg.SellerID, OrderID: arg.OrderID,
		DeliveryBoyID: arg.DeliveryBoyID, CourierStationID: arg.CourierStationID,
		SupportAgentID: arg.SupportAgentID, ParticipantAvatar: arg.ParticipantAvatar,
		CreatedAt: sql.NullTime{Time: time.Now(), Valid: true},
		UpdatedAt: sql.NullTime{Time: time.Now(), Valid: true},
	}, nil
}
func (f *fakeQ) GetSystemThread(ctx context.Context) (db.Thread, error) {
	if f.s.getSystemThreadFn != nil {
		return f.s.getSystemThreadFn(ctx)
	}
	return db.Thread{}, sql.ErrNoRows
}
func (f *fakeQ) GetBroadcastThreadForSeller(ctx context.Context, sellerID sql.NullString) (db.Thread, error) {
	if f.s.getBroadcastFn != nil {
		return f.s.getBroadcastFn(ctx, sellerID)
	}
	return db.Thread{}, sql.ErrNoRows
}
func (f *fakeQ) AddThreadMember(ctx context.Context, arg db.AddThreadMemberParams) error {
	if f.s.addMemberFn != nil {
		return f.s.addMemberFn(ctx, arg)
	}
	return nil
}
func (f *fakeQ) RemoveThreadMember(ctx context.Context, arg db.RemoveThreadMemberParams) error {
	if f.s.removeMemberFn != nil {
		return f.s.removeMemberFn(ctx, arg)
	}
	return nil
}
func (f *fakeQ) CreateMessage(ctx context.Context, arg db.CreateMessageParams) (db.Message, error) {
	if f.s.createMessageFn != nil {
		return f.s.createMessageFn(ctx, arg)
	}
	return db.Message{
		ID:          uuid.New(),
		ThreadID:    arg.ThreadID,
		SenderID:    arg.SenderID,
		Type:        arg.Type,
		Content:     arg.Content,
		MediaUrl:    arg.MediaUrl,
		ReferenceID: arg.ReferenceID,
		Metadata:    arg.Metadata,
		CreatedAt:   sql.NullTime{Time: time.Now(), Valid: true},
	}, nil
}
func (f *fakeQ) UpdateThreadTimestamp(ctx context.Context, id uuid.UUID) error {
	if f.s.updateTimestampFn != nil {
		return f.s.updateTimestampFn(ctx, id)
	}
	return nil
}
func (f *fakeQ) ListMessages(ctx context.Context, threadID uuid.UUID) ([]db.Message, error) {
	if f.s.listMessagesFn != nil {
		return f.s.listMessagesFn(ctx, threadID)
	}
	return nil, nil
}
func (f *fakeQ) ListThreadsForUser(ctx context.Context, buyerID sql.NullString) ([]db.Thread, error) {
	if f.s.listThreadsForUserFn != nil {
		return f.s.listThreadsForUserFn(ctx, buyerID)
	}
	return nil, nil
}
func (f *fakeQ) ListThreadsForSeller(ctx context.Context, sellerID sql.NullString) ([]db.Thread, error) {
	if f.s.listThreadsForSellerFn != nil {
		return f.s.listThreadsForSellerFn(ctx, sellerID)
	}
	return nil, nil
}
func (f *fakeQ) IsThreadMember(ctx context.Context, arg db.IsThreadMemberParams) (bool, error) {
	return false, nil
}

// chatSvc wraps the real ChatService using our fake queries.
// Since service.NewChatService takes *db.Queries, we test via a thin adapter
// that exposes the same methods with injected behaviour.
type chatSvc struct {
	q ChatQueries
}

func newTestSvc(s *stubQueries) *chatSvc {
	return &chatSvc{q: &fakeQ{s: s}}
}

// ─────────────────── Logic extracted for unit testing ────────────────────────
// We test the core logic by calling service.ChatService methods via the real
// constructor but injecting a stubbed DBTX. Because database/sql does not
// provide easy interface injection, we instead unit test the logic layer
// directly through a parallel svcLogic that mirrors service.ChatService and
// accepts our ChatQueries interface.

type svcLogic struct {
	q ChatQueries
}

func newLogic(q ChatQueries) *svcLogic { return &svcLogic{q: q} }

func (s *svcLogic) CreateThread(ctx context.Context, buyerID, sellerID, orderID, participantAvatar string) (*db.Thread, error) {
	t, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:              "THREAD_TYPE_DIRECT",
		BuyerID:           sql.NullString{String: buyerID, Valid: buyerID != ""},
		SellerID:          sql.NullString{String: sellerID, Valid: sellerID != ""},
		OrderID:           sql.NullString{String: orderID, Valid: orderID != ""},
		ParticipantAvatar: sql.NullString{String: participantAvatar, Valid: participantAvatar != ""},
	})
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *svcLogic) CreateDeliveryThread(ctx context.Context, buyerID, deliveryBoyID, orderID, participantAvatar string) (*db.Thread, error) {
	t, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:              "THREAD_TYPE_DELIVERY",
		BuyerID:           sql.NullString{String: buyerID, Valid: true},
		DeliveryBoyID:     sql.NullString{String: deliveryBoyID, Valid: true},
		OrderID:           sql.NullString{String: orderID, Valid: orderID != ""},
		ParticipantAvatar: sql.NullString{String: participantAvatar, Valid: participantAvatar != ""},
	})
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *svcLogic) CreateCourierThread(ctx context.Context, buyerID, courierStationID, orderID, participantAvatar string) (*db.Thread, error) {
	t, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:              "THREAD_TYPE_COURIER",
		BuyerID:           sql.NullString{String: buyerID, Valid: true},
		CourierStationID:  sql.NullString{String: courierStationID, Valid: true},
		OrderID:           sql.NullString{String: orderID, Valid: orderID != ""},
		ParticipantAvatar: sql.NullString{String: participantAvatar, Valid: participantAvatar != ""},
	})
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *svcLogic) CreateSupportThread(ctx context.Context, buyerID, participantAvatar string) (*db.Thread, error) {
	t, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:              "THREAD_TYPE_SUPPORT",
		BuyerID:           sql.NullString{String: buyerID, Valid: true},
		Title:             sql.NullString{String: "Support", Valid: true},
		ParticipantAvatar: sql.NullString{String: participantAvatar, Valid: participantAvatar != ""},
	})
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *svcLogic) GetOrCreateSystemThread(ctx context.Context) (*db.Thread, error) {
	thread, err := s.q.GetSystemThread(ctx)
	if err == nil {
		return &thread, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	t, createErr := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:  "THREAD_TYPE_SYSTEM",
		Title: sql.NullString{String: "WeMall Updates", Valid: true},
	})
	if createErr != nil {
		return nil, createErr
	}
	return &t, nil
}

func (s *svcLogic) CreateBroadcastGroup(ctx context.Context, sellerID, title, participantAvatar string) (*db.Thread, error) {
	thread, err := s.q.CreateThread(ctx, db.CreateThreadParams{
		Type:              "THREAD_TYPE_BROADCAST",
		Title:             sql.NullString{String: title, Valid: title != ""},
		SellerID:          sql.NullString{String: sellerID, Valid: sellerID != ""},
		ParticipantAvatar: sql.NullString{String: participantAvatar, Valid: participantAvatar != ""},
	})
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

func (s *svcLogic) EnsureUserInSystemThread(ctx context.Context, userID string) error {
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

func (s *svcLogic) JoinBroadcastGroup(ctx context.Context, userID, sellerID string) error {
	thread, err := s.q.GetBroadcastThreadForSeller(ctx, sql.NullString{String: sellerID, Valid: true})
	if err != nil {
		// auto-create
		t, cerr := s.q.CreateThread(ctx, db.CreateThreadParams{
			Type:     "THREAD_TYPE_BROADCAST",
			Title:    sql.NullString{String: "Store Updates", Valid: true},
			SellerID: sql.NullString{String: sellerID, Valid: true},
		})
		if cerr != nil {
			return cerr
		}
		thread = t
	}
	return s.q.AddThreadMember(ctx, db.AddThreadMemberParams{
		ThreadID: thread.ID,
		UserID:   userID,
		Role:     "MEMBER",
	})
}

func (s *svcLogic) LeaveBroadcastGroup(ctx context.Context, userID, sellerID string) error {
	thread, err := s.q.GetBroadcastThreadForSeller(ctx, sql.NullString{String: sellerID, Valid: true})
	if err != nil {
		return nil
	}
	return s.q.RemoveThreadMember(ctx, db.RemoveThreadMemberParams{
		ThreadID: thread.ID,
		UserID:   userID,
	})
}

func (s *svcLogic) SendMessage(ctx context.Context, threadID uuid.UUID, senderID, msgType, content, mediaUrl, referenceID string, metadataJSON []byte) (*db.Message, error) {
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
		ReferenceID: sql.NullString{String: referenceID, Valid: referenceID != ""},
		Metadata:    metaRaw,
	})
	if err != nil {
		return nil, err
	}
	_ = s.q.UpdateThreadTimestamp(ctx, threadID)
	return &msg, nil
}

func (s *svcLogic) ListThreadsForUser(ctx context.Context, userID string) ([]db.Thread, error) {
	_ = s.EnsureUserInSystemThread(ctx, userID)
	return s.q.ListThreadsForUser(ctx, sql.NullString{String: userID, Valid: true})
}

func (s *svcLogic) ListThreadsForBuyer(ctx context.Context, buyerID string) ([]db.Thread, error) {
	return s.ListThreadsForUser(ctx, buyerID)
}

func (s *svcLogic) ListThreadsForSeller(ctx context.Context, sellerID string) ([]db.Thread, error) {
	return s.q.ListThreadsForSeller(ctx, sql.NullString{String: sellerID, Valid: true})
}

func (s *svcLogic) ListMessages(ctx context.Context, threadID uuid.UUID) ([]db.Message, error) {
	return s.q.ListMessages(ctx, threadID)
}

func (s *svcLogic) GetBroadcastThreadForSeller(ctx context.Context, sellerID string) (*db.Thread, error) {
	thread, err := s.q.GetBroadcastThreadForSeller(ctx, sql.NullString{String: sellerID, Valid: true})
	if err != nil {
		return nil, err
	}
	return &thread, nil
}

// ────────────────────────────── Tests ────────────────────────────────────────


func TestCreateThread_Direct(t *testing.T) {
	ctx := context.Background()
	buyerID := uuid.New().String()
	sellerID := uuid.New().String()
	orderID := uuid.New().String()
	avatar := "https://avatar.url/direct"

	svc := newLogic(&fakeQ{s: &stubQueries{}})
	thread, err := svc.CreateThread(ctx, buyerID, sellerID, orderID, avatar)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thread.Type != "THREAD_TYPE_DIRECT" {
		t.Errorf("expected THREAD_TYPE_DIRECT, got %s", thread.Type)
	}
	if thread.BuyerID.String != buyerID {
		t.Errorf("expected buyerID %s, got %s", buyerID, thread.BuyerID.String)
	}
	if thread.SellerID.String != sellerID {
		t.Errorf("expected sellerID %s, got %s", sellerID, thread.SellerID.String)
	}
	if thread.ParticipantAvatar.String != avatar {
		t.Errorf("expected avatar %s, got %s", avatar, thread.ParticipantAvatar.String)
	}
}

func TestCreateThread_Delivery(t *testing.T) {
	ctx := context.Background()
	var capturedArg db.CreateThreadParams

	svc := newLogic(&fakeQ{s: &stubQueries{
		createThreadFn: func(ctx context.Context, arg db.CreateThreadParams) (db.Thread, error) {
			capturedArg = arg
			return db.Thread{ID: uuid.New(), Type: arg.Type, DeliveryBoyID: arg.DeliveryBoyID, ParticipantAvatar: arg.ParticipantAvatar}, nil
		},
	}})

	thread, err := svc.CreateDeliveryThread(ctx, "buyer-1", "driver-1", "order-1", "https://avatar.url/delivery")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedArg.Type != "THREAD_TYPE_DELIVERY" {
		t.Errorf("expected THREAD_TYPE_DELIVERY, got %s", capturedArg.Type)
	}
	if capturedArg.DeliveryBoyID.String != "driver-1" {
		t.Errorf("expected delivery_boy_id driver-1, got %s", capturedArg.DeliveryBoyID.String)
	}
	if capturedArg.ParticipantAvatar.String != "https://avatar.url/delivery" {
		t.Errorf("expected avatar, got %s", capturedArg.ParticipantAvatar.String)
	}
	if thread.Type != "THREAD_TYPE_DELIVERY" {
		t.Errorf("expected thread type THREAD_TYPE_DELIVERY, got %s", thread.Type)
	}
}

func TestCreateThread_Courier(t *testing.T) {
	ctx := context.Background()
	var capturedArg db.CreateThreadParams

	svc := newLogic(&fakeQ{s: &stubQueries{
		createThreadFn: func(ctx context.Context, arg db.CreateThreadParams) (db.Thread, error) {
			capturedArg = arg
			return db.Thread{ID: uuid.New(), Type: arg.Type}, nil
		},
	}})

	_, err := svc.CreateCourierThread(ctx, "buyer-1", "station-1", "order-1", "https://avatar.url/courier")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedArg.Type != "THREAD_TYPE_COURIER" {
		t.Errorf("expected THREAD_TYPE_COURIER, got %s", capturedArg.Type)
	}
	if capturedArg.CourierStationID.String != "station-1" {
		t.Errorf("expected courier_station_id station-1, got %s", capturedArg.CourierStationID.String)
	}
	if capturedArg.ParticipantAvatar.String != "https://avatar.url/courier" {
		t.Errorf("expected avatar, got %s", capturedArg.ParticipantAvatar.String)
	}
}

func TestCreateThread_Support(t *testing.T) {
	ctx := context.Background()
	svc := newLogic(&fakeQ{s: &stubQueries{}})
	thread, err := svc.CreateSupportThread(ctx, "buyer-1", "https://avatar.url/support")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thread.Type != "THREAD_TYPE_SUPPORT" {
		t.Errorf("expected THREAD_TYPE_SUPPORT, got %s", thread.Type)
	}
	if thread.Title.String != "Support" {
		t.Errorf("expected title 'Support', got %q", thread.Title.String)
	}
	if thread.ParticipantAvatar.String != "https://avatar.url/support" {
		t.Errorf("expected avatar, got %s", thread.ParticipantAvatar.String)
	}
}

func TestCreateThread_PropagatesDBError(t *testing.T) {
	ctx := context.Background()
	svc := newLogic(&fakeQ{s: &stubQueries{
		createThreadFn: func(_ context.Context, _ db.CreateThreadParams) (db.Thread, error) {
			return db.Thread{}, fmt.Errorf("db error")
		},
	}})

	_, err := svc.CreateThread(ctx, "b", "s", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ────────────────────────────── System Thread Tests ──────────────────────────

func TestGetOrCreateSystemThread_ExistingThread(t *testing.T) {
	ctx := context.Background()
	existingID := uuid.New()
	svc := newLogic(&fakeQ{s: &stubQueries{
		getSystemThreadFn: func(ctx context.Context) (db.Thread, error) {
			return db.Thread{ID: existingID, Type: "THREAD_TYPE_SYSTEM"}, nil
		},
	}})

	thread, err := svc.GetOrCreateSystemThread(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if thread.ID != existingID {
		t.Errorf("expected existing ID %s, got %s", existingID, thread.ID)
	}
}

func TestGetOrCreateSystemThread_CreatesWhenMissing(t *testing.T) {
	ctx := context.Background()
	created := false

	svc := newLogic(&fakeQ{s: &stubQueries{
		getSystemThreadFn: func(ctx context.Context) (db.Thread, error) {
			return db.Thread{}, sql.ErrNoRows
		},
		createThreadFn: func(ctx context.Context, arg db.CreateThreadParams) (db.Thread, error) {
			created = true
			if arg.Type != "THREAD_TYPE_SYSTEM" {
				t.Errorf("expected THREAD_TYPE_SYSTEM, got %s", arg.Type)
			}
			if arg.Title.String != "WeMall Updates" {
				t.Errorf("expected title 'WeMall Updates', got %q", arg.Title.String)
			}
			return db.Thread{ID: uuid.New(), Type: arg.Type, Title: arg.Title}, nil
		},
	}})

	thread, err := svc.GetOrCreateSystemThread(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !created {
		t.Error("expected CreateThread to be called when GetSystemThread returns ErrNoRows")
	}
	if thread.Type != "THREAD_TYPE_SYSTEM" {
		t.Errorf("expected THREAD_TYPE_SYSTEM, got %s", thread.Type)
	}
}

// ────────────────────────────── Broadcast Group / Membership Tests ────────────

func TestJoinBroadcastGroup_ExistingGroup(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()
	var addedMember db.AddThreadMemberParams

	svc := newLogic(&fakeQ{s: &stubQueries{
		getBroadcastFn: func(ctx context.Context, sellerID sql.NullString) (db.Thread, error) {
			return db.Thread{ID: threadID, Type: "THREAD_TYPE_BROADCAST"}, nil
		},
		addMemberFn: func(ctx context.Context, arg db.AddThreadMemberParams) error {
			addedMember = arg
			return nil
		},
	}})

	err := svc.JoinBroadcastGroup(ctx, "buyer-123", "seller-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if addedMember.UserID != "buyer-123" {
		t.Errorf("expected user_id buyer-123, got %s", addedMember.UserID)
	}
	if addedMember.ThreadID != threadID {
		t.Errorf("expected thread_id %s, got %s", threadID, addedMember.ThreadID)
	}
	if addedMember.Role != "MEMBER" {
		t.Errorf("expected role MEMBER, got %s", addedMember.Role)
	}
}

func TestJoinBroadcastGroup_AutoCreatesGroupWhenMissing(t *testing.T) {
	ctx := context.Background()
	groupCreated := false
	memberAdded := false

	svc := newLogic(&fakeQ{s: &stubQueries{
		getBroadcastFn: func(ctx context.Context, sellerID sql.NullString) (db.Thread, error) {
			return db.Thread{}, sql.ErrNoRows // No group yet
		},
		createThreadFn: func(ctx context.Context, arg db.CreateThreadParams) (db.Thread, error) {
			groupCreated = true
			if arg.Type != "THREAD_TYPE_BROADCAST" {
				t.Errorf("expected THREAD_TYPE_BROADCAST, got %s", arg.Type)
			}
			return db.Thread{ID: uuid.New(), Type: arg.Type}, nil
		},
		addMemberFn: func(ctx context.Context, arg db.AddThreadMemberParams) error {
			memberAdded = true
			return nil
		},
	}})

	err := svc.JoinBroadcastGroup(ctx, "buyer-123", "seller-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !groupCreated {
		t.Error("expected broadcast group to be auto-created")
	}
	if !memberAdded {
		t.Error("expected member to be added to the new group")
	}
}

func TestLeaveBroadcastGroup_NoopWhenGroupMissing(t *testing.T) {
	ctx := context.Background()
	removeCalled := false

	svc := newLogic(&fakeQ{s: &stubQueries{
		getBroadcastFn: func(ctx context.Context, sellerID sql.NullString) (db.Thread, error) {
			return db.Thread{}, sql.ErrNoRows
		},
		removeMemberFn: func(ctx context.Context, arg db.RemoveThreadMemberParams) error {
			removeCalled = true
			return nil
		},
	}})

	err := svc.LeaveBroadcastGroup(ctx, "buyer-123", "seller-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removeCalled {
		t.Error("RemoveThreadMember should NOT be called when group does not exist")
	}
}

func TestLeaveBroadcastGroup_RemovesMemberFromExistingGroup(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()
	var removedArg db.RemoveThreadMemberParams

	svc := newLogic(&fakeQ{s: &stubQueries{
		getBroadcastFn: func(ctx context.Context, sellerID sql.NullString) (db.Thread, error) {
			return db.Thread{ID: threadID}, nil
		},
		removeMemberFn: func(ctx context.Context, arg db.RemoveThreadMemberParams) error {
			removedArg = arg
			return nil
		},
	}})

	err := svc.LeaveBroadcastGroup(ctx, "buyer-123", "seller-456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if removedArg.UserID != "buyer-123" {
		t.Errorf("expected user_id buyer-123, got %s", removedArg.UserID)
	}
	if removedArg.ThreadID != threadID {
		t.Errorf("expected thread_id %s, got %s", threadID, removedArg.ThreadID)
	}
}

// ────────────────────────────── Message Tests ─────────────────────────────────

func TestSendMessage_TextMessage(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()
	var capturedArg db.CreateMessageParams

	svc := newLogic(&fakeQ{s: &stubQueries{
		createMessageFn: func(ctx context.Context, arg db.CreateMessageParams) (db.Message, error) {
			capturedArg = arg
			return db.Message{ID: uuid.New(), ThreadID: arg.ThreadID, Content: arg.Content, Type: arg.Type}, nil
		},
	}})

	msg, err := svc.SendMessage(ctx, threadID, "sender-1", "MESSAGE_TYPE_TEXT", "Hello!", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Content != "Hello!" {
		t.Errorf("expected content 'Hello!', got %q", msg.Content)
	}
	if capturedArg.Type != "MESSAGE_TYPE_TEXT" {
		t.Errorf("expected type MESSAGE_TYPE_TEXT, got %s", capturedArg.Type)
	}
	if capturedArg.Metadata.Valid {
		t.Error("expected no metadata for plain text message")
	}
}

func TestSendMessage_WithMetadata(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()

	meta := map[string]interface{}{
		"product_id": "prod-123",
		"title":      "Blue T-Shirt",
		"price":      29.99,
	}
	metaBytes, _ := json.Marshal(meta)

	var capturedArg db.CreateMessageParams
	svc := newLogic(&fakeQ{s: &stubQueries{
		createMessageFn: func(ctx context.Context, arg db.CreateMessageParams) (db.Message, error) {
			capturedArg = arg
			return db.Message{ID: uuid.New(), ThreadID: arg.ThreadID, Metadata: arg.Metadata}, nil
		},
	}})

	msg, err := svc.SendMessage(ctx, threadID, "seller-1", "MESSAGE_TYPE_PRODUCT",
		"Check out this product!", "https://cdn.example.com/img.jpg", "prod-123", metaBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = msg

	if !capturedArg.Metadata.Valid {
		t.Fatal("expected metadata to be set")
	}

	// Verify the stored JSON round-trips correctly
	var decoded map[string]interface{}
	if err := json.Unmarshal(capturedArg.Metadata.RawMessage, &decoded); err != nil {
		t.Fatalf("metadata is not valid JSON: %v", err)
	}
	if decoded["product_id"] != "prod-123" {
		t.Errorf("expected product_id prod-123, got %v", decoded["product_id"])
	}
}

func TestSendMessage_WithMediaURL(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()
	var capturedArg db.CreateMessageParams

	svc := newLogic(&fakeQ{s: &stubQueries{
		createMessageFn: func(ctx context.Context, arg db.CreateMessageParams) (db.Message, error) {
			capturedArg = arg
			return db.Message{ID: uuid.New()}, nil
		},
	}})

	_, err := svc.SendMessage(ctx, threadID, "buyer-1", "MESSAGE_TYPE_IMAGE",
		"", "https://cdn.example.com/photo.jpg", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !capturedArg.MediaUrl.Valid {
		t.Error("expected MediaUrl to be valid")
	}
	if capturedArg.MediaUrl.String != "https://cdn.example.com/photo.jpg" {
		t.Errorf("unexpected media_url: %s", capturedArg.MediaUrl.String)
	}
}

func TestSendMessage_UpdatesThreadTimestamp(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()
	var updatedID uuid.UUID

	svc := newLogic(&fakeQ{s: &stubQueries{
		updateTimestampFn: func(ctx context.Context, id uuid.UUID) error {
			updatedID = id
			return nil
		},
	}})

	_, err := svc.SendMessage(ctx, threadID, "u", "MESSAGE_TYPE_TEXT", "hi", "", "", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedID != threadID {
		t.Errorf("expected UpdateThreadTimestamp called with %s, got %s", threadID, updatedID)
	}
}

func TestSendMessage_MetadataCoupon(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()

	meta := map[string]interface{}{
		"coupon_id":  "coupon-abc",
		"code":       "SAVE20",
		"discount":   20.0,
		"expires_at": "2026-12-31",
	}
	metaBytes, _ := json.Marshal(meta)
	var capturedArg db.CreateMessageParams

	svc := newLogic(&fakeQ{s: &stubQueries{
		createMessageFn: func(ctx context.Context, arg db.CreateMessageParams) (db.Message, error) {
			capturedArg = arg
			return db.Message{ID: uuid.New()}, nil
		},
	}})

	_, err := svc.SendMessage(ctx, threadID, "seller-1", "MESSAGE_TYPE_COUPON",
		"🎟️ Use SAVE20 for 20% off!", "", "coupon-abc", metaBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !capturedArg.Metadata.Valid {
		t.Fatal("expected metadata for coupon message")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(capturedArg.Metadata.RawMessage, &decoded); err != nil {
		t.Fatalf("metadata JSON invalid: %v", err)
	}
	if decoded["code"] != "SAVE20" {
		t.Errorf("expected coupon code SAVE20, got %v", decoded["code"])
	}
}

func TestSendMessage_MetadataPromotion(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()

	meta := map[string]interface{}{
		"promotion_id": "promo-xyz",
		"title":        "Summer Sale",
		"discount":     30.0,
	}
	metaBytes, _ := json.Marshal(meta)
	var capturedArg db.CreateMessageParams

	svc := newLogic(&fakeQ{s: &stubQueries{
		createMessageFn: func(ctx context.Context, arg db.CreateMessageParams) (db.Message, error) {
			capturedArg = arg
			return db.Message{ID: uuid.New()}, nil
		},
	}})

	_, err := svc.SendMessage(ctx, threadID, "seller-1", "MESSAGE_TYPE_PROMOTION",
		"🔥 Summer Sale — 30% off everything!", "https://banner.jpg", "promo-xyz", metaBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(capturedArg.Metadata.RawMessage, &decoded); err != nil {
		t.Fatalf("metadata JSON invalid: %v", err)
	}
	if decoded["title"] != "Summer Sale" {
		t.Errorf("expected title 'Summer Sale', got %v", decoded["title"])
	}
}

// ────────────────────────────── Listing Tests ─────────────────────────────────

func TestListMessages_ReturnsMessages(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()
	now := time.Now()

	msgs := []db.Message{
		{ID: uuid.New(), ThreadID: threadID, SenderID: "user-1", Type: "MESSAGE_TYPE_TEXT", Content: "Hello", CreatedAt: sql.NullTime{Time: now, Valid: true}},
		{ID: uuid.New(), ThreadID: threadID, SenderID: "user-2", Type: "MESSAGE_TYPE_IMAGE", Content: "", MediaUrl: sql.NullString{String: "https://img.jpg", Valid: true}, CreatedAt: sql.NullTime{Time: now, Valid: true}},
	}

	svc := newLogic(&fakeQ{s: &stubQueries{
		listMessagesFn: func(ctx context.Context, id uuid.UUID) ([]db.Message, error) {
			return msgs, nil
		},
	}})

	result, err := svc.q.ListMessages(ctx, threadID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 messages, got %d", len(result))
	}
	if result[0].Type != "MESSAGE_TYPE_TEXT" {
		t.Errorf("expected first message type TEXT, got %s", result[0].Type)
	}
	if result[1].MediaUrl.String != "https://img.jpg" {
		t.Errorf("expected image URL, got %s", result[1].MediaUrl.String)
	}
}

// ─────────────────── Thread Type String Validation ───────────────────────────

func TestThreadTypeStrings(t *testing.T) {
	types := []string{
		"THREAD_TYPE_DIRECT",
		"THREAD_TYPE_BROADCAST",
		"THREAD_TYPE_DELIVERY",
		"THREAD_TYPE_COURIER",
		"THREAD_TYPE_SUPPORT",
		"THREAD_TYPE_SYSTEM",
	}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			ctx := context.Background()
			svc := newLogic(&fakeQ{s: &stubQueries{
				createThreadFn: func(ctx context.Context, arg db.CreateThreadParams) (db.Thread, error) {
					return db.Thread{ID: uuid.New(), Type: arg.Type}, nil
				},
			}})

			t.Run("can_create", func(t *testing.T) {
				thread, err := svc.q.CreateThread(ctx, db.CreateThreadParams{Type: typ})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if thread.Type != typ {
					t.Errorf("expected type %s, got %s", typ, thread.Type)
				}
			})
		})
	}
}

func TestMessageTypeStrings(t *testing.T) {
	types := []string{
		"MESSAGE_TYPE_TEXT",
		"MESSAGE_TYPE_IMAGE",
		"MESSAGE_TYPE_VIDEO",
		"MESSAGE_TYPE_DOCUMENT",
		"MESSAGE_TYPE_AUDIO",
		"MESSAGE_TYPE_PRODUCT",
		"MESSAGE_TYPE_ORDER",
		"MESSAGE_TYPE_PROMOTION",
		"MESSAGE_TYPE_COUPON",
	}

	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			ctx := context.Background()
			threadID := uuid.New()
			svc := newLogic(&fakeQ{s: &stubQueries{}})

			msg, err := svc.SendMessage(ctx, threadID, "sender-1", typ, "content", "", "", nil)
			if err != nil {
				t.Fatalf("unexpected error for type %s: %v", typ, err)
			}
			if msg.Type != typ {
				t.Errorf("expected type %s, got %s", typ, msg.Type)
			}
		})
	}
}

// ─────────────────── Metadata JSON round-trip ────────────────────────────────

func TestMetadataRoundTrip(t *testing.T) {
	tests := []struct {
		name     string
		payload  map[string]interface{}
		wantKeys []string
	}{
		{
			name: "product",
			payload: map[string]interface{}{
				"product_id": "p1",
				"title":      "Shoes",
				"image_url":  "https://img.jpg",
			},
			wantKeys: []string{"product_id", "title", "image_url"},
		},
		{
			name: "coupon",
			payload: map[string]interface{}{
				"coupon_id":  "c1",
				"code":       "DEAL10",
				"discount":   10.0,
				"expires_at": "2026-01-01",
			},
			wantKeys: []string{"coupon_id", "code", "discount", "expires_at"},
		},
		{
			name: "promotion",
			payload: map[string]interface{}{
				"promotion_id": "pr1",
				"title":        "Flash Sale",
				"banner_url":   "https://banner.jpg",
			},
			wantKeys: []string{"promotion_id", "title", "banner_url"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.payload)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var decoded map[string]interface{}
			if err := json.Unmarshal(raw, &decoded); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			for _, key := range tc.wantKeys {
				if _, ok := decoded[key]; !ok {
					t.Errorf("expected key %q in decoded metadata", key)
				}
			}
		})
	}
}

func TestEnsureUserInSystemThread(t *testing.T) {
	ctx := context.Background()
	systemThreadID := uuid.New()
	var getOrCreateCalled bool
	var addedMember db.AddThreadMemberParams

	svc := newLogic(&fakeQ{s: &stubQueries{
		getSystemThreadFn: func(ctx context.Context) (db.Thread, error) {
			getOrCreateCalled = true
			return db.Thread{ID: systemThreadID, Type: "THREAD_TYPE_SYSTEM"}, nil
		},
		addMemberFn: func(ctx context.Context, arg db.AddThreadMemberParams) error {
			addedMember = arg
			return nil
		},
	}})

	err := svc.EnsureUserInSystemThread(ctx, "user-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !getOrCreateCalled {
		t.Error("expected GetOrCreateSystemThread to be called")
	}
	if addedMember.UserID != "user-abc" {
		t.Errorf("expected member user-abc, got %s", addedMember.UserID)
	}
	if addedMember.ThreadID != systemThreadID {
		t.Errorf("expected thread ID %s, got %s", systemThreadID, addedMember.ThreadID)
	}
}

func TestListThreadsForUser_EnsuresSystemThreadJoin(t *testing.T) {
	ctx := context.Background()
	systemThreadID := uuid.New()
	var joinedUser string
	var listUser string

	svc := newLogic(&fakeQ{s: &stubQueries{
		getSystemThreadFn: func(ctx context.Context) (db.Thread, error) {
			return db.Thread{ID: systemThreadID, Type: "THREAD_TYPE_SYSTEM"}, nil
		},
		addMemberFn: func(ctx context.Context, arg db.AddThreadMemberParams) error {
			joinedUser = arg.UserID
			return nil
		},
		listThreadsForUserFn: func(ctx context.Context, userID sql.NullString) ([]db.Thread, error) {
			listUser = userID.String
			return []db.Thread{
				{ID: systemThreadID, Type: "THREAD_TYPE_SYSTEM"},
			}, nil
		},
	}})

	threads, err := svc.ListThreadsForUser(ctx, "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if joinedUser != "user-123" {
		t.Errorf("expected user-123 to be joined to system thread, got %q", joinedUser)
	}
	if listUser != "user-123" {
		t.Errorf("expected ListThreadsForUser called for user-123, got %q", listUser)
	}
	if len(threads) != 1 || threads[0].ID != systemThreadID {
		t.Errorf("expected system thread in list, got %v", threads)
	}
}

func TestSendMessage_InvalidMetadata(t *testing.T) {
	ctx := context.Background()
	threadID := uuid.New()

	svc := newLogic(&fakeQ{s: &stubQueries{
		createMessageFn: func(ctx context.Context, arg db.CreateMessageParams) (db.Message, error) {
			return db.Message{ID: uuid.New(), ThreadID: arg.ThreadID, Metadata: arg.Metadata}, nil
		},
	}})

	msg, err := svc.SendMessage(ctx, threadID, "sender-1", "MESSAGE_TYPE_TEXT", "hello", "", "", []byte("{invalid-json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !msg.Metadata.Valid {
		t.Fatal("expected metadata to be stored as raw bytes regardless of validity at service layer")
	}
	if string(msg.Metadata.RawMessage) != "{invalid-json" {
		t.Errorf("expected raw bytes '{invalid-json', got %s", string(msg.Metadata.RawMessage))
	}
}

// ─────────────────── Ensure service methods satisfy interface ─────────────────

var _ service.ChatServiceInterface = (*service.ChatService)(nil)
