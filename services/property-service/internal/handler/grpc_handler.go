package handler

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	propertyv1 "github.com/wemall/gen/property/v1"
	"github.com/wemall/property-service/internal/db"
	"github.com/wemall/property-service/internal/service"
)

type PropertyHandler struct {
	propertyv1.UnimplementedPropertyServiceServer
	svc *service.PropertyService
}

func NewPropertyHandler(svc *service.PropertyService) *PropertyHandler {
	return &PropertyHandler{svc: svc}
}

func (h *PropertyHandler) CreateProperty(ctx context.Context, req *propertyv1.CreatePropertyRequest) (*propertyv1.CreatePropertyResponse, error) {
	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid owner_id")
	}

	amenities := make([]db.CreatePropertyAmenityParams, len(req.Amenities))
	for i, am := range req.Amenities {
		amenities[i] = db.CreatePropertyAmenityParams{
			Name:     am.Name,
			Category: am.Category,
		}
	}

	prop, err := h.svc.CreateProperty(ctx, service.CreatePropertyInput{
		OwnerID:       ownerID,
		Type:          mapProtoPropertyType(req.Type),
		Title:         req.Title,
		Description:   req.Description,
		AddressLine1:  req.AddressLine1,
		AddressLine2:  optionalStr(req.AddressLine2),
		City:          req.City,
		StateProvince: req.StateProvince,
		Country:       req.Country,
		PostalCode:    optionalStr(req.PostalCode),
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		BedroomCount:  req.BedroomCount,
		BathroomCount: req.BathroomCount,
		MaxGuests:     req.MaxGuests,
		SquareMeters:  req.SquareMeters,
		ImageURLs:     req.ImageUrls,
		Amenities:     amenities,
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	return &propertyv1.CreatePropertyResponse{
		Property: mapProperty(prop),
	}, nil
}

func (h *PropertyHandler) GetProperty(ctx context.Context, req *propertyv1.GetPropertyRequest) (*propertyv1.GetPropertyResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid property id")
	}

	prop, err := h.svc.GetProperty(ctx, id)
	if err != nil {
		return nil, grpcErr(err)
	}

	return &propertyv1.GetPropertyResponse{
		Property: mapProperty(prop),
	}, nil
}

func (h *PropertyHandler) CreateListing(ctx context.Context, req *propertyv1.CreateListingRequest) (*propertyv1.CreateListingResponse, error) {
	propID, err := uuid.Parse(req.PropertyId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid property_id")
	}

	var cleaningFee, securityDeposit, escrowPercent, agentCommRate float64
	var minNights, maxNights, yearBuilt int32
	var checkInTime, checkOutTime string
	var includesFurniture bool
	var propertyTax float64

	if req.RentalMeta != nil {
		cleaningFee = req.RentalMeta.CleaningFee
		securityDeposit = req.RentalMeta.SecurityDeposit
		minNights = req.RentalMeta.MinNights
		maxNights = req.RentalMeta.MaxNights
		checkInTime = req.RentalMeta.CheckInTime
		checkOutTime = req.RentalMeta.CheckOutTime
	}

	if req.SalesMeta != nil {
		escrowPercent = req.SalesMeta.EscrowDepositPercent
		agentCommRate = req.SalesMeta.AgentCommissionRate
		includesFurniture = req.SalesMeta.IncludesFurniture
		yearBuilt = req.SalesMeta.YearBuilt
		propertyTax = req.SalesMeta.PropertyTaxAnnual
	}

	listing, err := h.svc.CreateListing(ctx, service.CreateListingInput{
		PropertyID:      propID,
		Type:            mapProtoListingType(req.Type),
		BasePrice:       req.BasePrice,
		Currency:        req.Currency,
		IsInstantBook:   req.IsInstantBook,
		CleaningFee:     cleaningFee,
		SecurityDeposit: securityDeposit,
		MinNights:       minNights,
		MaxNights:       maxNights,
		CheckInTime:     checkInTime,
		CheckOutTime:    checkOutTime,
		EscrowPercent:   escrowPercent,
		AgentCommRate:   agentCommRate,
		Furniture:       includesFurniture,
		YearBuilt:       optionalInt32(yearBuilt),
		PropertyTax:     optionalFloat64(propertyTax),
	})
	if err != nil {
		return nil, grpcErr(err)
	}

	return &propertyv1.CreateListingResponse{
		Listing: mapListing(listing),
	}, nil
}

