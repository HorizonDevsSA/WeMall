package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/wemall/ecocash-service/internal/db"
	"github.com/wemall/ecocash-service/internal/gateway"
)

// EcoCashService orchestrates all payment flows (charge, lookup, refund,
// reversal, payout) using ports-and-adapters conventions.
type EcoCashService struct {
	q       *db.Queries
	pool    *pgxpool.Pool
	gw      gateway.Gateway
	nc      *nats.Conn
	logger  zerolog.Logger
	notifyURL string // embedded in EcoCash charge requests
}

func NewEcoCashService(
	q *db.Queries,
	pool *pgxpool.Pool,
	gw gateway.Gateway,
	nc *nats.Conn,
	notifyURL string,
	logger zerolog.Logger,
) *EcoCashService {
	return &EcoCashService{
		q:         q,
		pool:      pool,
		gw:        gw,
		nc:        nc,
		notifyURL: notifyURL,
		logger:    logger,
	}
}

// ── ChargeCustomer — MER ──────────────────────────────────────────────────────

// ChargeCustomer initiates an EcoCash wallet charge for a given order.
// The DB row is written BEFORE the outbound call so a mid-call crash is
// recoverable by the reconciler. The correlator is <orderID>-<timestamp-ns>
// to ensure uniqueness while being deterministic on retry (callers reuse the
// same correlator by looking up an existing PENDING row first).
func (s *EcoCashService) ChargeCustomer(
	ctx context.Context,
	orderID uuid.UUID,
	msisdn string,
	amountCents int64,
	currency string,
) (*db.PaymentTransaction, error) {
	// 1. Validate inputs
	normalized, err := NormalizeMSISDN(msisdn)
	if err != nil {
		return nil, fmt.Errorf("invalid msisdn: %w", err)
	}
	if amountCents <= 0 {
		return nil, errors.New("amount must be positive")
	}
	if currency != "USD" && currency != "ZWG" {
		return nil, fmt.Errorf("unsupported currency: %s", currency)
	}

	masked := MaskMSISDN(normalized)

	// 2. Check for an existing PENDING transaction (idempotent re-entry).
	existing, err := s.q.GetExistingPendingCharge(ctx, db.GetExistingPendingChargeParams{
		OrderID:  orderID,
		TranType: "MER",
	})
	if err == nil && existing.Status == "PENDING" {
		// Return the existing row — caller should poll or wait for webhook.
		return &existing, nil
	}

	// 3. Generate a unique client_correlator stored before calling EcoCash.
	correlator := fmt.Sprintf("%s-%d", orderID.String(), time.Now().UnixNano())
	referenceCode := fmt.Sprintf("WM-%s", orderID.String()[:8])

	// 4. Persist PENDING row (outbound call happens after this).
	rawReq, _ := json.Marshal(map[string]interface{}{
		"order_id":          orderID.String(),
		"msisdn_masked":     masked,
		"amount_cents":      amountCents,
		"currency":          currency,
		"client_correlator": correlator,
	})

	txnRow, err := s.q.CreateTransaction(ctx, db.CreateTransactionParams{
		OrderID:          orderID,
		ClientCorrelator: correlator,
		TranType:         "MER",
		EndUserIDMasked:  masked,
		AmountCents:      amountCents,
		Currency:         currency,
		RawRequest:       rawReq,
	})
	if err != nil {
		return nil, fmt.Errorf("persist pending transaction: %w", err)
	}

	// 5. Build and send the EcoCash charge request.
	amountStr := centsToDecimalString(amountCents)
	chargeReq := gateway.ChargeRequest{
		ClientCorrelator: correlator,
		ReferenceCode:    referenceCode,
		TranType:         "MER",
		EndUserID:        normalized,
		Remark:           "WeMall Order " + referenceCode,
		Amount: gateway.PaymentAmount{
			ChargingInformation: gateway.ChargingInformation{
				Amount:      amountStr,
				Currency:    currency,
				Description: "WeMall Purchase " + referenceCode,
			},
			ChargingMetadata: gateway.ChargingMetadata{
				Amount:      amountStr,
				Currency:    currency,
				Description: "WeMall Purchase " + referenceCode,
			},
		},
	}

	gwResp, gwErr := s.gw.Charge(ctx, chargeReq)

	// 6. Determine final status from the gateway response.
	newStatus, statusCode, statusMsg := resolveStatus(gwResp.StatusMessage, gwResp.StatusCode, gwErr)

	// 7. Update the DB row in the same pass.
	rawResp, _ := json.Marshal(gwResp)
	updated, err := s.q.UpdateTransactionStatus(ctx, db.UpdateTransactionStatusParams{
		ID:                    txnRow.ID,
		Status:                newStatus,
		EcocashStatusCode:     &statusCode,
		EcocashStatusMsg:      &statusMsg,
		EcocashTransactionID:  &gwResp.TransactionID,
		ReferenceCode:         gwResp.ReferenceCode,
		RawResponse:           rawResp,
	})
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to update transaction status after charge")
		return &txnRow, nil // return the original row; reconciler will fix
	}

	// 8. Write outbox event in a separate DB write (best-effort; outbox relay
	//    will pick it up even if the service restarts immediately after).
	eventType := "payment.failed"
	if newStatus == "SUCCESS" {
		eventType = "payment.succeeded"
	} else if newStatus == "PENDING" {
		eventType = "payment.pending"
	}
	s.publishOutboxEvent(ctx, txnRow.ID, eventType, map[string]interface{}{
		"order_id":          orderID.String(),
		"transaction_id":    txnRow.ID.String(),
		"client_correlator": correlator,
		"amount_cents":      amountCents,
		"currency":          currency,
		"status":            newStatus,
	})

	return &updated, nil
}

