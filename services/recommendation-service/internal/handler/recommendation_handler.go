package handler

import (
	"context"

	recommendationv1 "github.com/wemall/gen/recommendation/v1"
	"github.com/wemall/recommendation-service/internal/service"
)

type RecommendationHandler struct {
	recommendationv1.UnimplementedRecommendationServiceServer
	svc *service.RecommendationService
}

func NewRecommendationHandler(svc *service.RecommendationService) *RecommendationHandler {
	return &RecommendationHandler{svc: svc}
}

func (h *RecommendationHandler) GetFrequentlyBoughtTogether(ctx context.Context, req *recommendationv1.GetFrequentlyBoughtTogetherRequest) (*recommendationv1.GetFrequentlyBoughtTogetherResponse, error) {
	return h.svc.GetFrequentlyBoughtTogether(ctx, req)
}

func (h *RecommendationHandler) GetPersonalizedRecommendations(ctx context.Context, req *recommendationv1.GetPersonalizedRecommendationsRequest) (*recommendationv1.GetPersonalizedRecommendationsResponse, error) {
	return h.svc.GetPersonalizedRecommendations(ctx, req)
}

func (h *RecommendationHandler) RecordProductView(ctx context.Context, req *recommendationv1.RecordProductViewRequest) (*recommendationv1.RecordProductViewResponse, error) {
	return h.svc.RecordProductView(ctx, req)
}
