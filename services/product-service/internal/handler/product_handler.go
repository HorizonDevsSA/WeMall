package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	productv1 "github.com/wemall/gen/product/v1"
	"github.com/wemall/product-service/internal/service"
)

type ProductHandler struct {
	productv1.UnimplementedProductServiceServer
	productSvc *service.ProductService
}

func NewProductHandler(productSvc *service.ProductService) *ProductHandler {
	return &ProductHandler{
		productSvc: productSvc,
	}
}

func (h *ProductHandler) ListCategories(ctx context.Context, req *productv1.ListCategoriesRequest) (*productv1.ListCategoriesResponse, error) {
	lang := "en"
	if req.Language != "" {
		lang = req.Language
	}

	roots, err := h.productSvc.ListCategories(ctx, lang)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list categories")
	}

	return &productv1.ListCategoriesResponse{
		Categories: roots,
	}, nil
}

func (h *ProductHandler) GetCategory(ctx context.Context, req *productv1.GetCategoryRequest) (*productv1.Category, error) {
	lang := "en"
	if req.Language != "" {
		lang = req.Language
	}

	cat, err := h.productSvc.GetCategory(ctx, req.Slug, lang)
	if err != nil {
		return nil, status.Error(codes.NotFound, "category not found")
	}

	return cat, nil
}

func (h *ProductHandler) ListProducts(ctx context.Context, req *productv1.ListProductsRequest) (*productv1.ListProductsResponse, error) {
	products, total, nextToken, err := h.productSvc.ListProducts(ctx, req)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list products")
	}

	return &productv1.ListProductsResponse{
		Products:      products,
		NextPageToken: nextToken,
		Total:         total,
	}, nil
}

func (h *ProductHandler) GetProduct(ctx context.Context, req *productv1.GetProductRequest) (*productv1.Product, error) {
	lang := "en"
	if req.Language != "" {
		lang = req.Language
	}

	prod, err := h.productSvc.GetProduct(ctx, req.Id, req.Slug, lang)
	if err != nil {
		return nil, status.Error(codes.NotFound, "product not found")
	}

	return prod, nil
}

func (h *ProductHandler) GetProductBatch(ctx context.Context, req *productv1.GetProductBatchRequest) (*productv1.GetProductBatchResponse, error) {
	lang := "en"
	if req.Language != "" {
		lang = req.Language
	}

	batch, err := h.productSvc.GetProductBatch(ctx, req.Ids, lang)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch batch")
	}

	return &productv1.GetProductBatchResponse{
		Products: batch,
	}, nil
}

func (h *ProductHandler) GetVariantBatch(ctx context.Context, req *productv1.GetVariantBatchRequest) (*productv1.GetVariantBatchResponse, error) {
	batch, err := h.productSvc.GetVariantBatch(ctx, req.Ids)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to fetch variant batch")
	}

	return &productv1.GetVariantBatchResponse{
		Variants: batch,
	}, nil
}

func (h *ProductHandler) CreateProduct(ctx context.Context, req *productv1.CreateProductRequest) (*productv1.Product, error) {
	prod, err := h.productSvc.CreateProduct(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to create product: %v", err)
	}

	return prod, nil
}

func (h *ProductHandler) UpdateProduct(ctx context.Context, req *productv1.UpdateProductRequest) (*productv1.Product, error) {
	prod, err := h.productSvc.UpdateProduct(ctx, req)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to update product")
	}

	return prod, nil
}

func (h *ProductHandler) DeleteProduct(ctx context.Context, req *productv1.DeleteProductRequest) (*emptypb.Empty, error) {
	err := h.productSvc.DeleteProduct(ctx, req.Id, req.SellerId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete product")
	}

	return &emptypb.Empty{}, nil
}

func (h *ProductHandler) ListNearbyProducts(ctx context.Context, req *productv1.ListNearbyProductsRequest) (*productv1.ListNearbyProductsResponse, error) {
	products, total, nextToken, err := h.productSvc.ListNearbyProducts(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list nearby products: %v", err)
	}

	return &productv1.ListNearbyProductsResponse{
		Products:      products,
		NextPageToken: nextToken,
		Total:         total,
	}, nil
}

func (h *ProductHandler) ListRecommendedProducts(ctx context.Context, req *productv1.ListRecommendedProductsRequest) (*productv1.ListRecommendedProductsResponse, error) {
	products, total, nextToken, err := h.productSvc.ListRecommendedProducts(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list recommended products: %v", err)
	}

	return &productv1.ListRecommendedProductsResponse{
		Products:      products,
		NextPageToken: nextToken,
		Total:         total,
	}, nil
}
