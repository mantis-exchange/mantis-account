package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/mantis-exchange/mantis-account/internal/model"
)

type BalanceService struct {
	balances *model.BalanceRepo
}

func NewBalanceService(balances *model.BalanceRepo) *BalanceService {
	return &BalanceService{balances: balances}
}

func (s *BalanceService) GetBalances(ctx context.Context, userID uuid.UUID) ([]model.Balance, error) {
	return s.balances.ListByUser(ctx, userID)
}

func (s *BalanceService) GetBalance(ctx context.Context, userID uuid.UUID, asset string) (*model.Balance, error) {
	return s.balances.GetByUserAndAsset(ctx, userID, asset)
}

func (s *BalanceService) Freeze(ctx context.Context, userID uuid.UUID, asset, amount string) error {
	return s.balances.Freeze(ctx, userID, asset, amount)
}

func (s *BalanceService) Unfreeze(ctx context.Context, userID uuid.UUID, asset, amount string) error {
	return s.balances.Unfreeze(ctx, userID, asset, amount)
}

func (s *BalanceService) Credit(ctx context.Context, userID uuid.UUID, asset, amount string) error {
	return s.balances.Credit(ctx, userID, asset, amount)
}

func (s *BalanceService) DeductFrozen(ctx context.Context, userID uuid.UUID, asset, amount string) error {
	return s.balances.DeductFrozen(ctx, userID, asset, amount)
}
