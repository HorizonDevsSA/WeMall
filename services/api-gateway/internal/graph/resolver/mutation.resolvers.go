package resolver

import (
	"context"
	"errors"

	"github.com/wemall/api-gateway/internal/graph/gqlerrors"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/wemall/api-gateway/internal/graph/model"
	"github.com/wemall/api-gateway/internal/middleware"
	notificationv1 "github.com/wemall/gen/notification/v1"
	orderv1 "github.com/wemall/gen/order/v1"
	productv1 "github.com/wemall/gen/product/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	userv1 "github.com/wemall/gen/user/v1"
	paymentv1 "github.com/wemall/gen/payment/v1"
)

// ── Auth Mutations ────────────────────────────────────────────────────────────

func (r *mutationResolver) BuyerGoogleAuth(ctx context.Context, code string, redirectURI *string) (*model.AuthPayload, error) {
	resp, err := r.Clients.User.BuyerGoogleAuth(ctx, &userv1.GoogleAuthRequest{Code: code, RedirectUri: derefStr(redirectURI)})
	if err != nil {
		return nil, err
	}
	return mapAuthPayload(resp), nil
}

func (r *mutationResolver) BuyerSendOtp(ctx context.Context, phone string) (*model.OTPPayload, error) {
	resp, err := r.Clients.User.BuyerSendOTP(ctx, &userv1.PhoneOTPRequest{Phone: phone})
	if err != nil {
		return nil, err
	}
	return &model.OTPPayload{Message: resp.Message, RequestID: resp.RequestId}, nil
}

func (r *mutationResolver) BuyerVerifyOtp(ctx context.Context, phone string, otp string) (*model.AuthPayload, error) {
	resp, err := r.Clients.User.BuyerVerifyOTP(ctx, &userv1.VerifyOTPRequest{Phone: phone, Otp: otp})
	if err != nil {
		return nil, err
	}
	return mapAuthPayload(resp), nil
}

func (r *mutationResolver) SellerRegister(ctx context.Context, email string, password string, fullName string) (*model.AuthPayload, error) {
	resp, err := r.Clients.User.SellerRegister(ctx, &userv1.SellerRegisterRequest{Email: email, Password: password, FullName: fullName})
	if err != nil {
		return nil, err
	}
	return mapAuthPayload(resp), nil
}

func (r *mutationResolver) SellerLogin(ctx context.Context, email string, password string) (*model.AuthPayload, error) {
	resp, err := r.Clients.User.SellerLogin(ctx, &userv1.SellerLoginRequest{Email: email, Password: password})
	if err != nil {
		return nil, err
	}
	return mapAuthPayload(resp), nil
}

func (r *mutationResolver) RefreshToken(ctx context.Context, refreshToken string) (*model.AuthPayload, error) {
	resp, err := r.Clients.User.RefreshToken(ctx, &userv1.RefreshTokenRequest{RefreshToken: refreshToken})
	if err != nil {
		return nil, err
	}
	return mapAuthPayload(resp), nil
}

// ── Profile Mutations ─────────────────────────────────────────────────────────

func (r *mutationResolver) UpdateProfile(ctx context.Context, fullName *string, avatarURL *string) (*model.User, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.User.UpdateProfile(ctx, &userv1.UpdateProfileRequest{UserId: uid, FullName: derefStr(fullName), AvatarUrl: derefStr(avatarURL)})
	if err != nil {
		return nil, err
	}
	return mapUser(resp), nil
}

func (r *mutationResolver) CreateAddress(ctx context.Context, input model.AddressInput) (*model.Address, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.User.CreateAddress(ctx, &userv1.CreateAddressRequest{
		UserId: uid, Label: derefStr(input.Label), FullName: input.FullName, Phone: input.Phone,
		AddressLine1: input.AddressLine1, AddressLine2: derefStr(input.AddressLine2),
		City: input.City, State: derefStr(input.State), PostalCode: derefStr(input.PostalCode),
		Country: input.Country, IsDefault: derefBool(input.IsDefault),
	})
	if err != nil {
		return nil, err
	}
	return mapAddress(resp), nil
}

