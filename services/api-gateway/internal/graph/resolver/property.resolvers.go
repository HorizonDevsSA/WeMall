package resolver

import (
	"context"
	"time"

	"github.com/wemall/api-gateway/internal/graph/gqlerrors"
	"github.com/wemall/api-gateway/internal/graph/model"
	"github.com/wemall/api-gateway/internal/middleware"
	propertyv1 "github.com/wemall/gen/property/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ── Property Mutations ────────────────────────────────────────────────────────

func (r *mutationResolver) CreateProperty(ctx context.Context, input model.CreatePropertyInput) (*model.Property, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	amenities := make([]*propertyv1.PropertyAmenityInput, len(input.Amenities))
	for i, am := range input.Amenities {
		amenities[i] = &propertyv1.PropertyAmenityInput{
			Name:     am.Name,
			Category: am.Category,
		}
	}

	resp, err := r.Clients.Property.CreateProperty(ctx, &propertyv1.CreatePropertyRequest{
		OwnerId:       uid,
		Type:          propertyv1.PropertyType(propertyv1.PropertyType_value["PROPERTY_TYPE_"+string(input.Type)]),
		Title:         input.Title,
		Description:   input.Description,
		AddressLine1:  input.AddressLine1,
		AddressLine2:  derefStr(input.AddressLine2),
		City:          input.City,
		StateProvince: input.StateProvince,
		Country:       input.Country,
		PostalCode:    derefStr(input.PostalCode),
		Latitude:      input.Latitude,
		Longitude:     input.Longitude,
		BedroomCount:  int32(input.BedroomCount),
		BathroomCount: input.BathroomCount,
		MaxGuests:     int32(input.MaxGuests),
		SquareMeters:  input.SquareMeters,
		ImageUrls:     input.ImageUrls,
		Amenities:     amenities,
	})
	if err != nil {
		return nil, err
	}
	return mapProperty(resp.Property), nil
}

func (r *mutationResolver) CreateListing(ctx context.Context, input model.CreateListingInput) (*model.Listing, error) {
	_, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	var rentalMeta *propertyv1.RentalListingMeta
	if input.RentalMeta != nil {
		rentalMeta = &propertyv1.RentalListingMeta{
			CleaningFee:     input.RentalMeta.CleaningFee,
			SecurityDeposit: input.RentalMeta.SecurityDeposit,
			MinNights:       int32(input.RentalMeta.MinNights),
			MaxNights:       int32(input.RentalMeta.MaxNights),
			CheckInTime:     input.RentalMeta.CheckInTime,
			CheckOutTime:    input.RentalMeta.CheckOutTime,
		}
	}

	var salesMeta *propertyv1.SalesListingMeta
	if input.SalesMeta != nil {
		var yearBuilt int32
		if input.SalesMeta.YearBuilt != nil {
			yearBuilt = int32(*input.SalesMeta.YearBuilt)
		}
		var tax float64
		if input.SalesMeta.PropertyTaxAnnual != nil {
			tax = *input.SalesMeta.PropertyTaxAnnual
		}
		salesMeta = &propertyv1.SalesListingMeta{
			EscrowDepositPercent: input.SalesMeta.EscrowDepositPercent,
			AgentCommissionRate:  input.SalesMeta.AgentCommissionRate,
			IncludesFurniture:    input.SalesMeta.IncludesFurniture,
			YearBuilt:            yearBuilt,
			PropertyTaxAnnual:    tax,
		}
	}

	resp, err := r.Clients.Property.CreateListing(ctx, &propertyv1.CreateListingRequest{
		PropertyId:    input.PropertyID,
		Type:          propertyv1.ListingType(propertyv1.ListingType_value["LISTING_TYPE_"+string(input.Type)]),
		BasePrice:     input.BasePrice,
		Currency:      input.Currency,
		IsInstantBook: input.IsInstantBook,
		RentalMeta:    rentalMeta,
		SalesMeta:     salesMeta,
	})
	if err != nil {
		return nil, err
	}
	return mapListing(resp.Listing), nil
}

func (r *mutationResolver) BookProperty(ctx context.Context, input model.BookPropertyInput) (*model.Booking, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Property.CreateBooking(ctx, &propertyv1.CreateBookingRequest{
		ListingId: input.ListingID,
		TenantId:  uid,
		StartDate: input.StartDate,
		EndDate:   input.EndDate,
	})
	if err != nil {
		return nil, err
	}
	return mapBooking(resp.Booking), nil
}

func (r *mutationResolver) MakePropertyOffer(ctx context.Context, input model.MakeOfferInput) (*model.Offer, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Property.CreateOffer(ctx, &propertyv1.CreateOfferRequest{
		ListingId:      input.ListingID,
		BuyerId:        uid,
		OfferPrice:     input.OfferPrice,
		ConditionsText: input.ConditionsText,
		ExpirationDate: timestamppb.New(input.ExpirationDate),
	})
	if err != nil {
		return nil, err
	}
	return mapOffer(resp.Offer), nil
}

func (r *mutationResolver) AcceptPropertyOffer(ctx context.Context, offerID string) (*model.Offer, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Property.AcceptOffer(ctx, &propertyv1.AcceptOfferRequest{
		Id:      offerID,
		OwnerId: uid,
	})
	if err != nil {
		return nil, err
	}
	return mapOffer(resp.Offer), nil
}

func (r *mutationResolver) ScheduleViewing(ctx context.Context, listingID string, scheduledTime time.Time, notes *string) (*model.ViewingAppointment, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Property.ScheduleViewing(ctx, &propertyv1.ScheduleViewingRequest{
		ListingId:     listingID,
		ClientId:      uid,
		ScheduledTime: timestamppb.New(scheduledTime),
		Notes:         derefStr(notes),
	})
	if err != nil {
		return nil, err
	}
	return mapViewingAppointment(resp.Appointment), nil
}

// ── Property Queries ──────────────────────────────────────────────────────────

func (r *queryResolver) Property(ctx context.Context, id string) (*model.Property, error) {
	resp, err := r.Clients.Property.GetProperty(ctx, &propertyv1.GetPropertyRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}
	return mapProperty(resp.Property), nil
}

func (r *queryResolver) Listing(ctx context.Context, id string) (*model.Listing, error) {
	resp, err := r.Clients.Property.GetListing(ctx, &propertyv1.GetListingRequest{
		Id: id,
	})
	if err != nil {
		return nil, err
	}
	return mapListing(resp.Listing), nil
}

func (r *queryResolver) NearbyProperties(ctx context.Context, lat float64, lng float64, radiusMeters float64, limit int, offset int) ([]*model.Listing, error) {
	resp, err := r.Clients.Property.SearchNearbyListings(ctx, &propertyv1.SearchNearbyListingsRequest{
		Latitude:     lat,
		Longitude:    lng,
		RadiusMeters: radiusMeters,
		Limit:        int32(limit),
		Offset:       int32(offset),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*model.Listing, len(resp.Listings))
	for i, l := range resp.Listings {
		out[i] = mapListing(l)
	}
	return out, nil
}

func (r *queryResolver) ViewportProperties(ctx context.Context, minLat float64, minLng float64, maxLat float64, maxLng float64) ([]*model.Listing, error) {
	resp, err := r.Clients.Property.GetListingsInViewport(ctx, &propertyv1.GetListingsInViewportRequest{
		MinLatitude:  minLat,
		MinLongitude: minLng,
		MaxLatitude:  maxLat,
		MaxLongitude: maxLng,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*model.Listing, len(resp.Listings))
	for i, l := range resp.Listings {
		out[i] = mapListing(l)
	}
	return out, nil
}
