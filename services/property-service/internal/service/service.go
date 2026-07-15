package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	werr "github.com/wemall/pkg/errors"
	"github.com/wemall/property-service/internal/db"
)

type PropertyWithDetails struct {
	Property  db.GetPropertyRow
	Images    []db.PropertyImage
	Amenities []db.PropertyAmenity
}

type ListingWithDetails struct {
	Listing    db.GetListingRow
	Images     []db.PropertyImage
	Amenities  []db.PropertyAmenity
	RentalMeta *db.RentalListingsMetum
	SalesMeta  *db.SalesListingsMetum
}

type PropertyService struct {
	q    *db.Queries
	pool *pgxpool.Pool
	nc   *nats.Conn
}

func NewPropertyService(q *db.Queries, pool *pgxpool.Pool, nc *nats.Conn) *PropertyService {
	return &PropertyService{q: q, pool: pool, nc: nc}
}

// ── Property Management ──────────────────────────────────────────────────────

type CreatePropertyInput struct {
	OwnerID       uuid.UUID
	Type          string
	Title         string
	Description   string
	AddressLine1  string
	AddressLine2  *string
	City          string
	StateProvince string
	Country       string
	PostalCode    *string
	Latitude      float64
	Longitude     float64
	BedroomCount  int32
	BathroomCount float64
	MaxGuests     int32
	SquareMeters  float64
	ImageURLs     []string
	Amenities     []db.CreatePropertyAmenityParams
}

