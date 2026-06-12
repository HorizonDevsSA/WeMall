package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wemall/admin-service/internal/db"
	adminv1 "github.com/wemall/gen/admin/v1"
	sellerv1 "github.com/wemall/gen/seller/v1"
	werr "github.com/wemall/pkg/errors"
)

type AdminService struct {
	q            *db.Queries
	adminPool    *pgxpool.Pool
	usersPool    *pgxpool.Pool
	sellersPool  *pgxpool.Pool
	disputesPool *pgxpool.Pool
	ordersPool   *pgxpool.Pool
	sellerClient sellerv1.SellerServiceClient
}

func NewAdminService(
	q *db.Queries,
	adminPool *pgxpool.Pool,
	usersPool *pgxpool.Pool,
	sellersPool *pgxpool.Pool,
	disputesPool *pgxpool.Pool,
	ordersPool *pgxpool.Pool,
	sellerClient sellerv1.SellerServiceClient,
) *AdminService {
	return &AdminService{
		q:            q,
		adminPool:    adminPool,
		usersPool:    usersPool,
		sellersPool:  sellersPool,
		disputesPool: disputesPool,
		ordersPool:   ordersPool,
		sellerClient: sellerClient,
	}
}

func (s *AdminService) SuspendSeller(ctx context.Context, sellerIDStr, reason, adminIDStr string) error {
	sellerID, err := uuid.Parse(sellerIDStr)
	if err != nil {
		return werr.InvalidArgument("invalid seller ID format")
	}
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		return werr.InvalidArgument("invalid admin ID format")
	}

	tx, err := s.adminPool.Begin(ctx)
	if err != nil {
		return werr.Internal(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	_, err = qtx.CreateSellerSuspension(ctx, db.CreateSellerSuspensionParams{
		SellerID: sellerID,
		Reason:   reason,
		AdminID:  adminID,
	})
	if err != nil {
		return werr.Internal(err)
	}

	// Update the seller status in seller-service
	_, err = s.sellerClient.UpdateSellerStatus(ctx, &sellerv1.UpdateSellerStatusRequest{
		SellerId: sellerIDStr,
		Status:   sellerv1.SellerStatus_SELLER_STATUS_SUSPENDED,
	})
	if err != nil {
		return werr.Internal(fmt.Errorf("failed to update seller status via gRPC: %w", err))
	}

	return tx.Commit(ctx)
}

func (s *AdminService) BanUser(ctx context.Context, userIDStr, reason, adminIDStr string) error {
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return werr.InvalidArgument("invalid user ID format")
	}
	adminID, err := uuid.Parse(adminIDStr)
	if err != nil {
		return werr.InvalidArgument("invalid admin ID format")
	}

	tx, err := s.adminPool.Begin(ctx)
	if err != nil {
		return werr.Internal(err)
	}
	defer tx.Rollback(ctx)

	qtx := s.q.WithTx(tx)

	_, err = qtx.CreateUserBan(ctx, db.CreateUserBanParams{
		UserID:  userID,
		Reason:  reason,
		AdminID: adminID,
	})
	if err != nil {
		return werr.Internal(err)
	}

	// Soft delete the user directly in users DB
	_, err = s.usersPool.Exec(ctx, "UPDATE users SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL", userID)
	if err != nil {
		return werr.Internal(fmt.Errorf("failed to soft-delete user in user-service DB: %w", err))
	}

	return tx.Commit(ctx)
}

func (s *AdminService) GetPlatformMetrics(ctx context.Context) (*adminv1.PlatformMetrics, error) {
	var totalUsers int64
	var totalSellers int64
	var activeDisputes int64
	var totalOrders int64

	// Query total users (excluding soft-deleted ones)
	err := s.usersPool.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&totalUsers)
	if err != nil {
		return nil, werr.Internal(fmt.Errorf("failed to query total users: %w", err))
	}

	// Query total sellers
	err = s.sellersPool.QueryRow(ctx, "SELECT COUNT(*) FROM sellers").Scan(&totalSellers)
	if err != nil {
		return nil, werr.Internal(fmt.Errorf("failed to query total sellers: %w", err))
	}

	// Query active disputes
	err = s.disputesPool.QueryRow(ctx, "SELECT COUNT(*) FROM disputes WHERE status IN ('DISPUTE_STATUS_OPEN', 'DISPUTE_STATUS_ESCALATED')").Scan(&activeDisputes)
	if err != nil {
		// If disputes table does not exist or has issue, default to 0 rather than failing the whole metrics query
		activeDisputes = 0
	}

	// Query total orders
	err = s.ordersPool.QueryRow(ctx, "SELECT COUNT(*) FROM orders").Scan(&totalOrders)
	if err != nil {
		return nil, werr.Internal(fmt.Errorf("failed to query total orders: %w", err))
	}

	return &adminv1.PlatformMetrics{
		TotalUsers:     int32(totalUsers),
		TotalSellers:   int32(totalSellers),
		ActiveDisputes: int32(activeDisputes),
		TotalOrders:    int32(totalOrders),
	}, nil
}
