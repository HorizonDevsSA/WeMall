package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/wemall/delivery-service/internal/db"
	"github.com/wemall/delivery-service/internal/partners"
	"github.com/wemall/delivery-service/internal/routing"
)

type DeliveryService struct {
	queries *db.Queries
	dbPool  *pgxpool.Pool
	rdb     *redis.Client
	nc      *nats.Conn
	log     zerolog.Logger
	geo     *routing.GeoIndex
	sf      *partners.SFExpressAdapter
	dhl     *partners.DHLAdapter
}

func NewDeliveryService(queries *db.Queries, dbPool *pgxpool.Pool, rdb *redis.Client, nc *nats.Conn, log zerolog.Logger) *DeliveryService {
	return &DeliveryService{
		queries: queries,
		dbPool:  dbPool,
		rdb:     rdb,
		nc:      nc,
		log:     log,
		geo:     routing.NewGeoIndex(rdb),
		sf:      partners.NewSFExpressAdapter(),
		dhl:     partners.NewDHLAdapter(),
	}
}

// FloatToNumeric converts a float64 value into pgtype.Numeric.
func FloatToNumeric(val float64) pgtype.Numeric {
	num := pgtype.Numeric{}
	_ = num.Scan(fmt.Sprintf("%.2f", val))
	num.Valid = true
	return num
}

// NumericToFloat converts a pgtype.Numeric value into float64.
func NumericToFloat(num pgtype.Numeric) float64 {
	if !num.Valid {
		return 0.0
	}
	var f float64
	if val, err := num.Value(); err == nil {
		if str, ok := val.(string); ok {
			_, _ = fmt.Sscanf(str, "%f", &f)
		}
	}
	return f
}

// GenerateTrackingNumber creates a unique tracking code.
func GenerateTrackingNumber() string {
	nBig, _ := rand.Int(rand.Reader, big.NewInt(900000))
	part1 := nBig.Int64() + 100000
	nBig2, _ := rand.Int(rand.Reader, big.NewInt(9000))
	part2 := nBig2.Int64() + 1000
	return fmt.Sprintf("WM-%d-%d", part1, part2)
}

type DeliveryOrderInput struct {
	OrderID              *uuid.UUID
	SenderType           string
	SenderID             uuid.UUID
	SenderName           string
	SenderPhone          string
	SenderAddressLine1   string
	SenderCity           string
	SenderCountry        string
	SenderLat            float64
	SenderLon            float64
	RecipientName        string
	RecipientPhone       string
	RecipientAddressLine1 string
	RecipientCity        string
	RecipientCountry     string
	RecipientLat         float64
	RecipientLon         float64
	DeliveryType         string
	OriginStationID      *uuid.UUID
	DestinationStationID *uuid.UUID
	WeightKg             float64
	LengthCm             int32
	WidthCm              int32
	HeightCm             int32
	PaymentStatus        string
}

// CreateShipment creates a shipping waybill, runs the routing selection, and posts NATS announcements.
func (s *DeliveryService) CreateShipment(ctx context.Context, input DeliveryOrderInput) (db.DeliveryOrder, error) {
	// Determine Carrier & Shipping Fee
	decision := routing.RoutePackage(
		input.SenderCity, input.RecipientCity,
		input.SenderLat, input.SenderLon,
		input.RecipientLat, input.RecipientLon,
		input.WeightKg, input.LengthCm, input.WidthCm, input.HeightCm,
	)

	trackingNo := GenerateTrackingNumber()
	dims := map[string]int32{
		"length": input.LengthCm,
		"width":  input.WidthCm,
		"height": input.HeightCm,
	}
	dimsBytes, _ := json.Marshal(dims)

	params := db.CreateDeliveryOrderParams{
		TrackingNumber:        trackingNo,
		OrderID:               input.OrderID,
		SenderType:            input.SenderType,
		SenderID:              input.SenderID,
		SenderName:            input.SenderName,
		SenderPhone:           input.SenderPhone,
		SenderAddressLine1:    input.SenderAddressLine1,
		SenderCity:            input.SenderCity,
		SenderCountry:         input.SenderCountry,
		StMakepoint:           input.SenderLon,
		StMakepoint_2:         input.SenderLat,
		RecipientName:         input.RecipientName,
		RecipientPhone:        input.RecipientPhone,
		RecipientAddressLine1: input.RecipientAddressLine1,
		RecipientCity:         input.RecipientCity,
		RecipientCountry:      input.RecipientCountry,
		StMakepoint_3:         input.RecipientLon,
		StMakepoint_4:         input.RecipientLat,
		DeliveryType:          input.DeliveryType,
		OriginStationID:       input.OriginStationID,
		DestinationStationID:  input.DestinationStationID,
		WeightKg:              FloatToNumeric(input.WeightKg),
		DimensionsCm:          dimsBytes,
		ShippingFee:           FloatToNumeric(decision.ShippingFee),
		PaymentStatus:         input.PaymentStatus,
		Status:                "created",
	}

	order, err := s.queries.CreateDeliveryOrder(ctx, params)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to create delivery order")
		return db.DeliveryOrder{}, err
	}

	// Create initial tracking log
	_, _ = s.queries.CreateTrackingLog(ctx, db.CreateTrackingLogParams{
		DeliveryOrderID: order.ID,
		Status:          "created",
		LocationDesc:    "Shipping order created",
		Column4:         input.SenderLon,
		Column5:         input.SenderLat,
	})

	s.log.Info().Str("tracking_no", trackingNo).Msg("delivery order created successfully")

	// Trigger Routing/Carrier Assignment Async
	go s.DispatchCarrierAssignment(context.Background(), order, decision)

	return order, nil
}

