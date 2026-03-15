package model

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Balance struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Asset     string    `json:"asset"`
	Available string    `json:"available"`
	Frozen    string    `json:"frozen"`
	UpdatedAt time.Time `json:"updated_at"`
}

type BalanceRepo struct {
	pool *pgxpool.Pool
}

func NewBalanceRepo(pool *pgxpool.Pool) *BalanceRepo {
	return &BalanceRepo{pool: pool}
}

func (r *BalanceRepo) GetByUserAndAsset(ctx context.Context, userID uuid.UUID, asset string) (*Balance, error) {
	b := &Balance{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, asset, available, frozen, updated_at
		 FROM balances WHERE user_id = $1 AND asset = $2`, userID, asset,
	).Scan(&b.ID, &b.UserID, &b.Asset, &b.Available, &b.Frozen, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *BalanceRepo) ListByUser(ctx context.Context, userID uuid.UUID) ([]Balance, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, asset, available, frozen, updated_at
		 FROM balances WHERE user_id = $1 ORDER BY asset`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []Balance
	for rows.Next() {
		var b Balance
		if err := rows.Scan(&b.ID, &b.UserID, &b.Asset, &b.Available, &b.Frozen, &b.UpdatedAt); err != nil {
			return nil, err
		}
		balances = append(balances, b)
	}
	return balances, nil
}

// Freeze moves amount from available to frozen. Used when placing an order.
func (r *BalanceRepo) Freeze(ctx context.Context, userID uuid.UUID, asset string, amount string) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE balances
		 SET available = (available::numeric - $1::numeric)::text,
		     frozen = (frozen::numeric + $1::numeric)::text,
		     updated_at = $2
		 WHERE user_id = $3 AND asset = $4
		   AND available::numeric >= $1::numeric`,
		amount, time.Now(), userID, asset,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient %s balance", asset)
	}
	return nil
}

// Unfreeze moves amount from frozen back to available. Used when cancelling an order.
func (r *BalanceRepo) Unfreeze(ctx context.Context, userID uuid.UUID, asset string, amount string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE balances
		 SET available = (available::numeric + $1::numeric)::text,
		     frozen = (frozen::numeric - $1::numeric)::text,
		     updated_at = $2
		 WHERE user_id = $3 AND asset = $4`,
		amount, time.Now(), userID, asset,
	)
	return err
}

// Credit adds amount to available balance.
func (r *BalanceRepo) Credit(ctx context.Context, userID uuid.UUID, asset string, amount string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO balances (id, user_id, asset, available, frozen, updated_at)
		 VALUES ($1, $2, $3, $4, '0', $5)
		 ON CONFLICT (user_id, asset)
		 DO UPDATE SET available = (balances.available::numeric + $4::numeric)::text, updated_at = $5`,
		uuid.New(), userID, asset, amount, time.Now(),
	)
	return err
}

// DeductFrozen removes amount from frozen balance. Used after trade settlement.
func (r *BalanceRepo) DeductFrozen(ctx context.Context, userID uuid.UUID, asset string, amount string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE balances
		 SET frozen = (frozen::numeric - $1::numeric)::text,
		     updated_at = $2
		 WHERE user_id = $3 AND asset = $4`,
		amount, time.Now(), userID, asset,
	)
	return err
}