func (r *mutationResolver) DeleteAddress(ctx context.Context, addressID string) (bool, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return false, gqlerrors.Unauthenticated("authentication required")
	}
	_, err := r.Clients.User.DeleteAddress(ctx, &userv1.DeleteAddressRequest{UserId: uid, AddressId: addressID})
	return err == nil, err
}

// ── Product Mutations ─────────────────────────────────────────────────────────
// Phase 1: seller_id = authenticated user's ID (seller-service added in Phase 2)

func (r *mutationResolver) CreateProduct(ctx context.Context, input model.CreateProductInput) (*model.Product, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	storeID := sellerStore.Id

	variants := make([]*productv1.CreateVariantInput, len(input.Variants))
	for i, v := range input.Variants {
		opts, _ := structpb.NewStruct(jsonToMap(v.Options))
		var initQty int32 = 0
		if v.InitialQuantity != nil {
			initQty = int32(*v.InitialQuantity)
		}
		variants[i] = &productv1.CreateVariantInput{
			Sku:             v.Sku,
			Options:         opts,
			Price:           v.Price,
			ComparePrice:    derefFloat(v.ComparePrice),
			InitialQuantity: initQty,
		}
	}
	attrs, _ := structpb.NewStruct(jsonToMap(input.Attributes))
	resp, err := r.Clients.Product.CreateProduct(ctx, &productv1.CreateProductRequest{
		SellerId: storeID, CategoryId: input.CategoryID, Title: input.Title,
		Description: derefStr(input.Description), Attributes: attrs, Brand: derefStr(input.Brand),
		OriginCountry: derefStr(input.OriginCountry), Variants: variants, Tags: input.Tags, Language: derefStr(input.Language),
		Latitude: sellerStore.Latitude, Longitude: sellerStore.Longitude,
		ProductType: unmapProductType(input.ProductType),
		ImageUrl:     derefStr(input.ImageURL),
		ThumbnailUrl: derefStr(input.ThumbnailURL),
		Images:       input.Images,
	})
	if err != nil {
		return nil, err
	}
	return mapProduct(resp), nil
}

func (r *mutationResolver) UpdateProduct(ctx context.Context, id string, input model.UpdateProductInput) (*model.Product, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	storeID := sellerStore.Id

	var attrs *structpb.Struct
	if input.Attributes != nil {
		attrs, _ = structpb.NewStruct(jsonToMap(input.Attributes))
	}
	statusStr := ""
	if input.Status != nil {
		statusStr = productStatusToProto(*input.Status)
	}
	resp, err := r.Clients.Product.UpdateProduct(ctx, &productv1.UpdateProductRequest{
		Id: id, SellerId: storeID, Title: derefStr(input.Title), Description: derefStr(input.Description),
		Attributes: attrs, Brand: derefStr(input.Brand), Language: derefStr(input.Language),
		Status: productv1.ProductStatus(productv1.ProductStatus_value["PRODUCT_STATUS_"+statusStr]),
		ImageUrl:     derefStr(input.ImageURL),
		ThumbnailUrl: derefStr(input.ThumbnailURL),
		Images:       input.Images,
	})
	if err != nil {
		return nil, err
	}
	return mapProduct(resp), nil
}

func (r *mutationResolver) DeleteProduct(ctx context.Context, id string) (bool, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return false, gqlerrors.Unauthenticated("authentication required")
	}
	sellerStore, err := r.Clients.Seller.GetSellerByUserID(ctx, &sellerv1.GetSellerByUserIDRequest{UserId: uid})
	if err != nil {
		return false, err
	}
	storeID := sellerStore.Id

	_, err = r.Clients.Product.DeleteProduct(ctx, &productv1.DeleteProductRequest{Id: id, SellerId: storeID})
	return err == nil, err
}

// ── Store Mutations (Seller) ─────────────────────────────────────────────────