// DispatchCarrierAssignment assigns the 3PL carrier or broadcasts bidding tasks to crowdsourced couriers.
func (s *DeliveryService) DispatchCarrierAssignment(ctx context.Context, order db.DeliveryOrder, rDec *routing.RoutingDecision) {
	if rDec.CarrierType == "3pl" {
		var info *partners.ShipmentInfo
		var err error

		if rDec.CarrierName == "SF Express" {
			info, err = s.sf.RegisterShipment(order.SenderName, order.SenderAddressLine1, order.RecipientName, order.RecipientAddressLine1, NumericToFloat(order.WeightKg))
		} else {
			info, err = s.dhl.RegisterShipment(order.SenderName, order.SenderAddressLine1, order.RecipientName, order.RecipientAddressLine1, NumericToFloat(order.WeightKg))
		}

		if err != nil {
			s.log.Error().Err(err).Str("order_id", order.ID.String()).Msg("failed to register 3PL shipment")
			return
		}

		cType := db.CarrierType3pl
		assignedOrder, err := s.queries.AssignCarrierToDeliveryOrder(ctx, db.AssignCarrierToDeliveryOrderParams{
			ID:                 order.ID,
			CarrierType:        &cType,
			CarrierPartnerID:   nil, // Optional: resolve partner uuid from config/DB
			CarrierCourierID:   nil,
			ExternalTrackingNo: &info.ExternalTrackingNo,
		})
		if err != nil {
			s.log.Error().Err(err).Msg("failed to assign carrier to order")
			return
		}

		// Log tracking update
		_, _ = s.queries.CreateTrackingLog(ctx, db.CreateTrackingLogParams{
			DeliveryOrderID: assignedOrder.ID,
			Status:          "assigned",
			LocationDesc:    fmt.Sprintf("Dispatched to carrier %s. Tracking Code: %s", rDec.CarrierName, info.ExternalTrackingNo),
		})

		// Publish event
		s.publishEvent("wemall.delivery.assigned", map[string]string{
			"delivery_id":      assignedOrder.ID.String(),
			"tracking_no":      assignedOrder.TrackingNumber,
			"carrier":          rDec.CarrierName,
			"external_track":   info.ExternalTrackingNo,
			"recipient_phone":  assignedOrder.RecipientPhone,
		})

	} else {
		// Crowdsourced route
		// Find closest active couriers within 5km of origin
		lon, lat := s.decodeLocation(order.SenderLocation)
		nearbyRiders, err := s.geo.GetNearbyCouriers(ctx, lat, lon, 5.0)
		if err != nil || len(nearbyRiders) == 0 {
			s.log.Warn().Str("order_id", order.ID.String()).Msg("no active couriers found within 5km, cascading to 3PL fallback")
			// Cascade fallback: SF Express
			rDec.CarrierType = "3pl"
			rDec.CarrierName = "SF Express"
			s.DispatchCarrierAssignment(ctx, order, rDec)
			return
		}

		// Broadcast tasks to nearby couriers
		s.log.Info().Int("riders_count", len(nearbyRiders)).Msg("broadcasting delivery offer to active couriers")
		for _, riderIDStr := range nearbyRiders {
			riderUUID, err := uuid.Parse(riderIDStr)
			if err != nil {
				continue
			}

			_, _ = s.queries.CreateCourierTask(ctx, db.CreateCourierTaskParams{
				DeliveryOrderID: order.ID,
				CourierID:       riderUUID,
				Status:          "offered",
			})

			// Publish event for mobile notification push to courier
			s.publishEvent("wemall.courier.task_offered", map[string]string{
				"delivery_id": order.ID.String(),
				"courier_id":  riderUUID.String(),
				"tracking_no": order.TrackingNumber,
				"fee":         fmt.Sprintf("%.2f", NumericToFloat(order.ShippingFee)),
			})
		}
	}
}

