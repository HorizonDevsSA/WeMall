package handler

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	userv1 "github.com/wemall/gen/user/v1"
	"github.com/wemall/user-service/internal/db"
	"github.com/wemall/user-service/internal/service"
)

type UserHandler struct {
	userv1.UnimplementedUserServiceServer
	authSvc *service.AuthService
	userSvc *service.UserService
}

func NewUserHandler(authSvc *service.AuthService, userSvc *service.UserService) *UserHandler {
	return &UserHandler{
		authSvc: authSvc,
		userSvc: userSvc,
	}
}

// ── Auth Handlers ────────────────────────────────────────────────────────────

func (h *UserHandler) BuyerGoogleAuth(ctx context.Context, req *userv1.GoogleAuthRequest) (*userv1.AuthResponse, error) {
	tokens, user, err := h.authSvc.BuyerGoogleAuth(ctx, req.Code, req.RedirectUri)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "google auth failed")
	}
	return &userv1.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         mapUser(user),
	}, nil
}

func (h *UserHandler) BuyerSendOTP(ctx context.Context, req *userv1.PhoneOTPRequest) (*userv1.PhoneOTPResponse, error) {
	reqID, err := h.authSvc.BuyerSendOTP(ctx, req.Phone)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to send OTP")
	}
	return &userv1.PhoneOTPResponse{
		Message:   "OTP sent successfully",
		RequestId: reqID,
	}, nil
}

func (h *UserHandler) BuyerVerifyOTP(ctx context.Context, req *userv1.VerifyOTPRequest) (*userv1.AuthResponse, error) {
	tokens, user, err := h.authSvc.BuyerVerifyOTP(ctx, req.Phone, req.Otp)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "OTP verification failed")
	}
	return &userv1.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         mapUser(user),
	}, nil
}

func (h *UserHandler) SellerRegister(ctx context.Context, req *userv1.SellerRegisterRequest) (*userv1.AuthResponse, error) {
	tokens, user, err := h.authSvc.SellerRegister(ctx, req.Email, req.Password, req.FullName)
	if err != nil {
		return nil, status.Error(codes.AlreadyExists, "registration failed")
	}
	return &userv1.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         mapUser(user),
	}, nil
}

func (h *UserHandler) SellerLogin(ctx context.Context, req *userv1.SellerLoginRequest) (*userv1.AuthResponse, error) {
	tokens, user, err := h.authSvc.SellerLogin(ctx, req.Email, req.Password)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "login failed")
	}
	return &userv1.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         mapUser(user),
	}, nil
}

func (h *UserHandler) RefreshToken(ctx context.Context, req *userv1.RefreshTokenRequest) (*userv1.AuthResponse, error) {
	tokens, user, err := h.authSvc.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "refresh token failed")
	}
	return &userv1.AuthResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User:         mapUser(user),
	}, nil
}

func (h *UserHandler) ValidateToken(ctx context.Context, req *userv1.ValidateTokenRequest) (*userv1.ValidateTokenResponse, error) {
	claims, err := h.authSvc.ValidateAccessToken(req.Token)
	if err != nil {
		return &userv1.ValidateTokenResponse{
			Valid: false,
		}, nil
	}
	return &userv1.ValidateTokenResponse{
		UserId: claims.UserID,
		Role:   mapRole(claims.Role),
		Valid:  true,
	}, nil
}

// ── Profile Handlers ─────────────────────────────────────────────────────────

func (h *UserHandler) GetUser(ctx context.Context, req *userv1.GetUserRequest) (*userv1.User, error) {
	uid, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	user, err := h.userSvc.GetUser(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	return mapUser(user), nil
}

func (h *UserHandler) GetUserBatch(ctx context.Context, req *userv1.GetUserBatchRequest) (*userv1.GetUserBatchResponse, error) {
	uids := make([]uuid.UUID, 0, len(req.Ids))
	for _, id := range req.Ids {
		uid, err := uuid.Parse(id)
		if err == nil {
			uids = append(uids, uid)
		}
	}
	users, err := h.userSvc.GetUserBatch(ctx, uids)
	if err != nil {
		return nil, status.Error(codes.Internal, "batch retrieval failed")
	}
	resUsers := make(map[string]*userv1.User)
	for k, v := range users {
		resUsers[k.String()] = mapUser(&v)
	}
	return &userv1.GetUserBatchResponse{
		Users: resUsers,
	}, nil
}

func (h *UserHandler) UpdateProfile(ctx context.Context, req *userv1.UpdateProfileRequest) (*userv1.User, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	user, err := h.userSvc.UpdateProfile(ctx, uid, req.FullName, req.AvatarUrl)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to update profile")
	}
	return mapUser(user), nil
}

