package ledger

import "github.com/lamboktulus1379/issuing-ledger-service/domain/model"

func AccountEntityToModel(entity AccountEntity) model.Account {
	return model.Account{
		ID:            entity.ID,
		Currency:      entity.Currency,
		CachedBalance: entity.Balance,
		UpdatedAt:     entity.UpdatedAt,
	}
}

func AccountModelToEntity(account model.Account) (AccountEntity, error) {
	return AccountEntity{
		ID:        account.ID,
		Balance:   account.CachedBalance,
		Currency:  account.Currency,
		UpdatedAt: account.UpdatedAt,
	}, nil
}

func LedgerEntryEntityToModel(entity LedgerEntryEntity) model.LedgerEntry {
	return model.LedgerEntry{
		ID:            entity.ID,
		TransactionID: entity.ReferenceID,
		AccountID:     entity.AccountID,
		Amount:        entity.Amount,
		CreatedAt:     entity.CreatedAt,
	}
}

func LedgerEntryModelToEntity(entry model.LedgerEntry) (LedgerEntryEntity, error) {
	return LedgerEntryEntity{
		ID:          entry.ID,
		AccountID:   entry.AccountID,
		Amount:      entry.Amount,
		ReferenceID: entry.TransactionID,
		CreatedAt:   entry.CreatedAt,
	}, nil
}