func (h *PropertyHandler) GetListing(ctx context.Context, req *propertyv1.GetListingRequest) (*propertyv1.GetListingResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid listing id")
	}

	listing, err := h.svc.GetListing(ctx, id)
	if err != nil {
		return nil, grpcErr(err)
	}

	return &propertyv1.GetListingResponse{
		Listing: mapListing(listing),
	}, nil
}

func (h *PropertyHandler) SearchNearbyListings(ctx context.Context, req *propertyv1.SearchNearbyListingsRequest) (*propertyv1.SearchNearbyListingsResponse, error) {
	listings, err := h.svc.SearchNearbyListings(ctx, req.Latitude, req.Longitude, req.RadiusMeters, req.Limit, req.Offset)
	if err != nil {
		return nil, grpcErr(err)
	}

	pbListings := make([]*propertyv1.Listing, len(listings))
	for i, l := range listings {
		pbListings[i] = mapListing(l)
	}

	return &propertyv1.SearchNearbyListingsResponse{
		Listings: pbListings,
	}, nil
}

func (h *PropertyHandler) GetListingsInViewport(ctx context.Context, req *propertyv1.GetListingsInViewportRequest) (*propertyv1.GetListingsInViewportResponse, error) {
	listings, err := h.svc.GetListingsInViewport(ctx, req.MinLatitude, req.MinLongitude, req.MaxLatitude, req.MaxLongitude)
	if err != nil {
		return nil, grpcErr(err)
	}

	pbListings := make([]*propertyv1.Listing, len(listings))
	for i, l := range listings {
		pbListings[i] = mapListing(l)
	}

	return &propertyv1.GetListingsInViewportResponse{
		Listings: pbListings,
	}, nil
}

func (h *PropertyHandler) CreateBooking(ctx context.Context, req *propertyv1.CreateBookingRequest) (*propertyv1.CreateBookingResponse, error) {
	listingID, err := uuid.Parse(req.ListingId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid listing_id")
	}
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tenant_id")
	}

	booking, err := h.svc.CreateBooking(ctx, listingID, tenantID, req.StartDate, req.EndDate)
	if err != nil {
		return nil, grpcErr(err)
	}

	return &propertyv1.CreateBookingResponse{
		Booking: mapBooking(booking, nil),
	}, nil
}

func (h *PropertyHandler) CreateOffer(ctx context.Context, req *propertyv1.CreateOfferRequest) (*propertyv1.CreateOfferResponse, error) {
	listingID, err := uuid.Parse(req.ListingId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid listing_id")
	}
	buyerID, err := uuid.Parse(req.BuyerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid buyer_id")
	}

	offer, err := h.svc.CreateOffer(ctx, listingID, buyerID, req.OfferPrice, req.ConditionsText, req.ExpirationDate.AsTime())
	if err != nil {
		return nil, grpcErr(err)
	}

	return &propertyv1.CreateOfferResponse{
		Offer: mapOffer(offer, nil),
	}, nil
}

func (h *PropertyHandler) AcceptOffer(ctx context.Context, req *propertyv1.AcceptOfferRequest) (*propertyv1.AcceptOfferResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid offer id")
	}
	ownerID, err := uuid.Parse(req.OwnerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid owner_id")
	}

	offer, err := h.svc.AcceptOffer(ctx, id, ownerID)
	if err != nil {
		return nil, grpcErr(err)
	}

	return &propertyv1.AcceptOfferResponse{
		Offer: mapOffer(offer, nil),
	}, nil
}

