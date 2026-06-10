package service

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	recommendationv1 "github.com/wemall/gen/recommendation/v1"
	"github.com/wemall/recommendation-service/internal/db"
)

type RecommendationService struct {
	queries *db.Queries
	db      *pgxpool.Pool
}

func NewRecommendationService(queries *db.Queries, dbPool *pgxpool.Pool) *RecommendationService {
	return &RecommendationService{queries: queries, db: dbPool}
}

func (s *RecommendationService) RecordProductView(ctx context.Context, req *recommendationv1.RecordProductViewRequest) (*recommendationv1.RecordProductViewResponse, error) {
	err := s.queries.UpsertProductView(ctx, db.UpsertProductViewParams{
		BuyerID:   req.BuyerId,
		ProductID: req.ProductId,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to record product view")
		return &recommendationv1.RecordProductViewResponse{Success: false}, fmt.Errorf("record view: %w", err)
	}

	return &recommendationv1.RecordProductViewResponse{Success: true}, nil
}

func (s *RecommendationService) GetFrequentlyBoughtTogether(ctx context.Context, req *recommendationv1.GetFrequentlyBoughtTogetherRequest) (*recommendationv1.GetFrequentlyBoughtTogetherResponse, error) {
	rows, err := s.queries.GetFrequentlyBoughtTogether(ctx, db.GetFrequentlyBoughtTogetherParams{
		ProductAID: req.ProductId,
		Limit:      10,
	})
	if err != nil {
		return nil, fmt.Errorf("get frequently bought together: %w", err)
	}

	var suggestions []*recommendationv1.ProductSuggestion
	for _, row := range rows {
		suggestions = append(suggestions, &recommendationv1.ProductSuggestion{
			ProductId: row.ProductID,
			Score:     float64(row.Score),
		})
	}

	return &recommendationv1.GetFrequentlyBoughtTogetherResponse{
		Suggestions: suggestions,
	}, nil
}

func (s *RecommendationService) GetPersonalizedRecommendations(ctx context.Context, req *recommendationv1.GetPersonalizedRecommendationsRequest) (*recommendationv1.GetPersonalizedRecommendationsResponse, error) {
	// 1. Get recent product views
	recentViews, err := s.queries.GetRecentProductViews(ctx, db.GetRecentProductViewsParams{
		BuyerID: req.BuyerId,
		Limit:   5, // Look at their 5 most recent views
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get recent views")
		// Continue to fallback
	}

	var suggestions []*recommendationv1.ProductSuggestion
	suggestionMap := make(map[string]float64)

	// 2. Collaborative Filtering based on Co-Purchases of viewed items
	if len(recentViews) > 0 {
		for _, viewedProductID := range recentViews {
			rows, _ := s.queries.GetFrequentlyBoughtTogether(ctx, db.GetFrequentlyBoughtTogetherParams{
				ProductAID: viewedProductID,
				Limit:      5,
			})
			for _, row := range rows {
				// Don't recommend a product they already viewed
				alreadyViewed := false
				for _, v := range recentViews {
					if v == row.ProductID {
						alreadyViewed = true
						break
					}
				}
				if !alreadyViewed {
					suggestionMap[row.ProductID] += float64(row.Score)
				}
			}
		}
	}

	// 3. Assemble and sort suggestions
	for pid, score := range suggestionMap {
		suggestions = append(suggestions, &recommendationv1.ProductSuggestion{
			ProductId: pid,
			Score:     score,
		})
	}

	// 4. Fallback to global top products if we don't have enough suggestions
	if len(suggestions) < 10 {
		topGlobal, err := s.queries.GetTopProductsGlobally(ctx, 20)
		if err == nil {
			for _, row := range topGlobal {
				if _, exists := suggestionMap[row.ProductID]; !exists {
					suggestions = append(suggestions, &recommendationv1.ProductSuggestion{
						ProductId: row.ProductID,
						Score:     float64(row.Score) * 0.1, // Lower weight for global fallback
					})
					if len(suggestions) >= 10 {
						break
					}
				}
			}
		}
	}

	// Basic sort descending by score
	for i := 0; i < len(suggestions); i++ {
		for j := i + 1; j < len(suggestions); j++ {
			if suggestions[i].Score < suggestions[j].Score {
				suggestions[i], suggestions[j] = suggestions[j], suggestions[i]
			}
		}
	}

	// Cap to 10
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}

	return &recommendationv1.GetPersonalizedRecommendationsResponse{
		Suggestions: suggestions,
	}, nil
}
