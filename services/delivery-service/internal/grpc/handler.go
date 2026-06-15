package grpc

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	deliveryv1 "github.com/wemall/gen/delivery/v1"
	"github.com/wemall/delivery-service/internal/db"
	"github.com/wemall/delivery-service/internal/routing"
	"github.com/wemall/delivery-service/internal/service"
	"github.com/wemall/delivery-service/internal/waybill"
)

type DeliveryHandler struct {
	deliveryv1.UnimplementedDeliveryServiceServer
	svc     *service.DeliveryService
	queries *db.Queries
}

func NewDeliveryHandler(svc *service.DeliveryService, queries *db.Queries) *DeliveryHandler {
	return &DeliveryHandler{
		svc:     svc,
		queries: queries,
	}
}

func (h *DeliveryHandler) CreateEcommerceShipment(ctx context.Context, req *deliveryv1.CreateEcommerceShipmentRequest) (*deliveryv1.CreateEcommerceShipmentResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, fmt.Errorf("invalid order_id: %w", err)
	}
	storeID, err := uuid.Parse(req.StoreId)
	if err != nil {
		return nil, fmt.Errorf("invalid store_id: %w", err)
	}

	var destStationID *uuid.UUID
	if req.DestinationStationId != "" {
		parsed, err := uuid.Parse(req.DestinationStationId)
		if err == nil {
			destStationID = &parsed
		}
	}

	input := service.DeliveryOrderInput{
		OrderID:               &orderID,
		SenderType:            "seller",
		SenderID:              storeID,
		SenderName:            req.SenderName,
		SenderPhone:           req.SenderPhone,
		SenderAddressLine1:    req.SenderAddress,
		SenderCity:            "Shenzhen", // Resolved or defaulting for mock
		SenderCountry:         "China",
		SenderLat:             req.SenderLocation.Latitude,
		SenderLon:             req.SenderLocation.Longitude,
		RecipientName:         req.RecipientName,
		RecipientPhone:        req.RecipientPhone,
		RecipientAddressLine1: req.RecipientAddress,
		RecipientCity:         "Shenzhen",
		RecipientCountry:      "China",
		RecipientLat:          req.RecipientLocation.Latitude,
		RecipientLon:          req.RecipientLocation.Longitude,
		DeliveryType:          req.DeliveryType,
		OriginStationID:       nil,
		DestinationStationID:  destStationID,
		WeightKg:              req.WeightKg,
		LengthCm:              req.LengthCm,
		WidthCm:               req.WidthCm,
		HeightCm:              req.HeightCm,
		PaymentStatus:         "paid", // E-commerce shipments are already paid at checkout
	}

	order, err := h.svc.CreateShipment(ctx, input)
	if err != nil {
		return nil, err
	}

	fee := service.NumericToFloat(order.ShippingFee)
	return &deliveryv1.CreateEcommerceShipmentResponse{
		DeliveryId:     order.ID.String(),
		TrackingNumber: order.TrackingNumber,
		ShippingFee:    fee,
	}, nil
}

func (h *DeliveryHandler) GetDeliveryByOrderID(ctx context.Context, req *deliveryv1.GetDeliveryByOrderIDRequest) (*deliveryv1.GetDeliveryByOrderIDResponse, error) {
	orderID, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, fmt.Errorf("invalid order_id: %w", err)
	}

	order, err := h.queries.GetDeliveryOrderByOrderID(ctx, &orderID)
	if err != nil {
		return nil, err
	}

	carrierName := "Unassigned"
	if order.CarrierType != nil {
		if *order.CarrierType == "crowdsourced" {
			carrierName = "WeMall Instant Rider"
		} else if order.CarrierPartnerID != nil {
			carrierName = "3PL Partner" // Simplified
		}
	}

	extTrackNo := ""
	if order.ExternalTrackingNo != nil {
		extTrackNo = *order.ExternalTrackingNo
	}

	return &deliveryv1.GetDeliveryByOrderIDResponse{
		DeliveryId:         order.ID.String(),
		TrackingNumber:     order.TrackingNumber,
		Status:             order.Status,
		CarrierName:        carrierName,
		ExternalTrackingNo: extTrackNo,
	}, nil
}

