package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	werr "github.com/wemall/pkg/errors"
	"github.com/wemall/seller-service/internal/db"
)

// SellerService implements store and payout business logic.
type SellerService struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewSellerService(q *db.Queries, pool *pgxpool.Pool) *SellerService {
	return &SellerService{q: q, pool: pool}
}

func (s *SellerService) GetSeller(ctx context.Context, id uuid.UUID) (*db.Seller, error) {
	seller, err := s.q.GetSellerByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("seller not found")
		}
		return nil, werr.Internal(err)
	}
	return &seller, nil
}

func (s *SellerService) GetSellerByUserID(ctx context.Context, userID uuid.UUID) (*db.Seller, error) {
	seller, err := s.q.GetSellerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("seller store not found")
		}
		return nil, werr.Internal(err)
	}
	return &seller, nil
}

func (s *SellerService) GetSellerBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]db.Seller, error) {
	if len(ids) == 0 {
		return map[uuid.UUID]db.Seller{}, nil
	}
	rows, err := s.q.GetSellersByIDs(ctx, ids)
	if err != nil {
		return nil, werr.Internal(err)
	}
	out := make(map[uuid.UUID]db.Seller, len(rows))
	for _, row := range rows {
		out[row.ID] = row
	}
	return out, nil
}

type CreateStoreInput struct {
	UserID      uuid.UUID
	StoreName   string
	Description *string
	LogoURL     *string
	BannerURL   *string
	Latitude    *float64
	Longitude   *float64
}

func (s *SellerService) CreateStore(ctx context.Context, in CreateStoreInput) (*db.Seller, error) {
	if in.StoreName == "" {
		return nil, werr.InvalidArgument("store_name is required")
	}

	// Validate coordinates if provided
	if err := validateCoordinates(in.Latitude, in.Longitude); err != nil {
		return nil, err
	}

	slug, err := s.uniqueSlug(ctx, Slugify(in.StoreName))
	if err != nil {
		return nil, err
	}

	seller, err := s.q.CreateSeller(ctx, db.CreateSellerParams{
		UserID:      in.UserID,
		StoreName:   in.StoreName,
		StoreSlug:   slug,
		LogoUrl:     in.LogoURL,
		BannerUrl:   in.BannerURL,
		Description: in.Description,
		Latitude:    in.Latitude,
		Longitude:   in.Longitude,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, werr.AlreadyExists("seller store already exists for this user")
		}
		return nil, werr.Internal(err)
	}
	return &seller, nil
}

type UpdateStoreInput struct {
	UserID      uuid.UUID
	StoreName   *string
	Description *string
	LogoURL     *string
	BannerURL   *string
	Latitude    *float64
	Longitude   *float64
}

func (s *SellerService) UpdateStore(ctx context.Context, in UpdateStoreInput) (*db.Seller, error) {
	current, err := s.GetSellerByUserID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}

	// Validate coordinates if provided
	if err := validateCoordinates(in.Latitude, in.Longitude); err != nil {
		return nil, err
	}

	var storeName, storeSlug *string
	if in.StoreName != nil && *in.StoreName != "" && *in.StoreName != current.StoreName {
		storeName = in.StoreName
		slug, slugErr := s.uniqueSlugExcluding(ctx, Slugify(*in.StoreName), current.ID)
		if slugErr != nil {
			return nil, slugErr
		}
		storeSlug = &slug
	}

	seller, err := s.q.UpdateSeller(ctx, db.UpdateSellerParams{
		UserID:      in.UserID,
		StoreName:   ptrToString(storeName),
		StoreSlug:   ptrToString(storeSlug),
		LogoUrl:     in.LogoURL,
		BannerUrl:   in.BannerURL,
		Description: in.Description,
		Latitude:    in.Latitude,
		Longitude:   in.Longitude,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, werr.AlreadyExists("store name or slug already taken")
		}
		return nil, werr.Internal(err)
	}
	return &seller, nil
}