// ── LookupTransaction ─────────────────────────────────────────────────────────

// LookupTransaction resolves a transaction's current EcoCash status.
// Called by the reconciler and as a safety check before retrying a charge
// after a timeout.
func (s *EcoCashService) LookupTransaction(
	ctx context.Context,
	orderID uuid.UUID,
	correlator string,
) (*db.PaymentTransaction, error) {
	txnRow, err := s.q.GetTransactionByCorrelator(ctx, correlator)
	if err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}

	// Re-derive the MSISDN from the masked form is not possible, so we pass
	// the end_user_id_masked as a lookup hint — EcoCash also accepts the
	// correlator alone on some endpoints; this is a best-effort call.
	gwResp, gwErr := s.gw.LookupTransaction(ctx, txnRow.EndUserIDMasked, correlator)
	if gwErr != nil {
		return &txnRow, fmt.Errorf("lookup failed: %w", gwErr)
	}

	newStatus, statusCode, statusMsg := resolveStatus(gwResp.StatusMessage, gwResp.StatusCode, nil)
	rawResp, _ := json.Marshal(gwResp)
	updated, err := s.q.UpdateTransactionStatus(ctx, db.UpdateTransactionStatusParams{
		ID:                   txnRow.ID,
		Status:               newStatus,
		EcocashStatusCode:    &statusCode,
		EcocashStatusMsg:     &statusMsg,
		EcocashTransactionID: &gwResp.TransactionID,
		ReferenceCode:        gwResp.TransactionID,
		RawResponse:          rawResp,
	})
	if err != nil {
		return &txnRow, nil
	}

	if newStatus == "SUCCESS" {
		s.publishOutboxEvent(ctx, txnRow.ID, "payment.succeeded", map[string]interface{}{
			"order_id":       orderID.String(),
			"transaction_id": txnRow.ID.String(),
			"status":         newStatus,
		})
	}

	return &updated, nil
}

// ── RefundTransaction — REF ───────────────────────────────────────────────────