func (h *DeliveryHandler) EstimateShippingRates(ctx context.Context, req *deliveryv1.EstimateShippingRatesRequest) (*deliveryv1.EstimateShippingRatesResponse, error) {
	decision := routing.RoutePackage(
		"Shenzhen", "Shenzhen", // Mocking sender/recipient city as same-city for estimations
		req.Origin.Latitude, req.Origin.Longitude,
		req.Destination.Latitude, req.Destination.Longitude,
		req.WeightKg, 20, 15, 10, // Default mock parcel dimensions (20x15x10cm)
	)

	estimate := &deliveryv1.ShippingRateEstimate{
		CarrierType:            decision.CarrierType,
		Name:                   decision.CarrierName,
		Cost:                   decision.ShippingFee,
		EstimatedDeliveryHours: 24,
	}

	if decision.CarrierType == "crowdsourced" {
		estimate.EstimatedDeliveryHours = 2
	}

	return &deliveryv1.EstimateShippingRatesResponse{
		Estimates: []*deliveryv1.ShippingRateEstimate{estimate},
	}, nil
}

// TrackPackage returns a delivery order and its tracking logs.
func (h *DeliveryHandler) TrackPackage(ctx context.Context, req *deliveryv1.TrackPackageRequest) (*deliveryv1.TrackPackageResponse, error) {
	order, err := h.queries.GetDeliveryOrderByTrackingNumber(ctx, req.TrackingNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery order: %w", err)
	}

	logs, err := h.queries.GetTrackingLogsByDeliveryOrderID(ctx, order.ID)
	if err != nil {
		logs = []db.TrackingLog{}
	}

	pbOrder := h.mapDeliveryOrder(ctx, order, logs)
	return &deliveryv1.TrackPackageResponse{
		DeliveryOrder: pbOrder,
	}, nil
}

// NearbyStations returns post stations near coordinates.
func (h *DeliveryHandler) NearbyStations(ctx context.Context, req *deliveryv1.NearbyStationsRequest) (*deliveryv1.NearbyStationsResponse, error) {
	stations, err := h.queries.GetStationsNearLocation(ctx, db.GetStationsNearLocationParams{
		StMakepoint:   req.Longitude,
		StMakepoint_2: req.Latitude,
		Limit:         20,
	})
	if err != nil {
		return nil, err
	}

	pbStations := make([]*deliveryv1.Station, len(stations))
	for i, row := range stations {
		st := db.Station{
			ID:                  row.ID,
			KeeperUserID:        row.KeeperUserID,
			Name:                row.Name,
			StoreType:           row.StoreType,
			Phone:               row.Phone,
			AddressLine1:        row.AddressLine1,
			City:                row.City,
			Country:             row.Country,
			Location:            row.Location,
			Status:              row.Status,
			CapacityPackages:    row.CapacityPackages,
			CurrentPackageCount: row.CurrentPackageCount,
			OperatingHours:      row.OperatingHours,
			CreatedAt:           row.CreatedAt,
			UpdatedAt:           row.UpdatedAt,
		}
		pbStations[i] = h.mapStation(st)
	}
	return &deliveryv1.NearbyStationsResponse{
		Stations: pbStations,
	}, nil
}

