package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	orderv1 "github.com/wemall/gen/order/v1"
	productv1 "github.com/wemall/gen/product/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	"github.com/wemall/order-service/internal/db"
)



type OrderService struct {
	q             *db.Queries
	pool          *pgxpool.Pool
	productClient productv1.ProductServiceClient
	sellerClient  sellerv1.SellerServiceClient
	nc            *nats.Conn
}

func NewOrderService(q *db.Queries, pool *pgxpool.Pool, pc productv1.ProductServiceClient, sc sellerv1.SellerServiceClient, nc *nats.Conn) *OrderService {
	return &OrderService{q: q, pool: pool, productClient: pc, sellerClient: sc, nc: nc}
}


func (s *OrderService) GetOrder(ctx context.Context, id, userID uuid.UUID) (*orderv1.Order, error) {
	o, err := s.q.GetOrder(ctx, db.GetOrderParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	items, err := s.q.GetOrderItems(ctx, o.ID)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}

	return assembleOrder(&o, items), nil
}

func (s *OrderService) ListOrders(ctx context.Context, userID uuid.UUID, pageSize int32, pageToken string) ([]*orderv1.Order, int32, string, error) {
	limit := int32(20)
	if pageSize > 0 {
		limit = pageSize
	}

	offset := int32(0)
	if pageToken != "" {
		var o int32
		if _, err := fmt.Sscanf(pageToken, "offset_%d", &o); err == nil {
			offset = o
		}
	}

	totalCount, err := s.q.CountOrdersByUser(ctx, userID)
	if err != nil {
		return nil, 0, "", fmt.Errorf("count orders: %w", err)
	}

	list, err := s.q.ListOrders(ctx, db.ListOrdersParams{
		UserID: userID,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, "", fmt.Errorf("list orders: %w", err)
	}

	res := make([]*orderv1.Order, len(list))
	for i := range list {
		items, _ := s.q.GetOrderItems(ctx, list[i].ID)
		res[i] = assembleOrder(&list[i], items)
	}

	nextPageToken := ""
	total := int32(totalCount)
	if offset+limit < total {
		nextPageToken = fmt.Sprintf("offset_%d", offset+limit)
	}

	return res, total, nextPageToken, nil
}

func (s *OrderService) Checkout(ctx context.Context, userID uuid.UUID, input *orderv1.CheckoutInput) (*orderv1.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	// 1. Get Cart
	cart, err := qtx.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	cartItems, err := qtx.GetCartItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}
	if len(cartItems) == 0 {
		return nil, fmt.Errorf("cart is empty")
	}

	// 2. Fetch Variant details from product-service
	variantIDs := make([]string, len(cartItems))
	for i, item := range cartItems {
		variantIDs[i] = item.VariantID.String()
	}

	vResp, err := s.productClient.GetVariantBatch(ctx, &productv1.GetVariantBatchRequest{
		Ids: variantIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("fetch variants: %w", err)
	}

	productIDs := make([]string, 0)
	for _, v := range vResp.Variants {
		productIDs = append(productIDs, v.ProductId)
	}

	pResp, err := s.productClient.GetProductBatch(ctx, &productv1.GetProductBatchRequest{
		Ids:      productIDs,
		Language: "en",
	})
	if err != nil {
		return nil, fmt.Errorf("fetch products: %w", err)
	}

	// 3. Compute Subtotal
	var subtotal float64
	for _, item := range cartItems {
		v, exists := vResp.Variants[item.VariantID.String()]
		if !exists {
			return nil, fmt.Errorf("variant %s no longer available", item.VariantID)
		}
		subtotal += float64(item.Quantity) * v.Price
	}

	// 4. Validate Coupon / Discount
	var discount float64
	if input.CouponCode != "" {
		row, err := qtx.GetCouponWithPromotion(ctx, input.CouponCode)
		if err == nil {
			minOrderVal := numericToFloat64(row.MinOrderValue)
			if !row.MinOrderValue.Valid || subtotal >= minOrderVal {
				if row.MaxUses == nil || row.UsedCount < *row.MaxUses {
					// Apply discount
					val := numericToFloat64(row.Value)
					if row.Type == "percentage" {
						discount = subtotal * (val / 100.0)
					} else { // fixed
						discount = val
					}
					maxDiscount := numericToFloat64(row.MaxDiscount)
					if row.MaxDiscount.Valid && discount > maxDiscount {
						discount = maxDiscount
					}
					// Increment coupon use count
					_ = qtx.IncrementCouponUses(ctx, input.CouponCode)
				}
			}
		}
	}

	shippingFee := 5.00 // flat rate for MVP
	total := subtotal + shippingFee - discount
	if total < 0 {
		total = 0
	}

	// 5. Create Order
	orderNumber := fmt.Sprintf("WM-%d-%s", time.Now().Unix(), uuid.New().String()[:6])
	shippingAddressJSON := structToJSON(input.ShippingAddress)

	couponCodePtr := &input.CouponCode
	if input.CouponCode == "" {
		couponCodePtr = nil
	}
	notesPtr := &input.Notes
	if input.Notes == "" {
		notesPtr = nil
	}
	orderID, err := qtx.CreateOrder(ctx, db.CreateOrderParams{
		OrderNumber:     orderNumber,
		UserID:          userID,
		Subtotal:        float64ToNumeric(subtotal),
		ShippingFee:     float64ToNumeric(shippingFee),
		DiscountAmount:  float64ToNumeric(discount),
		Total:           float64ToNumeric(total),
		ShippingAddress: shippingAddressJSON,
		CouponCode:      couponCodePtr,
		Notes:           notesPtr,
		Currency:        input.Currency,
	})
	if err != nil {
		return nil, fmt.Errorf("create order record: %w", err)
	}

	// Fetch Seller details
	sellerIDs := make([]string, 0)
	sellerIDSet := make(map[string]bool)
	for _, p := range pResp.Products {
		if !sellerIDSet[p.SellerId] {
			sellerIDSet[p.SellerId] = true
			sellerIDs = append(sellerIDs, p.SellerId)
		}
	}

	var sellers map[string]*sellerv1.Seller
	if len(sellerIDs) > 0 {
		sResp, err := s.sellerClient.GetSellerBatch(ctx, &sellerv1.GetSellerBatchRequest{
			Ids: sellerIDs,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch sellers: %w", err)
		}
		sellers = sResp.Sellers
	}

	// 6. Create Order Items & Snapshot
	var orderItems []db.OrderItem
	for _, item := range cartItems {
		v := vResp.Variants[item.VariantID.String()]
		p := pResp.Products[v.ProductId]

		var storeTitle, storeLogo string
		if sel, ok := sellers[p.SellerId]; ok {
			storeTitle = sel.StoreName
			storeLogo = sel.LogoUrl
		}

		// Build snapshot
		snapMap := map[string]interface{}{
			"title":                   p.Title,
			"sku":                     v.Sku,
			"options":                 v.Options.AsMap(),
			"image_url":               v.ImageUrl,
			"product_title":           p.Title,
			"variation":               formatVariation(v.Options),
			"variation_thumbnail":     getVariationThumbnail(v, p),
			"store_title":             storeTitle,
			"store_logo":              storeLogo,
			"product_type":            int32(p.ProductType),
		}
		snapBytes, _ := json.Marshal(snapMap)

		pUID, _ := uuid.Parse(p.Id)
		vUID, _ := uuid.Parse(v.Id)
		sUID, _ := uuid.Parse(p.SellerId)

		unitPriceNumeric := float64ToNumeric(v.Price)
		err = qtx.CreateOrderItem(ctx, db.CreateOrderItemParams{
			OrderID:   orderID,
			VariantID: vUID,
			ProductID: pUID,
			SellerID:  sUID,
			Quantity:  item.Quantity,
			UnitPrice: unitPriceNumeric,
			Snapshot:  snapBytes,
		})
		if err != nil {
			return nil, fmt.Errorf("create order item: %w", err)
		}

		orderItems = append(orderItems, db.OrderItem{
			OrderID:   orderID,
			VariantID: vUID,
			ProductID: pUID,
			SellerID:  sUID,
			Quantity:  item.Quantity,
			UnitPrice: unitPriceNumeric,
			Snapshot:  snapBytes,
			Status:    "pending",
		})
	}


	// 7. Clear Cart
	if err := qtx.ClearCartItems(ctx, cart.ID); err != nil {
		return nil, fmt.Errorf("clear cart: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit checkout: %w", err)
	}

	// 8. Publish NATS Event
	if s.nc != nil {
		event := map[string]interface{}{
			"order_id":     orderID.String(),
			"order_number": orderNumber,
			"user_id":      userID.String(),
			"total":        total,
			"currency":     input.Currency,
			"product_ids":  productIDs,
		}
		eventBytes, _ := json.Marshal(event)
		_ = s.nc.Publish("wemall.order.created", eventBytes)
	}

	// Get fully updated order
	o, err := s.q.GetOrder(ctx, db.GetOrderParams{
		ID:     orderID,
		UserID: userID,
	})
	if err != nil {
		return nil, err
	}

	return assembleOrder(&o, orderItems), nil
}

func (s *OrderService) CancelOrder(ctx context.Context, id, userID uuid.UUID) (*orderv1.Order, error) {
	o, err := s.q.CancelOrder(ctx, db.CancelOrderParams{
		ID:     id,
		UserID: userID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to cancel order: %w", err)
	}

	items, _ := s.q.GetOrderItems(ctx, o.ID)

	// Publish cancelled event to NATS
	if s.nc != nil {
		event := map[string]interface{}{
			"order_id":     o.ID.String(),
			"order_number": o.OrderNumber,
			"user_id":      userID.String(),
		}
		eventBytes, _ := json.Marshal(event)
		_ = s.nc.Publish("wemall.order.cancelled", eventBytes)
	}

	return assembleOrder(&o, items), nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func jsonToStruct(b []byte) *structpb.Struct {
	if len(b) == 0 {
		s, _ := structpb.NewStruct(map[string]interface{}{})
		return s
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		s, _ := structpb.NewStruct(map[string]interface{}{})
		return s
	}
	s, _ := structpb.NewStruct(m)
	return s
}

func structToJSON(s *structpb.Struct) []byte {
	if s == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(s.AsMap())
	if err != nil {
		return []byte("{}")
	}
	return b
}

func getVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func mapOrderStatus(statusStr string) orderv1.OrderStatus {
	switch statusStr {
	case "pending":
		return orderv1.OrderStatus_ORDER_STATUS_PENDING
	case "confirmed":
		return orderv1.OrderStatus_ORDER_STATUS_CONFIRMED
	case "shipped":
		return orderv1.OrderStatus_ORDER_STATUS_SHIPPED
	case "delivered":
		return orderv1.OrderStatus_ORDER_STATUS_DELIVERED
	case "cancelled":
		return orderv1.OrderStatus_ORDER_STATUS_CANCELLED
	case "refunded":
		return orderv1.OrderStatus_ORDER_STATUS_REFUNDED
	default:
		return orderv1.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}

func assembleOrder(o *db.Order, items []db.OrderItem) *orderv1.Order {
	resItems := make([]*orderv1.OrderItem, len(items))
	for i, item := range items {
		var snap map[string]interface{}
		if len(item.Snapshot) > 0 {
			_ = json.Unmarshal(item.Snapshot, &snap)
		}
		if snap == nil {
			snap = make(map[string]interface{})
		}

		productTitle, _ := snap["product_title"].(string)
		if productTitle == "" {
			productTitle, _ = snap["title"].(string)
		}

		variationVal, _ := snap["variation"].(string)
		if variationVal == "" {
			if optsMap, ok := snap["options"].(map[string]interface{}); ok {
				optStruct, _ := structpb.NewStruct(optsMap)
				variationVal = formatVariation(optStruct)
			}
		}

		variationThumbnail, _ := snap["variation_thumbnail"].(string)
		if variationThumbnail == "" {
			variationThumbnail, _ = snap["image_url"].(string)
		}

		storeTitle, _ := snap["store_title"].(string)
		storeLogo, _ := snap["store_logo"].(string)

		var options *structpb.Struct
		if optsMap, ok := snap["options"].(map[string]interface{}); ok {
			options, _ = structpb.NewStruct(optsMap)
		}
		if options == nil {
			options, _ = structpb.NewStruct(map[string]interface{}{})
		}

		var prodTypeVal orderv1.ProductType
		if ptVal, ok := snap["product_type"].(float64); ok {
			prodTypeVal = orderv1.ProductType(int32(ptVal))
		}

		resItems[i] = &orderv1.OrderItem{
			Id:                 item.ID.String(),
			VariantId:          item.VariantID.String(),
			ProductId:          item.ProductID.String(),
			SellerId:           item.SellerID.String(),
			Quantity:           item.Quantity,
			UnitPrice:          numericToFloat64(item.UnitPrice),
			Snapshot:           jsonToStruct(item.Snapshot),
			Status:             mapOrderStatus(item.Status),
			ProductTitle:       productTitle,
			Variation:          variationVal,
			VariationThumbnail: variationThumbnail,
			StoreTitle:         storeTitle,
			StoreLogo:          storeLogo,
			Options:            options,
			ProductType:        prodTypeVal,
		}
	}


	return &orderv1.Order{
		Id:              o.ID.String(),
		OrderNumber:     o.OrderNumber,
		UserId:          o.UserID.String(),
		Status:          mapOrderStatus(o.Status),
		Subtotal:        numericToFloat64(o.Subtotal),
		ShippingFee:     numericToFloat64(o.ShippingFee),
		DiscountAmount:  numericToFloat64(o.DiscountAmount),
		Total:           numericToFloat64(o.Total),
		ShippingAddress: jsonToStruct(o.ShippingAddress),
		Items:           resItems,
		CouponCode:      getVal(o.CouponCode),
		Notes:           getVal(o.Notes),
		Currency:        o.Currency,
		CreatedAt:       timestamppb.New(o.CreatedAt),
		UpdatedAt:       timestamppb.New(o.UpdatedAt),
	}
}

