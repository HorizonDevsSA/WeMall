package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/timestamppb"

	promotionv1 "github.com/wemall/gen/promotion/v1"
	"github.com/wemall/promotion-service/internal/db"
)

type PromotionService struct {
	queries *db.Queries
	db      *pgxpool.Pool
}

func NewPromotionService(queries *db.Queries, dbPool *pgxpool.Pool) *PromotionService {
	return &PromotionService{queries: queries, db: dbPool}
}

// Convert float64 to pgtype.Numeric
func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

// Convert pgtype.Numeric to float64
func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	val, err := n.Value()
	if err != nil {
		return 0
	}
	switch v := val.(type) {
	case string:
		var f float64
		fmt.Sscanf(v, "%f", &f)
		return f
	case float64:
		return v
	}
	return 0
}

func (s *PromotionService) CreateCoupon(ctx context.Context, req *promotionv1.CreateCouponRequest) (*promotionv1.Coupon, error) {
	sellerId := pgtype.Text{String: req.SellerId, Valid: req.SellerId != ""}
	
	params := db.CreateCouponParams{
		Code:          req.Code,
		SellerID:      sellerId,
		DiscountType:  req.DiscountType.String(),
		DiscountValue: float64ToNumeric(req.DiscountValue),
		MinOrderValue: float64ToNumeric(req.MinOrderValue),
		MaxDiscount:   float64ToNumeric(req.MaxDiscount),
		StartDate:     pgtype.Timestamptz{Time: req.StartDate.AsTime(), Valid: true},
		EndDate:       pgtype.Timestamptz{Time: req.EndDate.AsTime(), Valid: true},
		UsageLimit:    pgtype.Int4{Int32: req.UsageLimit, Valid: true},
	}

	c, err := s.queries.CreateCoupon(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create coupon: %w", err)
	}

	return s.mapCoupon(c), nil
}

func (s *PromotionService) ValidateCoupon(ctx context.Context, req *promotionv1.ValidateCouponRequest) (*promotionv1.ValidateCouponResponse, error) {
	c, err := s.queries.GetCouponByCode(ctx, req.Code)
	if err != nil {
		return &promotionv1.ValidateCouponResponse{
			IsValid:      false,
			ErrorMessage: "Coupon not found",
		}, nil
	}

	if !c.IsActive.Bool {
		return &promotionv1.ValidateCouponResponse{IsValid: false, ErrorMessage: "Coupon is disabled"}, nil
	}

	now := time.Now()
	if now.Before(c.StartDate.Time) {
		return &promotionv1.ValidateCouponResponse{IsValid: false, ErrorMessage: "Coupon is not yet active"}, nil
	}
	if now.After(c.EndDate.Time) {
		return &promotionv1.ValidateCouponResponse{IsValid: false, ErrorMessage: "Coupon has expired"}, nil
	}

	if c.UsageLimit.Int32 > 0 && c.UsageCount.Int32 >= c.UsageLimit.Int32 {
		return &promotionv1.ValidateCouponResponse{IsValid: false, ErrorMessage: "Coupon usage limit reached"}, nil
	}

	if c.SellerID.Valid && c.SellerID.String != "" && c.SellerID.String != req.SellerId {
		return &promotionv1.ValidateCouponResponse{IsValid: false, ErrorMessage: "Coupon not valid for this seller"}, nil
	}

	minOrder := numericToFloat64(c.MinOrderValue)
	if minOrder > 0 && req.CartTotal < minOrder {
		return &promotionv1.ValidateCouponResponse{
			IsValid:      false,
			ErrorMessage: fmt.Sprintf("Minimum order value is %.2f", minOrder),
		}, nil
	}

	var discountAmount float64
	discountValue := numericToFloat64(c.DiscountValue)
	
	if c.DiscountType == promotionv1.DiscountType_DISCOUNT_TYPE_PERCENTAGE.String() {
		discountAmount = req.CartTotal * (discountValue / 100.0)
		maxDiscount := numericToFloat64(c.MaxDiscount)
		if maxDiscount > 0 && discountAmount > maxDiscount {
			discountAmount = maxDiscount
		}
	} else if c.DiscountType == promotionv1.DiscountType_DISCOUNT_TYPE_FIXED_AMOUNT.String() {
		discountAmount = discountValue
		if discountAmount > req.CartTotal {
			discountAmount = req.CartTotal
		}
	}

	// Round to 2 decimal places
	discountAmount = math.Round(discountAmount*100) / 100

	return &promotionv1.ValidateCouponResponse{
		IsValid:        true,
		DiscountAmount: discountAmount,
	}, nil
}

func (s *PromotionService) ApplyCoupon(ctx context.Context, req *promotionv1.ApplyCouponRequest) (*promotionv1.ApplyCouponResponse, error) {
	c, err := s.queries.GetCouponByCode(ctx, req.Code)
	if err != nil {
		return &promotionv1.ApplyCouponResponse{Success: false}, fmt.Errorf("coupon not found: %w", err)
	}

	err = s.queries.IncrementCouponUsage(ctx, c.ID)
	if err != nil {
		return &promotionv1.ApplyCouponResponse{Success: false}, fmt.Errorf("failed to increment usage: %w", err)
	}

	return &promotionv1.ApplyCouponResponse{Success: true}, nil
}