// AvailableCourierTasks lists offered tasks for a gig courier.
func (h *DeliveryHandler) AvailableCourierTasks(ctx context.Context, req *deliveryv1.AvailableCourierTasksRequest) (*deliveryv1.AvailableCourierTasksResponse, error) {
	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	courier, err := h.queries.GetCourierByUserID(ctx, userUUID)
	if err != nil {
		return &deliveryv1.AvailableCourierTasksResponse{Tasks: []*deliveryv1.DeliveryOrder{}}, nil
	}

	orders, err := h.queries.GetAvailableCourierTasks(ctx, courier.ID)
	if err != nil {
		return nil, err
	}

	pbOrders := make([]*deliveryv1.DeliveryOrder, len(orders))
	for i, o := range orders {
		logs, _ := h.queries.GetTrackingLogsByDeliveryOrderID(ctx, o.ID)
		pbOrders[i] = h.mapDeliveryOrder(ctx, o, logs)
	}
	return &deliveryv1.AvailableCourierTasksResponse{
		Tasks: pbOrders,
	}, nil
}

// StationInventory returns station inventory.
func (h *DeliveryHandler) StationInventory(ctx context.Context, req *deliveryv1.StationInventoryRequest) (*deliveryv1.StationInventoryResponse, error) {
	stationID, err := uuid.Parse(req.StationId)
	if err != nil {
		return nil, fmt.Errorf("invalid station_id: %w", err)
	}

	pkgs, err := h.queries.GetStationInventory(ctx, db.GetStationInventoryParams{
		StationID: stationID,
		Column2:   req.UnclaimedOnly,
	})
	if err != nil {
		return nil, err
	}

	pbPackages := make([]*deliveryv1.StationPackage, len(pkgs))
	for i, p := range pkgs {
		sp := db.StationPackage{
			ID:               p.ID,
			StationID:        p.StationID,
			DeliveryOrderID:  p.DeliveryOrderID,
			Direction:        p.Direction,
			ShelfCode:        p.ShelfCode,
			VerificationCode: p.VerificationCode,
			CheckInAt:        p.CheckInAt,
			CheckInBy:        p.CheckInBy,
			CheckOutAt:       p.CheckOutAt,
			CheckOutBy:       p.CheckOutBy,
			CreatedAt:        p.CreatedAt,
		}
		pbPackages[i] = h.mapStationPackage(ctx, sp)
	}
	return &deliveryv1.StationInventoryResponse{
		Packages: pbPackages,
	}, nil
}

// CreatePersonalDelivery initiates a personal courier C2C task.
func (h *DeliveryHandler) CreatePersonalDelivery(ctx context.Context, req *deliveryv1.CreatePersonalDeliveryRequest) (*deliveryv1.CreatePersonalDeliveryResponse, error) {
	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	var destStationID *uuid.UUID
	if req.DestinationStationId != "" {
		parsed, err := uuid.Parse(req.DestinationStationId)
		if err == nil {
			destStationID = &parsed
		}
	}

	input := service.DeliveryOrderInput{
		OrderID:               nil,
		SenderType:            "buyer",
		SenderID:              userUUID,
		SenderName:            req.SenderName,
		SenderPhone:           req.SenderPhone,
		SenderAddressLine1:    req.SenderAddress,
		SenderCity:            req.SenderCity,
		SenderCountry:         req.SenderCountry,
		SenderLat:             req.SenderLocation.Latitude,
		SenderLon:             req.SenderLocation.Longitude,
		RecipientName:         req.RecipientName,
		RecipientPhone:        req.RecipientPhone,
		RecipientAddressLine1: req.RecipientAddress,
		RecipientCity:         req.RecipientCity,
		RecipientCountry:      req.RecipientCountry,
		RecipientLat:          req.RecipientLocation.Latitude,
		RecipientLon:          req.RecipientLocation.Longitude,
		DeliveryType:          req.DeliveryType,
		OriginStationID:       nil,
		DestinationStationID:  destStationID,
		WeightKg:              req.WeightKg,
		LengthCm:              req.LengthCm,
		WidthCm:               req.WidthCm,
		HeightCm:              req.HeightCm,
		PaymentStatus:         "pending",
	}

	order, err := h.svc.CreateShipment(ctx, input)
	if err != nil {
		return nil, err
	}

	logs, _ := h.queries.GetTrackingLogsByDeliveryOrderID(ctx, order.ID)
	pbOrder := h.mapDeliveryOrder(ctx, order, logs)

	paymentSecret := fmt.Sprintf("pi_%s_secret_%s", order.ID.String()[:8], uuid.NewString()[:8])
	paymentURL := fmt.Sprintf("https://checkout.wemall.com/pay/delivery/%s", order.ID.String())

	return &deliveryv1.CreatePersonalDeliveryResponse{
		DeliveryOrder: pbOrder,
		PaymentSecret: paymentSecret,
		PaymentUrl:    paymentURL,
	}, nil
}

