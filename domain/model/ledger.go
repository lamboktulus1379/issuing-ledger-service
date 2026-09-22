package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type Account struct {
	ID                string
	CorporateClientID string
	Type              string
	Currency          string
	CachedBalance     decimal.Decimal
	Status            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type LedgerTransaction struct {
	ID             string
	IdempotencyKey string
	Channel        string
	Metadata       map[string]interface{}
	CreatedAt      time.Time
}

type LedgerEntry struct {
	ID            string
	TransactionID string
	AccountID     string
	Amount        decimal.Decimal
	CreatedAt     time.Time
}