func (h *PropertyHandler) ScheduleViewing(ctx context.Context, req *propertyv1.ScheduleViewingRequest) (*propertyv1.ScheduleViewingResponse, error) {
	listingID, err := uuid.Parse(req.ListingId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid listing_id")
	}
	clientID, err := uuid.Parse(req.ClientId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid client_id")
	}
	hostID, err := uuid.Parse(req.HostId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid host_id")
	}

	app, err := h.svc.ScheduleViewing(ctx, listingID, clientID, hostID, req.ScheduledTime.AsTime(), req.Notes)
	if err != nil {
		return nil, grpcErr(err)
	}

	return &propertyv1.ScheduleViewingResponse{
		Appointment: mapViewingAppointment(app, nil),
	}, nil
}

// ── Mapping Helpers ──────────────────────────────────────────────────────────

func mapProperty(p *service.PropertyWithDetails) *propertyv1.Property {
	if p == nil {
		return nil
	}

	imgs := make([]*propertyv1.PropertyImage, len(p.Images))
	for i, img := range p.Images {
		imgs[i] = &propertyv1.PropertyImage{
			Id:           img.ID.String(),
			PropertyId:   img.PropertyID.String(),
			Url:          img.Url,
			DisplayOrder: img.DisplayOrder,
			IsCover:      img.IsCover,
		}
	}

	amens := make([]*propertyv1.PropertyAmenity, len(p.Amenities))
	for i, am := range p.Amenities {
		amens[i] = &propertyv1.PropertyAmenity{
			Id:         am.ID.String(),
			PropertyId: am.PropertyID.String(),
			Name:       am.Name,
			Category:   am.Category,
		}
	}

	return &propertyv1.Property{
		Id:            p.Property.ID.String(),
		OwnerId:       p.Property.OwnerID.String(),
		Type:          mapPropertyType(p.Property.Type),
		Title:         p.Property.Title,
		Description:   p.Property.Description,
		AddressLine1:  p.Property.AddressLine1,
		AddressLine2:  deref(p.Property.AddressLine2),
		City:          p.Property.City,
		StateProvince: p.Property.StateProvince,
		Country:       p.Property.Country,
		PostalCode:    deref(p.Property.PostalCode),
		Latitude:      p.Property.Latitude,
		Longitude:     p.Property.Longitude,
		BedroomCount:  p.Property.BedroomCount,
		BathroomCount: numericToFloat64(p.Property.BathroomCount),
		MaxGuests:     p.Property.MaxGuests,
		SquareMeters:  numericToFloat64(p.Property.SquareMeters),
		IsVerified:     p.Property.IsVerified,
		CreatedAt:     timestamppb.New(p.Property.CreatedAt),
		UpdatedAt:     timestamppb.New(p.Property.UpdatedAt),
		Images:        imgs,
		Amenities:     amens,
	}
}