// RegisterAsCourier registers a buyer profile as a crowd-sourced courier.
func (s *DeliveryService) RegisterAsCourier(ctx context.Context, userID uuid.UUID, vehicleType string, plateNumber *string) (db.Courier, error) {
	params := db.CreateCourierParams{
		UserID:             userID,
		VehicleType:        vehicleType,
		PlateNumber:        plateNumber,
		IsOnline:           false,
		VerificationStatus: "approved", // auto approve for dev verification
		Rating:             FloatToNumeric(5.00),
	}
	return s.queries.CreateCourier(ctx, params)
}

// SetCourierOnlineStatus changes courier availability and geo-index.
func (s *DeliveryService) SetCourierOnlineStatus(ctx context.Context, courierUserID uuid.UUID, isOnline bool) (db.Courier, error) {
	courier, err := s.queries.GetCourierByUserID(ctx, courierUserID)
	if err != nil {
		return db.Courier{}, err
	}

	updated, err := s.queries.UpdateCourierStatus(ctx, db.UpdateCourierStatusParams{
		ID:       courier.ID,
		IsOnline: isOnline,
	})
	if err != nil {
		return db.Courier{}, err
	}

	if !isOnline {
		_ = s.geo.RemoveCourier(ctx, courier.ID.String())
	}
	return updated, nil
}

// UpdateCourierLocation ingests courier location and updates Redis Geohash index.
func (s *DeliveryService) UpdateCourierLocation(ctx context.Context, courierUserID uuid.UUID, lat, lon float64) error {
	courier, err := s.queries.GetCourierByUserID(ctx, courierUserID)
	if err != nil {
		return err
	}

	_, err = s.queries.UpdateCourierLocation(ctx, db.UpdateCourierLocationParams{
		ID:            courier.ID,
		StMakepoint:   lon,
		StMakepoint_2: lat,
	})
	if err != nil {
		return err
	}

	if courier.IsOnline {
		return s.geo.UpdateCourierLocation(ctx, courier.ID.String(), lat, lon)
	}
	return nil
}

// AcceptCourierTask processes task locks for crowd-sourced couriers.
func (s *DeliveryService) AcceptCourierTask(ctx context.Context, courierUserID uuid.UUID, orderID uuid.UUID) (bool, error) {
	courier, err := s.queries.GetCourierByUserID(ctx, courierUserID)
	if err != nil {
		return false, err
	}

	// Fetch current task status
	task, err := s.queries.GetCourierTask(ctx, db.GetCourierTaskParams{
		DeliveryOrderID: orderID,
		CourierID:       courier.ID,
	})
	if err != nil {
		return false, errors.New("task not found for this courier")
	}

	if task.Status != "offered" {
		return false, errors.New("task is no longer available")
	}

	// Start database transaction to lock the delivery order
	tx, err := s.dbPool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// Fetch order with lock
	order, err := qtx.GetDeliveryOrder(ctx, orderID)
	if err != nil {
		return false, err
	}

	if order.Status != "created" && order.Status != "assigned" {
		return false, errors.New("order has already been assigned to another courier")
	}

	// Update order to assigned
	cType := db.CarrierTypeCrowdsourced
	_, err = qtx.AssignCarrierToDeliveryOrder(ctx, db.AssignCarrierToDeliveryOrderParams{
		ID:                 orderID,
		CarrierType:        &cType,
		CarrierCourierID:   &courier.ID,
		CarrierPartnerID:   nil,
		ExternalTrackingNo: nil,
	})
	if err != nil {
		return false, err
	}

	// Update task status
	_, err = qtx.UpdateCourierTaskStatus(ctx, db.UpdateCourierTaskStatusParams{
		DeliveryOrderID: orderID,
		CourierID:       courier.ID,
		Status:          "accepted",
	})
	if err != nil {
		return false, err
	}

	// Log tracking update
	_, err = qtx.CreateTrackingLog(ctx, db.CreateTrackingLogParams{
		DeliveryOrderID: orderID,
		Status:          "assigned",
		LocationDesc:    "Rider assigned to pickup package",
	})
	if err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	s.publishEvent("wemall.delivery.assigned", map[string]string{
		"delivery_id":     orderID.String(),
		"tracking_no":     order.TrackingNumber,
		"courier_id":      courier.ID.String(),
		"recipient_phone": order.RecipientPhone,
	})

	return true, nil
}

