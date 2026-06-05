package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/types/known/structpb"

	orderv1 "github.com/wemall/gen/order/v1"
	productv1 "github.com/wemall/gen/product/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	"github.com/wemall/order-service/internal/db"
)

func float64ToNumeric(f float64) pgtype.Numeric {
	var n pgtype.Numeric
	_ = n.Scan(fmt.Sprintf("%.2f", f))
	return n
}

func numericToFloat64(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	val, err := n.Value()
	if err != nil {
		return 0
	}
	if s, ok := val.(string); ok {
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	}
	return 0
}

type CartService struct {
	q             *db.Queries
	productClient productv1.ProductServiceClient
	sellerClient  sellerv1.SellerServiceClient
}

func NewCartService(q *db.Queries, pc productv1.ProductServiceClient, sc sellerv1.SellerServiceClient) *CartService {
	return &CartService{q: q, productClient: pc, sellerClient: sc}
}

func (s *CartService) GetCart(ctx context.Context, userID uuid.UUID) (*orderv1.Cart, error) {
	cart, err := s.q.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get or create cart: %w", err)
	}
	return s.assembleCart(ctx, cart)
}

func (s *CartService) AddToCart(ctx context.Context, userID uuid.UUID, variantID uuid.UUID, quantity int32) (*orderv1.Cart, error) {
	// Call product-service gRPC to get variant details
	resp, err := s.productClient.GetVariantBatch(ctx, &productv1.GetVariantBatchRequest{
		Ids: []string{variantID.String()},
	})
	if err != nil {
		return nil, fmt.Errorf("get variant from product-service: %w", err)
	}

	v, exists := resp.Variants[variantID.String()]
	if !exists {
		return nil, fmt.Errorf("variant %s not found", variantID)
	}

	cart, err := s.q.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	vUID, _ := uuid.Parse(v.Id)
	pUID, _ := uuid.Parse(v.ProductId)
	err = s.q.AddToCartItem(ctx, db.AddToCartItemParams{
		CartID:    cart.ID,
		VariantID: vUID,
		ProductID: pUID,
		Quantity:  quantity,
		UnitPrice: float64ToNumeric(v.Price),
	})
	if err != nil {
		return nil, fmt.Errorf("add item to cart: %w", err)
	}

	return s.assembleCart(ctx, cart)
}

func (s *CartService) UpdateCartItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID, quantity int32) (*orderv1.Cart, error) {
	cart, err := s.q.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	err = s.q.UpdateCartItemQuantity(ctx, db.UpdateCartItemQuantityParams{
		CartID:   cart.ID,
		ID:       itemID,
		Quantity: quantity,
	})
	if err != nil {
		return nil, fmt.Errorf("update cart item quantity: %w", err)
	}

	return s.assembleCart(ctx, cart)
}

func (s *CartService) RemoveCartItem(ctx context.Context, userID uuid.UUID, itemID uuid.UUID) (*orderv1.Cart, error) {
	cart, err := s.q.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	err = s.q.RemoveCartItem(ctx, db.RemoveCartItemParams{
		CartID: cart.ID,
		ID:     itemID,
	})
	if err != nil {
		return nil, fmt.Errorf("remove cart item: %w", err)
	}

	return s.assembleCart(ctx, cart)
}

