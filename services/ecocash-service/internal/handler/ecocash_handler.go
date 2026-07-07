package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	ecocashv1 "github.com/wemall/gen/ecocash/v1"
	"github.com/wemall/ecocash-service/internal/db"
	"github.com/wemall/ecocash-service/internal/service"
)

// EcoCashHandler implements the ecocashv1.EcoCashServiceServer gRPC interface.
type EcoCashHandler struct {
	ecocashv1.UnimplementedEcoCashServiceServer
	svc *service.EcoCashService
}

func NewEcoCashHandler(svc *service.EcoCashService) *EcoCashHandler {
	return &EcoCashHandler{svc: svc}
}

// ── Payment Flows ─────────────────────────────────────────────────────────────

func (h *EcoCashHandler) ChargeCustomer(ctx context.Context, req *ecocashv1.ChargeCustomerRequest) (*ecocashv1.ChargeCustomerResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	txn, err := h.svc.ChargeCustomer(ctx, orderID, req.Msisdn, req.AmountCents, req.Currency)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	return &ecocashv1.ChargeCustomerResponse{
		Transaction: mapTransaction(txn),
		StatusMsg:   safeStr(txn.EcocashStatusMsg),
	}, nil
}

func (h *EcoCashHandler) LookupTransaction(ctx context.Context, req *ecocashv1.LookupTransactionRequest) (*ecocashv1.LookupTransactionResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	txn, err := h.svc.LookupTransaction(ctx, orderID, req.ClientCorrelator)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "lookup failed: %v", err)
	}

	return &ecocashv1.LookupTransactionResponse{
		Transaction: mapTransaction(txn),
	}, nil
}

func (h *EcoCashHandler) RefundTransaction(ctx context.Context, req *ecocashv1.RefundTransactionRequest) (*ecocashv1.RefundTransactionResponse, error) {
	origID, err := uuid.Parse(req.OriginalTxnId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid original_txn_id: %v", err)
	}

	refund, err := h.svc.RefundTransaction(ctx, origID, req.AmountCents, req.Reason)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	return &ecocashv1.RefundTransactionResponse{
		Refund: mapRefundRecord(refund),
	}, nil
}

func (h *EcoCashHandler) ReverseTransaction(ctx context.Context, req *ecocashv1.ReverseTransactionRequest) (*ecocashv1.ReverseTransactionResponse, error) {
	origID, err := uuid.Parse(req.OriginalTxnId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid original_txn_id: %v", err)
	}

	reversal, err := h.svc.ReverseTransaction(ctx, origID, req.Reason)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	return &ecocashv1.ReverseTransactionResponse{
		Reversal: mapRefundRecord(reversal),
	}, nil
}

// ── Transaction Queries ───────────────────────────────────────────────────────

func (h *EcoCashHandler) GetTransaction(ctx context.Context, req *ecocashv1.GetTransactionRequest) (*ecocashv1.Transaction, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	txn, err := h.svc.GetTransaction(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "transaction not found")
	}

	return mapTransaction(txn), nil
}

func (h *EcoCashHandler) ListTransactionsByOrder(ctx context.Context, req *ecocashv1.ListTransactionsByOrderRequest) (*ecocashv1.ListTransactionsByOrderResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_id: %v", err)
	}

	txns, err := h.svc.ListTransactionsByOrder(ctx, orderID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list transactions failed")
	}

	proto := make([]*ecocashv1.Transaction, len(txns))
	for i := range txns {
		proto[i] = mapTransaction(&txns[i])
	}

	return &ecocashv1.ListTransactionsByOrderResponse{Transactions: proto}, nil
}

// ── Payout Management ─────────────────────────────────────────────────────────

func (h *EcoCashHandler) CreatePayout(ctx context.Context, req *ecocashv1.CreatePayoutRequest) (*ecocashv1.CreatePayoutResponse, error) {
	sellerID, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller_id: %v", err)
	}

	payout, err := h.svc.CreatePayout(ctx, sellerID, req.AmountCents, req.Currency)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	return &ecocashv1.CreatePayoutResponse{Payout: mapPayout(payout)}, nil
}

func (h *EcoCashHandler) GetPayout(ctx context.Context, req *ecocashv1.GetPayoutRequest) (*ecocashv1.Payout, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	payout, err := h.svc.GetPayout(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "payout not found")
	}

	return mapPayout(payout), nil
}

func (h *EcoCashHandler) UpdatePayoutStatus(ctx context.Context, req *ecocashv1.UpdatePayoutStatusRequest) (*ecocashv1.UpdatePayoutStatusResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid id: %v", err)
	}

	statusStr := mapPayoutStatusToString(req.Status)
	payout, err := h.svc.UpdatePayoutStatus(ctx, id, statusStr, req.ProviderRef)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update payout status failed: %v", err)
	}

	return &ecocashv1.UpdatePayoutStatusResponse{Payout: mapPayout(payout)}, nil
}

// ── Webhook Handler ───────────────────────────────────────────────────────────

func (h *EcoCashHandler) HandleWebhook(ctx context.Context, req *ecocashv1.HandleWebhookRequest) (*emptypb.Empty, error) {
	if err := h.svc.HandleWebhook(
		ctx,
		req.ClientCorrelator,
		req.StatusCode,
		req.StatusMessage,
		req.TransactionId,
		req.ReferenceCode,
	); err != nil {
		// Do not surface internal errors on the webhook endpoint — log and ack.
		return &emptypb.Empty{}, nil
	}
	return &emptypb.Empty{}, nil
}

