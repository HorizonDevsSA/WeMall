package resolver

import (
	"context"
	"sync"
	"time"

	"github.com/wemall/api-gateway/internal/graph/model"
	inventoryv1 "github.com/wemall/gen/inventory/v1"
	productv1 "github.com/wemall/gen/product/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
)

// Seller resolves the seller field for a Product by fetching seller data from seller service.
func (r *productResolver) Seller(ctx context.Context, obj *model.Product) (*model.Seller, error) {
	resp, err := r.Clients.Seller.GetSeller(ctx, &sellerv1.GetSellerRequest{
		Id: obj.SellerID,
	})
	if err != nil {
		return nil, err
	}
	return mapSeller(resp), nil
}

// Inventory resolves the inventory field for a ProductVariant.
func (r *productVariantResolver) Inventory(ctx context.Context, obj *model.ProductVariant) (*model.InventoryInfo, error) {
	resp, err := r.Clients.Inventory.GetStock(ctx, &inventoryv1.GetStockRequest{
		VariantId: obj.ID,
	})
	if err != nil {
		// Return zero inventory if stock service is unavailable or variant not found
		return &model.InventoryInfo{
			Quantity:  0,
			UpdatedAt: time.Now(),
		}, nil
	}
	return mapInventoryInfo(resp), nil
}

// ProductWithDetails fetches all related product data in parallel.
func (r *queryResolver) ProductWithDetails(ctx context.Context, id *string, slug *string, language *string) (*model.Product, error) {
	lang := "en"
	if language != nil && *language != "" {
		lang = *language
	}

	// First get the basic product data
	productResp, err := r.Clients.Product.GetProduct(ctx, &productv1.GetProductRequest{
		Id:       derefStr(id),
		Slug:     derefStr(slug),
		Language: lang,
	})
	if err != nil {
		return nil, err
	}

	product := mapProduct(productResp)

	// Fetch seller and inventory data in parallel
	var wg sync.WaitGroup
	var seller *model.Seller
	var sellerErr error
	var inventoryMap map[string]*model.InventoryInfo
	var inventoryErr error

	// Fetch seller data
	wg.Add(1)
	go func() {
		defer wg.Done()
		sellerResp, err := r.Clients.Seller.GetSeller(ctx, &sellerv1.GetSellerRequest{
			Id: product.SellerID,
		})
		if err != nil {
			sellerErr = err
			return
		}
		seller = mapSeller(sellerResp)
	}()

	// Fetch inventory data for all variants
	if len(product.Variants) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			variantIDs := make([]string, len(product.Variants))
			for i, v := range product.Variants {
				variantIDs[i] = v.ID
			}

			inventoryResp, err := r.Clients.Inventory.GetStockBatch(ctx, &inventoryv1.GetStockBatchRequest{
				VariantIds: variantIDs,
			})
			if err != nil {
				inventoryErr = err
				return
			}

			inventoryMap = make(map[string]*model.InventoryInfo)
			for variantID, stock := range inventoryResp.Stocks {
				inventoryMap[variantID] = mapInventoryInfo(stock)
			}
		}()
	}

	wg.Wait()

	// Handle errors gracefully - if seller or inventory service is down, still return product data
	if sellerErr == nil {
		product.Seller = seller
	}

	if inventoryErr == nil && inventoryMap != nil {
		for _, variant := range product.Variants {
			if inventory, exists := inventoryMap[variant.ID]; exists {
				variant.Inventory = inventory
			} else {
				// Default to zero inventory if not found
				variant.Inventory = &model.InventoryInfo{
					Quantity:  0,
					UpdatedAt: product.UpdatedAt, // Use product update time as fallback
				}
			}
		}
	}

	return product, nil
}
