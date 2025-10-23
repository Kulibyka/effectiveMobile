package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kulibyka/effective-mobile/internal/lib/uuid"

	domain "github.com/Kulibyka/effective-mobile/internal/domain/subscription"
)

const baseSelect = "SELECT id, service_name, price, user_id, start_month, end_month FROM subscriptions"

func (s *Storage) CreateSubscription(ctx context.Context, input domain.CreateInput) (domain.Subscription, error) {
	const op = "storage.postgresql.CreateSubscription"

	query := `INSERT INTO subscriptions (service_name, price, user_id, start_month, end_month)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, service_name, price, user_id, start_month, end_month`

	var sub domain.Subscription
	err := s.db.QueryRowContext(ctx, query,
		input.ServiceName,
		input.Price,
		input.UserID,
		input.StartMonth,
		sqlNullTime(input.EndMonth),
	).Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartMonth, &sub.EndMonth)
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("%s: %w", op, err)
	}

	return sub, nil
}

func (s *Storage) GetSubscription(ctx context.Context, id uuid.UUID) (domain.Subscription, error) {
	const op = "storage.postgresql.GetSubscription"

	query := baseSelect + " WHERE id = $1"

	var sub domain.Subscription
	err := s.db.QueryRowContext(ctx, query, id).Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartMonth, &sub.EndMonth)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Subscription{}, domain.ErrNotFound
		}
		return domain.Subscription{}, fmt.Errorf("%s: %w", op, err)
	}

	return sub, nil
}

func (s *Storage) UpdateSubscription(ctx context.Context, id uuid.UUID, input domain.UpdateInput) (domain.Subscription, error) {
	const op = "storage.postgresql.UpdateSubscription"

	query := `UPDATE subscriptions
SET service_name = $1,
    price = $2,
    start_month = $3,
    end_month = $4
WHERE id = $5
RETURNING id, service_name, price, user_id, start_month, end_month`

	var sub domain.Subscription
	err := s.db.QueryRowContext(ctx, query,
		input.ServiceName,
		input.Price,
		input.StartMonth,
		sqlNullTime(input.EndMonth),
		id,
	).Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartMonth, &sub.EndMonth)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Subscription{}, domain.ErrNotFound
		}
		return domain.Subscription{}, fmt.Errorf("%s: %w", op, err)
	}

	return sub, nil
}

func (s *Storage) DeleteSubscription(ctx context.Context, id uuid.UUID) error {
	const op = "storage.postgresql.DeleteSubscription"

	res, err := s.db.ExecContext(ctx, "DELETE FROM subscriptions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if affected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (s *Storage) ListSubscriptions(ctx context.Context, filter domain.ListFilter) ([]domain.Subscription, error) {
	const op = "storage.postgresql.ListSubscriptions"

	query := baseSelect
	var conditions []string
	var args []any

	if filter.UserID != nil {
		args = append(args, *filter.UserID)
		conditions = append(conditions, fmt.Sprintf("user_id = $%d", len(args)))
	}

	if filter.ServiceName != nil {
		args = append(args, *filter.ServiceName)
		conditions = append(conditions, fmt.Sprintf("service_name = $%d", len(args)))
	}

	if filter.StartMonthFrom != nil {
		args = append(args, *filter.StartMonthFrom)
		conditions = append(conditions, fmt.Sprintf("start_month >= $%d", len(args)))
	}

	if filter.StartMonthTo != nil {
		args = append(args, *filter.StartMonthTo)
		conditions = append(conditions, fmt.Sprintf("start_month <= $%d", len(args)))
	}

	if filter.ActivePeriodFrom != nil && filter.ActivePeriodTo != nil {
		args = append(args, *filter.ActivePeriodTo)
		conditions = append(conditions, fmt.Sprintf("start_month <= $%d", len(args)))

		args = append(args, *filter.ActivePeriodFrom)
		conditions = append(conditions, fmt.Sprintf("(end_month IS NULL OR end_month >= $%d)", len(args)))
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY start_month"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", filter.Limit)
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", filter.Offset)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	defer rows.Close()

	var result []domain.Subscription
	for rows.Next() {
		var sub domain.Subscription
		if err := rows.Scan(&sub.ID, &sub.ServiceName, &sub.Price, &sub.UserID, &sub.StartMonth, &sub.EndMonth); err != nil {
			return nil, fmt.Errorf("%s: %w", op, err)
		}
		result = append(result, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil
}

func sqlNullTime(t *time.Time) any {
	if t == nil {
		return sql.NullTime{}
	}

	return sql.NullTime{Time: *t, Valid: true}
}