// GenerateEWaybillLabel generates waybill pdf label encoded in base64.
func (h *DeliveryHandler) GenerateEWaybillLabel(ctx context.Context, req *deliveryv1.GenerateEWaybillLabelRequest) (*deliveryv1.GenerateEWaybillLabelResponse, error) {
	orderID, err := uuid.Parse(req.DeliveryOrderId)
	if err != nil {
		return nil, fmt.Errorf("invalid delivery_order_id: %w", err)
	}

	order, err := h.queries.GetDeliveryOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	carrierName := "Unassigned"
	if order.CarrierType != nil {
		if *order.CarrierType == "crowdsourced" {
			carrierName = "WeMall Instant Rider"
		} else {
			carrierName = "3PL Partner"
		}
	}

	extTrackNo := ""
	if order.ExternalTrackingNo != nil {
		extTrackNo = *order.ExternalTrackingNo
	}

	weight := service.NumericToFloat(order.WeightKg)

	labelBase64 := waybill.GenerateEWaybill(
		order.TrackingNumber,
		order.SenderName,
		order.SenderAddressLine1,
		order.RecipientName,
		order.RecipientAddressLine1,
		carrierName,
		extTrackNo,
		weight,
	)

	return &deliveryv1.GenerateEWaybillLabelResponse{
		LabelBase64: labelBase64,
	}, nil
}

// RegisterAsCourier registers user.
func (h *DeliveryHandler) RegisterAsCourier(ctx context.Context, req *deliveryv1.RegisterAsCourierRequest) (*deliveryv1.RegisterAsCourierResponse, error) {
	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	var plateNo *string
	if req.PlateNumber != "" {
		p := req.PlateNumber
		plateNo = &p
	}

	courier, err := h.svc.RegisterAsCourier(ctx, userUUID, req.VehicleType, plateNo)
	if err != nil {
		return nil, err
	}

	return &deliveryv1.RegisterAsCourierResponse{
		Courier: h.mapCourier(courier),
	}, nil
}

// SetCourierOnlineStatus goes online/offline.
func (h *DeliveryHandler) SetCourierOnlineStatus(ctx context.Context, req *deliveryv1.SetCourierOnlineStatusRequest) (*deliveryv1.SetCourierOnlineStatusResponse, error) {
	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	courier, err := h.svc.SetCourierOnlineStatus(ctx, userUUID, req.IsOnline)
	if err != nil {
		return nil, err
	}

	return &deliveryv1.SetCourierOnlineStatusResponse{
		Courier: h.mapCourier(courier),
	}, nil
}

// AcceptCourierTask accepts the offered courier task.
func (h *DeliveryHandler) AcceptCourierTask(ctx context.Context, req *deliveryv1.AcceptCourierTaskRequest) (*deliveryv1.AcceptCourierTaskResponse, error) {
	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	orderUUID, err := uuid.Parse(req.DeliveryOrderId)
	if err != nil {
		return nil, fmt.Errorf("invalid delivery_order_id: %w", err)
	}

	success, err := h.svc.AcceptCourierTask(ctx, userUUID, orderUUID)
	if err != nil {
		return nil, err
	}

	return &deliveryv1.AcceptCourierTaskResponse{
		Success: success,
	}, nil
}