// ── Address Handlers ─────────────────────────────────────────────────────────

func (h *UserHandler) ListAddresses(ctx context.Context, req *userv1.ListAddressesRequest) (*userv1.ListAddressesResponse, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	addresses, err := h.userSvc.ListAddresses(ctx, uid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list addresses")
	}
	resAddrs := make([]*userv1.Address, len(addresses))
	for i := range addresses {
		resAddrs[i] = mapAddress(&addresses[i])
	}
	return &userv1.ListAddressesResponse{
		Addresses: resAddrs,
	}, nil
}

func (h *UserHandler) CreateAddress(ctx context.Context, req *userv1.CreateAddressRequest) (*userv1.Address, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	addr, err := h.userSvc.CreateAddress(ctx, db.CreateAddressParams{
		UserID:       uid,
		Label:        &req.Label,
		FullName:     req.FullName,
		Phone:        req.Phone,
		AddressLine1: req.AddressLine1,
		AddressLine2: &req.AddressLine2,
		City:         req.City,
		State:        &req.State,
		PostalCode:   &req.PostalCode,
		Country:      req.Country,
		IsDefault:    req.IsDefault,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create address")
	}
	return mapAddress(addr), nil
}

func (h *UserHandler) DeleteAddress(ctx context.Context, req *userv1.DeleteAddressRequest) (*emptypb.Empty, error) {
	uid, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user id")
	}
	aid, err := uuid.Parse(req.AddressId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid address id")
	}
	err = h.userSvc.DeleteAddress(ctx, aid, uid)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to delete address")
	}
	return &emptypb.Empty{}, nil
}

func (h *UserHandler) SendReviewStatusEmail(ctx context.Context, req *userv1.SendReviewStatusEmailRequest) (*emptypb.Empty, error) {
	err := h.authSvc.SendReviewStatusEmail(ctx, req.Email, req.FullName, req.StoreName, req.Status)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send review status email: %v", err)
	}
	return &emptypb.Empty{}, nil
}

// ── Helpers ───────────────────────────────────────────────────────────────────


func mapRole(role string) userv1.UserRole {
	switch role {
	case "buyer":
		return userv1.UserRole_USER_ROLE_BUYER
	case "seller":
		return userv1.UserRole_USER_ROLE_SELLER
	case "admin":
		return userv1.UserRole_USER_ROLE_ADMIN
	default:
		return userv1.UserRole_USER_ROLE_UNSPECIFIED
	}
}

func mapProvider(provider string) userv1.AuthProvider {
	switch provider {
	case "email":
		return userv1.AuthProvider_AUTH_PROVIDER_EMAIL
	case "google":
		return userv1.AuthProvider_AUTH_PROVIDER_GOOGLE
	case "phone":
		return userv1.AuthProvider_AUTH_PROVIDER_PHONE
	default:
		return userv1.AuthProvider_AUTH_PROVIDER_UNSPECIFIED
	}
}

func mapUser(u *db.User) *userv1.User {
	if u == nil {
		return nil
	}
	return &userv1.User{
		Id:           u.ID.String(),
		Email:        getVal(u.Email),
		Phone:        getVal(u.Phone),
		FullName:     u.FullName,
		AvatarUrl:    getVal(u.AvatarUrl),
		Role:         mapRole(u.Role),
		IsVerified:   u.IsVerified,
		AuthProvider: mapProvider(u.AuthProvider),
		CreatedAt:    timestamppb.New(u.CreatedAt),
		UpdatedAt:    timestamppb.New(u.UpdatedAt),
	}
}

func mapAddress(a *db.Address) *userv1.Address {
	if a == nil {
		return nil
	}
	return &userv1.Address{
		Id:           a.ID.String(),
		UserId:       a.UserID.String(),
		Label:        getVal(a.Label),
		FullName:     a.FullName,
		Phone:        a.Phone,
		AddressLine1: a.AddressLine1,
		AddressLine2: getVal(a.AddressLine2),
		City:         a.City,
		State:        getVal(a.State),
		PostalCode:   getVal(a.PostalCode),
		Country:      a.Country,
		IsDefault:    a.IsDefault,
	}
}

func getVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