func (r *mutationResolver) CreateStore(ctx context.Context, input model.CreateStoreInput) (*model.Seller, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Seller.CreateStore(ctx, &sellerv1.CreateStoreRequest{
		UserId:      uid,
		StoreName:   input.StoreName,
		Description: derefStr(input.Description),
		LogoUrl:     derefStr(input.LogoURL),
		BannerUrl:   derefStr(input.BannerURL),
		Latitude:    derefFloat(input.Latitude),
		Longitude:   derefFloat(input.Longitude),
	})
	if err != nil {
		return nil, err
	}
	return mapSeller(resp), nil
}

func (r *mutationResolver) UpdateStore(ctx context.Context, input model.UpdateStoreInput) (*model.Seller, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Seller.UpdateStore(ctx, &sellerv1.UpdateStoreRequest{
		UserId:      uid,
		StoreName:   derefStr(input.StoreName),
		Description: derefStr(input.Description),
		LogoUrl:     derefStr(input.LogoURL),
		BannerUrl:   derefStr(input.BannerURL),
		Latitude:    derefFloat(input.Latitude),
		Longitude:   derefFloat(input.Longitude),
	})
	if err != nil {
		return nil, err
	}
	return mapSeller(resp), nil
}

func (r *mutationResolver) UpdateSellerStatus(ctx context.Context, sellerID string, status model.SellerStatus) (*model.Seller, error) {
	var protoStatus sellerv1.SellerStatus
	switch status {
	case model.SellerStatusPending:
		protoStatus = sellerv1.SellerStatus_SELLER_STATUS_PENDING
	case model.SellerStatusProcessing:
		protoStatus = sellerv1.SellerStatus_SELLER_STATUS_PROCESSING
	case model.SellerStatusVerified:
		protoStatus = sellerv1.SellerStatus_SELLER_STATUS_VERIFIED
	case model.SellerStatusSuspended:
		protoStatus = sellerv1.SellerStatus_SELLER_STATUS_SUSPENDED
	default:
		protoStatus = sellerv1.SellerStatus_SELLER_STATUS_UNSPECIFIED
	}

	resp, err := r.Clients.Seller.UpdateSellerStatus(ctx, &sellerv1.UpdateSellerStatusRequest{
		SellerId: sellerID,
		Status:   protoStatus,
	})
	if err != nil {
		return nil, err
	}

	// Trigger review status notification email asynchronously
	go func() {
		userResp, userErr := r.Clients.User.GetUser(context.Background(), &userv1.GetUserRequest{Id: resp.UserId})
		if userErr == nil && userResp.Email != "" {
			_, emailErr := r.Clients.User.SendReviewStatusEmail(context.Background(), &userv1.SendReviewStatusEmailRequest{
				Email:     userResp.Email,
				FullName:  userResp.FullName,
				StoreName: resp.StoreName,
				Status:    string(status),
			})
			if emailErr != nil {
				println("failed to send status email:", emailErr.Error())
			}
		}
	}()

	return mapSeller(resp), nil
}

// ── Store Follow Mutations ────────────────────────────────────────────────────


func (r *mutationResolver) FollowStore(ctx context.Context, sellerID string) (bool, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return false, gqlerrors.Unauthenticated("authentication required")
	}
	_, err := r.Clients.Seller.FollowStore(ctx, &sellerv1.FollowStoreRequest{
		UserId:   uid,
		SellerId: sellerID,
	})
	return err == nil, err
}

func (r *mutationResolver) UnfollowStore(ctx context.Context, sellerID string) (bool, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return false, gqlerrors.Unauthenticated("authentication required")
	}
	_, err := r.Clients.Seller.UnfollowStore(ctx, &sellerv1.UnfollowStoreRequest{
		UserId:   uid,
		SellerId: sellerID,
	})
	return err == nil, err
}

// ── Cart & Order Mutations ────────────────────────────────────────────────────

func (r *mutationResolver) AddToCart(ctx context.Context, variantID string, quantity int) (*model.Cart, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Order.AddToCart(ctx, &orderv1.AddToCartRequest{UserId: uid, VariantId: variantID, Quantity: int32(quantity)})
	if err != nil {
		return nil, err
	}
	return mapCart(resp), nil
}