func mapListing(l *service.ListingWithDetails) *propertyv1.Listing {
	if l == nil {
		return nil
	}

	var pbRental *propertyv1.RentalListingMeta
	if l.RentalMeta != nil {
		pbRental = &propertyv1.RentalListingMeta{
			CleaningFee:     numericToFloat64(l.RentalMeta.CleaningFee),
			SecurityDeposit: numericToFloat64(l.RentalMeta.SecurityDeposit),
			MinNights:       l.RentalMeta.MinNights,
			MaxNights:       l.RentalMeta.MaxNights,
			CheckInTime:     formatPgTime(l.RentalMeta.CheckInTime),
			CheckOutTime:    formatPgTime(l.RentalMeta.CheckOutTime),
		}
	}

	var pbSales *propertyv1.SalesListingMeta
	if l.SalesMeta != nil {
		pbSales = &propertyv1.SalesListingMeta{
			EscrowDepositPercent: numericToFloat64(l.SalesMeta.EscrowDepositPercent),
			AgentCommissionRate:  numericToFloat64(l.SalesMeta.AgentCommissionRate),
			IncludesFurniture:     l.SalesMeta.IncludesFurniture,
			YearBuilt:             derefInt32(l.SalesMeta.YearBuilt),
			PropertyTaxAnnual:    numericToFloat64(l.SalesMeta.PropertyTaxAnnual),
		}
	}

	propDetails := &service.PropertyWithDetails{
		Property: db.GetPropertyRow{
			ID:            l.Listing.PropertyID,
			OwnerID:       l.Listing.OwnerID,
			Type:          l.Listing.PropertyType,
			Title:         l.Listing.Title,
			Description:   l.Listing.Description,
			AddressLine1:  l.Listing.AddressLine1,
			AddressLine2:  l.Listing.AddressLine2,
			City:          l.Listing.City,
			StateProvince: l.Listing.StateProvince,
			Country:       l.Listing.Country,
			PostalCode:    l.Listing.PostalCode,
			Latitude:      l.Listing.Latitude,
			Longitude:     l.Listing.Longitude,
			BedroomCount:  l.Listing.BedroomCount,
			BathroomCount: l.Listing.BathroomCount,
			MaxGuests:     l.Listing.MaxGuests,
			SquareMeters:  l.Listing.SquareMeters,
			IsVerified:     l.Listing.IsVerified,
		},
		Images:    l.Images,
		Amenities: l.Amenities,
	}

	return &propertyv1.Listing{
		Id:            l.Listing.ID.String(),
		PropertyId:    l.Listing.PropertyID.String(),
		Type:          mapListingType(l.Listing.Type),
		Status:        mapListingStatus(l.Listing.Status),
		BasePrice:     numericToFloat64(l.Listing.BasePrice),
		Currency:      l.Listing.Currency,
		IsInstantBook: l.Listing.IsInstantBook,
		CreatedAt:     timestamppb.New(l.Listing.CreatedAt),
		UpdatedAt:     timestamppb.New(l.Listing.UpdatedAt),
		RentalMeta:    pbRental,
		SalesMeta:     pbSales,
		Property:      mapProperty(propDetails),
	}
}

func mapBooking(b *db.RentalBooking, listing *service.ListingWithDetails) *propertyv1.Booking {
	if b == nil {
		return nil
	}

	return &propertyv1.Booking{
		Id:              b.ID.String(),
		ListingId:       b.ListingID.String(),
		TenantId:        b.TenantID.String(),
		StartDate:       b.StartDate.Time.Format("2006-01-02"),
		EndDate:         b.EndDate.Time.Format("2006-01-02"),
		NightlyPrice:    numericToFloat64(b.NightlyPrice),
		CleaningFee:     numericToFloat64(b.CleaningFee),
		SecurityDeposit: numericToFloat64(b.SecurityDeposit),
		TotalPrice:      numericToFloat64(b.TotalPrice),
		Status:          mapBookingStatus(b.Status),
		PaymentIntentId: deref(b.PaymentIntentID),
		CreatedAt:       timestamppb.New(b.CreatedAt),
		UpdatedAt:       timestamppb.New(b.UpdatedAt),
		Listing:         mapListing(listing),
	}
}

func mapOffer(o *db.SalesOffer, listing *service.ListingWithDetails) *propertyv1.Offer {
	if o == nil {
		return nil
	}

	return &propertyv1.Offer{
		Id:                 o.ID.String(),
		ListingId:          o.ListingID.String(),
		BuyerId:            o.BuyerID.String(),
		OfferPrice:         numericToFloat64(o.OfferPrice),
		EscrowDepositPaid:  numericToFloat64(o.EscrowDepositPaid),
		Status:             mapOfferStatus(o.Status),
		ConditionsText:     deref(o.ConditionsText),
		ExpirationDate:     timestamppb.New(o.ExpirationDate),
		CreatedAt:          timestamppb.New(o.CreatedAt),
		UpdatedAt:          timestamppb.New(o.UpdatedAt),
		Listing:            mapListing(listing),
	}
}

