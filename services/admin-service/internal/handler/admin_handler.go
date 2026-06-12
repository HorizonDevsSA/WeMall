package handler

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/wemall/admin-service/internal/service"
	adminv1 "github.com/wemall/gen/admin/v1"
)

type AdminHandler struct {
	adminv1.UnimplementedAdminServiceServer
	adminSvc *service.AdminService
}

func NewAdminHandler(adminSvc *service.AdminService) *AdminHandler {
	return &AdminHandler{
		adminSvc: adminSvc,
	}
}

func (h *AdminHandler) SuspendSeller(ctx context.Context, req *adminv1.SuspendSellerRequest) (*emptypb.Empty, error) {
	err := h.adminSvc.SuspendSeller(ctx, req.SellerId, req.Reason, req.AdminId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *AdminHandler) BanUser(ctx context.Context, req *adminv1.BanUserRequest) (*emptypb.Empty, error) {
	err := h.adminSvc.BanUser(ctx, req.UserId, req.Reason, req.AdminId)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return &emptypb.Empty{}, nil
}

func (h *AdminHandler) GetPlatformMetrics(ctx context.Context, req *emptypb.Empty) (*adminv1.PlatformMetrics, error) {
	metrics, err := h.adminSvc.GetPlatformMetrics(ctx)
	if err != nil {
		return nil, toGRPCError(err)
	}
	return metrics, nil
}

func toGRPCError(err error) error {
	if _, ok := status.FromError(err); ok {
		return err
	}
	return status.Error(codes.Internal, err.Error())
}
