package promotionv1

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type DiscountType int32

const (
	DiscountType_DISCOUNT_TYPE_UNSPECIFIED DiscountType = 0
	DiscountType_DISCOUNT_TYPE_PERCENTAGE    DiscountType = 1
	DiscountType_DISCOUNT_TYPE_FIXED_AMOUNT  DiscountType = 2
)

var DiscountType_name = map[int32]string{
	0: "DISCOUNT_TYPE_UNSPECIFIED",
	1: "DISCOUNT_TYPE_PERCENTAGE",
	2: "DISCOUNT_TYPE_FIXED_AMOUNT",
}
var DiscountType_value = map[string]int32{
	"DISCOUNT_TYPE_UNSPECIFIED":  0,
	"DISCOUNT_TYPE_PERCENTAGE":     1,
	"DISCOUNT_TYPE_FIXED_AMOUNT":   2,
}

func (x DiscountType) String() string {
	return DiscountType_name[int32(x)]
}

type FlashSaleStatus int32

const (
	FlashSaleStatus_FLASH_SALE_STATUS_UNSPECIFIED FlashSaleStatus = 0
	FlashSaleStatus_FLASH_SALE_STATUS_SCHEDULED   FlashSaleStatus = 1
	FlashSaleStatus_FLASH_SALE_STATUS_ACTIVE      FlashSaleStatus = 2
	FlashSaleStatus_FLASH_SALE_STATUS_ENDED       FlashSaleStatus = 3
)

var FlashSaleStatus_name = map[int32]string{
	0: "FLASH_SALE_STATUS_UNSPECIFIED",
	1: "FLASH_SALE_STATUS_SCHEDULED",
	2: "FLASH_SALE_STATUS_ACTIVE",
	3: "FLASH_SALE_STATUS_ENDED",
}
var FlashSaleStatus_value = map[string]int32{
	"FLASH_SALE_STATUS_UNSPECIFIED": 0,
	"FLASH_SALE_STATUS_SCHEDULED":   1,
	"FLASH_SALE_STATUS_ACTIVE":      2,
	"FLASH_SALE_STATUS_ENDED":       3,
}

func (x FlashSaleStatus) String() string {
	return FlashSaleStatus_name[int32(x)]
}

type Coupon struct {
	Id             string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Code           string                 `protobuf:"bytes,2,opt,name=code,proto3" json:"code,omitempty"`
	SellerId       string                 `protobuf:"bytes,3,opt,name=seller_id,json=sellerId,proto3" json:"seller_id,omitempty"`
	DiscountType   DiscountType           `protobuf:"varint,4,opt,name=discount_type,json=discountType,proto3,enum=promotion.v1.DiscountType" json:"discount_type,omitempty"`
	DiscountValue  float64                `protobuf:"fixed64,5,opt,name=discount_value,json=discountValue,proto3" json:"discount_value,omitempty"`
	MinOrderValue  float64                `protobuf:"fixed64,6,opt,name=min_order_value,json=minOrderValue,proto3" json:"min_order_value,omitempty"`
	MaxDiscount    float64                `protobuf:"fixed64,7,opt,name=max_discount,json=maxDiscount,proto3" json:"max_discount,omitempty"`
	StartDate      *timestamppb.Timestamp `protobuf:"bytes,8,opt,name=start_date,json=startDate,proto3" json:"start_date,omitempty"`
	EndDate        *timestamppb.Timestamp `protobuf:"bytes,9,opt,name=end_date,json=endDate,proto3" json:"end_date,omitempty"`
	UsageLimit     int32                  `protobuf:"varint,10,opt,name=usage_limit,json=usageLimit,proto3" json:"usage_limit,omitempty"`
	UsageCount     int32                  `protobuf:"varint,11,opt,name=usage_count,json=usageCount,proto3" json:"usage_count,omitempty"`
}