func mapViewingAppointment(a *db.ViewingAppointment, listing *service.ListingWithDetails) *propertyv1.ViewingAppointment {
	if a == nil {
		return nil
	}

	return &propertyv1.ViewingAppointment{
		Id:            a.ID.String(),
		ListingId:     a.ListingID.String(),
		ClientId:      a.ClientID.String(),
		HostId:        a.HostID.String(),
		ScheduledTime: timestamppb.New(a.ScheduledTime),
		Status:        mapAppointmentStatus(a.Status),
		Notes:         deref(a.Notes),
		CreatedAt:     timestamppb.New(a.CreatedAt),
		UpdatedAt:     timestamppb.New(a.UpdatedAt),
		Listing:       mapListing(listing),
	}
}

func mapPropertyType(t string) propertyv1.PropertyType {
	switch t {
	case "apartment":
		return propertyv1.PropertyType_PROPERTY_TYPE_APARTMENT
	case "house":
		return propertyv1.PropertyType_PROPERTY_TYPE_HOUSE
	case "villa":
		return propertyv1.PropertyType_PROPERTY_TYPE_VILLA
	case "condo":
		return propertyv1.PropertyType_PROPERTY_TYPE_CONDO
	case "townhouse":
		return propertyv1.PropertyType_PROPERTY_TYPE_TOWNHOUSE
	case "cabin":
		return propertyv1.PropertyType_PROPERTY_TYPE_CABIN
	case "studio":
		return propertyv1.PropertyType_PROPERTY_TYPE_STUDIO
	case "land":
		return propertyv1.PropertyType_PROPERTY_TYPE_LAND
	default:
		return propertyv1.PropertyType_PROPERTY_TYPE_UNSPECIFIED
	}
}

func mapProtoPropertyType(t propertyv1.PropertyType) string {
	switch t {
	case propertyv1.PropertyType_PROPERTY_TYPE_APARTMENT:
		return "apartment"
	case propertyv1.PropertyType_PROPERTY_TYPE_HOUSE:
		return "house"
	case propertyv1.PropertyType_PROPERTY_TYPE_VILLA:
		return "villa"
	case propertyv1.PropertyType_PROPERTY_TYPE_CONDO:
		return "condo"
	case propertyv1.PropertyType_PROPERTY_TYPE_TOWNHOUSE:
		return "townhouse"
	case propertyv1.PropertyType_PROPERTY_TYPE_CABIN:
		return "cabin"
	case propertyv1.PropertyType_PROPERTY_TYPE_STUDIO:
		return "studio"
	case propertyv1.PropertyType_PROPERTY_TYPE_LAND:
		return "land"
	default:
		return "apartment"
	}
}

func mapListingType(t string) propertyv1.ListingType {
	switch t {
	case "rental":
		return propertyv1.ListingType_LISTING_TYPE_RENTAL
	case "sale":
		return propertyv1.ListingType_LISTING_TYPE_SALE
	default:
		return propertyv1.ListingType_LISTING_TYPE_UNSPECIFIED
	}
}

func mapProtoListingType(t propertyv1.ListingType) string {
	switch t {
	case propertyv1.ListingType_LISTING_TYPE_RENTAL:
		return "rental"
	case propertyv1.ListingType_LISTING_TYPE_SALE:
		return "sale"
	default:
		return "rental"
	}
}

func mapListingStatus(s string) propertyv1.ListingStatus {
	switch s {
	case "draft":
		return propertyv1.ListingStatus_LISTING_STATUS_DRAFT
	case "pending_approval":
		return propertyv1.ListingStatus_LISTING_STATUS_PENDING_APPROVAL
	case "active":
		return propertyv1.ListingStatus_LISTING_STATUS_ACTIVE
	case "suspended":
		return propertyv1.ListingStatus_LISTING_STATUS_SUSPENDED
	case "sold":
		return propertyv1.ListingStatus_LISTING_STATUS_SOLD
	case "rented":
		return propertyv1.ListingStatus_LISTING_STATUS_RENTED
	case "inactive":
		return propertyv1.ListingStatus_LISTING_STATUS_INACTIVE
	default:
		return propertyv1.ListingStatus_LISTING_STATUS_UNSPECIFIED
	}
}

