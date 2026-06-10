package handler

import (
	"context"

	"google.golang.org/protobuf/types/known/emptypb"

	promotionv1 "github.com/wemall/gen/promotion/v1"
	"github.com/wemall/promotion-service/internal/service"
)

type PromotionHandler struct {
	promotionv1.UnimplementedPromotionServiceServer
	svc *service.PromotionService
}

func NewPromotionHandler(svc *service.PromotionService) *PromotionHandler {
	return &PromotionHandler{svc: svc}
}

func (h *PromotionHandler) ValidateCoupon(ctx context.Context, req *promotionv1.ValidateCouponRequest) (*promotionv1.ValidateCouponResponse, error) {
	return h.svc.ValidateCoupon(ctx, req)
}

func (h *PromotionHandler) ApplyCoupon(ctx context.Context, req *promotionv1.ApplyCouponRequest) (*promotionv1.ApplyCouponResponse, error) {
	return h.svc.ApplyCoupon(ctx, req)
}

func (h *PromotionHandler) CreateCoupon(ctx context.Context, req *promotionv1.CreateCouponRequest) (*promotionv1.Coupon, error) {
	return h.svc.CreateCoupon(ctx, req)
}

func (h *PromotionHandler) ListCoupons(ctx context.Context, req *promotionv1.ListCouponsRequest) (*promotionv1.ListCouponsResponse, error) {
	return h.svc.ListCoupons(ctx, req)
}

func (h *PromotionHandler) CreateFlashSale(ctx context.Context, req *promotionv1.CreateFlashSaleRequest) (*promotionv1.FlashSale, error) {
	return h.svc.CreateFlashSale(ctx, req)
}

func (h *PromotionHandler) AddFlashSaleItem(ctx context.Context, req *promotionv1.AddFlashSaleItemRequest) (*promotionv1.FlashSaleItem, error) {
	return h.svc.AddFlashSaleItem(ctx, req)
}

func (h *PromotionHandler) ListActiveFlashSales(ctx context.Context, req *emptypb.Empty) (*promotionv1.ListActiveFlashSalesResponse, error) {
	return h.svc.ListActiveFlashSales(ctx)
}