type FlashSaleItem struct {
	Id            string  `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	ProductId     string  `protobuf:"bytes,2,opt,name=product_id,json=productId,proto3" json:"product_id,omitempty"`
	DiscountPrice float64 `protobuf:"fixed64,3,opt,name=discount_price,json=discountPrice,proto3" json:"discount_price,omitempty"`
	StockLimit    int32   `protobuf:"varint,4,opt,name=stock_limit,json=stockLimit,proto3" json:"stock_limit,omitempty"`
}

type FlashSale struct {
	Id        string                 `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Name      string                 `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	StartTime *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	EndTime   *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=end_time,json=endTime,proto3" json:"end_time,omitempty"`
	Status    FlashSaleStatus        `protobuf:"varint,5,opt,name=status,proto3,enum=promotion.v1.FlashSaleStatus" json:"status,omitempty"`
	Items     []*FlashSaleItem       `protobuf:"bytes,6,rep,name=items,proto3" json:"items,omitempty"`
}

type ValidateCouponRequest struct {
	Code      string  `protobuf:"bytes,1,opt,name=code,proto3" json:"code,omitempty"`
	CartTotal float64 `protobuf:"fixed64,2,opt,name=cart_total,json=cartTotal,proto3" json:"cart_total,omitempty"`
	SellerId  string  `protobuf:"bytes,3,opt,name=seller_id,json=sellerId,proto3" json:"seller_id,omitempty"`
	BuyerId   string  `protobuf:"bytes,4,opt,name=buyer_id,json=buyerId,proto3" json:"buyer_id,omitempty"`
}

type ValidateCouponResponse struct {
	IsValid        bool    `protobuf:"varint,1,opt,name=is_valid,json=isValid,proto3" json:"is_valid,omitempty"`
	DiscountAmount float64 `protobuf:"fixed64,2,opt,name=discount_amount,json=discountAmount,proto3" json:"discount_amount,omitempty"`
	ErrorMessage   string  `protobuf:"bytes,3,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
}

type ApplyCouponRequest struct {
	Code    string `protobuf:"bytes,1,opt,name=code,proto3" json:"code,omitempty"`
	OrderId string `protobuf:"bytes,2,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
	BuyerId string `protobuf:"bytes,3,opt,name=buyer_id,json=buyerId,proto3" json:"buyer_id,omitempty"`
}

type ApplyCouponResponse struct {
	Success bool `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
}

type CreateCouponRequest struct {
	Code          string                 `protobuf:"bytes,1,opt,name=code,proto3" json:"code,omitempty"`
	SellerId      string                 `protobuf:"bytes,2,opt,name=seller_id,json=sellerId,proto3" json:"seller_id,omitempty"`
	DiscountType  DiscountType           `protobuf:"varint,3,opt,name=discount_type,json=discountType,proto3,enum=promotion.v1.DiscountType" json:"discount_type,omitempty"`
	DiscountValue float64                `protobuf:"fixed64,4,opt,name=discount_value,json=discountValue,proto3" json:"discount_value,omitempty"`
	MinOrderValue float64                `protobuf:"fixed64,5,opt,name=min_order_value,json=minOrderValue,proto3" json:"min_order_value,omitempty"`
	MaxDiscount   float64                `protobuf:"fixed64,6,opt,name=max_discount,json=maxDiscount,proto3" json:"max_discount,omitempty"`
	StartDate     *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=start_date,json=startDate,proto3" json:"start_date,omitempty"`
	EndDate       *timestamppb.Timestamp `protobuf:"bytes,8,opt,name=end_date,json=endDate,proto3" json:"end_date,omitempty"`
	UsageLimit    int32                  `protobuf:"varint,9,opt,name=usage_limit,json=usageLimit,proto3" json:"usage_limit,omitempty"`
}