// RefundTransaction refunds a completed charge (full or partial).
// Validates locally before calling EcoCash to avoid unnecessary API calls for
// clearly invalid requests (mirrors E009/E012 locally).
func (s *EcoCashService) RefundTransaction(
	ctx context.Context,
	originalTxnID uuid.UUID,
	amountCents int64,
	reason string,
) (*db.RefundTransaction, error) {
	// 1. Load original transaction
	orig, err := s.q.GetTransactionByID(ctx, originalTxnID)
	if err != nil {
		return nil, fmt.Errorf("original transaction not found: %w", err)
	}
	if orig.Status != "SUCCESS" {
		return nil, fmt.Errorf("cannot refund a transaction in status %s", orig.Status)
	}

	// 2. Calculate already-refunded amount to prevent over-refunding (E012).
	alreadyRefunded, err := s.q.SumRefundedAmountForTxn(ctx, originalTxnID)
	if err != nil {
		alreadyRefunded = 0
	}

	if amountCents == 0 {
		amountCents = orig.AmountCents - alreadyRefunded
	}
	if amountCents <= 0 {
		return nil, errors.New("no refundable balance remaining")
	}
	if amountCents > orig.AmountCents-alreadyRefunded {
		return nil, fmt.Errorf("refund amount exceeds remaining refundable balance (%d cents)", orig.AmountCents-alreadyRefunded)
	}

	// 3. Generate correlator and persist PENDING refund row.
	correlator := fmt.Sprintf("ref-%s-%d", originalTxnID.String(), time.Now().UnixNano())
	rawReq, _ := json.Marshal(map[string]interface{}{
		"original_txn_id":  originalTxnID.String(),
		"amount_cents":     amountCents,
		"reason":           reason,
		"client_correlator": correlator,
	})

	refundRow, err := s.q.CreateRefundTransaction(ctx, db.CreateRefundTransactionParams{
		OriginalTxnID:    originalTxnID,
		ClientCorrelator: correlator,
		TranType:         "REF",
		AmountCents:      amountCents,
		RawRequest:       rawReq,
	})
	if err != nil {
		return nil, fmt.Errorf("persist pending refund: %w", err)
	}

	// 4. Call EcoCash refund endpoint.
	amountStr := centsToDecimalString(amountCents)
	origAmountStr := centsToDecimalString(orig.AmountCents)
	refundReq := gateway.RefundRequest{
		ClientCorrelator:  correlator,
		ReferenceCode:     fmt.Sprintf("REF-%s", originalTxnID.String()[:8]),
		TranType:          "REF",
		OriginalTxnID:     safeStr(orig.EcocashTransactionID),
		OriginalReference: orig.ReferenceCode,
		EndUserID:         orig.EndUserIDMasked,
		Amount: gateway.PaymentAmount{
			ChargingInformation: gateway.ChargingInformation{
				Amount:      amountStr,
				Currency:    orig.Currency,
				Description: reason,
			},
			ChargingMetadata: gateway.ChargingMetadata{
				Amount:      origAmountStr,
				Currency:    orig.Currency,
				Description: reason,
			},
		},
	}

	gwResp, gwErr := s.gw.Refund(ctx, refundReq)

	// 5. Map gateway result back to local status.
	newStatus := "FAILED"
	statusCode := ""
	statusMsg := ""
	if gwErr == nil {
		newStatus = "SUCCESS"
		statusCode = gwResp.StatusCode
		statusMsg = gwResp.StatusMessage
	} else {
		statusMsg = gwErr.Error()
		// Move to manual review for transient failures.
		if gateway.IsRetryable(gwErr) {
			newStatus = "FAILED" // reconciler can retry
		}
	}

	rawResp, _ := json.Marshal(gwResp)
	updated, err := s.q.UpdateRefundStatus(ctx, db.UpdateRefundStatusParams{
		ID:                 refundRow.ID,
		Status:             newStatus,
		EcocashStatusCode:  &statusCode,
		EcocashStatusMsg:   &statusMsg,
		RawResponse:        rawResp,
	})
	if err != nil {
		return &refundRow, nil
	}

	// 6. On success, mark original transaction as REFUNDED (or PARTIALLY if
	//    more balance remains).
	if newStatus == "SUCCESS" {
		remaining := orig.AmountCents - alreadyRefunded - amountCents
		parentStatus := "REFUNDED"
		if remaining > 0 {
			parentStatus = "REFUNDED" // extend model to PARTIALLY_REFUNDED as needed
		}
		emptyStr := ""
		_ , _ = s.q.UpdateTransactionStatus(ctx, db.UpdateTransactionStatusParams{
			ID:                   orig.ID,
			Status:               parentStatus,
			EcocashStatusCode:    &emptyStr,
			EcocashStatusMsg:     &emptyStr,
			EcocashTransactionID: orig.EcocashTransactionID,
			ReferenceCode:        orig.ReferenceCode,
			RawResponse:          nil,
		})

		s.publishOutboxEvent(ctx, orig.ID, "refund.completed", map[string]interface{}{
			"order_id":       orig.OrderID.String(),
			"transaction_id": orig.ID.String(),
			"refund_id":      refundRow.ID.String(),
			"amount_cents":   amountCents,
			"currency":       orig.Currency,
		})
	}

	return &updated, nil
}

// ── ReverseTransaction — REV ──────────────────────────────────────────────────

