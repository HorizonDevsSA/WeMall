package handler

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	reviewv1 "github.com/wemall/gen/review/v1"
	"github.com/wemall/review-service/internal/db"
	"github.com/wemall/review-service/internal/service"
)

type ReviewHandler struct {
	reviewv1.UnimplementedReviewServiceServer
	svc *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

func (h *ReviewHandler) CreateReview(ctx context.Context, req *reviewv1.CreateReviewRequest) (*reviewv1.Review, error) {
	oid, err := uuid.Parse(req.OrderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid order_id")
	}
	bid, err := uuid.Parse(req.BuyerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid buyer_id")
	}
	sid, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid seller_id")
	}
	pid, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	vid, err := uuid.Parse(req.VariantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid variant_id")
	}

	review, media, err := h.svc.CreateReview(ctx, service.CreateReviewInput{
		OrderID:           oid,
		BuyerID:           bid,
		SellerID:          sid,
		ProductID:         pid,
		VariantID:         vid,
		RatingDescription: req.RatingDescription,
		RatingService:     req.RatingService,
		RatingDelivery:    req.RatingDelivery,
		Content:           req.Content,
		IsAnonymous:       req.IsAnonymous,
		MediaURLs:         req.MediaUrls,
	})
	if err != nil {
		return nil, err
	}

	return mapReview(review, media, nil, nil, nil), nil
}

func (h *ReviewHandler) AppendReview(ctx context.Context, req *reviewv1.AppendReviewRequest) (*reviewv1.AppendReviewResponse, error) {
	rid, err := uuid.Parse(req.ReviewId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid review_id")
	}
	bid, err := uuid.Parse(req.BuyerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid buyer_id")
	}

	appRev, _, err := h.svc.AppendReview(ctx, service.AppendReviewInput{
		ReviewID:  rid,
		BuyerID:   bid,
		Content:   req.Content,
		MediaURLs: req.MediaUrls,
	})
	if err != nil {
		return nil, err
	}

	return &reviewv1.AppendReviewResponse{
		Id:        appRev.ID.String(),
		ReviewId:  appRev.ReviewID.String(),
		CreatedAt: timestamppb.New(appRev.CreatedAt.Time),
	}, nil
}

func (h *ReviewHandler) UpdateReview(ctx context.Context, req *reviewv1.UpdateReviewRequest) (*reviewv1.Review, error) {
	rid, err := uuid.Parse(req.ReviewId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid review_id")
	}
	bid, err := uuid.Parse(req.BuyerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid buyer_id")
	}

	review, err := h.svc.UpdateReview(ctx, service.UpdateReviewInput{
		ReviewID:          rid,
		BuyerID:           bid,
		RatingDescription: req.RatingDescription,
		RatingService:     req.RatingService,
		RatingDelivery:    req.RatingDelivery,
		Content:           req.Content,
	})
	if err != nil {
		return nil, err
	}

	// Fetch fresh media/replies
	_, media, appRev, appMedia, replies, err := h.svc.GetReview(ctx, review.ID)
	if err != nil {
		return nil, err
	}

	return mapReview(review, media, appRev, appMedia, replies), nil
}

func (h *ReviewHandler) DeleteReview(ctx context.Context, req *reviewv1.DeleteReviewRequest) (*emptypb.Empty, error) {
	rid, err := uuid.Parse(req.ReviewId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid review_id")
	}
	bid, err := uuid.Parse(req.BuyerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid buyer_id")
	}

	err = h.svc.DeleteReview(ctx, rid, bid)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func (h *ReviewHandler) CreateSellerReply(ctx context.Context, req *reviewv1.CreateSellerReplyRequest) (*reviewv1.SellerReply, error) {
	rid, err := uuid.Parse(req.ReviewId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid review_id")
	}
	sid, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid seller_id")
	}

	reply, err := h.svc.CreateSellerReply(ctx, service.CreateSellerReplyInput{
		ReviewID:  rid,
		SellerID:  sid,
		ReplyType: req.ReplyType,
		Content:   req.Content,
	})
	if err != nil {
		return nil, err
	}

	return &reviewv1.SellerReply{
		Id:        reply.ID.String(),
		ReplyType: reply.ReplyType,
		Content:   reply.Content,
		CreatedAt: timestamppb.New(reply.CreatedAt.Time),
	}, nil
}

func (h *ReviewHandler) GetReview(ctx context.Context, req *reviewv1.GetReviewRequest) (*reviewv1.Review, error) {
	rid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id")
	}

	review, media, appRev, appMedia, replies, err := h.svc.GetReview(ctx, rid)
	if err != nil {
		return nil, err
	}

	return mapReview(review, media, appRev, appMedia, replies), nil
}

func (h *ReviewHandler) ListProductReviews(ctx context.Context, req *reviewv1.ListProductReviewsRequest) (*reviewv1.ListProductReviewsResponse, error) {
	pid, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}

	list, total, err := h.svc.ListProductReviews(ctx, pid, req.FilterType, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}

	reviews := make([]*reviewv1.Review, len(list))
	for i := range list {
		// Fetch media and replies for each in the list
		_, media, appRev, appMedia, replies, _ := h.svc.GetReview(ctx, list[i].ID)
		reviews[i] = mapReview(&list[i], media, appRev, appMedia, replies)
	}

	return &reviewv1.ListProductReviewsResponse{
		Reviews: reviews,
		Total:   total,
	}, nil
}

