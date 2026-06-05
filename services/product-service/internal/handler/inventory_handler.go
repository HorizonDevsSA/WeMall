package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	inventoryv1 "github.com/wemall/gen/inventory/v1"
	"github.com/wemall/product-service/internal/service"
)

type InventoryHandler struct {
	inventoryv1.UnimplementedInventoryServiceServer
	productSvc *service.ProductService
}

func NewInventoryHandler(productSvc *service.ProductService) *InventoryHandler {
	return &InventoryHandler{
		productSvc: productSvc,
	}
}

func (h *InventoryHandler) UpsertStock(ctx context.Context, req *inventoryv1.UpsertStockRequest) (*inventoryv1.StockItem, error) {
	if req.VariantId == "" {
		return nil, status.Error(codes.InvalidArgument, "variant_id is required")
	}
	if req.Quantity < 0 {
		return nil, status.Error(codes.InvalidArgument, "quantity cannot be negative")
	}
	item, err := h.productSvc.UpsertStock(ctx, req.VariantId, req.Quantity)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to upsert stock")
	}
	return item, nil
}

func (h *InventoryHandler) GetStock(ctx context.Context, req *inventoryv1.GetStockRequest) (*inventoryv1.StockItem, error) {
	if req.VariantId == "" {
		return nil, status.Error(codes.InvalidArgument, "variant_id is required")
	}
	item, err := h.productSvc.GetStock(ctx, req.VariantId)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get stock")
	}
	return item, nil
}

func (h *InventoryHandler) GetStockBatch(ctx context.Context, req *inventoryv1.GetStockBatchRequest) (*inventoryv1.GetStockBatchResponse, error) {
	stocks, err := h.productSvc.GetStockBatch(ctx, req.VariantIds)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get stock batch")
	}
	return &inventoryv1.GetStockBatchResponse{
		Stocks: stocks,
	}, nil
}
