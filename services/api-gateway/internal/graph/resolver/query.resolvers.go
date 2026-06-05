package resolver

import (
	"context"

	"github.com/wemall/api-gateway/internal/graph/gqlerrors"
	"github.com/wemall/api-gateway/internal/graph/model"
	"github.com/wemall/api-gateway/internal/middleware"
	notificationv1 "github.com/wemall/gen/notification/v1"
	orderv1 "github.com/wemall/gen/order/v1"
	productv1 "github.com/wemall/gen/product/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	userv1 "github.com/wemall/gen/user/v1"
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
