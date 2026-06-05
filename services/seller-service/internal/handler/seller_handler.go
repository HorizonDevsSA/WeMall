package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	sellerv1 "github.com/wemall/gen/seller/v1"
	"github.com/wemall/seller-service/internal/db"
	"github.com/wemall/seller-service/internal/service"
)

type SellerHandler struct {
	sellerv1.UnimplementedSellerServiceServer
	svc *service.SellerService
}

func NewSellerHandler(svc *service.SellerService) *SellerHandler {
	return &SellerHandler{svc: svc}
}

func (h *SellerHandler) GetSeller(ctx context.Context, req *sellerv1.GetSellerRequest) (*sellerv1.Seller, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller id")
	}
	seller, err := h.svc.GetSeller(ctx, id)
	return mapSeller(seller), grpcErr(err)
}

func (h *SellerHandler) GetSellerByUserID(ctx context.Context, req *sellerv1.GetSellerByUserIDRequest) (*sellerv1.Seller, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id")
	}
	seller, err := h.svc.GetSellerByUserID(ctx, uid)
	return mapSeller(seller), grpcErr(err)
}

func (h *SellerHandler) GetSellerBatch(ctx context.Context, req *sellerv1.GetSellerBatchRequest) (*sellerv1.GetSellerBatchResponse, error) {
	ids := make([]uuid.UUID, 0, len(req.Ids))
	for _, id := range req.Ids {
		uid, err := uuid.Parse(id)
		if err == nil {
			ids = append(ids, uid)
		}
	}
	sellers, err := h.svc.GetSellerBatch(ctx, ids)
	if err != nil {
		return nil, grpcErr(err)
	}
	out := make(map[string]*sellerv1.Seller, len(sellers))
	for id, s := range sellers {
		out[id.String()] = mapSeller(&s)
	}
	return &sellerv1.GetSellerBatchResponse{Sellers: out}, nil
}

func (h *SellerHandler) CreateStore(ctx context.Context, req *sellerv1.CreateStoreRequest) (*sellerv1.Seller, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id")
	}

	var lat, lon *float64
	if req.Latitude != 0 {
		lat = &req.Latitude
	}
	if req.Longitude != 0 {
		lon = &req.Longitude
	}

	seller, err := h.svc.CreateStore(ctx, service.CreateStoreInput{
		UserID:      uid,
		StoreName:   req.StoreName,
		Description: strPtr(req.Description),
		LogoURL:     strPtr(req.LogoUrl),
		BannerURL:   strPtr(req.BannerUrl),
		Latitude:    lat,
		Longitude:   lon,
	})
	return mapSeller(seller), grpcErr(err)
}

func (h *SellerHandler) UpdateStore(ctx context.Context, req *sellerv1.UpdateStoreRequest) (*sellerv1.Seller, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user id")
	}

	var lat, lon *float64
	if req.Latitude != 0 {
		lat = &req.Latitude
	}
	if req.Longitude != 0 {
		lon = &req.Longitude
	}

	seller, err := h.svc.UpdateStore(ctx, service.UpdateStoreInput{
		UserID:      uid,
		StoreName:   optionalStr(req.StoreName),
		Description: optionalStr(req.Description),
		LogoURL:     optionalStr(req.LogoUrl),
		BannerURL:   optionalStr(req.BannerUrl),
		Latitude:    lat,
		Longitude:   lon,
	})
	return mapSeller(seller), grpcErr(err)
}

func (h *SellerHandler) VerifySeller(ctx context.Context, req *sellerv1.VerifySellerRequest) (*sellerv1.Seller, error) {
	id, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller id")
	}
	seller, err := h.svc.VerifySeller(ctx, id, req.Verified)
	return mapSeller(seller), grpcErr(err)
}

func (h *SellerHandler) UpdateSellerStatus(ctx context.Context, req *sellerv1.UpdateSellerStatusRequest) (*sellerv1.Seller, error) {
	id, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller id")
	}

	dbStatus := protoStatusToDBStatus(req.Status)
	seller, err := h.svc.UpdateSellerStatus(ctx, id, dbStatus)
	return mapSeller(seller), grpcErr(err)
}

// ── Store Follow RPCs ─────────────────────────────────────────────────────────

func (h *SellerHandler) FollowStore(ctx context.Context, req *sellerv1.FollowStoreRequest) (*emptypb.Empty, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id")
	}
	sellerID, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller_id")
	}
	err = h.svc.FollowStore(ctx, userID, sellerID)
	if err != nil {
		return nil, grpcErr(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *SellerHandler) UnfollowStore(ctx context.Context, req *sellerv1.UnfollowStoreRequest) (*emptypb.Empty, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id")
	}
	sellerID, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller_id")
	}
	err = h.svc.UnfollowStore(ctx, userID, sellerID)
	if err != nil {
		return nil, grpcErr(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *SellerHandler) IsFollowingStore(ctx context.Context, req *sellerv1.IsFollowingStoreRequest) (*sellerv1.IsFollowingStoreResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id")
	}
	sellerID, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller_id")
	}
	following, err := h.svc.IsFollowingStore(ctx, userID, sellerID)
	if err != nil {
		return nil, grpcErr(err)
	}
	return &sellerv1.IsFollowingStoreResponse{IsFollowing: following}, nil
}

func (h *SellerHandler) ListFollowedStores(ctx context.Context, req *sellerv1.ListFollowedStoresRequest) (*sellerv1.ListFollowedStoresResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid user_id")
	}

	stores, total, nextToken, err := h.svc.ListFollowedStores(ctx, userID, req.PageSize, req.PageToken)
	if err != nil {
		return nil, grpcErr(err)
	}

	sellers := make([]*sellerv1.Seller, len(stores))
	for i := range stores {
		sellers[i] = mapSeller(&stores[i])
	}

	return &sellerv1.ListFollowedStoresResponse{
		Sellers:       sellers,
		NextPageToken: nextToken,
		Total:         total,
	}, nil
}