// ReverseTransaction reverses a pending/unsettled charge before settlement.
func (s *EcoCashService) ReverseTransaction(
	ctx context.Context,
	originalTxnID uuid.UUID,
	reason string,
) (*db.RefundTransaction, error) {
	orig, err := s.q.GetTransactionByID(ctx, originalTxnID)
	if err != nil {
		return nil, fmt.Errorf("original transaction not found: %w", err)
	}
	if orig.Status != "PENDING" {
		return nil, fmt.Errorf("can only reverse a PENDING transaction, current status: %s", orig.Status)
	}

	correlator := fmt.Sprintf("rev-%s-%d", originalTxnID.String(), time.Now().UnixNano())
	rawReq, _ := json.Marshal(map[string]interface{}{
		"original_txn_id":   originalTxnID.String(),
		"reason":            reason,
		"client_correlator": correlator,
	})

	reversalRow, err := s.q.CreateRefundTransaction(ctx, db.CreateRefundTransactionParams{
		OriginalTxnID:    originalTxnID,
		ClientCorrelator: correlator,
		TranType:         "REV",
		AmountCents:      orig.AmountCents,
		RawRequest:       rawReq,
	})
	if err != nil {
		return nil, fmt.Errorf("persist pending reversal: %w", err)
	}

	amountStr := centsToDecimalString(orig.AmountCents)
	reverseReq := gateway.RefundRequest{
		ClientCorrelator:  correlator,
		ReferenceCode:     fmt.Sprintf("REV-%s", originalTxnID.String()[:8]),
		TranType:          "REV",
		OriginalTxnID:     safeStr(orig.EcocashTransactionID),
		OriginalReference: orig.ReferenceCode,
		EndUserID:         orig.EndUserIDMasked,
		Amount: gateway.PaymentAmount{
			ChargingInformation: gateway.ChargingInformation{
				Amount:      amountStr,
				Currency:    orig.Currency,
				Description: reason,
			},
			ChargingMetadata: gateway.ChargingMetadata{
				Amount:      amountStr,
				Currency:    orig.Currency,
				Description: reason,
			},
		},
	}

	gwResp, gwErr := s.gw.Refund(ctx, reverseReq)

	newStatus := "FAILED"
	statusCode := ""
	statusMsg := ""
	if gwErr == nil {
		newStatus = "SUCCESS"
		statusCode = gwResp.StatusCode
		statusMsg = gwResp.StatusMessage
	} else {
		statusMsg = gwErr.Error()
	}

	rawResp, _ := json.Marshal(gwResp)
	updated, err := s.q.UpdateRefundStatus(ctx, db.UpdateRefundStatusParams{
		ID:                reversalRow.ID,
		Status:            newStatus,
		EcocashStatusCode: &statusCode,
		EcocashStatusMsg:  &statusMsg,
		RawResponse:       rawResp,
	})
	if err != nil {
		return &reversalRow, nil
	}

	if newStatus == "SUCCESS" {
		emptyStr := ""
		_, _ = s.q.UpdateTransactionStatus(ctx, db.UpdateTransactionStatusParams{
			ID:                   orig.ID,
			Status:               "REVERSED",
			EcocashStatusCode:    &emptyStr,
			EcocashStatusMsg:     &emptyStr,
			EcocashTransactionID: orig.EcocashTransactionID,
			ReferenceCode:        orig.ReferenceCode,
			RawResponse:          nil,
		})
	}

	return &updated, nil
}

// ── HandleWebhook ─────────────────────────────────────────────────────────────

// HandleWebhook processes an EcoCash notifyUrl callback.
// Since EcoCash webhooks are unsigned, we treat the payload as a trigger to
// re-verify via LookupTransaction rather than trusting the callback directly.
// Only processes correlators that are already known and in PENDING state.
func (s *EcoCashService) HandleWebhook(
	ctx context.Context,
	correlator string,
	statusCode string,
	statusMsg string,
	transactionID string,
	referenceCode string,
) error {
	txn, err := s.q.GetTransactionByCorrelator(ctx, correlator)
	if err != nil {
		s.logger.Warn().Str("correlator", correlator).Msg("webhook received for unknown correlator — ignoring")
		return nil // not an error; just unknown
	}

	// Idempotency: ignore if already terminal
	if txn.Status == "SUCCESS" || txn.Status == "FAILED" ||
		txn.Status == "REFUNDED" || txn.Status == "REVERSED" {
		s.logger.Debug().Str("correlator", correlator).Str("status", txn.Status).
			Msg("webhook received for already-terminal transaction — ignoring")
		return nil
	}

	// Re-confirm via LookupTransaction (webhook is not authoritative on its own).
	_, lookupErr := s.LookupTransaction(ctx, txn.OrderID, correlator)
	if lookupErr != nil {
		s.logger.Error().Err(lookupErr).Str("correlator", correlator).
			Msg("lookup after webhook failed — reconciler will retry")
	}
	return nil
}

