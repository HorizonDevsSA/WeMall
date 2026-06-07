package resolver

import (
	"context"
	"strconv"

	"github.com/wemall/api-gateway/internal/graph/gqlerrors"
	"github.com/wemall/api-gateway/internal/graph/model"
	"github.com/wemall/api-gateway/internal/middleware"
	productv1 "github.com/wemall/gen/product/v1"
	reviewv1 "github.com/wemall/gen/review/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	userv1 "github.com/wemall/gen/user/v1"
)

// ── Mutation Review Resolvers ───────────────────────────────────────────────

func (r *mutationResolver) CreateReview(ctx context.Context, input model.CreateReviewInput) (*model.Review, error) {
	buyerID, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	// Fetch product details to get the seller ID
	productResp, err := r.Clients.Product.GetProduct(ctx, &productv1.GetProductRequest{
		Id: input.ProductID,
	})
	if err != nil {
		return nil, err
	}

	resp, err := r.Clients.Review.CreateReview(ctx, &reviewv1.CreateReviewRequest{
		OrderId:           input.OrderID,
		BuyerId:           buyerID,
		SellerId:          productResp.SellerId,
		ProductId:         input.ProductID,
		VariantId:         input.VariantID,
		RatingDescription: int32(input.RatingDescription),
		RatingService:     int32(input.RatingService),
		RatingDelivery:    int32(input.RatingDelivery),
		Content:           derefStr(input.Content),
		IsAnonymous:       derefBool(input.IsAnonymous),
		MediaUrls:         input.MediaUrls,
	})
	if err != nil {
		return nil, err
	}

	review := mapReview(resp)

	// Fetch buyer info to populate name/avatar
	buyer, err := r.Clients.User.GetUser(ctx, &userv1.GetUserRequest{Id: buyerID})
	if err == nil {
		review.BuyerName = buyer.FullName
		review.BuyerAvatar = strPtr(buyer.AvatarUrl)
	}

	return review, nil
}

func (r *mutationResolver) AppendReview(ctx context.Context, input model.AppendReviewInput) (*model.AppendReview, error) {
	buyerID, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Review.AppendReview(ctx, &reviewv1.AppendReviewRequest{
		ReviewId:  input.ReviewID,
		BuyerId:   buyerID,
		Content:   input.Content,
		MediaUrls: input.MediaUrls,
	})
	if err != nil {
		return nil, err
	}

	return &model.AppendReview{
		ID:        resp.Id,
		Content:   input.Content,
		HasMedia:  len(input.MediaUrls) > 0,
		Media:     mapMediaUrlsToReviewMedia(resp.Id, input.MediaUrls),
		CreatedAt: resp.CreatedAt.AsTime(),
	}, nil
}

func (r *mutationResolver) UpdateReview(ctx context.Context, input model.UpdateReviewInput) (*model.Review, error) {
	buyerID, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Review.UpdateReview(ctx, &reviewv1.UpdateReviewRequest{
		ReviewId:          input.ReviewID,
		BuyerId:           buyerID,
		RatingDescription: int32(input.RatingDescription),
		RatingService:     int32(input.RatingService),
		RatingDelivery:    int32(input.RatingDelivery),
		Content:           derefStr(input.Content),
	})
	if err != nil {
		return nil, err
	}

	review := mapReview(resp)

	// Fetch buyer info
	buyer, err := r.Clients.User.GetUser(ctx, &userv1.GetUserRequest{Id: buyerID})
	if err == nil {
		review.BuyerName = buyer.FullName
		review.BuyerAvatar = strPtr(buyer.AvatarUrl)
	}

	return review, nil
}

func (r *mutationResolver) DeleteReview(ctx context.Context, reviewID string) (bool, error) {
	buyerID, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return false, gqlerrors.Unauthenticated("authentication required")
	}

	_, err := r.Clients.Review.DeleteReview(ctx, &reviewv1.DeleteReviewRequest{
		ReviewId: reviewID,
		BuyerId:  buyerID,
	})
	return err == nil, err
}

func (r *mutationResolver) CreateSellerReply(ctx context.Context, input model.SellerReplyInput) (*model.SellerReply, error) {
	sellerUserID, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	// Fetch seller store to get seller ID
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: sellerUserID})
	if err != nil {
		return nil, err
	}

	resp, err := r.Clients.Review.CreateSellerReply(ctx, &reviewv1.CreateSellerReplyRequest{
		ReviewId:  input.ReviewID,
		SellerId:  sellerStore.Id,
		ReplyType: input.ReplyType,
		Content:   input.Content,
	})
	if err != nil {
		return nil, err
	}

	return mapSellerReply(resp), nil
}

// ── Product Review Resolvers ───────────────────────────────────────────────

func (r *productResolver) Reviews(ctx context.Context, obj *model.Product, filter *model.ReviewFilterType, page *int, limit *int) (*model.ProductReviewConnection, error) {
	var fType string
	if filter != nil {
		fType = string(*filter)
	} else {
		fType = "ALL"
	}

	resp, err := r.Clients.Review.ListProductReviews(ctx, &reviewv1.ListProductReviewsRequest{
		ProductId:  obj.ID,
		FilterType: fType,
		Page:       int32(derefInt(page, 1)),
		Limit:      int32(derefInt(limit, 10)),
	})
	if err != nil {
		return nil, err
	}

	edges := make([]*model.Review, len(resp.Reviews))
	var buyerIDs []string
	for i, rev := range resp.Reviews {
		edges[i] = mapReview(rev)
		buyerIDs = append(buyerIDs, rev.BuyerId)
	}

	// Batch fetch buyer profiles to avoid N+1 query problem
	if len(buyerIDs) > 0 {
		userBatch, err := r.Clients.User.GetUserBatch(ctx, &userv1.GetUserBatchRequest{Ids: buyerIDs})
		if err == nil && userBatch != nil {
			usersMap := userBatch.GetUsers()
			for _, edge := range edges {
				if user, exists := usersMap[edge.BuyerID]; exists {
					edge.BuyerName = user.FullName
					edge.BuyerAvatar = strPtr(user.AvatarUrl)
				}
			}
		}
	}

	return &model.ProductReviewConnection{
		Edges:      edges,
		TotalCount: int(resp.Total),
	}, nil
}

func (r *productResolver) ReviewStats(ctx context.Context, obj *model.Product) (*model.ProductReviewStats, error) {
	resp, err := r.Clients.Review.GetProductRatingStats(ctx, &reviewv1.GetProductRatingStatsRequest{
		ProductId: obj.ID,
	})
	if err != nil {
		return nil, err
	}

	return mapProductReviewStats(resp), nil
}

// ── Seller Review Resolvers ──────────────────────────────────────────────────

func (r *sellerResolver) Dsr(ctx context.Context, obj *model.Seller) (*model.SellerDsr, error) {
	resp, err := r.Clients.Review.GetSellerDSR(ctx, &reviewv1.GetSellerDSRRequest{
		SellerId: obj.ID,
	})
	if err != nil {
		return nil, err
	}

	return mapSellerDsr(resp), nil
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func mapMediaUrlsToReviewMedia(reviewID string, urls []string) []*model.ReviewMedia {
	media := make([]*model.ReviewMedia, len(urls))
	for i, url := range urls {
		media[i] = &model.ReviewMedia{
			ID:        reviewID + "_media_" + strconv.Itoa(i),
			MediaURL:  url,
			MediaType: "image",
		}
	}
	return media
}