func (h *SellerHandler) ListStoreFollowers(ctx context.Context, req *sellerv1.ListStoreFollowersRequest) (*sellerv1.ListStoreFollowersResponse, error) {
	sellerID, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller_id")
	}
	userIDs, err := h.svc.ListStoreFollowers(ctx, sellerID)
	if err != nil {
		return nil, grpcErr(err)
	}
	ids := make([]string, len(userIDs))
	for i, id := range userIDs {
		ids[i] = id.String()
	}
	return &sellerv1.ListStoreFollowersResponse{UserIds: ids}, nil
}

// ── Payout RPCs ───────────────────────────────────────────────────────────────

func (h *SellerHandler) ListPayouts(ctx context.Context, req *sellerv1.ListPayoutsRequest) (*sellerv1.ListPayoutsResponse, error) {
	sid, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller id")
	}
	rows, total, nextToken, err := h.svc.ListPayouts(ctx, sid, req.PageSize, req.PageToken)
	if err != nil {
		return nil, grpcErr(err)
	}
	payouts := make([]*sellerv1.Payout, len(rows))
	for i := range rows {
		payouts[i] = mapPayout(&rows[i])
	}
	return &sellerv1.ListPayoutsResponse{
		Payouts:       payouts,
		NextPageToken: nextToken,
		Total:         total,
	}, nil
}

func (h *SellerHandler) GetPayout(ctx context.Context, req *sellerv1.GetPayoutRequest) (*sellerv1.Payout, error) {
	pid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid payout id")
	}
	sid, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller id")
	}
	payout, err := h.svc.GetPayout(ctx, pid, sid)
	return mapPayout(payout), grpcErr(err)
}

func (h *SellerHandler) CreatePayout(ctx context.Context, req *sellerv1.CreatePayoutRequest) (*sellerv1.Payout, error) {
	sid, err := uuid.Parse(req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid seller id")
	}
	payout, err := h.svc.CreatePayout(ctx, sid, req.Amount, req.Currency)
	return mapPayout(payout), grpcErr(err)
}

// ── Mappers ───────────────────────────────────────────────────────────────────

func mapSeller(s *db.Seller) *sellerv1.Seller {
	if s == nil {
		return nil
	}
	seller := &sellerv1.Seller{
		Id:          s.ID.String(),
		UserId:      s.UserID.String(),
		StoreName:   s.StoreName,
		StoreSlug:   s.StoreSlug,
		LogoUrl:     deref(s.LogoUrl),
		BannerUrl:   deref(s.BannerUrl),
		Description: deref(s.Description),
		Rating:      s.Rating,
		TotalSales:  s.TotalSales,
		IsVerified:  s.IsVerified,
		Status:      dbStatusToProtoStatus(s.Status),
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}

	// Add coordinates if they exist
	if s.Latitude != nil {
		seller.Latitude = *s.Latitude
	}
	if s.Longitude != nil {
		seller.Longitude = *s.Longitude
	}

	return seller
}

func mapPayout(p *db.SellerPayout) *sellerv1.Payout {
	if p == nil {
		return nil
	}
	out := &sellerv1.Payout{
		Id:        p.ID.String(),
		SellerId:  p.SellerID.String(),
		Amount:    p.Amount,
		Currency:  p.Currency,
		Status:    mapPayoutStatus(p.Status),
		CreatedAt: timestamppb.New(p.CreatedAt),
	}
	if p.ProviderRef != nil {
		out.ProviderRef = *p.ProviderRef
	}
	if p.PaidAt.Valid {
		out.PaidAt = timestamppb.New(p.PaidAt.Time)
	}
	return out
}

func mapPayoutStatus(s string) sellerv1.PayoutStatus {
	switch s {
	case "pending":
		return sellerv1.PayoutStatus_PAYOUT_STATUS_PENDING
	case "processing":
		return sellerv1.PayoutStatus_PAYOUT_STATUS_PROCESSING
	case "paid":
		return sellerv1.PayoutStatus_PAYOUT_STATUS_PAID
	case "failed":
		return sellerv1.PayoutStatus_PAYOUT_STATUS_FAILED
	default:
		return sellerv1.PayoutStatus_PAYOUT_STATUS_UNSPECIFIED
	}
}

func dbStatusToProtoStatus(s db.SellerStatus) sellerv1.SellerStatus {
	switch s {
	case db.SellerStatusPending:
		return sellerv1.SellerStatus_SELLER_STATUS_PENDING
	case db.SellerStatusProcessing:
		return sellerv1.SellerStatus_SELLER_STATUS_PROCESSING
	case db.SellerStatusVerified:
		return sellerv1.SellerStatus_SELLER_STATUS_VERIFIED
	case db.SellerStatusSuspended:
		return sellerv1.SellerStatus_SELLER_STATUS_SUSPENDED
	default:
		return sellerv1.SellerStatus_SELLER_STATUS_UNSPECIFIED
	}
}

func protoStatusToDBStatus(s sellerv1.SellerStatus) db.SellerStatus {
	switch s {
	case sellerv1.SellerStatus_SELLER_STATUS_PENDING:
		return db.SellerStatusPending
	case sellerv1.SellerStatus_SELLER_STATUS_PROCESSING:
		return db.SellerStatusProcessing
	case sellerv1.SellerStatus_SELLER_STATUS_VERIFIED:
		return db.SellerStatusVerified
	case sellerv1.SellerStatus_SELLER_STATUS_SUSPENDED:
		return db.SellerStatusSuspended
	default:
		return db.SellerStatusPending
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func grpcErr(err error) error {
	return err
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func optionalStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
