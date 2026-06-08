package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	paymentv1 "github.com/wemall/gen/payment/v1"
	"github.com/wemall/payment-service/internal/db"
	"github.com/wemall/payment-service/internal/service"
)

type PaymentHandler struct {
	paymentv1.UnimplementedPaymentServiceServer
	svc *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

func (h *PaymentHandler) CreatePayment(ctx context.Context, req *paymentv1.CreatePaymentRequest) (*paymentv1.CreatePaymentResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order id: %v", err)
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id: %v", err)
	}

	payment, clientSecret, err := h.svc.CreatePayment(ctx, orderID, userID, req.Amount, req.Currency, req.Provider)
	if err != nil {
		return nil, err
	}

	return &paymentv1.CreatePaymentResponse{
		Payment:      mapPayment(payment),
		ClientSecret: clientSecret,
	}, nil
}

func (h *PaymentHandler) ProcessPayment(ctx context.Context, req *paymentv1.ProcessPaymentRequest) (*paymentv1.ProcessPaymentResponse, error) {
	paymentID, err := uuid.Parse(req.PaymentId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payment id: %v", err)
	}

	payment, err := h.svc.ProcessPayment(ctx, paymentID, req.Token)
	if err != nil {
		return nil, err
	}

	return &paymentv1.ProcessPaymentResponse{
		Payment: mapPayment(payment),
	}, nil
}

func (h *PaymentHandler) GetPayment(ctx context.Context, req *paymentv1.GetPaymentRequest) (*paymentv1.Payment, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payment id: %v", err)
	}

	payment, err := h.svc.GetPayment(ctx, id)
	if err != nil {
		return nil, err
	}

	return mapPayment(payment), nil
}

func (h *PaymentHandler) RefundPayment(ctx context.Context, req *paymentv1.RefundPaymentRequest) (*paymentv1.RefundPaymentResponse, error) {
	paymentID, err := uuid.Parse(req.PaymentId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payment id: %v", err)
	}

	payment, err := h.svc.RefundPayment(ctx, paymentID, req.Amount)
	if err != nil {
		return nil, err
	}

	return &paymentv1.RefundPaymentResponse{
		Payment: mapPayment(payment),
	}, nil
}

// Helper mapping functions
func mapPayment(p *db.Payment) *paymentv1.Payment {
	if p == nil {
		return nil
	}

	txnID := ""
	if p.TransactionID != nil {
		txnID = *p.TransactionID
	}

	return &paymentv1.Payment{
		Id:            p.ID.String(),
		OrderId:       p.OrderID.String(),
		UserId:        p.UserID.String(),
		Amount:        p.Amount,
		Currency:      p.Currency,
		Provider:      mapProviderToProto(p.Provider),
		Status:        mapStatusToProto(p.Status),
		TransactionId: txnID,
		CreatedAt:     timestamppb.New(p.CreatedAt),
		UpdatedAt:     timestamppb.New(p.UpdatedAt),
	}
}

func mapProviderToProto(p string) paymentv1.PaymentProvider {
	switch p {
	case "google_pay":
		return paymentv1.PaymentProvider_PAYMENT_PROVIDER_GOOGLE_PAY
	case "stripe":
		return paymentv1.PaymentProvider_PAYMENT_PROVIDER_STRIPE
	default:
		return paymentv1.PaymentProvider_PAYMENT_PROVIDER_UNSPECIFIED
	}
}

func mapStatusToProto(s string) paymentv1.PaymentStatus {
	switch s {
	case "pending":
		return paymentv1.PaymentStatus_PAYMENT_STATUS_PENDING
	case "completed":
		return paymentv1.PaymentStatus_PAYMENT_STATUS_COMPLETED
	case "failed":
		return paymentv1.PaymentStatus_PAYMENT_STATUS_FAILED
	case "refunded":
		return paymentv1.PaymentStatus_PAYMENT_STATUS_REFUNDED
	default:
		return paymentv1.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}
