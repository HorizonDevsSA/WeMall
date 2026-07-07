package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/wemall/ecocash-service/internal/service"
)

// WebhookPayload mirrors the EcoCash notifyUrl callback body.
// Since EcoCash webhooks are unsigned, we validate the correlator against our
// DB and re-confirm via LookupTransaction before trusting any status update.
type WebhookPayload struct {
	ClientCorrelator string `json:"clientCorrelator"`
	TransactionID    string `json:"transactionId"`
	StatusCode       string `json:"statusCode"`
	StatusMessage    string `json:"statusMessage"`
	ReferenceCode    string `json:"referenceCode"`
	EndUserID        string `json:"endUserId"`
}

// WebhookHandler handles inbound HTTP POST calls from EcoCash.
// It is mounted by the cmd/api server at a secret/unguessable path to reduce
// spoofing risk, per §11 of the integration plan.
type WebhookHandler struct {
	svc    *service.EcoCashService
	logger zerolog.Logger
}

func NewWebhookHandler(svc *service.EcoCashService, logger zerolog.Logger) *WebhookHandler {
	return &WebhookHandler{svc: svc, logger: logger}
}

// ServeHTTP satisfies the http.Handler interface.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to read webhook body")
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var payload WebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Warn().Err(err).Msg("failed to parse webhook payload")
		// Always return 200 to EcoCash so it does not retry endlessly.
		w.WriteHeader(http.StatusOK)
		return
	}

	if payload.ClientCorrelator == "" {
		h.logger.Warn().Msg("webhook received with empty client_correlator — ignoring")
		w.WriteHeader(http.StatusOK)
		return
	}

	h.logger.Info().
		Str("client_correlator", payload.ClientCorrelator).
		Str("status_code", payload.StatusCode).
		Str("status_message", payload.StatusMessage).
		Msg("ecocash webhook received")

	if err := h.svc.HandleWebhook(
		r.Context(),
		payload.ClientCorrelator,
		payload.StatusCode,
		payload.StatusMessage,
		payload.TransactionID,
		payload.ReferenceCode,
	); err != nil {
		h.logger.Error().Err(err).Msg("webhook processing error")
	}

	// Always ack with 200 — EcoCash interprets any non-200 as a delivery failure
	// and may retry, causing duplicate processing.
	w.WriteHeader(http.StatusOK)
}
