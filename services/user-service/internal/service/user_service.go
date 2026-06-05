package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/wemall/user-service/internal/db"
)

type UserService struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

func NewUserService(q *db.Queries, pool *pgxpool.Pool) *UserService {
	return &UserService{q: q, pool: pool}
}

func (s *UserService) GetUser(ctx context.Context, id uuid.UUID) (*db.User, error) {
	user, err := s.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &user, nil
}

func (s *UserService) GetUserBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]db.User, error) {
	users, err := s.q.GetUsersByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("get users batch: %w", err)
	}
	result := make(map[uuid.UUID]db.User)
	for _, u := range users {
		result[u.ID] = u
	}
	return result, nil
}

func (s *UserService) UpdateProfile(ctx context.Context, id uuid.UUID, fullName, avatarURL string) (*db.User, error) {
	user, err := s.q.UpdateUser(ctx, db.UpdateUserParams{
		ID:        id,
		FullName:  fullName,
		AvatarUrl: avatarURL,
	})
	if err != nil {
		return nil, fmt.Errorf("update user profile: %w", err)
	}
	return &user, nil
}

func (s *UserService) ListAddresses(ctx context.Context, userID uuid.UUID) ([]db.Address, error) {
	addresses, err := s.q.ListAddressesByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	return addresses, nil
}

func (s *UserService) CreateAddress(ctx context.Context, arg db.CreateAddressParams) (*db.Address, error) {
	if arg.IsDefault {
		// Run in transaction to unset previous default addresses
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return nil, fmt.Errorf("begin transaction: %w", err)
		}
		defer tx.Rollback(ctx)

		qtx := s.q.WithTx(tx)
		if err := qtx.UnsetDefaultAddresses(ctx, arg.UserID); err != nil {
			return nil, fmt.Errorf("unset default addresses: %w", err)
		}

		addr, err := qtx.CreateAddress(ctx, arg)
		if err != nil {
			return nil, fmt.Errorf("create default address: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit transaction: %w", err)
		}
		return &addr, nil
	}

	addr, err := s.q.CreateAddress(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("create address: %w", err)
	}
	return &addr, nil
}

func (s *UserService) DeleteAddress(ctx context.Context, addressID, userID uuid.UUID) error {
	err := s.q.DeleteAddress(ctx, db.DeleteAddressParams{
		ID:     addressID,
		UserID: userID,
	})
	if err != nil {
		return fmt.Errorf("delete address: %w", err)
	}
	return nil
}