func (r *mutationResolver) UpdateCartItem(ctx context.Context, itemID string, quantity int) (*model.Cart, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Order.UpdateCartItem(ctx, &orderv1.UpdateCartItemRequest{UserId: uid, ItemId: itemID, Quantity: int32(quantity)})
	if err != nil {
		return nil, err
	}
	return mapCart(resp), nil
}

func (r *mutationResolver) RemoveCartItem(ctx context.Context, itemID string) (*model.Cart, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Order.RemoveCartItem(ctx, &orderv1.RemoveCartItemRequest{UserId: uid, ItemId: itemID})
	if err != nil {
		return nil, err
	}
	return mapCart(resp), nil
}

func (r *mutationResolver) ClearCart(ctx context.Context) (*model.Cart, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Order.ClearCart(ctx, &orderv1.ClearCartRequest{UserId: uid})
	if err != nil {
		return nil, err
	}
	return mapCart(resp), nil
}

func (r *mutationResolver) Checkout(ctx context.Context, input model.CheckoutInput) (*model.Order, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	addrStruct, _ := structpb.NewStruct(map[string]interface{}{
		"full_name": input.ShippingAddress.FullName, "phone": input.ShippingAddress.Phone,
		"address_line1": input.ShippingAddress.AddressLine1, "address_line2": derefStr(input.ShippingAddress.AddressLine2),
		"city": input.ShippingAddress.City, "state": derefStr(input.ShippingAddress.State),
		"postal_code": derefStr(input.ShippingAddress.PostalCode), "country": input.ShippingAddress.Country,
	})
	currency := "USD"
	if input.Currency != nil {
		currency = string(*input.Currency)
	}
	resp, err := r.Clients.Order.Checkout(ctx, &orderv1.CheckoutRequest{
		UserId: uid,
		Input:  &orderv1.CheckoutInput{ShippingAddress: addrStruct, CouponCode: derefStr(input.CouponCode), Notes: derefStr(input.Notes), Currency: currency},
	})
	if err != nil {
		return nil, err
	}
	return mapOrder(resp), nil
}

func (r *mutationResolver) CancelOrder(ctx context.Context, id string) (*model.Order, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Order.CancelOrder(ctx, &orderv1.CancelOrderRequest{Id: id, UserId: uid})
	if err != nil {
		return nil, err
	}
	return mapOrder(resp), nil
}

// ── Notification Mutations ───────────────────────────────────────────────────

func (r *mutationResolver) RegisterDeviceToken(ctx context.Context, token string, platform string, deviceName *string) (bool, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return false, gqlerrors.Unauthenticated("authentication required")
	}
	var devName string
	if deviceName != nil {
		devName = *deviceName
	}
	_, err := r.Clients.Notification.RegisterDeviceToken(ctx, &notificationv1.RegisterDeviceTokenRequest{
		UserId:     uid,
		Token:      token,
		Platform:   platform,
		DeviceName: devName,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) DeregisterDeviceToken(ctx context.Context, token string) (bool, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return false, gqlerrors.Unauthenticated("authentication required")
	}
	_, err := r.Clients.Notification.DeregisterDeviceToken(ctx, &notificationv1.DeregisterDeviceTokenRequest{
		UserId: uid,
		Token:  token,
	})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *mutationResolver) UpdateNotificationPreferences(ctx context.Context, category model.NotificationCategory, emailEnabled bool, pushEnabled bool) (*model.NotificationPreference, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}
	resp, err := r.Clients.Notification.UpdateNotificationPreferences(ctx, &notificationv1.UpdateNotificationPreferencesRequest{
		UserId:       uid,
		Category:     unmapNotificationCategory(category),
		EmailEnabled: emailEnabled,
		PushEnabled:  pushEnabled,
	})
	if err != nil {
		return nil, err
	}
	return mapNotificationPreference(resp), nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func productStatusToProto(s model.ProductStatus) string {
	switch s {
	case model.ProductStatusDraft:
		return "DRAFT"
	case model.ProductStatusActive:
		return "ACTIVE"
	case model.ProductStatusPaused:
		return "PAUSED"
	case model.ProductStatusBanned:
		return "BANNED"
	default:
		return ""
	}
}