type ListCouponsRequest struct {
	SellerId  string `protobuf:"bytes,1,opt,name=seller_id,json=sellerId,proto3" json:"seller_id,omitempty"`
	PageSize  int32  `protobuf:"varint,2,opt,name=page_size,json=pageSize,proto3" json:"page_size,omitempty"`
	PageToken string `protobuf:"bytes,3,opt,name=page_token,json=pageToken,proto3" json:"page_token,omitempty"`
}

type ListCouponsResponse struct {
	Coupons       []*Coupon `protobuf:"bytes,1,rep,name=coupons,proto3" json:"coupons,omitempty"`
	NextPageToken string    `protobuf:"bytes,2,opt,name=next_page_token,json=nextPageToken,proto3" json:"next_page_token,omitempty"`
}

type CreateFlashSaleRequest struct {
	Name      string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	StartTime *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	EndTime   *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=end_time,json=endTime,proto3" json:"end_time,omitempty"`
}

type AddFlashSaleItemRequest struct {
	FlashSaleId   string  `protobuf:"bytes,1,opt,name=flash_sale_id,json=flashSaleId,proto3" json:"flash_sale_id,omitempty"`
	ProductId     string  `protobuf:"bytes,2,opt,name=product_id,json=productId,proto3" json:"product_id,omitempty"`
	DiscountPrice float64 `protobuf:"fixed64,3,opt,name=discount_price,json=discountPrice,proto3" json:"discount_price,omitempty"`
	StockLimit    int32   `protobuf:"varint,4,opt,name=stock_limit,json=stockLimit,proto3" json:"stock_limit,omitempty"`
}

type ListActiveFlashSalesResponse struct {
	Sales []*FlashSale `protobuf:"bytes,1,rep,name=sales,proto3" json:"sales,omitempty"`
}

type PromotionServiceServer interface {
	ValidateCoupon(context.Context, *ValidateCouponRequest) (*ValidateCouponResponse, error)
	ApplyCoupon(context.Context, *ApplyCouponRequest) (*ApplyCouponResponse, error)
	CreateCoupon(context.Context, *CreateCouponRequest) (*Coupon, error)
	ListCoupons(context.Context, *ListCouponsRequest) (*ListCouponsResponse, error)
	CreateFlashSale(context.Context, *CreateFlashSaleRequest) (*FlashSale, error)
	AddFlashSaleItem(context.Context, *AddFlashSaleItemRequest) (*FlashSaleItem, error)
	ListActiveFlashSales(context.Context, *emptypb.Empty) (*ListActiveFlashSalesResponse, error)
	mustEmbedUnimplementedPromotionServiceServer()
}

type UnimplementedPromotionServiceServer struct{}

func (UnimplementedPromotionServiceServer) ValidateCoupon(context.Context, *ValidateCouponRequest) (*ValidateCouponResponse, error) { return nil, nil }
func (UnimplementedPromotionServiceServer) ApplyCoupon(context.Context, *ApplyCouponRequest) (*ApplyCouponResponse, error) { return nil, nil }
func (UnimplementedPromotionServiceServer) CreateCoupon(context.Context, *CreateCouponRequest) (*Coupon, error) { return nil, nil }
func (UnimplementedPromotionServiceServer) ListCoupons(context.Context, *ListCouponsRequest) (*ListCouponsResponse, error) { return nil, nil }
func (UnimplementedPromotionServiceServer) CreateFlashSale(context.Context, *CreateFlashSaleRequest) (*FlashSale, error) { return nil, nil }
func (UnimplementedPromotionServiceServer) AddFlashSaleItem(context.Context, *AddFlashSaleItemRequest) (*FlashSaleItem, error) { return nil, nil }
func (UnimplementedPromotionServiceServer) ListActiveFlashSales(context.Context, *emptypb.Empty) (*ListActiveFlashSalesResponse, error) { return nil, nil }
func (UnimplementedPromotionServiceServer) mustEmbedUnimplementedPromotionServiceServer() {}

func RegisterPromotionServiceServer(s grpc.ServiceRegistrar, srv PromotionServiceServer) {
	// mock implementation
}