func (h *ReviewHandler) ListSellerReviews(ctx context.Context, req *reviewv1.ListSellerReviewsRequest) (*reviewv1.ListSellerReviewsResponse, error) {
	sid, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid seller_id")
	}

	list, total, err := h.svc.ListSellerReviews(ctx, sid, req.Page, req.Limit)
	if err != nil {
		return nil, err
	}

	reviews := make([]*reviewv1.Review, len(list))
	for i := range list {
		_, media, appRev, appMedia, replies, _ := h.svc.GetReview(ctx, list[i].ID)
		reviews[i] = mapReview(&list[i], media, appRev, appMedia, replies)
	}

	return &reviewv1.ListSellerReviewsResponse{
		Reviews: reviews,
		Total:   total,
	}, nil
}

func (h *ReviewHandler) GetProductRatingStats(ctx context.Context, req *reviewv1.GetProductRatingStatsRequest) (*reviewv1.ProductRatingStats, error) {
	pid, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}

	avgRating, total, good, neutral, bad, hasMedia, appends, tags, err := h.svc.GetProductRatingStats(ctx, pid)
	if err != nil {
		return nil, err
	}

	return &reviewv1.ProductRatingStats{
		AverageRating: float32(avgRating),
		TotalReviews:  total,
		GoodCount:     good,
		NeutralCount:  neutral,
		BadCount:      bad,
		HasMediaCount: hasMedia,
		AppendCount:   appends,
		TopNlpTags:    tags,
	}, nil
}

func (h *ReviewHandler) GetSellerDSR(ctx context.Context, req *reviewv1.GetSellerDSRRequest) (*reviewv1.SellerDSR, error) {
	sid, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid seller_id")
	}

	avgDesc, avgServ, avgDeliv, reputation, err := h.svc.GetSellerDSR(ctx, sid)
	if err != nil {
		return nil, err
	}

	return &reviewv1.SellerDSR{
		AvgDescription:  float32(avgDesc),
		AvgService:      float32(avgServ),
		AvgDelivery:     float32(avgDeliv),
		ReputationScore: reputation,
	}, nil
}

// ── Mappers ───────────────────────────────────────────────────────────────────

func mapReview(r *db.Review, media []db.ReviewMedium, appRev *db.AppendReview, appMedia []db.AppendReviewMedium, replies []db.SellerReply) *reviewv1.Review {
	if r == nil {
		return nil
	}

	var nlpTags []string
	if len(r.NlpTags) > 0 {
		_ = json.Unmarshal(r.NlpTags, &nlpTags)
	}

	resMedia := make([]*reviewv1.Media, len(media))
	for i, m := range media {
		var sortOrder int32
		if m.SortOrder != nil {
			sortOrder = *m.SortOrder
		}
		resMedia[i] = &reviewv1.Media{
			Id:        m.ID.String(),
			Url:       m.MediaUrl,
			MediaType: m.MediaType,
			SortOrder: sortOrder,
		}
	}

	resReplies := make([]*reviewv1.SellerReply, len(replies))
	for i, rep := range replies {
		resReplies[i] = &reviewv1.SellerReply{
			Id:        rep.ID.String(),
			ReplyType: rep.ReplyType,
			Content:   rep.Content,
			CreatedAt: timestamppb.New(rep.CreatedAt.Time),
		}
	}

	var mappedAppRev *reviewv1.AppendReview
	if appRev != nil {
		mappedAppMedia := make([]*reviewv1.Media, len(appMedia))
		for i, am := range appMedia {
			var sortOrder int32
			if am.SortOrder != nil {
				sortOrder = *am.SortOrder
			}
			mappedAppMedia[i] = &reviewv1.Media{
				Id:        am.ID.String(),
				Url:       am.MediaUrl,
				MediaType: am.MediaType,
				SortOrder: sortOrder,
			}
		}
		var hasMedia bool
		if appRev.HasMedia != nil {
			hasMedia = *appRev.HasMedia
		}
		mappedAppRev = &reviewv1.AppendReview{
			Id:        appRev.ID.String(),
			Content:   appRev.Content,
			HasMedia:  hasMedia,
			Media:     mappedAppMedia,
			CreatedAt: timestamppb.New(appRev.CreatedAt.Time),
		}
	}

	var reviewType string
	if r.ReviewType != nil {
		reviewType = *r.ReviewType
	}
	var content string
	if r.Content != nil {
		content = *r.Content
	}
	var isAnonymous bool
	if r.IsAnonymous != nil {
		isAnonymous = *r.IsAnonymous
	}
	var hasMedia bool
	if r.HasMedia != nil {
		hasMedia = *r.HasMedia
	}
	var isSystemGenerated bool
	if r.IsSystemGenerated != nil {
		isSystemGenerated = *r.IsSystemGenerated
	}

	return &reviewv1.Review{
		Id:                r.ID.String(),
		OrderId:           r.OrderID.String(),
		BuyerId:           r.BuyerID.String(),
		SellerId:          r.SellerID.String(),
		ProductId:         r.ProductID.String(),
		VariantId:         r.VariantID.String(),
		RatingDescription: r.RatingDescription,
		RatingService:     r.RatingService,
		RatingDelivery:    r.RatingDelivery,
		ReviewType:        reviewType,
		Content:           content,
		IsAnonymous:       isAnonymous,
		HasMedia:          hasMedia,
		Media:             resMedia,
		NlpTags:           nlpTags,
		IsSystemGenerated: isSystemGenerated,
		CreatedAt:         timestamppb.New(r.CreatedAt.Time),
		UpdatedAt:         timestamppb.New(r.UpdatedAt.Time),
		AppendReview:      mappedAppRev,
		Replies:           resReplies,
	}
}