// UpdateDeliveryProgress handles tracking progression (e.g. PICKED_UP, DELIVERED).
func (s *DeliveryService) UpdateDeliveryProgress(ctx context.Context, courierUserID uuid.UUID, orderID uuid.UUID, status string, lat, lon float64, details *string) (bool, error) {
	courier, err := s.queries.GetCourierByUserID(ctx, courierUserID)
	if err != nil {
		return false, err
	}

	order, err := s.queries.GetDeliveryOrder(ctx, orderID)
	if err != nil {
		return false, err
	}

	if order.CarrierCourierID == nil || *order.CarrierCourierID != courier.ID {
		return false, errors.New("unauthorized: this order is not assigned to you")
	}

	// Update order status
	_, err = s.queries.UpdateDeliveryOrderStatus(ctx, db.UpdateDeliveryOrderStatusParams{
		ID:     orderID,
		Status: status,
	})
	if err != nil {
		return false, err
	}

	locDesc := s.getLocationDescription(status)
	_, _ = s.queries.CreateTrackingLog(ctx, db.CreateTrackingLogParams{
		DeliveryOrderID: orderID,
		Status:          status,
		LocationDesc:    locDesc,
		Column4:         lon,
		Column5:         lat,
		Details:         details,
		OperatorID:      &courier.ID,
	})

	s.publishEvent("wemall.delivery.status_changed", map[string]string{
		"delivery_id":     orderID.String(),
		"tracking_no":     order.TrackingNumber,
		"status":          status,
		"recipient_phone": order.RecipientPhone,
	})

	// Process earnings payout on delivery success
	if status == "delivered" {
		earning := NumericToFloat(order.ShippingFee) * 0.8 // Courier gets 80% cut
		_, _ = s.queries.UpdateCourierBalance(ctx, db.UpdateCourierBalanceParams{
			ID:             courier.ID,
			WalletBalance: FloatToNumeric(earning),
		})
		s.log.Info().Str("courier_id", courier.ID.String()).Float64("earning", earning).Msg("courier wallet credited")
	}

	return true, nil
}