// UpdateDeliveryProgress updates courier status/progress.
func (h *DeliveryHandler) UpdateDeliveryProgress(ctx context.Context, req *deliveryv1.UpdateDeliveryProgressRequest) (*deliveryv1.UpdateDeliveryProgressResponse, error) {
	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	orderUUID, err := uuid.Parse(req.DeliveryOrderId)
	if err != nil {
		return nil, fmt.Errorf("invalid delivery_order_id: %w", err)
	}

	var details *string
	if req.Details != "" {
		d := req.Details
		details = &d
	}

	// First ingest courier location update in DB and redis
	_ = h.svc.UpdateCourierLocation(ctx, userUUID, req.Latitude, req.Longitude)

	success, err := h.svc.UpdateDeliveryProgress(ctx, userUUID, orderUUID, req.Status, req.Latitude, req.Longitude, details)
	if err != nil {
		return nil, err
	}

	return &deliveryv1.UpdateDeliveryProgressResponse{
		Success: success,
	}, nil
}

// StationCheckInPackage keeper registers package inside station.
func (h *DeliveryHandler) StationCheckInPackage(ctx context.Context, req *deliveryv1.StationCheckInPackageRequest) (*deliveryv1.StationCheckInPackageResponse, error) {
	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	stationUUID, err := uuid.Parse(req.StationId)
	if err != nil {
		return nil, fmt.Errorf("invalid station_id: %w", err)
	}

	pkg, err := h.svc.CheckInPackage(ctx, userUUID, stationUUID, req.TrackingNumber, req.ShelfCode, req.Direction)
	if err != nil {
		return nil, err
	}

	return &deliveryv1.StationCheckInPackageResponse{
		Package: h.mapStationPackage(ctx, pkg),
	}, nil
}

// StationCheckOutPackage keeper releases package with OTP.
func (h *DeliveryHandler) StationCheckOutPackage(ctx context.Context, req *deliveryv1.StationCheckOutPackageRequest) (*deliveryv1.StationCheckOutPackageResponse, error) {
	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, fmt.Errorf("invalid user_id: %w", err)
	}

	stationUUID, err := uuid.Parse(req.StationId)
	if err != nil {
		return nil, fmt.Errorf("invalid station_id: %w", err)
	}

	success, err := h.svc.CheckOutPackage(ctx, userUUID, stationUUID, req.TrackingNumber, req.VerificationCode)
	if err != nil {
		return nil, err
	}

	return &deliveryv1.StationCheckOutPackageResponse{
		Success: success,
	}, nil
}

// Helper mapping methods

func (h *DeliveryHandler) mapStation(s db.Station) *deliveryv1.Station {
	lat, lon := parseGeometryPoint(s.Location)
	return &deliveryv1.Station{
		Id:             s.ID.String(),
		Name:           s.Name,
		StoreType:      s.StoreType,
		Phone:          s.Phone,
		AddressLine1:   s.AddressLine1,
		City:           s.City,
		Country:        s.Country,
		Location: &deliveryv1.Location{
			Latitude:  lat,
			Longitude: lon,
		},
		OperatingHours: string(s.OperatingHours),
	}
}

func (h *DeliveryHandler) mapCourier(c db.Courier) *deliveryv1.Courier {
	rating := service.NumericToFloat(c.Rating)
	return &deliveryv1.Courier{
		Id:          c.ID.String(),
		VehicleType: c.VehicleType,
		PlateNumber: derefStrPtr(c.PlateNumber),
		IsOnline:    c.IsOnline,
		Rating:      rating,
	}
}

