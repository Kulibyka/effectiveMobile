package subscriptions

import (
	"context"
	"errors"
	"log/slog"
	"time"

	domain "github.com/Kulibyka/effective-mobile/internal/domain/subscription"
	"github.com/Kulibyka/effective-mobile/internal/lib/uuid"
)

type Repository interface {
	CreateSubscription(ctx context.Context, input domain.CreateInput) (domain.Subscription, error)
	GetSubscription(ctx context.Context, id uuid.UUID) (domain.Subscription, error)
	UpdateSubscription(ctx context.Context, id uuid.UUID, input domain.UpdateInput) (domain.Subscription, error)
	DeleteSubscription(ctx context.Context, id uuid.UUID) error
	ListSubscriptions(ctx context.Context, filter domain.ListFilter) ([]domain.Subscription, error)
}

type Service struct {
	repo   Repository
	logger *slog.Logger
}

func New(repo Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger.WithGroup("subscriptions_service")}
}

func (s *Service) Create(ctx context.Context, input domain.CreateInput) (domain.Subscription, error) {
	s.logger.InfoContext(ctx, "creating subscription", slog.String("service", input.ServiceName), slog.String("user_id", input.UserID.String()))

	sub, err := s.repo.CreateSubscription(ctx, input)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create subscription", slog.String("user_id", input.UserID.String()), slog.Any("error", err))
		return domain.Subscription{}, err
	}

	return sub, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (domain.Subscription, error) {
	sub, err := s.repo.GetSubscription(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.logger.WarnContext(ctx, "subscription not found", slog.String("subscription_id", id.String()))
		} else {
			s.logger.ErrorContext(ctx, "failed to get subscription", slog.String("subscription_id", id.String()), slog.Any("error", err))
		}
		return domain.Subscription{}, err
	}

	return sub, nil
}

func (s *Service) Update(ctx context.Context, id uuid.UUID, input domain.UpdateInput) (domain.Subscription, error) {
	s.logger.InfoContext(ctx, "updating subscription", slog.String("subscription_id", id.String()))

	sub, err := s.repo.UpdateSubscription(ctx, id, input)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.logger.WarnContext(ctx, "subscription not found", slog.String("subscription_id", id.String()))
		} else {
			s.logger.ErrorContext(ctx, "failed to update subscription", slog.String("subscription_id", id.String()), slog.Any("error", err))
		}
		return domain.Subscription{}, err
	}

	return sub, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	s.logger.InfoContext(ctx, "deleting subscription", slog.String("subscription_id", id.String()))

	if err := s.repo.DeleteSubscription(ctx, id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			s.logger.WarnContext(ctx, "subscription not found", slog.String("subscription_id", id.String()))
		} else {
			s.logger.ErrorContext(ctx, "failed to delete subscription", slog.String("subscription_id", id.String()), slog.Any("error", err))
		}
		return err
	}

	return nil
}

func (s *Service) List(ctx context.Context, filter domain.ListFilter) ([]domain.Subscription, error) {
	subs, err := s.repo.ListSubscriptions(ctx, filter)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list subscriptions", slog.Any("error", err))
		return nil, err
	}

	return subs, nil
}

func (s *Service) Sum(ctx context.Context, input domain.SummaryFilter) (int, error) {
	listFilter := domain.ListFilter{
		UserID:           input.UserID,
		ServiceName:      input.ServiceName,
		ActivePeriodFrom: &input.PeriodStart,
		ActivePeriodTo:   &input.PeriodEnd,
	}

	subs, err := s.repo.ListSubscriptions(ctx, listFilter)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list subscriptions for summary", slog.Any("error", err))
		return 0, err
	}

	total := 0
	for _, sub := range subs {
		overlapStart := maxTime(sub.StartMonth, input.PeriodStart)

		subEnd := input.PeriodEnd
		if sub.EndMonth != nil && sub.EndMonth.Before(subEnd) {
			subEnd = *sub.EndMonth
		}

		if overlapStart.After(subEnd) {
			continue
		}

		months := monthsBetween(overlapStart, subEnd)
		total += sub.Price * months
	}

	return total, nil
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func monthsBetween(start, end time.Time) int {
	y := end.Year() - start.Year()
	m := int(end.Month()) - int(start.Month())
	months := y*12 + m + 1
	if months < 0 {
		return 0
	}
	return months
}