func (s *SellerService) UpdateSellerStatus(ctx context.Context, sellerID uuid.UUID, status db.SellerStatus) (*db.Seller, error) {
	// Validate the status transition
	validStatuses := map[db.SellerStatus]bool{
		db.SellerStatusPending:    true,
		db.SellerStatusProcessing: true,
		db.SellerStatusVerified:   true,
		db.SellerStatusSuspended:  true,
	}
	if !validStatuses[status] {
		return nil, werr.InvalidArgument("invalid seller status")
	}

	seller, err := s.q.UpdateSellerStatus(ctx, db.UpdateSellerStatusParams{
		ID:     sellerID,
		Status: status,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("seller not found")
		}
		return nil, werr.Internal(err)
	}

	// If status is verified, also set is_verified = true
	if status == db.SellerStatusVerified {
		seller, err = s.q.VerifySeller(ctx, db.VerifySellerParams{ID: sellerID, IsVerified: true})
		if err != nil {
			return nil, werr.Internal(err)
		}
	} else if status == db.SellerStatusSuspended {
		seller, err = s.q.VerifySeller(ctx, db.VerifySellerParams{ID: sellerID, IsVerified: false})
		if err != nil {
			return nil, werr.Internal(err)
		}
	}

	return &seller, nil
}

func (s *SellerService) VerifySeller(ctx context.Context, sellerID uuid.UUID, verified bool) (*db.Seller, error) {
	seller, err := s.q.VerifySeller(ctx, db.VerifySellerParams{
		ID:         sellerID,
		IsVerified: verified,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("seller not found")
		}
		return nil, werr.Internal(err)
	}
	return &seller, nil
}

// ── Store Follows ─────────────────────────────────────────────────────────────

func (s *SellerService) FollowStore(ctx context.Context, userID, sellerID uuid.UUID) error {
	// Verify seller exists
	if _, err := s.GetSeller(ctx, sellerID); err != nil {
		return err
	}
	if err := s.q.FollowStore(ctx, db.FollowStoreParams{UserID: userID, SellerID: sellerID}); err != nil {
		return werr.Internal(err)
	}
	return nil
}

func (s *SellerService) UnfollowStore(ctx context.Context, userID, sellerID uuid.UUID) error {
	if err := s.q.UnfollowStore(ctx, db.UnfollowStoreParams{UserID: userID, SellerID: sellerID}); err != nil {
		return werr.Internal(err)
	}
	return nil
}

func (s *SellerService) IsFollowingStore(ctx context.Context, userID, sellerID uuid.UUID) (bool, error) {
	following, err := s.q.IsFollowingStore(ctx, db.IsFollowingStoreParams{UserID: userID, SellerID: sellerID})
	if err != nil {
		return false, werr.Internal(err)
	}
	return following, nil
}

