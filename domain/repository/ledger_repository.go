package repository

import (
	"context"

	"github.com/lamboktulus1379/issuing-ledger-service/domain/model"
)

type LedgerRepository interface {
	FindAccountByID(ctx context.Context, accountID string) (model.Account, error)
	FindAccountForUpdate(ctx context.Context, accountID string) (model.Account, error)
	SaveTransaction(ctx context.Context, transaction model.LedgerTransaction, entries []model.LedgerEntry) error
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