func (s *CartService) ClearCart(ctx context.Context, userID uuid.UUID) (*orderv1.Cart, error) {
	cart, err := s.q.GetOrCreateCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	err = s.q.ClearCartItems(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("clear cart: %w", err)
	}

	return s.assembleCart(ctx, cart)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func (s *CartService) assembleCart(ctx context.Context, cart db.Cart) (*orderv1.Cart, error) {
	items, err := s.q.GetCartItems(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("get cart items: %w", err)
	}

	// 1. Fetch Variant details from product-service
	varVids := make([]string, 0, len(items))
	for _, item := range items {
		varVids = append(varVids, item.VariantID.String())
	}

	var variants map[string]*productv1.ProductVariant
	var products map[string]*productv1.Product
	var sellers map[string]*sellerv1.Seller

	if len(varVids) > 0 {
		vResp, err := s.productClient.GetVariantBatch(ctx, &productv1.GetVariantBatchRequest{Ids: varVids})
		if err != nil {
			return nil, fmt.Errorf("cart assemble get variants: %w", err)
		}
		variants = vResp.Variants

		pIds := make([]string, 0)
		pIdSet := make(map[string]bool)
		for _, v := range variants {
			if !pIdSet[v.ProductId] {
				pIdSet[v.ProductId] = true
				pIds = append(pIds, v.ProductId)
			}
		}

		if len(pIds) > 0 {
			pResp, err := s.productClient.GetProductBatch(ctx, &productv1.GetProductBatchRequest{Ids: pIds, Language: "en"})
			if err != nil {
				return nil, fmt.Errorf("cart assemble get products: %w", err)
			}
			products = pResp.Products

			sIds := make([]string, 0)
			sIdSet := make(map[string]bool)
			for _, p := range products {
				if !sIdSet[p.SellerId] {
					sIdSet[p.SellerId] = true
					sIds = append(sIds, p.SellerId)
				}
			}

			if len(sIds) > 0 {
				sResp, err := s.sellerClient.GetSellerBatch(ctx, &sellerv1.GetSellerBatchRequest{Ids: sIds})
				if err != nil {
					return nil, fmt.Errorf("cart assemble get sellers: %w", err)
				}
				sellers = sResp.Sellers
			}
		}
	}

	resItems := make([]*orderv1.CartItem, len(items))
	var totalQuantity int32
	var subtotal float64

	for i, item := range items {
		unitPrice := numericToFloat64(item.UnitPrice)

		var title, sellerID, storeTitle, storeLogo, variationVal, variationThumbnail string
		var options *structpb.Struct
		var productType orderv1.ProductType

		if v, exists := variants[item.VariantID.String()]; exists {
			variationVal = formatVariation(v.Options)
			options = v.Options

			if p, pExists := products[v.ProductId]; pExists {
				title = p.Title
				sellerID = p.SellerId
				productType = orderv1.ProductType(p.ProductType)
				variationThumbnail = getVariationThumbnail(v, p)

				if sel, sExists := sellers[p.SellerId]; sExists {
					storeTitle = sel.StoreName
					storeLogo = sel.LogoUrl
				}
			}
		}

		if options == nil {
			options, _ = structpb.NewStruct(map[string]interface{}{})
		}

		resItems[i] = &orderv1.CartItem{
			Id:                 item.ID.String(),
			VariantId:          item.VariantID.String(),
			ProductId:          item.ProductID.String(),
			Quantity:           item.Quantity,
			UnitPrice:          unitPrice,
			ProductTitle:       title,
			Variation:          variationVal,
			VariationThumbnail: variationThumbnail,
			SellerId:           sellerID,
			StoreTitle:         storeTitle,
			StoreLogo:          storeLogo,
			Options:            options,
			ProductType:        productType,
		}
		totalQuantity += item.Quantity
		subtotal += float64(item.Quantity) * unitPrice
	}

	return &orderv1.Cart{
		Id:        cart.ID.String(),
		UserId:    cart.UserID.String(),
		Items:     resItems,
		ItemCount: totalQuantity,
		Subtotal:  subtotal,
	}, nil
}

func formatVariation(options *structpb.Struct) string {
	if options == nil {
		return ""
	}
	m := options.AsMap()
	if len(m) == 0 {
		return ""
	}
	var parts []string
	if val, ok := m["color"]; ok {
		parts = append(parts, fmt.Sprintf("%v", val))
	}
	if val, ok := m["size"]; ok {
		parts = append(parts, fmt.Sprintf("%v", val))
	}
	for k, val := range m {
		if k != "color" && k != "size" {
			parts = append(parts, fmt.Sprintf("%s: %v", k, val))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " / ")
}

func getVariationThumbnail(v *productv1.ProductVariant, p *productv1.Product) string {
	if v != nil && v.ImageUrl != "" {
		return v.ImageUrl
	}
	if p != nil && len(p.Images) > 0 {
		for _, img := range p.Images {
			if img.IsPrimary {
				return img.Url
			}
		}
		return p.Images[0].Url
	}
	return ""
}