// CheckInPackage registers parcel inside station hub.
func (s *DeliveryService) CheckInPackage(ctx context.Context, keeperUserID uuid.UUID, stationID uuid.UUID, trackingNo string, shelfCode string, direction string) (db.StationPackage, error) {
	station, err := s.queries.GetStation(ctx, stationID)
	if err != nil {
		return db.StationPackage{}, err
	}

	if station.KeeperUserID != keeperUserID {
		return db.StationPackage{}, errors.New("unauthorized: you do not manage this station")
	}

	order, err := s.queries.GetDeliveryOrderByTrackingNumber(ctx, trackingNo)
	if err != nil {
		return db.StationPackage{}, errors.New("package tracking number not found")
	}

	// Generate high entropy pickup OTP
	pickupCode := s.generateOTP()

	tx, err := s.dbPool.Begin(ctx)
	if err != nil {
		return db.StationPackage{}, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// Update order status to arrived_at_station
	_, err = qtx.UpdateDeliveryOrderStatus(ctx, db.UpdateDeliveryOrderStatusParams{
		ID:     order.ID,
		Status: "arrived_at_station",
	})
	if err != nil {
		return db.StationPackage{}, err
	}

	// Check-in package record
	pkg, err := qtx.CheckInStationPackage(ctx, db.CheckInStationPackageParams{
		StationID:        stationID,
		DeliveryOrderID:  order.ID,
		Direction:        direction,
		ShelfCode:        shelfCode,
		VerificationCode: pickupCode,
		CheckInBy:        keeperUserID,
	})
	if err != nil {
		return db.StationPackage{}, err
	}

	// Increment package counts
	_, _ = qtx.UpdateStationPackageCount(ctx, db.UpdateStationPackageCountParams{
		ID:            stationID,
		CurrentPackageCount: 1,
	})

	// Log tracking step
	_, err = qtx.CreateTrackingLog(ctx, db.CreateTrackingLogParams{
		DeliveryOrderID: order.ID,
		Status:          "arrived_at_station",
		LocationDesc:    fmt.Sprintf("Arrived at Station: %s. Shelf Location: %s", station.Name, shelfCode),
	})
	if err != nil {
		return db.StationPackage{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return db.StationPackage{}, err
	}

	s.publishEvent("wemall.delivery.station_in", map[string]string{
		"delivery_id":      order.ID.String(),
		"tracking_no":      order.TrackingNumber,
		"station_name":     station.Name,
		"shelf_code":       shelfCode,
		"pickup_otp":       pickupCode,
		"recipient_phone":  order.RecipientPhone,
		"recipient_name":   order.RecipientName,
	})

	return pkg, nil
}

// CheckOutPackage releases parcel from station hub upon secure code validation.
func (s *DeliveryService) CheckOutPackage(ctx context.Context, keeperUserID uuid.UUID, stationID uuid.UUID, trackingNo string, code string) (bool, error) {
	station, err := s.queries.GetStation(ctx, stationID)
	if err != nil {
		return false, err
	}

	if station.KeeperUserID != keeperUserID {
		return false, errors.New("unauthorized: you do not manage this station")
	}

	pkg, err := s.queries.GetStationPackageByTrackingNumber(ctx, trackingNo)
	if err != nil {
		return false, errors.New("package not checked in at this station")
	}

	if pkg.StationID != stationID {
		return false, errors.New("package is located in a different station")
	}

	if pkg.CheckOutAt != nil {
		return false, errors.New("package has already been checked out")
	}

	if pkg.VerificationCode != code {
		return false, errors.New("invalid verification code")
	}

	tx, err := s.dbPool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	qtx := s.queries.WithTx(tx)

	// Update order status to delivered
	_, err = qtx.UpdateDeliveryOrderStatus(ctx, db.UpdateDeliveryOrderStatusParams{
		ID:     pkg.DeliveryOrderID,
		Status: "delivered",
	})
	if err != nil {
		return false, err
	}

	// Checkout package
	_, err = qtx.CheckOutStationPackage(ctx, db.CheckOutStationPackageParams{
		DeliveryOrderID: pkg.DeliveryOrderID,
		CheckOutBy:      &keeperUserID,
	})
	if err != nil {
		return false, err
	}

	// Decrement package counts
	_, _ = qtx.UpdateStationPackageCount(ctx, db.UpdateStationPackageCountParams{
		ID:            stationID,
		CurrentPackageCount: -1,
	})

	// Log tracking step
	_, err = qtx.CreateTrackingLog(ctx, db.CreateTrackingLogParams{
		DeliveryOrderID: pkg.DeliveryOrderID,
		Status:          "delivered",
		LocationDesc:    fmt.Sprintf("Picked up by recipient from Station %s", station.Name),
	})
	if err != nil {
		return false, err
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}

	order, _ := s.queries.GetDeliveryOrder(ctx, pkg.DeliveryOrderID)
	s.publishEvent("wemall.delivery.delivered", map[string]string{
		"delivery_id":     pkg.DeliveryOrderID.String(),
		"tracking_no":     order.TrackingNumber,
		"recipient_phone": order.RecipientPhone,
	})

	return true, nil
}

// Helpers

func (s *DeliveryService) publishEvent(subject string, payload map[string]string) {
	if s.nc == nil {
		return
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = s.nc.Publish(subject, bytes)
}

func (s *DeliveryService) decodeLocation(geom interface{}) (float64, float64) {
	// Simple stub for geom point extraction. Returns default coordinates if type unknown.
	return 114.0578, 22.5430 // Shenzhen coordinates
}

func (s *DeliveryService) generateOTP() string {
	nBig, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", nBig.Int64()+100000)
}

func (s *DeliveryService) getLocationDescription(status string) string {
	switch status {
	case "picked_up":
		return "Package collected by courier"
	case "in_transit":
		return "Package is on its way"
	case "out_for_delivery":
		return "Courier is delivering package to destination"
	case "delivered":
		return "Package delivered successfully"
	case "failed":
		return "Delivery attempt failed"
	default:
		return "Package status updated"
	}
}
