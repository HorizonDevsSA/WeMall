package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/wemall/delivery-service/internal/db"
)

type WebhookHandler struct {
	queries *db.Queries
	log     zerolog.Logger
}

type CarrierStatusWebhook struct {
	ExternalTrackingNo string   `json:"external_tracking_no"`
	Status             string   `json:"status"` // 'in_transit' | 'delivered' | 'failed'
	LocationDesc       string   `json:"location_desc"`
	Latitude           *float64 `json:"latitude"`
	Longitude          *float64 `json:"longitude"`
	Details            string   `json:"details"`
}

func NewWebhookHandler(queries *db.Queries, log zerolog.Logger) *WebhookHandler {
	return &WebhookHandler{
		queries: queries,
		log:     log,
	}
}

func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var payload CarrierStatusWebhook
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.log.Error().Err(err).Msg("failed to decode webhook request")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid json"}`))
		return
	}

	h.log.Info().Str("external_tracking_no", payload.ExternalTrackingNo).Str("status", payload.Status).Msg("received 3PL webhook update")

	ctx := r.Context()
	order, err := h.queries.GetDeliveryOrderByTrackingNumber(ctx, payload.ExternalTrackingNo)
	if err != nil {
		// Try resolving by external tracking number field
		// We'll search the DB directly or log an error.
		// For mock robustness, we assume external tracking matches standard tracking or log.
		h.log.Warn().Str("tracking_no", payload.ExternalTrackingNo).Msg("could not find order matching tracking number")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"tracking number not found"}`))
		return
	}

	// Update delivery status
	_, err = h.queries.UpdateDeliveryOrderStatus(ctx, db.UpdateDeliveryOrderStatusParams{
		ID:     order.ID,
		Status: payload.Status,
	})
	if err != nil {
		h.log.Error().Err(err).Msg("failed to update delivery order status")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Log tracking step
	var latVal, lonVal float64
	if payload.Latitude != nil && payload.Longitude != nil {
		latVal = *payload.Latitude
		lonVal = *payload.Longitude
	}

	var detailsPtr *string
	if payload.Details != "" {
		detailsPtr = &payload.Details
	}

	var systemOperatorID *uuid.UUID = nil

	_, _ = h.queries.CreateTrackingLog(ctx, db.CreateTrackingLogParams{
		DeliveryOrderID: order.ID,
		Status:          payload.Status,
		LocationDesc:    payload.LocationDesc,
		Column4:         lonVal,
		Column5:         latVal,
		Details:         detailsPtr,
		OperatorID:      systemOperatorID,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"success"}`))
}