func (s *PropertyService) CreateProperty(ctx context.Context, in CreatePropertyInput) (*PropertyWithDetails, error) {
	if in.Title == "" || in.City == "" || in.Country == "" {
		return nil, werr.InvalidArgument("title, city, and country are required")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, werr.Internal(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	// 1. Insert Property
	prop, err := qtx.CreateProperty(ctx, db.CreatePropertyParams{
		OwnerID:       in.OwnerID,
		Type:          in.Type,
		Title:         in.Title,
		Description:   in.Description,
		AddressLine1:  in.AddressLine1,
		AddressLine2:  in.AddressLine2,
		City:          in.City,
		StateProvince: in.StateProvince,
		Country:       in.Country,
		PostalCode:    in.PostalCode,
		Column11:      in.Longitude,
		Column12:      in.Latitude,
		BedroomCount:  in.BedroomCount,
		BathroomCount: toNumeric(in.BathroomCount),
		MaxGuests:     in.MaxGuests,
		SquareMeters:  toNumeric(in.SquareMeters),
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	// 2. Insert Images
	var dbImages []db.PropertyImage
	for idx, url := range in.ImageURLs {
		img, err := qtx.CreatePropertyImage(ctx, db.CreatePropertyImageParams{
			PropertyID:   prop.ID,
			Url:          url,
			DisplayOrder: int32(idx),
			IsCover:      idx == 0,
		})
		if err != nil {
			return nil, werr.Internal(err)
		}
		dbImages = append(dbImages, img)
	}

	// 3. Insert Amenities
	var dbAmenities []db.PropertyAmenity
	for _, am := range in.Amenities {
		am.PropertyID = prop.ID
		dbAm, err := qtx.CreatePropertyAmenity(ctx, am)
		if err != nil {
			return nil, werr.Internal(err)
		}
		dbAmenities = append(dbAmenities, dbAm)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, werr.Internal(err)
	}

	// Publish NATS event
	s.publishEvent("wemall.property.created", map[string]interface{}{
		"property_id": prop.ID.String(),
		"owner_id":    prop.OwnerID.String(),
		"title":       prop.Title,
		"type":        prop.Type,
		"city":        prop.City,
		"created_at":  prop.CreatedAt,
	})

	// Get properties formatted back with latitude/longitude
	getProp, err := s.q.GetProperty(ctx, prop.ID)
	if err != nil {
		return nil, werr.Internal(err)
	}

	return &PropertyWithDetails{
		Property:  getProp,
		Images:    dbImages,
		Amenities: dbAmenities,
	}, nil
}

func (s *PropertyService) GetProperty(ctx context.Context, id uuid.UUID) (*PropertyWithDetails, error) {
	prop, err := s.q.GetProperty(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("property not found")
		}
		return nil, werr.Internal(err)
	}

	images, err := s.q.GetPropertyImages(ctx, id)
	if err != nil {
		return nil, werr.Internal(err)
	}

	amenities, err := s.q.GetPropertyAmenities(ctx, id)
	if err != nil {
		return nil, werr.Internal(err)
	}

	return &PropertyWithDetails{
		Property:  prop,
		Images:    images,
		Amenities: amenities,
	}, nil
}

// ── Listing Management ───────────────────────────────────────────────────────

type CreateListingInput struct {
	PropertyID     uuid.UUID
	Type           string
	BasePrice      float64
	Currency       string
	IsInstantBook  bool
	CleaningFee    float64
	SecurityDeposit float64
	MinNights      int32
	MaxNights      int32
	CheckInTime    string
	CheckOutTime   string
	EscrowPercent  float64
	AgentCommRate  float64
	Furniture      bool
	YearBuilt      *int32
	PropertyTax    *float64
}

func (s *PropertyService) CreateListing(ctx context.Context, in CreateListingInput) (*ListingWithDetails, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, werr.Internal(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	// Check if property exists
	_, err = qtx.GetProperty(ctx, in.PropertyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("property not found")
		}
		return nil, werr.Internal(err)
	}

	// 1. Create Core Listing
	listing, err := qtx.CreateListing(ctx, db.CreateListingParams{
		PropertyID:    in.PropertyID,
		Type:          in.Type,
		Status:        "active",
		BasePrice:     toNumeric(in.BasePrice),
		Currency:      in.Currency,
		IsInstantBook: in.IsInstantBook,
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	var rentalMeta *db.RentalListingsMetum
	var salesMeta *db.SalesListingsMetum

	// 2. Create Listing Sub-type Metadata
	if in.Type == "rental" {
		checkIn, err := time.Parse("15:04:05", in.CheckInTime)
		if err != nil {
			checkIn = time.Date(0, 1, 1, 15, 0, 0, 0, time.UTC)
		}
		checkOut, err := time.Parse("15:04:05", in.CheckOutTime)
		if err != nil {
			checkOut = time.Date(0, 1, 1, 11, 0, 0, 0, time.UTC)
		}

		meta, err := qtx.CreateRentalListingMeta(ctx, db.CreateRentalListingMetaParams{
			ListingID:       listing.ID,
			CleaningFee:     toNumeric(in.CleaningFee),
			SecurityDeposit: toNumeric(in.SecurityDeposit),
			MinNights:       in.MinNights,
			MaxNights:       in.MaxNights,
			CheckInTime:     pgtype.Time{Microseconds: int64(checkIn.Hour()*3600+checkIn.Minute()*60+checkIn.Second()) * 1000000, Valid: true},
			CheckOutTime:    pgtype.Time{Microseconds: int64(checkOut.Hour()*3600+checkOut.Minute()*60+checkOut.Second()) * 1000000, Valid: true},
		})
		if err != nil {
			return nil, werr.Internal(err)
		}
		rentalMeta = &meta
	} else if in.Type == "sale" {
		var tax pgtype.Numeric
		if in.PropertyTax != nil {
			tax = toNumeric(*in.PropertyTax)
		}
		meta, err := qtx.CreateSalesListingMeta(ctx, db.CreateSalesListingMetaParams{
			ListingID:             listing.ID,
			EscrowDepositPercent: toNumeric(in.EscrowPercent),
			AgentCommissionRate:  toNumeric(in.AgentCommRate),
			IncludesFurniture:     in.Furniture,
			YearBuilt:             in.YearBuilt,
			PropertyTaxAnnual:    tax,
		})
		if err != nil {
			return nil, werr.Internal(err)
		}
		salesMeta = &meta
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, werr.Internal(err)
	}

	// Fetch detailed listing output
	getListing, err := s.q.GetListing(ctx, listing.ID)
	if err != nil {
		return nil, werr.Internal(err)
	}

	images, err := s.q.GetPropertyImages(ctx, in.PropertyID)
	if err != nil {
		return nil, werr.Internal(err)
	}

	amenities, err := s.q.GetPropertyAmenities(ctx, in.PropertyID)
	if err != nil {
		return nil, werr.Internal(err)
	}

	return &ListingWithDetails{
		Listing:    getListing,
		Images:     images,
		Amenities:  amenities,
		RentalMeta: rentalMeta,
		SalesMeta:  salesMeta,
	}, nil
}

func (s *PropertyService) GetListing(ctx context.Context, id uuid.UUID) (*ListingWithDetails, error) {
	listing, err := s.q.GetListing(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("listing not found")
		}
		return nil, werr.Internal(err)
	}

	images, err := s.q.GetPropertyImages(ctx, listing.PropertyID)
	if err != nil {
		return nil, werr.Internal(err)
	}

	amenities, err := s.q.GetPropertyAmenities(ctx, listing.PropertyID)
	if err != nil {
		return nil, werr.Internal(err)
	}

	var rentalMeta *db.RentalListingsMetum
	if listing.Type == "rental" {
		meta, err := s.q.GetRentalListingMeta(ctx, id)
		if err == nil {
			rentalMeta = &meta
		}
	}

	var salesMeta *db.SalesListingsMetum
	if listing.Type == "sale" {
		meta, err := s.q.GetSalesListingMeta(ctx, id)
		if err == nil {
			salesMeta = &meta
		}
	}

	return &ListingWithDetails{
		Listing:    listing,
		Images:     images,
		Amenities:  amenities,
		RentalMeta: rentalMeta,
		SalesMeta:  salesMeta,
	}, nil
}

// ── GIS Search ───────────────────────────────────────────────────────────────

func (s *PropertyService) SearchNearbyListings(ctx context.Context, lat, lng, radius float64, limit, offset int32) ([]*ListingWithDetails, error) {
	rows, err := s.q.SearchNearbyListings(ctx, db.SearchNearbyListingsParams{
		Column1: lng,
		Column2: lat,
		Column3: radius,
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	var listings []*ListingWithDetails
	for _, r := range rows {
		// Populate mock/empty details for each nearby result (since this query flat-joins listing & property)
		listingDetails := db.GetListingRow{
			ID:             r.ListingID,
			PropertyID:     r.PropertyID,
			Type:           r.ListingType,
			Status:         r.ListingStatus,
			BasePrice:      r.BasePrice,
			Currency:       r.Currency,
			IsInstantBook:  r.IsInstantBook,
			CreatedAt:      r.CreatedAt,
			UpdatedAt:      r.UpdatedAt,
			OwnerID:        r.OwnerID,
			PropertyType:   r.PropertyType,
			Title:          r.Title,
			Description:    r.Description,
			AddressLine1:   r.AddressLine1,
			AddressLine2:   r.AddressLine2,
			City:           r.City,
			StateProvince:  r.StateProvince,
			Country:        r.Country,
			PostalCode:     r.PostalCode,
			Latitude:       r.Latitude,
			Longitude:      r.Longitude,
			BedroomCount:   r.BedroomCount,
			BathroomCount:  r.BathroomCount,
			MaxGuests:      r.MaxGuests,
			SquareMeters:   r.SquareMeters,
			IsVerified:     r.IsVerified,
		}

		listings = append(listings, &ListingWithDetails{
			Listing: listingDetails,
		})
	}

	return listings, nil
}

func (s *PropertyService) GetListingsInViewport(ctx context.Context, minLat, minLng, maxLat, maxLng float64) ([]*ListingWithDetails, error) {
	rows, err := s.q.GetListingsInViewport(ctx, db.GetListingsInViewportParams{
		Column1: minLng,
		Column2: minLat,
		Column3: maxLng,
		Column4: maxLat,
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	var listings []*ListingWithDetails
	for _, r := range rows {
		listingDetails := db.GetListingRow{
			ID:             r.ListingID,
			PropertyID:     r.PropertyID,
			Type:           r.ListingType,
			Status:         r.ListingStatus,
			BasePrice:      r.BasePrice,
			Currency:       r.Currency,
			IsInstantBook:  r.IsInstantBook,
			CreatedAt:      r.CreatedAt,
			UpdatedAt:      r.UpdatedAt,
			OwnerID:        r.OwnerID,
			PropertyType:   r.PropertyType,
			Title:          r.Title,
			Description:    r.Description,
			AddressLine1:   r.AddressLine1,
			AddressLine2:   r.AddressLine2,
			City:           r.City,
			StateProvince:  r.StateProvince,
			Country:        r.Country,
			PostalCode:     r.PostalCode,
			Latitude:       r.Latitude,
			Longitude:      r.Longitude,
			BedroomCount:   r.BedroomCount,
			BathroomCount:  r.BathroomCount,
			MaxGuests:      r.MaxGuests,
			SquareMeters:   r.SquareMeters,
			IsVerified:     r.IsVerified,
		}

		listings = append(listings, &ListingWithDetails{
			Listing: listingDetails,
		})
	}

	return listings, nil
}

// ── Rental Bookings ──────────────────────────────────────────────────────────

func (s *PropertyService) CreateBooking(ctx context.Context, listingID, tenantID uuid.UUID, startStr, endStr string) (*db.RentalBooking, error) {
	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return nil, werr.InvalidArgument("invalid start date format (use YYYY-MM-DD)")
	}
	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return nil, werr.InvalidArgument("invalid end date format (use YYYY-MM-DD)")
	}

	if !endDate.After(startDate) {
		return nil, werr.InvalidArgument("checkout date must be after checkin date")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, werr.Internal(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	// Fetch Listing
	listing, err := qtx.GetListing(ctx, listingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("listing not found")
		}
		return nil, werr.Internal(err)
	}

	if listing.Type != "rental" {
		return nil, werr.InvalidArgument("cannot book a sale-type listing")
	}

	rentalMeta, err := qtx.GetRentalListingMeta(ctx, listingID)
	if err != nil {
		return nil, werr.Internal(err)
	}

	nights := int32(endDate.Sub(startDate).Hours() / 24)
	if nights < rentalMeta.MinNights || nights > rentalMeta.MaxNights {
		return nil, werr.InvalidArgument(fmt.Sprintf("booking duration must be between %d and %d nights", rentalMeta.MinNights, rentalMeta.MaxNights))
	}

	totalPrice := (numericToFloat(listing.BasePrice) * float64(nights)) + numericToFloat(rentalMeta.CleaningFee)

	// Save Booking (exclusion constraint in DB prevents overlaps)
	booking, err := qtx.CreateBooking(ctx, db.CreateBookingParams{
		ListingID:       listingID,
		TenantID:        tenantID,
		StartDate:       pgtype.Date{Time: startDate, Valid: true},
		EndDate:         pgtype.Date{Time: endDate, Valid: true},
		NightlyPrice:    listing.BasePrice,
		CleaningFee:     rentalMeta.CleaningFee,
		SecurityDeposit: rentalMeta.SecurityDeposit,
		TotalPrice:      toNumeric(totalPrice),
		Status:          "requested",
	})
	if err != nil {
		// Detect PG exclude constraint overlap error
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23P01" {
			return nil, werr.AlreadyExists("property is already booked for the selected dates")
		}
		return nil, werr.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, werr.Internal(err)
	}

	// Publish Event
	s.publishEvent("wemall.rental.booking_created", map[string]interface{}{
		"booking_id": booking.ID.String(),
		"listing_id": booking.ListingID.String(),
		"tenant_id":  booking.TenantID.String(),
		"total_price": booking.TotalPrice,
		"start_date": booking.StartDate.Time.Format("2006-01-02"),
		"end_date":   booking.EndDate.Time.Format("2006-01-02"),
	})

	return &booking, nil
}

// ── Bidding / Offers ──────────────────────────────────────────────────────────

func (s *PropertyService) CreateOffer(ctx context.Context, listingID, buyerID uuid.UUID, offerPrice float64, conditions string, expiration time.Time) (*db.SalesOffer, error) {
	listing, err := s.q.GetListing(ctx, listingID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("listing not found")
		}
		return nil, werr.Internal(err)
	}

	if listing.Type != "sale" {
		return nil, werr.InvalidArgument("cannot place an offer on a rental listing")
	}

	offer, err := s.q.CreateOffer(ctx, db.CreateOfferParams{
		ListingID:      listingID,
		BuyerID:        buyerID,
		OfferPrice:     toNumeric(offerPrice),
		Status:         "submitted",
		ConditionsText: &conditions,
		ExpirationDate: expiration,
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	// Publish Event
	s.publishEvent("wemall.sales.offer_placed", map[string]interface{}{
		"offer_id":    offer.ID.String(),
		"listing_id":  offer.ListingID.String(),
		"buyer_id":    offer.BuyerID.String(),
		"offer_price": offer.OfferPrice,
	})

	return &offer, nil
}

func (s *PropertyService) AcceptOffer(ctx context.Context, offerID, ownerID uuid.UUID) (*db.SalesOffer, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, werr.Internal(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	// Fetch Offer
	offer, err := qtx.GetOffer(ctx, offerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("offer not found")
		}
		return nil, werr.Internal(err)
	}

	// Validate ownership
	listing, err := qtx.GetListing(ctx, offer.ListingID)
	if err != nil {
		return nil, werr.Internal(err)
	}

	if listing.OwnerID != ownerID {
		return nil, werr.PermissionDenied("only the property owner can accept offers")
	}

	// Accept Offer
	acceptedOffer, err := qtx.AcceptOffer(ctx, offerID)
	if err != nil {
		return nil, werr.Internal(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, werr.Internal(err)
	}

	// Publish Event
	s.publishEvent("wemall.sales.offer_accepted", map[string]interface{}{
		"offer_id":   acceptedOffer.ID.String(),
		"listing_id": acceptedOffer.ListingID.String(),
		"buyer_id":   acceptedOffer.BuyerID.String(),
	})

	return &acceptedOffer, nil
}

// ── Viewing Appointments ─────────────────────────────────────────────────────

func (s *PropertyService) ScheduleViewing(ctx context.Context, listingID, clientID, hostID uuid.UUID, scheduledTime time.Time, notes string) (*db.ViewingAppointment, error) {
	app, err := s.q.ScheduleViewing(ctx, db.ScheduleViewingParams{
		ListingID:     listingID,
		ClientID:      clientID,
		HostID:        hostID,
		ScheduledTime: scheduledTime,
		Notes:         &notes,
		Status:        "scheduled",
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	return &app, nil
}

// ── Helper functions ─────────────────────────────────────────────────────────

func (s *PropertyService) publishEvent(subject string, payload interface{}) {
	if s.nc == nil {
		return
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = s.nc.Publish(subject, bytes)
}

func toNumeric(f float64) pgtype.Numeric {
	var num pgtype.Numeric
	_ = num.Scan(fmt.Sprintf("%.4f", f))
	return num
}

func numericToFloat(num pgtype.Numeric) float64 {
	if !num.Valid {
		return 0
	}
	var f float64
	_ = num.Scan(&f)
	return f
}
