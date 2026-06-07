package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"

	werr "github.com/wemall/pkg/errors"
	"github.com/wemall/review-service/internal/db"
)

type ReviewService struct {
	q    *db.Queries
	pool *pgxpool.Pool
	nc   *nats.Conn
}

func NewReviewService(q *db.Queries, pool *pgxpool.Pool, nc *nats.Conn) *ReviewService {
	return &ReviewService{q: q, pool: pool, nc: nc}
}

type CreateReviewInput struct {
	OrderID           uuid.UUID
	BuyerID           uuid.UUID
	SellerID          uuid.UUID
	ProductID         uuid.UUID
	VariantID         uuid.UUID
	RatingDescription int32
	RatingService     int32
	RatingDelivery    int32
	Content           string
	IsAnonymous       bool
	MediaURLs         []string
	IsSystemGenerated bool
}

func (s *ReviewService) CreateReview(ctx context.Context, in CreateReviewInput) (*db.Review, []db.ReviewMedium, error) {
	// 1. Validate inputs
	if in.RatingDescription < 1 || in.RatingDescription > 5 ||
		in.RatingService < 1 || in.RatingService > 5 ||
		in.RatingDelivery < 1 || in.RatingDelivery > 5 {
		return nil, nil, werr.InvalidArgument("ratings must be between 1 and 5")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, werr.Internal(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	// Check if already reviewed
	_, err = qtx.GetReviewByOrderAndVariant(ctx, db.GetReviewByOrderAndVariantParams{
		OrderID:   in.OrderID,
		VariantID: in.VariantID,
	})
	if err == nil {
		return nil, nil, werr.AlreadyExists("item in this order has already been reviewed")
	}

	// 2. Generate NLP tags mock
	nlpTags := generateMockNLPTags(in.Content)
	nlpTagsBytes, _ := json.Marshal(nlpTags)

	// 3. Insert Review
	hasMedia := len(in.MediaURLs) > 0
	var contentPtr *string
	if in.Content != "" {
		contentPtr = &in.Content
	}
	review, err := qtx.CreateReview(ctx, db.CreateReviewParams{
		OrderID:           in.OrderID,
		BuyerID:           in.BuyerID,
		SellerID:          in.SellerID,
		ProductID:         in.ProductID,
		VariantID:         in.VariantID,
		RatingDescription: in.RatingDescription,
		RatingService:     in.RatingService,
		RatingDelivery:    in.RatingDelivery,
		Content:           contentPtr,
		IsAnonymous:       &in.IsAnonymous,
		HasMedia:          &hasMedia,
		NlpTags:           nlpTagsBytes,
		IsSystemGenerated: &in.IsSystemGenerated,
	})
	if err != nil {
		return nil, nil, werr.Internal(err)
	}

	// 4. Insert Media
	var mediaList []db.ReviewMedium
	for i, url := range in.MediaURLs {
		mediaType := "image"
		if strings.HasSuffix(strings.ToLower(url), ".mp4") || strings.HasSuffix(strings.ToLower(url), ".mov") {
			mediaType = "video"
		}
		sortOrder := int32(i)
		m, err := qtx.CreateReviewMedia(ctx, db.CreateReviewMediaParams{
			ReviewID:  review.ID,
			MediaUrl:  url,
			MediaType: mediaType,
			SortOrder: &sortOrder,
		})
		if err != nil {
			return nil, nil, werr.Internal(err)
		}
		mediaList = append(mediaList, m)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, werr.Internal(err)
	}

	// 5. Publish NATS Event
	s.publishReviewEvent("wemall.review.created", &review)

	return &review, mediaList, nil
}

type AppendReviewInput struct {
	ReviewID  uuid.UUID
	BuyerID   uuid.UUID
	Content   string
	MediaURLs []string
}

func (s *ReviewService) AppendReview(ctx context.Context, in AppendReviewInput) (*db.AppendReview, []db.AppendReviewMedium, error) {
	if in.Content == "" {
		return nil, nil, werr.InvalidArgument("content is required for append review")
	}

	// Fetch parent review to ensure it exists and matches buyer
	review, err := s.q.GetReview(ctx, in.ReviewID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, werr.NotFound("original review not found")
		}
		return nil, nil, werr.Internal(err)
	}

	if review.BuyerID != in.BuyerID {
		return nil, nil, werr.PermissionDenied("only the original reviewer can append to this review")
	}

	// Taobao append reviews can be added up to 180 days post-review creation
	if time.Since(review.CreatedAt.Time) > 180*24*time.Hour {
		return nil, nil, werr.PermissionDenied("cannot append review after 180 days")
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, nil, werr.Internal(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	// Check if append review already exists
	_, err = qtx.GetAppendReview(ctx, in.ReviewID)
	if err == nil {
		return nil, nil, werr.AlreadyExists("an append review has already been submitted for this item")
	}

	// Insert Append Review
	hasMedia := len(in.MediaURLs) > 0
	appendRev, err := qtx.CreateAppendReview(ctx, db.CreateAppendReviewParams{
		ReviewID: in.ReviewID,
		Content:  in.Content,
		HasMedia: &hasMedia,
	})
	if err != nil {
		return nil, nil, werr.Internal(err)
	}

	// Insert Media
	var mediaList []db.AppendReviewMedium
	for i, url := range in.MediaURLs {
		mediaType := "image"
		if strings.HasSuffix(strings.ToLower(url), ".mp4") || strings.HasSuffix(strings.ToLower(url), ".mov") {
			mediaType = "video"
		}
		sortOrder := int32(i)
		m, err := qtx.CreateAppendReviewMedia(ctx, db.CreateAppendReviewMediaParams{
			AppendReviewID: appendRev.ID,
			MediaUrl:       url,
			MediaType:      mediaType,
			SortOrder:      &sortOrder,
		})
		if err != nil {
			return nil, nil, werr.Internal(err)
		}
		mediaList = append(mediaList, m)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, werr.Internal(err)
	}

	s.publishReviewEvent("wemall.review.updated", &review)

	return &appendRev, mediaList, nil
}

type UpdateReviewInput struct {
	ReviewID          uuid.UUID
	BuyerID           uuid.UUID
	RatingDescription int32
	RatingService     int32
	RatingDelivery    int32
	Content           string
}

func (s *ReviewService) UpdateReview(ctx context.Context, in UpdateReviewInput) (*db.Review, error) {
	review, err := s.q.GetReview(ctx, in.ReviewID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("review not found")
		}
		return nil, werr.Internal(err)
	}

	if review.BuyerID != in.BuyerID {
		return nil, werr.PermissionDenied("only the original reviewer can modify this review")
	}

	// 1. Goodwill window: Must be within 30 days of creation
	if time.Since(review.CreatedAt.Time) > 30*24*time.Hour {
		return nil, werr.PermissionDenied("modification window of 30 days has expired")
	}

	// 2. Taobao rules: Cannot downgrade rating, and cannot edit a review that is already "Good" (description >= 4)
	isOriginalGood := review.RatingDescription >= 4
	if isOriginalGood {
		return nil, werr.PermissionDenied("positive reviews cannot be modified or downgraded")
	}

	// New description rating must be "Good" (4 or 5)
	if in.RatingDescription < 4 {
		return nil, werr.PermissionDenied("modifications are only allowed to upgrade ratings to a Positive (4 or 5 stars) review")
	}

	// Save update
	var contentPtr *string
	if in.Content != "" {
		contentPtr = &in.Content
	}
	updated, err := s.q.UpdateReview(ctx, db.UpdateReviewParams{
		ID:                in.ReviewID,
		RatingDescription: in.RatingDescription,
		RatingService:     in.RatingService,
		RatingDelivery:    in.RatingDelivery,
		Content:           contentPtr,
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	s.publishReviewEvent("wemall.review.updated", &updated)

	return &updated, nil
}

func (s *ReviewService) DeleteReview(ctx context.Context, id, buyerID uuid.UUID) error {
	review, err := s.q.GetReview(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return werr.NotFound("review not found")
		}
		return werr.Internal(err)
	}

	if review.BuyerID != buyerID {
		return werr.PermissionDenied("only the original reviewer can delete this review")
	}

	// Goodwill window: Must be within 30 days of creation
	if time.Since(review.CreatedAt.Time) > 30*24*time.Hour {
		return werr.PermissionDenied("deletion window of 30 days has expired")
	}

	_, err = s.q.DeleteReview(ctx, id)
	if err != nil {
		return werr.Internal(err)
	}

	s.publishReviewEvent("wemall.review.updated", &review)

	return nil
}

type CreateSellerReplyInput struct {
	ReviewID  uuid.UUID
	SellerID  uuid.UUID
	ReplyType string // "initial" | "append"
	Content   string
}

func (s *ReviewService) CreateSellerReply(ctx context.Context, in CreateSellerReplyInput) (*db.SellerReply, error) {
	if in.Content == "" {
		return nil, werr.InvalidArgument("content is required for reply")
	}
	if in.ReplyType != "initial" && in.ReplyType != "append" {
		return nil, werr.InvalidArgument("reply_type must be either 'initial' or 'append'")
	}

	review, err := s.q.GetReview(ctx, in.ReviewID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, werr.NotFound("review not found")
		}
		return nil, werr.Internal(err)
	}

	if review.SellerID != in.SellerID {
		return nil, werr.PermissionDenied("only the merchant seller can reply to this review")
	}

	reply, err := s.q.CreateSellerReply(ctx, db.CreateSellerReplyParams{
		ReviewID:  in.ReviewID,
		ReplyType: in.ReplyType,
		Content:   in.Content,
	})
	if err != nil {
		return nil, werr.Internal(err)
	}

	return &reply, nil
}

func (s *ReviewService) GetReview(ctx context.Context, id uuid.UUID) (*db.Review, []db.ReviewMedium, *db.AppendReview, []db.AppendReviewMedium, []db.SellerReply, error) {
	review, err := s.q.GetReview(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil, nil, nil, werr.NotFound("review not found")
		}
		return nil, nil, nil, nil, nil, werr.Internal(err)
	}

	media, _ := s.q.GetReviewMedia(ctx, review.ID)
	replies, _ := s.q.GetSellerReplies(ctx, review.ID)

	var appReview *db.AppendReview
	var appMedia []db.AppendReviewMedium

	ar, err := s.q.GetAppendReview(ctx, review.ID)
	if err == nil {
		appReview = &ar
		appMedia, _ = s.q.GetAppendReviewMedia(ctx, ar.ID)
	}

	return &review, media, appReview, appMedia, replies, nil
}

func (s *ReviewService) ListProductReviews(ctx context.Context, productID uuid.UUID, filterType string, page, limit int32) ([]db.Review, int32, error) {
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	total, err := s.q.CountProductReviews(ctx, db.CountProductReviewsParams{
		ProductID: productID,
		Column2:   filterType,
	})
	if err != nil {
		return nil, 0, werr.Internal(err)
	}

	list, err := s.q.ListProductReviews(ctx, db.ListProductReviewsParams{
		ProductID: productID,
		Column2:   filterType,
		Offset:    offset,
		Limit:     limit,
	})
	if err != nil {
		return nil, 0, werr.Internal(err)
	}

	return list, int32(total), nil
}

func (s *ReviewService) ListSellerReviews(ctx context.Context, sellerID uuid.UUID, page, limit int32) ([]db.Review, int32, error) {
	if limit <= 0 {
		limit = 10
	}
	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	total, err := s.q.CountSellerReviews(ctx, sellerID)
	if err != nil {
		return nil, 0, werr.Internal(err)
	}

	list, err := s.q.ListSellerReviews(ctx, db.ListSellerReviewsParams{
		SellerID: sellerID,
		Offset:   offset,
		Limit:    limit,
	})
	if err != nil {
		return nil, 0, werr.Internal(err)
	}

	return list, int32(total), nil
}

func (s *ReviewService) GetProductRatingStats(ctx context.Context, productID uuid.UUID) (float64, int32, int32, int32, int32, int32, int32, []string, error) {
	stats, err := s.q.GetProductRatingStats(ctx, productID)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, 0, nil, werr.Internal(err)
	}

	appends, _ := s.q.GetProductAppendCount(ctx, productID)

	// In a real NLP app, we would query the top tags from a separate table.
	// For now, we mock some tags based on the product stats.
	mockTags := []string{"good quality", "fast delivery"}
	if stats.BadCount > 0 {
		mockTags = append(mockTags, "slow shipping")
	}

	return stats.AvgRating, stats.TotalReviews, stats.GoodCount, stats.NeutralCount, stats.BadCount, stats.HasMediaCount, appends, mockTags, nil
}

