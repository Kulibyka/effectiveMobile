package subscription

import (
	"errors"
	"time"

	"github.com/Kulibyka/effective-mobile/internal/lib/uuid"
)

var ErrNotFound = errors.New("subscription not found")

const MonthLayout = "01-2006"

type Subscription struct {
	ID          uuid.UUID
	ServiceName string
	Price       int
	UserID      uuid.UUID
	StartMonth  time.Time
	EndMonth    *time.Time
}

type CreateInput struct {
	ServiceName string
	Price       int
	UserID      uuid.UUID
	StartMonth  time.Time
	EndMonth    *time.Time
}

type UpdateInput struct {
	ServiceName string
	Price       int
	StartMonth  time.Time
	EndMonth    *time.Time
}

type ListFilter struct {
	UserID           *uuid.UUID
	ServiceName      *string
	StartMonthFrom   *time.Time
	StartMonthTo     *time.Time
	ActivePeriodFrom *time.Time
	ActivePeriodTo   *time.Time
	Limit            int
	Offset           int
}

type SummaryFilter struct {
	UserID      *uuid.UUID
	ServiceName *string
	PeriodStart time.Time
	PeriodEnd   time.Time
}
