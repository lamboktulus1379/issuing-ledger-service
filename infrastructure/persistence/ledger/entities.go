package ledger

import (
	"time"

	"github.com/shopspring/decimal"
)

type AccountEntity struct {
	ID        string          `db:"id"`
	Balance   decimal.Decimal `db:"balance"`
	Currency  string          `db:"currency"`
	UpdatedAt time.Time       `db:"updated_at"`
}

type LedgerEntryEntity struct {
	ID          string          `db:"id"`
	AccountID   string          `db:"account_id"`
	Amount      decimal.Decimal `db:"amount"`
	EntryType   string          `db:"entry_type"`
	ReferenceID string          `db:"reference_id"`
	CreatedAt   time.Time       `db:"created_at"`
}

type LedgerTransactionEntity struct {
	ID             string
	IdempotencyKey string
	Channel        string
	Metadata       []byte
	CreatedAt      time.Time
}