func (s *PromotionService) ListCoupons(ctx context.Context, req *promotionv1.ListCouponsRequest) (*promotionv1.ListCouponsResponse, error) {
	sellerId := pgtype.Text{String: req.SellerId, Valid: req.SellerId != ""}
	dbCoupons, err := s.queries.ListCouponsBySeller(ctx, sellerId)
	if err != nil {
		return nil, fmt.Errorf("list coupons: %w", err)
	}

	var coupons []*promotionv1.Coupon
	for _, c := range dbCoupons {
		coupons = append(coupons, s.mapCoupon(c))
	}

	return &promotionv1.ListCouponsResponse{
		Coupons: coupons,
	}, nil
}

func (s *PromotionService) CreateFlashSale(ctx context.Context, req *promotionv1.CreateFlashSaleRequest) (*promotionv1.FlashSale, error) {
	params := db.CreateFlashSaleParams{
		Name:      req.Name,
		StartTime: pgtype.Timestamptz{Time: req.StartTime.AsTime(), Valid: true},
		EndTime:   pgtype.Timestamptz{Time: req.EndTime.AsTime(), Valid: true},
		Status:    promotionv1.FlashSaleStatus_FLASH_SALE_STATUS_SCHEDULED.String(),
	}

	fs, err := s.queries.CreateFlashSale(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("create flash sale: %w", err)
	}

	return s.mapFlashSale(fs, nil), nil
}

func (s *PromotionService) AddFlashSaleItem(ctx context.Context, req *promotionv1.AddFlashSaleItemRequest) (*promotionv1.FlashSaleItem, error) {
	fsId, err := uuid.Parse(req.FlashSaleId)
	if err != nil {
		return nil, errors.New("invalid flash_sale_id")
	}

	params := db.AddFlashSaleItemParams{
		FlashSaleID:   fsId,
		ProductID:     req.ProductId,
		DiscountPrice: float64ToNumeric(req.DiscountPrice),
		StockLimit:    req.StockLimit,
	}

	item, err := s.queries.AddFlashSaleItem(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("add flash sale item: %w", err)
	}

	return s.mapFlashSaleItem(item), nil
}

func (s *PromotionService) ListActiveFlashSales(ctx context.Context) (*promotionv1.ListActiveFlashSalesResponse, error) {
	dbSales, err := s.queries.ListActiveFlashSales(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active flash sales: %w", err)
	}

	var sales []*promotionv1.FlashSale
	for _, fs := range dbSales {
		items, err := s.queries.GetFlashSaleItems(ctx, fs.ID)
		if err != nil {
			return nil, fmt.Errorf("get flash sale items: %w", err)
		}
		sales = append(sales, s.mapFlashSale(fs, items))
	}

	return &promotionv1.ListActiveFlashSalesResponse{
		Sales: sales,
	}, nil
}

func (s *PromotionService) mapCoupon(c db.Coupon) *promotionv1.Coupon {
	return &promotionv1.Coupon{
		Id:            c.ID.String(),
		Code:          c.Code,
		SellerId:      c.SellerID.String,
		DiscountType:  promotionv1.DiscountType(promotionv1.DiscountType_value[c.DiscountType]),
		DiscountValue: numericToFloat64(c.DiscountValue),
		MinOrderValue: numericToFloat64(c.MinOrderValue),
		MaxDiscount:   numericToFloat64(c.MaxDiscount),
		StartDate:     timestamppb.New(c.StartDate.Time),
		EndDate:       timestamppb.New(c.EndDate.Time),
		UsageLimit:    c.UsageLimit.Int32,
		UsageCount:    c.UsageCount.Int32,
	}
}

func (s *PromotionService) mapFlashSale(fs db.FlashSale, dbItems []db.FlashSaleItem) *promotionv1.FlashSale {
	var items []*promotionv1.FlashSaleItem
	for _, item := range dbItems {
		items = append(items, s.mapFlashSaleItem(item))
	}

	return &promotionv1.FlashSale{
		Id:        fs.ID.String(),
		Name:      fs.Name,
		StartTime: timestamppb.New(fs.StartTime.Time),
		EndTime:   timestamppb.New(fs.EndTime.Time),
		Status:    promotionv1.FlashSaleStatus(promotionv1.FlashSaleStatus_value[fs.Status]),
		Items:     items,
	}
}

func (s *PromotionService) mapFlashSaleItem(item db.FlashSaleItem) *promotionv1.FlashSaleItem {
	return &promotionv1.FlashSaleItem{
		Id:            item.ID.String(),
		ProductId:     item.ProductID,
		DiscountPrice: numericToFloat64(item.DiscountPrice),
		StockLimit:    item.StockLimit,
	}
}