func (h *DeliveryHandler) mapStationPackage(ctx context.Context, p db.StationPackage) *deliveryv1.StationPackage {
	var station *deliveryv1.Station
	if st, err := h.queries.GetStation(ctx, p.StationID); err == nil {
		station = h.mapStation(st)
	}

	var order *deliveryv1.DeliveryOrder
	if ord, err := h.queries.GetDeliveryOrder(ctx, p.DeliveryOrderID); err == nil {
		logs, _ := h.queries.GetTrackingLogsByDeliveryOrderID(ctx, ord.ID)
		order = h.mapDeliveryOrder(ctx, ord, logs)
	}

	checkOutStr := ""
	if p.CheckOutAt != nil {
		checkOutStr = p.CheckOutAt.Format(time.RFC3339)
	}

	return &deliveryv1.StationPackage{
		Id:            p.ID.String(),
		Station:       station,
		DeliveryOrder: order,
		ShelfCode:     p.ShelfCode,
		CheckInAt:     p.CheckInAt.Format(time.RFC3339),
		CheckOutAt:    checkOutStr,
	}
}

func (h *DeliveryHandler) mapDeliveryOrder(ctx context.Context, o db.DeliveryOrder, logs []db.TrackingLog) *deliveryv1.DeliveryOrder {
	var originStation *deliveryv1.Station
	if o.OriginStationID != nil {
		if st, err := h.queries.GetStation(ctx, *o.OriginStationID); err == nil {
			originStation = h.mapStation(st)
		}
	}

	var destStation *deliveryv1.Station
	if o.DestinationStationID != nil {
		if st, err := h.queries.GetStation(ctx, *o.DestinationStationID); err == nil {
			destStation = h.mapStation(st)
		}
	}

	pbLogs := make([]*deliveryv1.TrackingLog, len(logs))
	for i, l := range logs {
		pbLogs[i] = &deliveryv1.TrackingLog{
			Id:           l.ID.String(),
			Status:       l.Status,
			LocationDesc: l.LocationDesc,
			Details:      derefStrPtr(l.Details),
			CreatedAt:    l.CreatedAt.Format(time.RFC3339),
		}
	}

	fee := service.NumericToFloat(o.ShippingFee)
	weight := service.NumericToFloat(o.WeightKg)

	return &deliveryv1.DeliveryOrder{
		Id:                 o.ID.String(),
		TrackingNumber:     o.TrackingNumber,
		OrderId:            derefUUIDPtr(o.OrderID),
		SenderName:         o.SenderName,
		SenderPhone:        o.SenderPhone,
		RecipientName:      o.RecipientName,
		RecipientPhone:     o.RecipientPhone,
		DeliveryType:       o.DeliveryType,
		OriginStation:      originStation,
		DestinationStation: destStation,
		WeightKg:           weight,
		ShippingFee:        fee,
		Status:             o.Status,
		TrackingLogs:       pbLogs,
		CreatedAt:          o.CreatedAt.Format(time.RFC3339),
	}
}

func derefStrPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefUUIDPtr(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}

func parseGeometryPoint(geom interface{}) (lat, lon float64) {
	defaultLon, defaultLat := 114.0578, 22.5430
	if geom == nil {
		return defaultLat, defaultLon
	}
	var geomStr string
	switch v := geom.(type) {
	case string:
		geomStr = v
	case []byte:
		geomStr = string(v)
	default:
		return defaultLat, defaultLon
	}
	if len(geomStr) < 50 {
		return defaultLat, defaultLon
	}
	b, err := hex.DecodeString(geomStr)
	if err != nil || len(b) < 25 {
		return defaultLat, defaultLon
	}
	isLittleEndian := b[0] == 1
	var byteOrder binary.ByteOrder
	if isLittleEndian {
		byteOrder = binary.LittleEndian
	} else {
		byteOrder = binary.BigEndian
	}
	if len(b) >= 25 {
		xOffset := len(b) - 16
		yOffset := len(b) - 8
		
		xBits := byteOrder.Uint64(b[xOffset : xOffset+8])
		yBits := byteOrder.Uint64(b[yOffset : yOffset+8])
		
		x := math.Float64frombits(xBits)
		y := math.Float64frombits(yBits)
		
		return y, x
	}
	return defaultLat, defaultLon
}