// ── Payout Management ─────────────────────────────────────────────────────────

// CreatePayout queues a manual seller payout for finance team settlement.
func (s *EcoCashService) CreatePayout(
	ctx context.Context,
	sellerID uuid.UUID,
	amountCents int64,
	currency string,
) (*db.Payout, error) {
	if amountCents <= 0 {
		return nil, errors.New("payout amount must be positive")
	}

	payout, err := s.q.CreatePayout(ctx, db.CreatePayoutParams{
		SellerID:    sellerID,
		AmountCents: amountCents,
		Currency:    currency,
	})
	if err != nil {
		return nil, fmt.Errorf("create payout: %w", err)
	}
	return &payout, nil
}

// UpdatePayoutStatus allows the finance reconciliation process to mark a payout
// as PAID/FAILED after the manual settlement has been executed.
func (s *EcoCashService) UpdatePayoutStatus(
	ctx context.Context,
	payoutID uuid.UUID,
	status string,
	providerRef string,
) (*db.Payout, error) {
	updated, err := s.q.UpdatePayoutStatus(ctx, db.UpdatePayoutStatusParams{
		ID:          payoutID,
		Status:      status,
		ProviderRef: &providerRef,
	})
	if err != nil {
		return nil, fmt.Errorf("update payout status: %w", err)
	}

	if status == "PAID" {
		s.publishOutboxEvent(ctx, payoutID, "payout.paid", map[string]interface{}{
			"payout_id":    payoutID.String(),
			"seller_id":    updated.SellerID.String(),
			"amount_cents": updated.AmountCents,
			"currency":     updated.Currency,
			"provider_ref": providerRef,
		})
	}

	return &updated, nil
}

// GetTransaction fetches a single payment transaction by ID.
func (s *EcoCashService) GetTransaction(ctx context.Context, id uuid.UUID) (*db.PaymentTransaction, error) {
	txn, err := s.q.GetTransactionByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("transaction not found: %w", err)
	}
	return &txn, nil
}

// ListTransactionsByOrder returns all transactions for a given order.
func (s *EcoCashService) ListTransactionsByOrder(ctx context.Context, orderID uuid.UUID) ([]db.PaymentTransaction, error) {
	txns, err := s.q.ListTransactionsByOrder(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("list transactions: %w", err)
	}
	return txns, nil
}

// GetPayout fetches a payout by ID.
func (s *EcoCashService) GetPayout(ctx context.Context, id uuid.UUID) (*db.Payout, error) {
	p, err := s.q.GetPayoutByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("payout not found: %w", err)
	}
	return &p, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// publishOutboxEvent writes an outbox row for the NATS relay worker.
// Failures are logged but do not surface as errors — the relay worker will
// retry any unpublished events.
func (s *EcoCashService) publishOutboxEvent(
	ctx context.Context,
	aggregateID uuid.UUID,
	eventType string,
	payload map[string]interface{},
) {
	payloadBytes, _ := json.Marshal(payload)
	if _, err := s.q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
		AggregateID: aggregateID,
		EventType:   eventType,
		Payload:     payloadBytes,
	}); err != nil {
		s.logger.Error().Err(err).Str("event_type", eventType).Msg("failed to write outbox event")
	}
}

// resolveStatus maps a raw EcoCash statusMessage/statusCode pair to our
// internal status string. Returns (status, statusCode, statusMsg).
func resolveStatus(statusMessage, statusCode string, gwErr error) (string, string, string) {
	if gwErr != nil {
		return "FAILED", statusCode, gwErr.Error()
	}

	switch {
	case isSuccess(statusMessage):
		return "SUCCESS", statusCode, statusMessage
	case isPending(statusMessage):
		return "PENDING", statusCode, statusMessage
	default:
		return "FAILED", statusCode, statusMessage
	}
}

func isSuccess(msg string) bool {
	return msg == "Transaction Successful" || msg == "SUCCESS"
}

func isPending(msg string) bool {
	return msg == "Pending" || msg == "PENDING" || msg == ""
}

// centsToDecimalString converts an integer amount in cents to a 2-decimal
// string, e.g. 1050 → "10.50".
func centsToDecimalString(cents int64) string {
	whole := cents / 100
	frac := cents % 100
	return fmt.Sprintf("%d.%02d", whole, frac)
}

func safeStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