func mapBookingStatus(s string) propertyv1.BookingStatus {
	switch s {
	case "pending_payment":
		return propertyv1.BookingStatus_BOOKING_STATUS_PENDING_PAYMENT
	case "requested":
		return propertyv1.BookingStatus_BOOKING_STATUS_REQUESTED
	case "confirmed":
		return propertyv1.BookingStatus_BOOKING_STATUS_CONFIRMED
	case "cancelled":
		return propertyv1.BookingStatus_BOOKING_STATUS_CANCELLED
	case "active":
		return propertyv1.BookingStatus_BOOKING_STATUS_ACTIVE
	case "completed":
		return propertyv1.BookingStatus_BOOKING_STATUS_COMPLETED
	case "disputed":
		return propertyv1.BookingStatus_BOOKING_STATUS_DISPUTED
	default:
		return propertyv1.BookingStatus_BOOKING_STATUS_UNSPECIFIED
	}
}

func mapOfferStatus(s string) propertyv1.OfferStatus {
	switch s {
	case "submitted":
		return propertyv1.OfferStatus_OFFER_STATUS_SUBMITTED
	case "accepted":
		return propertyv1.OfferStatus_OFFER_STATUS_ACCEPTED
	case "rejected":
		return propertyv1.OfferStatus_OFFER_STATUS_REJECTED
	case "under_escrow":
		return propertyv1.OfferStatus_OFFER_STATUS_UNDER_ESCROW
	case "funds_released":
		return propertyv1.OfferStatus_OFFER_STATUS_FUNDS_RELEASED
	case "cancelled":
		return propertyv1.OfferStatus_OFFER_STATUS_CANCELLED
	case "disputed":
		return propertyv1.OfferStatus_OFFER_STATUS_DISPUTED
	default:
		return propertyv1.OfferStatus_OFFER_STATUS_UNSPECIFIED
	}
}

func mapAppointmentStatus(s string) propertyv1.AppointmentStatus {
	switch s {
	case "scheduled":
		return propertyv1.AppointmentStatus_APPOINTMENT_STATUS_SCHEDULED
	case "confirmed":
		return propertyv1.AppointmentStatus_APPOINTMENT_STATUS_CONFIRMED
	case "completed":
		return propertyv1.AppointmentStatus_APPOINTMENT_STATUS_COMPLETED
	case "cancelled":
		return propertyv1.AppointmentStatus_APPOINTMENT_STATUS_CANCELLED
	case "no_show":
		return propertyv1.AppointmentStatus_APPOINTMENT_STATUS_NO_SHOW
	default:
		return propertyv1.AppointmentStatus_APPOINTMENT_STATUS_UNSPECIFIED
	}
}

func grpcErr(err error) error {
	return err
}

func optionalStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalInt32(i int32) *int32 {
	if i == 0 {
		return nil
	}
	return &i
}

func optionalFloat64(f float64) *float64 {
	if f == 0.0 {
		return nil
	}
	return &f
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func derefFloat64(f *float64) float64 {
	if f == nil {
		return 0.0
	}
	return *f
}

func formatPgTime(t pgtype.Time) string {
	if !t.Valid {
		return ""
	}
	// Microseconds to time
	hr := t.Microseconds / 3600000000
	rem := t.Microseconds % 3600000000
	min := rem / 60000000
	rem = rem % 60000000
	sec := rem / 1000000
	return fmt.Sprintf("%02d:%02d:%02d", hr, min, sec)
}

func numericToFloat64(num pgtype.Numeric) float64 {
	if !num.Valid {
		return 0
	}
	var f float64
	_ = num.Scan(&f)
	return f
}
