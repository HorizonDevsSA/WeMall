package resolver

import (
	"context"
	"errors"
	"time"

	"github.com/wemall/api-gateway/internal/graph/gqlerrors"
	"github.com/wemall/api-gateway/internal/graph/model"
	"github.com/wemall/api-gateway/internal/middleware"
	deliveryv1 "github.com/wemall/gen/delivery/v1"
	notificationv1 "github.com/wemall/gen/notification/v1"
	orderv1 "github.com/wemall/gen/order/v1"
	paymentv1 "github.com/wemall/gen/payment/v1"
	productv1 "github.com/wemall/gen/product/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	userv1 "github.com/wemall/gen/user/v1"
	promotionv1 "github.com/wemall/gen/promotion/v1"
	chatv1 "github.com/wemall/gen/chat/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ── User Queries ──────────────────────────────────────────────────────────────

func (r *queryResolver) Me(ctx context.Context) (*model.User, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.User.GetUser(ctx, &userv1.GetUserRequest{Id: uid})
	if err != nil {
		return nil, err
	}
	return mapUser(resp), nil
}

func (r *queryResolver) User(ctx context.Context, id string) (*model.User, error) {
	resp, err := r.Clients.User.GetUser(ctx, &userv1.GetUserRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return mapUser(resp), nil
}

func (r *queryResolver) Addresses(ctx context.Context) ([]*model.Address, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.User.ListAddresses(ctx, &userv1.ListAddressesRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	out := make([]*model.Address, len(resp.Addresses))
	for i, a := range resp.Addresses {
		out[i] = mapAddress(a)
	}
	return out, nil
}

// ── Category Queries ──────────────────────────────────────────────────────────

func (r *queryResolver) Categories(ctx context.Context, language *string) ([]*model.Category, error) {
	lang := "en"
	if language != nil && *language != "" {
		lang = *language
	}
	resp, err := r.Clients.Product.ListCategories(ctx, &productv1.ListCategoriesRequest{Language: lang})
	if err != nil {
		return nil, err
	}
	out := make([]*model.Category, len(resp.Categories))
	for i, c := range resp.Categories {
		out[i] = mapCategory(c)
	}
	return out, nil
}

func (r *queryResolver) Category(ctx context.Context, slug string, language *string) (*model.Category, error) {
	lang := "en"
	if language != nil && *language != "" {
		lang = *language
	}
	resp, err := r.Clients.Product.GetCategory(ctx, &productv1.GetCategoryRequest{Slug: slug, Language: lang})
	if err != nil {
		return nil, err
	}
	return mapCategory(resp), nil
}

// ── Product Queries ───────────────────────────────────────────────────────────

func (r *queryResolver) Products(ctx context.Context, filter *model.ProductFilterInput, pageSize *int, pageToken *string, language *string) (*model.ProductList, error) {
	lang := "en"
	if language != nil && *language != "" {
		lang = *language
	}

	req := &productv1.ListProductsRequest{
		Language:  lang,
		PageSize:  int32(derefInt(pageSize, 20)),
		PageToken: derefStr(pageToken),
	}

	if filter != nil {
		req.Filter = mapProductFilter(filter)
	}

	resp, err := r.Clients.Product.ListProducts(ctx, req)
	if err != nil {
		return nil, err
	}

	products := make([]*model.Product, len(resp.Products))
	for i, p := range resp.Products {
		products[i] = mapProduct(p)
	}

	return &model.ProductList{
		Products:      products,
		NextPageToken: strPtr(resp.NextPageToken),
		Total:         int(resp.Total),
	}, nil
}

func (r *queryResolver) Product(ctx context.Context, id *string, slug *string, language *string) (*model.Product, error) {
	return r.ProductWithDetails(ctx, id, slug, language)
}

func (r *queryResolver) RecommendedProducts(ctx context.Context, pageSize *int, pageToken *string, language *string) (*model.ProductList, error) {
	lang := "en"
	if language != nil && *language != "" {
		lang = *language
	}

	req := &productv1.ListRecommendedProductsRequest{
		Language:  lang,
		PageSize:  int32(derefInt(pageSize, 20)),
		PageToken: derefStr(pageToken),
	}

	resp, err := r.Clients.Product.ListRecommendedProducts(ctx, req)
	if err != nil {
		return nil, err
	}

	products := make([]*model.Product, len(resp.Products))
	for i, p := range resp.Products {
		products[i] = mapProduct(p)
	}

	return &model.ProductList{
		Products:      products,
		NextPageToken: strPtr(resp.NextPageToken),
		Total:         int(resp.Total),
	}, nil
}

// ── Seller Queries ───────────────────────────────────────────────────────────

func (r *queryResolver) MyStore(ctx context.Context) (*model.Seller, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	return mapSeller(resp), nil
}

func (r *queryResolver) RevealBankDetails(ctx context.Context, pin string) (*model.Seller, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	// 1. Fetch the seller profile first
	sellerProto, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}

	// 2. Fetch the unmasked bank details
	revealResp, err := r.Clients.Seller.RevealBankDetails(ctx, &sellerv1.RevealBankDetailsRequest{
		UserId: uid,
		Pin:    pin,
	})
	if err != nil {
		return nil, err
	}

	// 3. Map to model and overwrite with decrypted plaintext details
	sellerModel := mapSeller(sellerProto)
	sellerModel.BankName = revealResp.BankName
	sellerModel.BankAccountNumber = revealResp.BankAccountNumber
	sellerModel.EcocashNumber = revealResp.EcocashNumber

	return sellerModel, nil
}

func (r *queryResolver) Seller(ctx context.Context, id string) (*model.Seller, error) {
	resp, err := r.Clients.Seller.GetSeller(ctx, &sellerv1.GetSellerRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return mapSeller(resp), nil
}

func (r *queryResolver) IsFollowingStore(ctx context.Context, sellerID string) (bool, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return false, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Seller.IsFollowingStore(ctx, &sellerv1.IsFollowingStoreRequest{
		UserId:   uid,
		SellerId: sellerID,
	})
	if err != nil {
		return false, err
	}
	return resp.IsFollowing, nil
}

func (r *queryResolver) MyFollowedStores(ctx context.Context, pageSize *int, pageToken *string) (*model.FollowedStoresList, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Seller.ListFollowedStores(ctx, &sellerv1.ListFollowedStoresRequest{
		UserId:    uid,
		PageSize:  int32(derefInt(pageSize, 20)),
		PageToken: derefStr(pageToken),
	})
	if err != nil {
		return nil, err
	}
	sellers := make([]*model.Seller, len(resp.Sellers))
	for i, s := range resp.Sellers {
		sellers[i] = mapSeller(s)
	}
	return &model.FollowedStoresList{
		Sellers:       sellers,
		NextPageToken: strPtr(resp.NextPageToken),
		Total:         int(resp.Total),
	}, nil
}

// ── Cart & Order Queries ──────────────────────────────────────────────────────

func (r *queryResolver) Cart(ctx context.Context) (*model.Cart, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Order.GetCart(ctx, &orderv1.GetCartRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	return mapCart(resp), nil
}

func (r *queryResolver) Order(ctx context.Context, id string) (*model.Order, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Order.GetOrder(ctx, &orderv1.GetOrderRequest{Id: id, UserId: uid})
	if err != nil {
		return nil, err
	}
	return mapOrder(resp), nil
}

func (r *queryResolver) Orders(ctx context.Context, pageSize *int, pageToken *string) (*model.OrderList, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Order.ListOrders(ctx, &orderv1.ListOrdersRequest{
		UserId:    uid,
		PageSize:  int32(derefInt(pageSize, 20)),
		PageToken: derefStr(pageToken),
	})
	if err != nil {
		return nil, err
	}
	orders := make([]*model.Order, len(resp.Orders))
	for i, o := range resp.Orders {
		orders[i] = mapOrder(o)
	}
	return &model.OrderList{
		Orders:        orders,
		NextPageToken: strPtr(resp.NextPageToken),
		Total:         int(resp.Total),
	}, nil
}

func (r *queryResolver) NearbyProducts(ctx context.Context, latitude float64, longitude float64, radiusMeters float64, pageSize *int, pageToken *string) ([]*model.ProductWithDistance, error) {
	req := &productv1.ListNearbyProductsRequest{
		Latitude:     latitude,
		Longitude:    longitude,
		RadiusMeters: radiusMeters,
		PageSize:     int32(derefInt(pageSize, 20)),
		PageToken:    derefStr(pageToken),
	}

	resp, err := r.Clients.Product.ListNearbyProducts(ctx, req)
	if err != nil {
		return nil, err
	}

	out := make([]*model.ProductWithDistance, len(resp.Products))
	for i, p := range resp.Products {
		out[i] = &model.ProductWithDistance{
			Product:  mapProduct(p),
			Distance: p.Distance,
		}
	}

	return out, nil
}

// ── Notification Queries ──────────────────────────────────────────────────────

func (r *queryResolver) NotificationPreferences(ctx context.Context) ([]*model.NotificationPreference, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Notification.GetNotificationPreferences(ctx, &notificationv1.GetNotificationPreferencesRequest{
		UserId: uid,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*model.NotificationPreference, len(resp.Preferences))
	for i, p := range resp.Preferences {
		out[i] = mapNotificationPreference(p)
	}
	return out, nil
}

func (r *queryResolver) MyNotifications(ctx context.Context, limit *int, offset *int) ([]*model.NotificationLog, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	var lim int32 = 20
	if limit != nil {
		lim = int32(*limit)
	}
	var off int32 = 0
	if offset != nil {
		off = int32(*offset)
	}

	resp, err := r.Clients.Notification.ListNotifications(ctx, &notificationv1.ListNotificationsRequest{
		UserId: uid,
		Limit:  lim,
		Offset: off,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*model.NotificationLog, len(resp.Notifications))
	for i, l := range resp.Notifications {
		out[i] = mapNotificationLog(l)
	}
	return out, nil
}

// ── Payment Queries ──────────────────────────────────────────────────────────

func (r *queryResolver) Payment(ctx context.Context, id string) (*model.Payment, error) {
	_, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Payment.GetPayment(ctx, &paymentv1.GetPaymentRequest{Id: id})
	if err != nil {
		return nil, err
	}
	return mapPayment(resp), nil
}

// ── Scaffolded Queries (Placeholder implementations) ─────────────────────────

func (r *queryResolver) MyChatThreads(ctx context.Context) ([]*model.ChatThread, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	// 1. Get seller profile to get store ID
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	storeID := sellerStore.Id

	// 2. Call chat service to list threads
	resp, err := r.Clients.Chat.ListThreads(ctx, &chatv1.ListThreadsRequest{
		UserId: storeID,
		Role:   "SELLER",
	})
	if err != nil {
		return nil, err
	}

	// Collect buyer IDs
	buyerIDs := []string{}
	buyerMap := make(map[string]bool)
	for _, t := range resp.Threads {
		if t.BuyerId != "" && !buyerMap[t.BuyerId] {
			buyerMap[t.BuyerId] = true
			buyerIDs = append(buyerIDs, t.BuyerId)
		}
	}

	// Fetch buyer users in batch
	users := make(map[string]*userv1.User)
	if len(buyerIDs) > 0 {
		batchResp, err := r.Clients.User.GetUserBatch(ctx, &userv1.GetUserBatchRequest{Ids: buyerIDs})
		if err == nil && batchResp != nil {
			users = batchResp.Users
		}
	}

	// Map to model.ChatThread
	out := make([]*model.ChatThread, len(resp.Threads))
	for i, t := range resp.Threads {
		buyerName := "Customer"
		if u, exists := users[t.BuyerId]; exists && u != nil {
			buyerName = u.FullName
		}

		lastMsg := "No messages yet"
		timestampStr := ""
		if t.UpdatedAt != nil {
			timestampStr = t.UpdatedAt.AsTime().Format(time.RFC3339)
		} else if t.CreatedAt != nil {
			timestampStr = t.CreatedAt.AsTime().Format(time.RFC3339)
		}

		// Query the last message of this thread
		msgResp, err := r.Clients.Chat.ListMessages(ctx, &chatv1.ListMessagesRequest{
			ThreadId: t.Id,
			PageSize: 1,
		})
		if err == nil && msgResp != nil && len(msgResp.Messages) > 0 {
			lastMsg = msgResp.Messages[0].Content
			if msgResp.Messages[0].CreatedAt != nil {
				timestampStr = msgResp.Messages[0].CreatedAt.AsTime().Format(time.RFC3339)
			}
		}

		var orderIDPtr *string
		if t.OrderId != "" {
			orderIDPtr = &t.OrderId
		}

		var createdAt, updatedAt time.Time
		if t.CreatedAt != nil {
			createdAt = t.CreatedAt.AsTime()
		}
		if t.UpdatedAt != nil {
			updatedAt = t.UpdatedAt.AsTime()
		}

		out[i] = &model.ChatThread{
			ID:          t.Id,
			BuyerID:     t.BuyerId,
			SellerID:    t.SellerId,
			OrderID:     orderIDPtr,
			BuyerName:   buyerName,
			LastMessage: lastMsg,
			Timestamp:   timestampStr,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		}
	}

	return out, nil
}

func (r *queryResolver) ChatMessages(ctx context.Context, threadID string, pageToken *string, pageSize *int) ([]*model.ChatMessage, error) {
	var size int32 = 50
	if pageSize != nil {
		size = int32(*pageSize)
	}
	var token string
	if pageToken != nil {
		token = *pageToken
	}

	resp, err := r.Clients.Chat.ListMessages(ctx, &chatv1.ListMessagesRequest{
		ThreadId:  threadID,
		PageToken: token,
		PageSize:  size,
	})
	if err != nil {
		return nil, err
	}

	out := make([]*model.ChatMessage, len(resp.Messages))
	for i, m := range resp.Messages {
		var createdAt time.Time
		if m.CreatedAt != nil {
			createdAt = m.CreatedAt.AsTime()
		}
		out[i] = &model.ChatMessage{
			ID:        m.Id,
			ThreadID:  m.ThreadId,
			SenderID:  m.SenderId,
			Content:   m.Content,
			IsRead:    m.IsRead,
			Timestamp: createdAt.Format(time.RFC3339),
			CreatedAt: createdAt,
		}
	}
	return out, nil
}

func (r *queryResolver) SellerDashboard(ctx context.Context) (*model.SellerDashboard, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	// 1. Get seller profile to get store ID
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	storeID := sellerStore.Id

	// 2. Query products for this seller
	prodResp, err := r.Clients.Product.ListProducts(ctx, &productv1.ListProductsRequest{
		Filter: &productv1.ProductFilter{
			SellerId: storeID,
		},
		PageSize: 100,
	})
	productsCount := 0
	activeProductsCount := 0
	if err == nil && prodResp != nil {
		productsCount = int(prodResp.Total)
		for _, p := range prodResp.Products {
			if p.Status == productv1.ProductStatus_PRODUCT_STATUS_ACTIVE {
				activeProductsCount++
			}
		}
	}

	// 3. Query seller orders
	var pendingOrdersCount int
	var totalOrdersCount int
	weeklyRevenue := make([]float64, 7)
	recentOrders := []*model.Order{}

	ordersResp, err := r.Clients.Order.ListSellerOrders(ctx, &orderv1.ListSellerOrdersRequest{
		SellerId:  storeID,
		PageSize:  100,
		PageToken: "",
	})
	if err == nil && ordersResp != nil {
		totalOrdersCount = int(ordersResp.Total)
		now := time.Now()
		
		getSellerOrderRevenue := func(o *orderv1.Order, sellerID string) float64 {
			var rev float64
			for _, item := range o.Items {
				if item.SellerId == sellerID {
					rev += item.UnitPrice * float64(item.Quantity)
				}
			}
			return rev
		}

		for _, o := range ordersResp.Orders {
			if o.Status == orderv1.OrderStatus_ORDER_STATUS_PENDING {
				pendingOrdersCount++
			}
			
			if o.CreatedAt != nil {
				t := o.CreatedAt.AsTime()
				daysDiff := int(now.Sub(t).Hours() / 24)
				if daysDiff >= 0 && daysDiff < 7 {
					weeklyRevenue[6-daysDiff] += getSellerOrderRevenue(o, storeID)
				}
			}
		}

		recentCount := len(ordersResp.Orders)
		if recentCount > 5 {
			recentCount = 5
		}
		recentOrders = make([]*model.Order, recentCount)
		for i := 0; i < recentCount; i++ {
			recentOrders[i] = mapOrder(ordersResp.Orders[i])
		}
	} else {
		// Fallback mocks if service fails or is empty to ensure smooth UI
		pendingOrdersCount = 0
		totalOrdersCount = 0
		weeklyRevenue = []float64{0, 0, 0, 0, 0, 0, 0}
	}

	// Map seller to model.Seller
	modelStore := mapSeller(sellerStore)

	return &model.SellerDashboard{
		Store:               modelStore,
		ProductsCount:       productsCount,
		ActiveProductsCount: activeProductsCount,
		PendingOrdersCount:  pendingOrdersCount,
		TotalOrdersCount:    totalOrdersCount,
		RecentOrders:        recentOrders,
		WeeklyRevenue:       weeklyRevenue,
	}, nil
}

func (r *queryResolver) MySellerOrders(ctx context.Context, pageSize *int, pageToken *string, status *model.OrderStatus) (*model.SellerOrderList, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	// 1. Get seller profile to get store ID
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	storeID := sellerStore.Id

	ps := int32(20)
	if pageSize != nil {
		ps = int32(*pageSize)
	}
	pt := ""
	if pageToken != nil {
		pt = *pageToken
	}

	resp, err := r.Clients.Order.ListSellerOrders(ctx, &orderv1.ListSellerOrdersRequest{
		SellerId:  storeID,
		PageSize:  ps,
		PageToken: pt,
	})
	if err != nil {
		return nil, err
	}

	orders := []*model.Order{}
	for _, o := range resp.Orders {
		mo := mapOrder(o)
		if status != nil && mo.Status != *status {
			continue
		}
		orders = append(orders, mo)
	}

	var nextPageToken *string
	if resp.NextPageToken != "" {
		nextPageToken = &resp.NextPageToken
	}

	return &model.SellerOrderList{
		Orders:        orders,
		NextPageToken: nextPageToken,
		Total:         int(resp.Total),
	}, nil
}

func (r *queryResolver) MyDisputes(ctx context.Context) (*model.DisputeList, error) {
	return nil, errors.New("dispute service not implemented")
}

func (r *queryResolver) Dispute(ctx context.Context, id string) (*model.Dispute, error) {
	return nil, errors.New("dispute service not implemented")
}

func (r *queryResolver) DisputeMessages(ctx context.Context, disputeID string) (*model.DisputeMessageList, error) {
	return nil, errors.New("dispute service not implemented")
}

func (r *queryResolver) PlatformMetrics(ctx context.Context) (*model.PlatformMetrics, error) {
	resp, err := r.Clients.Admin.GetPlatformMetrics(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return &model.PlatformMetrics{
		TotalUsers:     int(resp.TotalUsers),
		TotalSellers:   int(resp.TotalSellers),
		ActiveDisputes: int(resp.ActiveDisputes),
		TotalOrders:    int(resp.TotalOrders),
	}, nil
}

func (r *queryResolver) ActiveFlashSales(ctx context.Context) ([]*model.FlashSale, error) {
	resp, err := r.Clients.Promotion.ListActiveFlashSales(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	out := make([]*model.FlashSale, len(resp.Sales))
	for i, fs := range resp.Sales {
		out[i] = mapFlashSale(fs)
	}

	// Fetch product details for items to populate visual fields (thumbnail, title, rating)
	var productIDs []string
	seenIDs := make(map[string]bool)
	for _, fs := range out {
		for _, item := range fs.Items {
			if item.ProductID != "" && !seenIDs[item.ProductID] {
				seenIDs[item.ProductID] = true
				productIDs = append(productIDs, item.ProductID)
			}
		}
	}

	if len(productIDs) > 0 {
		batchResp, err := r.Clients.Product.GetProductBatch(ctx, &productv1.GetProductBatchRequest{
			Ids:      productIDs,
			Language: "en",
		})
		if err == nil && batchResp != nil && batchResp.Products != nil {
			for _, fs := range out {
				for _, item := range fs.Items {
					if prod, exists := batchResp.Products[item.ProductID]; exists && prod != nil {
						title := prod.Title
						item.ProductTitle = &title

						thumb := prod.Thumbnail
						if thumb == "" {
							thumb = prod.ImageUrl
						}
						if thumb != "" {
							item.Thumbnail = &thumb
						}

						rating := prod.Rating
						item.Rating = &rating
					}
				}
			}
		}
	}

	return out, nil
}

func (r *queryResolver) Coupons(ctx context.Context, sellerID *string) ([]*model.Coupon, error) {
	var sID string
	if sellerID != nil {
		sID = *sellerID
	}
	resp, err := r.Clients.Promotion.ListCoupons(ctx, &promotionv1.ListCouponsRequest{
		SellerId: sID,
		PageSize: 100,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*model.Coupon, len(resp.Coupons))
	for i, c := range resp.Coupons {
		out[i] = mapCoupon(c)
	}
	return out, nil
}

func (r *queryResolver) FrequentlyBoughtTogether(ctx context.Context, productID string) ([]*model.Product, error) {
	return nil, errors.New("recommendation service not implemented")
}

func (r *queryResolver) PersonalizedRecommendations(ctx context.Context) ([]*model.Product, error) {
	return nil, errors.New("recommendation service not implemented")
}

// ── Delivery Queries ──────────────────────────────────────────────────────────

func (r *queryResolver) TrackPackage(ctx context.Context, trackingNumber string) (*model.DeliveryOrder, error) {
	resp, err := r.Clients.Delivery.TrackPackage(ctx, &deliveryv1.TrackPackageRequest{
		TrackingNumber: trackingNumber,
	})
	if err != nil {
		return nil, err
	}
	return mapDeliveryOrder(resp.DeliveryOrder), nil
}

func (r *queryResolver) NearbyStations(ctx context.Context, latitude float64, longitude float64, radiusMeters float64) ([]*model.Station, error) {
	resp, err := r.Clients.Delivery.NearbyStations(ctx, &deliveryv1.NearbyStationsRequest{
		Latitude:     latitude,
		Longitude:    longitude,
		RadiusMeters: radiusMeters,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*model.Station, len(resp.Stations))
	for i, s := range resp.Stations {
		out[i] = mapStation(s)
	}
	return out, nil
}

func (r *queryResolver) AvailableCourierTasks(ctx context.Context, latitude float64, longitude float64) ([]*model.DeliveryOrder, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Delivery.AvailableCourierTasks(ctx, &deliveryv1.AvailableCourierTasksRequest{
		Latitude:  latitude,
		Longitude: longitude,
		UserId:    uid,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*model.DeliveryOrder, len(resp.Tasks))
	for i, t := range resp.Tasks {
		out[i] = mapDeliveryOrder(t)
	}
	return out, nil
}

func (r *queryResolver) StationInventory(ctx context.Context, stationID string, unclaimedOnly bool) ([]*model.StationPackage, error) {
	_, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Delivery.StationInventory(ctx, &deliveryv1.StationInventoryRequest{
		StationId:     stationID,
		UnclaimedOnly: unclaimedOnly,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*model.StationPackage, len(resp.Packages))
	for i, p := range resp.Packages {
		out[i] = mapStationPackage(p)
	}
	return out, nil
}

func (r *queryResolver) DeliveryByOrderID(ctx context.Context, orderID string) (*model.DeliveryOrder, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	// 1. Verify that the order belongs to the signed in user
	_, err := r.Clients.Order.GetOrder(ctx, &orderv1.GetOrderRequest{
		Id:     orderID,
		UserId: uid,
	})
	if err != nil {
		return nil, errors.New("access denied: order does not belong to you or does not exist")
	}

	// 2. Query delivery service for this order ID
	delResp, err := r.Clients.Delivery.GetDeliveryByOrderID(ctx, &deliveryv1.GetDeliveryByOrderIDRequest{
		OrderId: orderID,
	})
	if err != nil {
		// Return nil if no delivery exists yet for this order (unshipped)
		return nil, nil
	}

	// 3. Get full tracking information using tracking number
	trackResp, err := r.Clients.Delivery.TrackPackage(ctx, &deliveryv1.TrackPackageRequest{
		TrackingNumber: delResp.TrackingNumber,
	})
	if err != nil {
		return nil, err
	}

	return mapDeliveryOrder(trackResp.DeliveryOrder), nil
}

// ── Monetization & Payout Queries ─────────────────────────────────────────────

func (r *queryResolver) MySellerBalance(ctx context.Context) (*model.SellerBalance, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	resp, err := r.Clients.Seller.GetSellerBalance(ctx, &sellerv1.GetSellerBalanceRequest{SellerId: sellerStore.Id})
	if err != nil {
		return nil, err
	}
	return mapSellerBalance(resp), nil
}

func (r *queryResolver) MySellerEarningsLedger(ctx context.Context, pageSize *int, pageToken *string, statusFilter *string) (*model.SellerEarningsLedgerConnection, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	ps := int32(20)
	if pageSize != nil {
		ps = int32(*pageSize)
	}
	pt := ""
	if pageToken != nil {
		pt = *pageToken
	}
	sf := ""
	if statusFilter != nil {
		sf = *statusFilter
	}
	resp, err := r.Clients.Seller.GetSellerEarningsLedger(ctx, &sellerv1.GetSellerEarningsLedgerRequest{
		SellerId:     sellerStore.Id,
		PageSize:     ps,
		PageToken:    pt,
		StatusFilter: sf,
	})
	if err != nil {
		return nil, err
	}
	entries := make([]*model.EarningsLedgerEntry, len(resp.Entries))
	for i, entry := range resp.Entries {
		entries[i] = mapEarningsLedgerEntry(entry)
	}
	return &model.SellerEarningsLedgerConnection{
		Entries:       entries,
		NextPageToken: strPtr(resp.NextPageToken),
	}, nil
}

func (r *queryResolver) MyPayouts(ctx context.Context, pageSize *int, pageToken *string) (*model.PayoutList, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	ps := int32(20)
	if pageSize != nil {
		ps = int32(*pageSize)
	}
	pt := ""
	if pageToken != nil {
		pt = *pageToken
	}
	resp, err := r.Clients.Seller.ListPayouts(ctx, &sellerv1.ListPayoutsRequest{
		SellerId:  sellerStore.Id,
		PageSize:  ps,
		PageToken: pt,
	})
	if err != nil {
		return nil, err
	}
	payouts := make([]*model.Payout, len(resp.Payouts))
	for i, payout := range resp.Payouts {
		payouts[i] = mapPayout(payout)
	}
	return &model.PayoutList{
		Payouts:       payouts,
		NextPageToken: strPtr(resp.NextPageToken),
		Total:         int(resp.Total),
	}, nil
}

func (r *queryResolver) Payout(ctx context.Context, id string) (*model.Payout, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	resp, err := r.Clients.Seller.GetPayout(ctx, &sellerv1.GetPayoutRequest{
		Id:       id,
		SellerId: sellerStore.Id,
	})
	if err != nil {
		return nil, err
	}
	return mapPayout(resp), nil
}

func (r *queryResolver) MySellerMonetizationConfig(ctx context.Context) (*model.SellerMonetizationConfig, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	resp, err := r.Clients.Seller.GetSellerMonetizationConfig(ctx, &sellerv1.GetSellerMonetizationConfigRequest{SellerId: sellerStore.Id})
	if err != nil {
		return nil, err
	}
	return mapSellerMonetizationConfig(resp), nil
}