// ── Mappers ───────────────────────────────────────────────────────────────────

func mapTransaction(t *db.PaymentTransaction) *ecocashv1.Transaction {
	if t == nil {
		return nil
	}
	return &ecocashv1.Transaction{
		Id:                   t.ID.String(),
		OrderId:              t.OrderID.String(),
		ClientCorrelator:     t.ClientCorrelator,
		ReferenceCode:        t.ReferenceCode,
		TranType:             mapTranType(t.TranType),
		MsisdnMasked:         t.EndUserIDMasked,
		AmountCents:          t.AmountCents,
		Currency:             t.Currency,
		Status:               mapTxnStatus(t.Status),
		EcocashStatusCode:    safeStr(t.EcocashStatusCode),
		EcocashStatusMsg:     safeStr(t.EcocashStatusMsg),
		EcocashTransactionId: safeStr(t.EcocashTransactionID),
		CreatedAt:            timestamppb.New(t.CreatedAt),
		UpdatedAt:            timestamppb.New(t.UpdatedAt),
	}
}

func mapRefundRecord(r *db.RefundTransaction) *ecocashv1.RefundRecord {
	if r == nil {
		return nil
	}
	return &ecocashv1.RefundRecord{
		Id:                 r.ID.String(),
		OriginalTxnId:      r.OriginalTxnID.String(),
		ClientCorrelator:   r.ClientCorrelator,
		AmountCents:        r.AmountCents,
		Status:             mapTxnStatus(r.Status),
		EcocashStatusCode:  safeStr(r.EcocashStatusCode),
		EcocashStatusMsg:   safeStr(r.EcocashStatusMsg),
		CreatedAt:          timestamppb.New(r.CreatedAt),
	}
}

func mapPayout(p *db.Payout) *ecocashv1.Payout {
	if p == nil {
		return nil
	}
	return &ecocashv1.Payout{
		Id:          p.ID.String(),
		SellerId:    p.SellerID.String(),
		AmountCents: p.AmountCents,
		Currency:    p.Currency,
		Status:      mapPayoutStatus(p.Status),
		Method:      p.Method,
		ProviderRef: safeStr(p.ProviderRef),
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}

func mapTxnStatus(s string) ecocashv1.TransactionStatus {
	switch s {
	case "PENDING":
		return ecocashv1.TransactionStatus_TRANSACTION_STATUS_PENDING
	case "SUCCESS":
		return ecocashv1.TransactionStatus_TRANSACTION_STATUS_SUCCESS
	case "FAILED":
		return ecocashv1.TransactionStatus_TRANSACTION_STATUS_FAILED
	case "REFUNDED":
		return ecocashv1.TransactionStatus_TRANSACTION_STATUS_REFUNDED
	case "REVERSED":
		return ecocashv1.TransactionStatus_TRANSACTION_STATUS_REVERSED
	case "MANUAL_REVIEW":
		return ecocashv1.TransactionStatus_TRANSACTION_STATUS_MANUAL_REVIEW
	default:
		return ecocashv1.TransactionStatus_TRANSACTION_STATUS_UNSPECIFIED
	}
}

func mapTranType(t string) ecocashv1.TransactionType {
	switch t {
	case "MER":
		return ecocashv1.TransactionType_TRANSACTION_TYPE_MER
	case "REF":
		return ecocashv1.TransactionType_TRANSACTION_TYPE_REF
	case "REV":
		return ecocashv1.TransactionType_TRANSACTION_TYPE_REV
	default:
		return ecocashv1.TransactionType_TRANSACTION_TYPE_UNSPECIFIED
	}
}

func mapPayoutStatus(s string) ecocashv1.PayoutStatus {
	switch s {
	case "QUEUED":
		return ecocashv1.PayoutStatus_PAYOUT_STATUS_QUEUED
	case "PROCESSING":
		return ecocashv1.PayoutStatus_PAYOUT_STATUS_PROCESSING
	case "PAID":
		return ecocashv1.PayoutStatus_PAYOUT_STATUS_PAID
	case "FAILED":
		return ecocashv1.PayoutStatus_PAYOUT_STATUS_FAILED
	case "MANUAL_REVIEW":
		return ecocashv1.PayoutStatus_PAYOUT_STATUS_MANUAL_REVIEW
	default:
		return ecocashv1.PayoutStatus_PAYOUT_STATUS_UNSPECIFIED
	}
}

func mapPayoutStatusToString(s ecocashv1.PayoutStatus) string {
	switch s {
	case ecocashv1.PayoutStatus_PAYOUT_STATUS_QUEUED:
		return "QUEUED"
	case ecocashv1.PayoutStatus_PAYOUT_STATUS_PROCESSING:
		return "PROCESSING"
	case ecocashv1.PayoutStatus_PAYOUT_STATUS_PAID:
		return "PAID"
	case ecocashv1.PayoutStatus_PAYOUT_STATUS_FAILED:
		return "FAILED"
	case ecocashv1.PayoutStatus_PAYOUT_STATUS_MANUAL_REVIEW:
		return "MANUAL_REVIEW"
	default:
		return "QUEUED"
	}
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
