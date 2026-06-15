package handler

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog"

	"github.com/wemall/delivery-service/internal/service"
)

type NatsHandler struct {
	nc  *nats.Conn
	svc *service.DeliveryService
	log zerolog.Logger
}

type OrderPaidEvent struct {
	OrderID              string  `json:"order_id"`
	StoreID              string  `json:"store_id"`
	BuyerID              string  `json:"buyer_id"`
	SenderName           string  `json:"sender_name"`
	SenderPhone          string  `json:"sender_phone"`
	SenderAddress        string  `json:"sender_address"`
	SenderLat            float64 `json:"sender_lat"`
	SenderLon            float64 `json:"sender_lon"`
	RecipientName        string  `json:"recipient_name"`
	RecipientPhone       string  `json:"recipient_phone"`
	RecipientAddress     string  `json:"recipient_address"`
	RecipientLat         float64 `json:"recipient_lat"`
	RecipientLon         float64 `json:"recipient_lon"`
	WeightKg             float64 `json:"weight_kg"`
	LengthCm             int32   `json:"length_cm"`
	WidthCm              int32   `json:"width_cm"`
	HeightCm             int32   `json:"height_cm"`
	DeliveryType         string  `json:"delivery_type"`
	DestinationStationID string  `json:"destination_station_id,omitempty"`
}

func NewNatsHandler(nc *nats.Conn, svc *service.DeliveryService, log zerolog.Logger) *NatsHandler {
	return &NatsHandler{
		nc:  nc,
		svc: svc,
		log: log,
	}
}

func (h *NatsHandler) Start(ctx context.Context) error {
	if h.nc == nil {
		h.log.Warn().Msg("NATS client is nil, skipping subscriber setup")
		return nil
	}

	_, err := h.nc.Subscribe("wemall.order.paid", func(msg *nats.Msg) {
		h.log.Info().Str("subject", msg.Subject).Msg("received NATS message")

		var ev OrderPaidEvent
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			h.log.Error().Err(err).Msg("failed to unmarshal order paid event")
			return
		}

		orderUUID, err := uuid.Parse(ev.OrderID)
		if err != nil {
			h.log.Error().Err(err).Msg("invalid order ID in paid event")
			return
		}
		storeUUID, err := uuid.Parse(ev.StoreID)
		if err != nil {
			h.log.Error().Err(err).Msg("invalid store ID in paid event")
			return
		}

		var destStationID *uuid.UUID
		if ev.DestinationStationID != "" {
			parsed, err := uuid.Parse(ev.DestinationStationID)
			if err == nil {
				destStationID = &parsed
			}
		}

		input := service.DeliveryOrderInput{
			OrderID:               &orderUUID,
			SenderType:            "seller",
			SenderID:              storeUUID,
			SenderName:            ev.SenderName,
			SenderPhone:           ev.SenderPhone,
			SenderAddressLine1:    ev.SenderAddress,
			SenderCity:            "Shenzhen", // mock resolved city
			SenderCountry:         "China",
			SenderLat:             ev.SenderLat,
			SenderLon:             ev.SenderLon,
			RecipientName:         ev.RecipientName,
			RecipientPhone:        ev.RecipientPhone,
			RecipientAddressLine1: ev.RecipientAddress,
			RecipientCity:         "Shenzhen",
			RecipientCountry:      "China",
			RecipientLat:          ev.RecipientLat,
			RecipientLon:          ev.RecipientLon,
			DeliveryType:          ev.DeliveryType,
			OriginStationID:       nil,
			DestinationStationID:  destStationID,
			WeightKg:              ev.WeightKg,
			LengthCm:              ev.LengthCm,
			WidthCm:               ev.WidthCm,
			HeightCm:              ev.HeightCm,
			PaymentStatus:         "paid",
		}

		_, err = h.svc.CreateShipment(context.Background(), input)
		if err != nil {
			h.log.Error().Err(err).Msg("failed to process order paid shipment creation")
			return
		}

		_ = msg.Ack()
	})

	if err != nil {
		return err
	}

	h.log.Info().Msg("subscribed to wemall.order.paid successfully")
	return nil
}
