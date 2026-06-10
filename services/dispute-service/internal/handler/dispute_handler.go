package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	disputev1 "github.com/wemall/gen/dispute/v1"
	"github.com/wemall/dispute-service/internal/db"
	"github.com/wemall/dispute-service/internal/service"
	werr "github.com/wemall/pkg/errors"
)

type DisputeHandler struct {
	disputev1.UnimplementedDisputeServiceServer
	svc *service.DisputeService
}

func NewDisputeHandler(svc *service.DisputeService) *DisputeHandler {
	return &DisputeHandler{svc: svc}
}

func (h *DisputeHandler) OpenDispute(ctx context.Context, req *disputev1.OpenDisputeRequest) (*disputev1.OpenDisputeResponse, error) {
	// Note: in a real system we should fetch seller_id from the order-service.
	// We'll mock it here or expect it via context.
	sellerID := "mock_seller_id"

	dispute, err := h.svc.OpenDispute(ctx, req.OrderId, req.BuyerId, sellerID, req.Reason, req.EvidenceUrls)
	if err != nil {
		return nil, err
	}

	return &disputev1.OpenDisputeResponse{
		Dispute: mapToPbDispute(dispute),
	}, nil
}

func (h *DisputeHandler) ReplyToDispute(ctx context.Context, req *disputev1.ReplyToDisputeRequest) (*disputev1.ReplyToDisputeResponse, error) {
	uid, err := uuid.Parse(req.DisputeId)
	if err != nil {
		return nil, werr.InvalidArgument("invalid dispute_id")
	}

	msg, err := h.svc.ReplyToDispute(ctx, uid, req.SenderId, req.Message, req.EvidenceUrls)
	if err != nil {
		return nil, err
	}

	return &disputev1.ReplyToDisputeResponse{
		Message: mapToPbMessage(msg),
	}, nil
}

func (h *DisputeHandler) EscalateDispute(ctx context.Context, req *disputev1.EscalateDisputeRequest) (*disputev1.Dispute, error) {
	uid, err := uuid.Parse(req.DisputeId)
	if err != nil {
		return nil, werr.InvalidArgument("invalid dispute_id")
	}

	dispute, err := h.svc.EscalateDispute(ctx, uid)
	if err != nil {
		return nil, err
	}

	return mapToPbDispute(dispute), nil
}

func (h *DisputeHandler) ResolveDispute(ctx context.Context, req *disputev1.ResolveDisputeRequest) (*disputev1.Dispute, error) {
	uid, err := uuid.Parse(req.DisputeId)
	if err != nil {
		return nil, werr.InvalidArgument("invalid dispute_id")
	}

	resStr := ""
	switch req.Resolution {
	case disputev1.DisputeStatus_DISPUTE_STATUS_RESOLVED_REFUNDED:
		resStr = "DISPUTE_STATUS_RESOLVED_REFUNDED"
	case disputev1.DisputeStatus_DISPUTE_STATUS_RESOLVED_REJECTED:
		resStr = "DISPUTE_STATUS_RESOLVED_REJECTED"
	default:
		return nil, werr.InvalidArgument("invalid resolution status")
	}

	dispute, err := h.svc.ResolveDispute(ctx, uid, resStr)
	if err != nil {
		return nil, err
	}

	return mapToPbDispute(dispute), nil
}

func (h *DisputeHandler) GetDispute(ctx context.Context, req *disputev1.GetDisputeRequest) (*disputev1.Dispute, error) {
	uid, err := uuid.Parse(req.DisputeId)
	if err != nil {
		return nil, werr.InvalidArgument("invalid dispute_id")
	}

	dispute, err := h.svc.GetDispute(ctx, uid)
	if err != nil {
		return nil, err
	}

	return mapToPbDispute(dispute), nil
}

func (h *DisputeHandler) ListDisputes(ctx context.Context, req *disputev1.ListDisputesRequest) (*disputev1.ListDisputesResponse, error) {
	var disputes []db.Dispute
	var err error

	if req.Role == "BUYER" {
		disputes, err = h.svc.ListDisputesByBuyer(ctx, req.UserId)
	} else if req.Role == "SELLER" {
		disputes, err = h.svc.ListDisputesBySeller(ctx, req.UserId)
	} else {
		disputes, err = h.svc.ListAllDisputes(ctx)
	}

	if err != nil {
		return nil, err
	}

	pbDisputes := make([]*disputev1.Dispute, 0, len(disputes))
	for _, d := range disputes {
		pbDisputes = append(pbDisputes, mapToPbDispute(&d))
	}

	return &disputev1.ListDisputesResponse{
		Disputes: pbDisputes,
	}, nil
}

func (h *DisputeHandler) ListDisputeMessages(ctx context.Context, req *disputev1.ListDisputeMessagesRequest) (*disputev1.ListDisputeMessagesResponse, error) {
	uid, err := uuid.Parse(req.DisputeId)
	if err != nil {
		return nil, werr.InvalidArgument("invalid dispute_id")
	}

	msgs, err := h.svc.ListDisputeMessages(ctx, uid)
	if err != nil {
		return nil, err
	}

	pbMsgs := make([]*disputev1.DisputeMessage, 0, len(msgs))
	for _, m := range msgs {
		pbMsgs = append(pbMsgs, mapToPbMessage(&m))
	}

	return &disputev1.ListDisputeMessagesResponse{
		Messages: pbMsgs,
	}, nil
}

func mapToPbDispute(d *db.Dispute) *disputev1.Dispute {
	status := disputev1.DisputeStatus_DISPUTE_STATUS_UNSPECIFIED
	switch d.Status {
	case "DISPUTE_STATUS_OPEN":
		status = disputev1.DisputeStatus_DISPUTE_STATUS_OPEN
	case "DISPUTE_STATUS_SELLER_REVIEW":
		status = disputev1.DisputeStatus_DISPUTE_STATUS_SELLER_REVIEW
	case "DISPUTE_STATUS_ESCALATED":
		status = disputev1.DisputeStatus_DISPUTE_STATUS_ESCALATED
	case "DISPUTE_STATUS_RESOLVED_REFUNDED":
		status = disputev1.DisputeStatus_DISPUTE_STATUS_RESOLVED_REFUNDED
	case "DISPUTE_STATUS_RESOLVED_REJECTED":
		status = disputev1.DisputeStatus_DISPUTE_STATUS_RESOLVED_REJECTED
	}

	return &disputev1.Dispute{
		Id:        d.ID.String(),
		OrderId:   d.OrderID,
		BuyerId:   d.BuyerID,
		SellerId:  d.SellerID,
		Reason:    d.Reason,
		Status:    status,
		CreatedAt: timestamppb.New(d.CreatedAt),
		UpdatedAt: timestamppb.New(d.UpdatedAt),
	}
}

func mapToPbMessage(m *db.DisputeMessage) *disputev1.DisputeMessage {
	return &disputev1.DisputeMessage{
		Id:           m.ID.String(),
		DisputeId:    m.DisputeID.String(),
		SenderId:     m.SenderID,
		Content:      m.Content,
		EvidenceUrls: m.EvidenceUrls,
		CreatedAt:    timestamppb.New(m.CreatedAt),
	}
}
