package service

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/wemall/dispute-service/internal/db"
	werr "github.com/wemall/pkg/errors"
)

type DisputeService struct {
	q  *db.Queries
	nc *nats.Conn
}

func NewDisputeService(q *db.Queries, nc *nats.Conn) *DisputeService {
	return &DisputeService{
		q:  q,
		nc: nc,
	}
}

func (s *DisputeService) OpenDispute(ctx context.Context, orderID, buyerID, sellerID, reason string, evidenceUrls []string) (*db.Dispute, error) {
	dispute, err := s.q.CreateDispute(ctx, db.CreateDisputeParams{
		OrderID:  orderID,
		BuyerID:  buyerID,
		SellerID: sellerID,
		Reason:   reason,
		Status:   "DISPUTE_STATUS_OPEN",
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	// Publish NATS event
	if s.nc != nil {
		event := map[string]interface{}{
			"dispute_id": dispute.ID.String(),
			"order_id":   orderID,
			"buyer_id":   buyerID,
			"seller_id":  sellerID,
			"reason":     reason,
		}
		eb, _ := json.Marshal(event)
		_ = s.nc.Publish("wemall.dispute.opened", eb)
	}

	return &dispute, nil
}

func (s *DisputeService) ReplyToDispute(ctx context.Context, disputeID uuid.UUID, senderID, message string, evidenceUrls []string) (*db.DisputeMessage, error) {
	msg, err := s.q.CreateDisputeMessage(ctx, db.CreateDisputeMessageParams{
		DisputeID:    disputeID,
		SenderID:     senderID,
		Content:      message,
		EvidenceUrls: evidenceUrls,
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	// In a real app we'd update the dispute updated_at in a tx here.
	return &msg, nil
}

func (s *DisputeService) EscalateDispute(ctx context.Context, disputeID uuid.UUID) (*db.Dispute, error) {
	dispute, err := s.q.UpdateDisputeStatus(ctx, db.UpdateDisputeStatusParams{
		ID:     disputeID,
		Status: "DISPUTE_STATUS_ESCALATED",
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	return &dispute, nil
}

func (s *DisputeService) ResolveDispute(ctx context.Context, disputeID uuid.UUID, resolution string) (*db.Dispute, error) {
	dispute, err := s.q.UpdateDisputeStatus(ctx, db.UpdateDisputeStatusParams{
		ID:     disputeID,
		Status: resolution,
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	// Publish NATS event to trigger refund if RESOLVED_REFUNDED
	if s.nc != nil {
		event := map[string]interface{}{
			"dispute_id": dispute.ID.String(),
			"order_id":   dispute.OrderID,
			"resolution": resolution,
		}
		eb, _ := json.Marshal(event)
		_ = s.nc.Publish("wemall.dispute.resolved", eb)
	}

	return &dispute, nil
}

func (s *DisputeService) GetDispute(ctx context.Context, id uuid.UUID) (*db.Dispute, error) {
	dispute, err := s.q.GetDispute(ctx, id)
	if err != nil {
		return nil, werr.Internal(err)
	}
	return &dispute, nil
}

func (s *DisputeService) ListDisputesByBuyer(ctx context.Context, buyerID string) ([]db.Dispute, error) {
	return s.q.ListDisputesByBuyer(ctx, buyerID)
}

func (s *DisputeService) ListDisputesBySeller(ctx context.Context, sellerID string) ([]db.Dispute, error) {
	return s.q.ListDisputesBySeller(ctx, sellerID)
}

func (s *DisputeService) ListAllDisputes(ctx context.Context) ([]db.Dispute, error) {
	return s.q.ListAllDisputes(ctx)
}

func (s *DisputeService) ListDisputeMessages(ctx context.Context, disputeID uuid.UUID) ([]db.DisputeMessage, error) {
	return s.q.ListDisputeMessages(ctx, disputeID)
}