func (s *SellerService) ListFollowedStores(ctx context.Context, userID uuid.UUID, pageSize int32, pageToken string) ([]db.Seller, int32, string, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := int32(0)
	if pageToken != "" {
		n, err := strconv.ParseInt(pageToken, 10, 32)
		if err != nil || n < 0 {
			return nil, 0, "", werr.InvalidArgument("invalid page_token")
		}
		offset = int32(n)
	}

	total, err := s.q.CountFollowedStores(ctx, userID)
	if err != nil {
		return nil, 0, "", werr.Internal(err)
	}

	rows, err := s.q.ListFollowedStores(ctx, db.ListFollowedStoresParams{
		UserID: userID,
		Limit:  pageSize,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, "", werr.Internal(err)
	}

	nextToken := ""
	if int(offset)+len(rows) < int(total) {
		nextToken = strconv.FormatInt(int64(offset)+int64(pageSize), 10)
	}

	return rows, int32(total), nextToken, nil
}

func (s *SellerService) ListStoreFollowers(ctx context.Context, sellerID uuid.UUID) ([]uuid.UUID, error) {
	userIDs, err := s.q.ListStoreFollowers(ctx, sellerID)
	if err != nil {
		return nil, werr.Internal(err)
	}
	return userIDs, nil
}

// ── Payouts ───────────────────────────────────────────────────────────────────

func (s *SellerService) ListPayouts(ctx context.Context, sellerID uuid.UUID, pageSize int32, pageToken string) ([]db.SellerPayout, int32, string, error) {
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := int32(0)
	if pageToken != "" {
		n, err := strconv.ParseInt(pageToken, 10, 32)
		if err != nil || n < 0 {
			return nil, 0, "", werr.InvalidArgument("invalid page_token")
		}
		offset = int32(n)
	}

	total, err := s.q.CountPayoutsBySeller(ctx, sellerID)
	if err != nil {
		return nil, 0, "", werr.Internal(err)
	}

	rows, err := s.q.ListPayoutsBySeller(ctx, db.ListPayoutsBySellerParams{
		SellerID: sellerID,
		Limit:    pageSize,
		Offset:   offset,
	})
	if err != nil {
		return nil, 0, "", werr.Internal(err)
	}

	nextToken := ""
	if int(offset)+len(rows) < int(total) {
		nextToken = strconv.FormatInt(int64(offset)+int64(pageSize), 10)
	}

	return rows, total, nextToken, nil
}

func (s *SellerService) GetPayout(ctx context.Context, payoutID, sellerID uuid.UUID) (*db.SellerPayout, error) {
	payout, err := s.q.GetPayoutByIDForSeller(ctx, db.GetPayoutByIDForSellerParams{
		ID:       payoutID,
		SellerID: sellerID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("payout not found")
		}
		return nil, werr.Internal(err)
	}
	return &payout, nil
}

func (s *SellerService) CreatePayout(ctx context.Context, sellerID uuid.UUID, amount float64, currency string) (*db.SellerPayout, error) {
	if amount <= 0 {
		return nil, werr.InvalidArgument("amount must be positive")
	}
	if currency == "" {
		currency = "USD"
	}

	if _, err := s.GetSeller(ctx, sellerID); err != nil {
		return nil, err
	}

	payout, err := s.q.CreatePayout(ctx, db.CreatePayoutParams{
		SellerID: sellerID,
		Amount:   amount,
		Currency: currency,
	})
	if err != nil {
		return nil, werr.Internal(err)
	}
	return &payout, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (s *SellerService) uniqueSlug(ctx context.Context, base string) (string, error) {
	return s.uniqueSlugExcluding(ctx, base, uuid.Nil)
}

func (s *SellerService) uniqueSlugExcluding(ctx context.Context, base string, excludeID uuid.UUID) (string, error) {
	if base == "" {
		base = "store"
	}
	candidate := base
	for i := 0; i < 100; i++ {
		existing, err := s.q.GetSellerByStoreSlug(ctx, candidate)
		if errors.Is(err, pgx.ErrNoRows) {
			return candidate, nil
		}
		if err != nil {
			return "", werr.Internal(err)
		}
		if excludeID != uuid.Nil && existing.ID == excludeID {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s-%d", base, i+2)
	}
	return "", werr.Internal(fmt.Errorf("could not generate unique slug"))
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}

// validateCoordinates ensures latitude and longitude are within valid ranges
func validateCoordinates(lat, lon *float64) error {
	if lat != nil {
		if *lat < -90 || *lat > 90 {
			return werr.InvalidArgument("latitude must be between -90 and 90")
		}
	}
	if lon != nil {
		if *lon < -180 || *lon > 180 {
			return werr.InvalidArgument("longitude must be between -180 and 180")
		}
	}
	// Both coordinates should be provided together or not at all
	if (lat != nil) != (lon != nil) {
		return werr.InvalidArgument("both latitude and longitude must be provided together")
	}
	return nil
}

// ptrToString converts a *string to string, returning empty string if nil
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