func (s *ReviewService) GetSellerDSR(ctx context.Context, sellerID uuid.UUID) (float64, float64, float64, int32, error) {
	dsr, err := s.q.GetSellerDSR(ctx, sellerID)
	if err != nil {
		return 0, 0, 0, 0, werr.Internal(err)
	}
	return dsr.AvgDescription, dsr.AvgService, dsr.AvgDelivery, dsr.ReputationScore, nil
}

// ── NATS Publishers ───────────────────────────────────────────────────────────

func (s *ReviewService) publishReviewEvent(subject string, review *db.Review) {
	if s.nc == nil {
		return
	}
	var isAnonymous bool
	if review.IsAnonymous != nil {
		isAnonymous = *review.IsAnonymous
	}
	var content string
	if review.Content != nil {
		content = *review.Content
	}
	event := map[string]interface{}{
		"review_id":          review.ID.String(),
		"product_id":         review.ProductID.String(),
		"seller_id":          review.SellerID.String(),
		"rating_description": review.RatingDescription,
		"rating_service":     review.RatingService,
		"rating_delivery":    review.RatingDelivery,
		"review_type":        review.ReviewType,
		"is_anonymous":       isAnonymous,
		"content":            content,
	}
	eventBytes, _ := json.Marshal(event)
	_ = s.nc.Publish(subject, eventBytes)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func generateMockNLPTags(content string) []string {
	tags := []string{}
	lower := strings.ToLower(content)
	if strings.Contains(lower, "good") || strings.Contains(lower, "great") || strings.Contains(lower, "excellent") || strings.Contains(lower, "perfect") {
		tags = append(tags, "good quality")
	}
	if strings.Contains(lower, "slow") || strings.Contains(lower, "late") || strings.Contains(lower, "delay") {
		tags = append(tags, "slow shipping")
	}
	if strings.Contains(lower, "fast") || strings.Contains(lower, "quick") || strings.Contains(lower, "rapid") {
		tags = append(tags, "fast delivery")
	}
	if strings.Contains(lower, "size") || strings.Contains(lower, "fit") || strings.Contains(lower, "small") || strings.Contains(lower, "large") {
		tags = append(tags, "accurate size")
	}
	if strings.Contains(lower, "bad") || strings.Contains(lower, "poor") || strings.Contains(lower, "cheap") {
		tags = append(tags, "poor quality")
	}
	if strings.Contains(lower, "recommend") {
		tags = append(tags, "highly recommended")
	}
	if len(tags) == 0 && content != "" {
		tags = append(tags, "general review")
	}
	return tags
}
