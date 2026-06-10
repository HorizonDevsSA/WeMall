package recommendationv1

import (
	"context"

	"google.golang.org/grpc"
)

type ProductSuggestion struct {
	ProductId string  `protobuf:"bytes,1,opt,name=product_id,json=productId,proto3" json:"product_id,omitempty"`
	Score     float64 `protobuf:"fixed64,2,opt,name=score,proto3" json:"score,omitempty"`
}

type GetFrequentlyBoughtTogetherRequest struct {
	ProductId string `protobuf:"bytes,1,opt,name=product_id,json=productId,proto3" json:"product_id,omitempty"`
}

type GetFrequentlyBoughtTogetherResponse struct {
	Suggestions []*ProductSuggestion `protobuf:"bytes,1,rep,name=suggestions,proto3" json:"suggestions,omitempty"`
}

type GetPersonalizedRecommendationsRequest struct {
	BuyerId string `protobuf:"bytes,1,opt,name=buyer_id,json=buyerId,proto3" json:"buyer_id,omitempty"`
}

type GetPersonalizedRecommendationsResponse struct {
	Suggestions []*ProductSuggestion `protobuf:"bytes,1,rep,name=suggestions,proto3" json:"suggestions,omitempty"`
}

type RecordProductViewRequest struct {
	BuyerId   string `protobuf:"bytes,1,opt,name=buyer_id,json=buyerId,proto3" json:"buyer_id,omitempty"`
	ProductId string `protobuf:"bytes,2,opt,name=product_id,json=productId,proto3" json:"product_id,omitempty"`
}

type RecordProductViewResponse struct {
	Success bool `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
}

type RecommendationServiceServer interface {
	GetFrequentlyBoughtTogether(context.Context, *GetFrequentlyBoughtTogetherRequest) (*GetFrequentlyBoughtTogetherResponse, error)
	GetPersonalizedRecommendations(context.Context, *GetPersonalizedRecommendationsRequest) (*GetPersonalizedRecommendationsResponse, error)
	RecordProductView(context.Context, *RecordProductViewRequest) (*RecordProductViewResponse, error)
	mustEmbedUnimplementedRecommendationServiceServer()
}

type UnimplementedRecommendationServiceServer struct{}

func (UnimplementedRecommendationServiceServer) GetFrequentlyBoughtTogether(context.Context, *GetFrequentlyBoughtTogetherRequest) (*GetFrequentlyBoughtTogetherResponse, error) {
	return nil, nil
}
func (UnimplementedRecommendationServiceServer) GetPersonalizedRecommendations(context.Context, *GetPersonalizedRecommendationsRequest) (*GetPersonalizedRecommendationsResponse, error) {
	return nil, nil
}
func (UnimplementedRecommendationServiceServer) RecordProductView(context.Context, *RecordProductViewRequest) (*RecordProductViewResponse, error) {
	return nil, nil
}
func (UnimplementedRecommendationServiceServer) mustEmbedUnimplementedRecommendationServiceServer() {}

func RegisterRecommendationServiceServer(s grpc.ServiceRegistrar, srv RecommendationServiceServer) {
	// mock implementation
}