// ── Payment Mutations ────────────────────────────────────────────────────────

func (r *mutationResolver) InitiatePayment(ctx context.Context, orderID string, provider model.PaymentProvider) (*model.InitiatePaymentResponse, error) {
	uid, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	// 1. Fetch Order details to get the amount/currency (from order-service)
	order, err := r.Clients.Order.GetOrder(ctx, &orderv1.GetOrderRequest{
		Id:     orderID,
		UserId: uid,
	})
	if err != nil {
		return nil, err
	}

	// 2. Call payment-service to create payment
	resp, err := r.Clients.Payment.CreatePayment(ctx, &paymentv1.CreatePaymentRequest{
		OrderId:  orderID,
		UserId:   uid,
		Amount:   order.Total,
		Currency: order.Currency,
		Provider: unmapPaymentProvider(provider),
	})
	if err != nil {
		return nil, err
	}

	return &model.InitiatePaymentResponse{
		Payment:      mapPayment(resp.Payment),
		ClientSecret: &resp.ClientSecret,
	}, nil
}

func (r *mutationResolver) ProcessPayment(ctx context.Context, paymentID string, token string) (*model.Payment, error) {
	_, ok := middleware.UserIDFromCtx(ctx)
	if !ok {
		return nil, gqlerrors.Unauthenticated("authentication required")
	}

	resp, err := r.Clients.Payment.ProcessPayment(ctx, &paymentv1.ProcessPaymentRequest{
		PaymentId: paymentID,
		Token:     token,
	})
	if err != nil {
		return nil, err
	}

	return mapPayment(resp.Payment), nil
}

// ── Scaffolded Mutations (Placeholder implementations) ───────────────────────

func (r *mutationResolver) CreateChatThread(ctx context.Context, sellerID string, orderID *string) (*model.ChatThread, error) {
	return nil, errors.New("chat service not implemented")
}

func (r *mutationResolver) SendChatMessage(ctx context.Context, threadID string, content string) (*model.ChatMessage, error) {
	return nil, errors.New("chat service not implemented")
}

func (r *mutationResolver) OpenDispute(ctx context.Context, orderID string, reason string, evidenceUrls []string) (*model.Dispute, error) {
	return nil, errors.New("dispute service not implemented")
}

func (r *mutationResolver) ReplyToDispute(ctx context.Context, disputeID string, message string, evidenceUrls []string) (*model.DisputeMessage, error) {
	return nil, errors.New("dispute service not implemented")
}

func (r *mutationResolver) EscalateDispute(ctx context.Context, disputeID string) (*model.Dispute, error) {
	return nil, errors.New("dispute service not implemented")
}

func (r *mutationResolver) ResolveDispute(ctx context.Context, disputeID string, resolution model.DisputeStatus) (*model.Dispute, error) {
	return nil, errors.New("admin service not implemented")
}

func (r *mutationResolver) SuspendSeller(ctx context.Context, sellerID string, reason string) (bool, error) {
	return false, errors.New("admin service not implemented")
}

func (r *mutationResolver) BanBuyer(ctx context.Context, buyerID string, reason string) (bool, error) {
	return false, errors.New("admin service not implemented")
}

func (r *mutationResolver) ApplyCoupon(ctx context.Context, code string, cartID string) (*model.Cart, error) {
	return nil, errors.New("promotion service not implemented")
}

func (r *mutationResolver) CreateCoupon(ctx context.Context, input model.CreateCouponInput) (*model.Coupon, error) {
	return nil, errors.New("promotion service not implemented")
}

func (r *mutationResolver) RecordProductView(ctx context.Context, productID string) (bool, error) {
	return false, errors.New("recommendation service not implemented")
}
