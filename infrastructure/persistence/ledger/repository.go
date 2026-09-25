package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/persistence/outbox"
	"github.com/shopspring/decimal"
)

var (
	ErrNilDatabase    = errors.New("database connection is nil")
	ErrNilTransaction = errors.New("database transaction is nil")
	ErrNilCallback    = errors.New("transaction callback is nil")
)

type MariaDBRepository struct {
	db *sql.DB
}

func NewMariaDBRepository(db *sql.DB) (*MariaDBRepository, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}
	return &MariaDBRepository{db: db}, nil
}

func (r *MariaDBRepository) ExecuteInTransaction(ctx context.Context, fn func(tx *sql.Tx) error) (err error) {
	if r == nil || r.db == nil {
		return ErrNilDatabase
	}
	if ctx == nil {
		return errors.New("transaction context is nil")
	}
	if fn == nil {
		return ErrNilCallback
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin ledger transaction: %w", err)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = fn(tx); err != nil {
		return fmt.Errorf("ledger transaction callback failed: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit ledger transaction: %w", err)
	}
	return nil
}

func (r *MariaDBRepository) GetAccountForUpdate(ctx context.Context, tx *sql.Tx, accountID string) (*AccountEntity, error) {
	if r == nil || r.db == nil {
		return nil, ErrNilDatabase
	}
	if ctx == nil {
		return nil, errors.New("account lookup context is nil")
	}
	if tx == nil {
		return nil, ErrNilTransaction
	}
	if accountID == "" {
		return nil, errors.New("account ID is empty")
	}

	account := &AccountEntity{}
	// FOR UPDATE holds the account row lock until tx.Commit or tx.Rollback,
	// preventing concurrent authorizations from racing on the same balance.
	err := tx.QueryRowContext(ctx, `
		SELECT id, balance, currency, updated_at
		FROM accounts
		WHERE id = ?
		FOR UPDATE`, accountID).Scan(
		&account.ID,
		&account.Balance,
		&account.Currency,
		&account.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("account %q not found: %w", accountID, sql.ErrNoRows)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to lock account %q: %w", accountID, err)
	}
	return account, nil
}

func (r *MariaDBRepository) UpdateAccountBalance(ctx context.Context, tx *sql.Tx, accountID string, newBalance decimal.Decimal) error {
	if r == nil || r.db == nil {
		return ErrNilDatabase
	}
	if ctx == nil {
		return errors.New("account update context is nil")
	}
	if tx == nil {
		return ErrNilTransaction
	}
	if accountID == "" {
		return errors.New("account ID is empty")
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE accounts
		SET balance = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`, newBalance, accountID)
	if err != nil {
		return fmt.Errorf("failed to update account %q balance: %w", accountID, err)
	}
	if affected, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("failed to verify account %q balance update: %w", accountID, err)
	} else if affected != 1 {
		return fmt.Errorf("account %q was not updated", accountID)
	}
	return nil
}

func (r *MariaDBRepository) CreateLedgerEntry(ctx context.Context, tx *sql.Tx, entry LedgerEntryEntity) error {
	if r == nil || r.db == nil {
		return ErrNilDatabase
	}
	if ctx == nil {
		return errors.New("ledger entry context is nil")
	}
	if tx == nil {
		return ErrNilTransaction
	}
	if entry.ID == "" || entry.AccountID == "" {
		return errors.New("ledger entry ID and account ID are required")
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO ledger_entries (id, transaction_id, account_id, amount, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		entry.ID,
		entry.ReferenceID,
		entry.AccountID,
		entry.Amount,
		entry.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create ledger entry %q: %w", entry.ID, err)
	}
	return nil
}

func (r *MariaDBRepository) CreateLedgerTransaction(ctx context.Context, tx *sql.Tx, transaction LedgerTransactionEntity) error {
	if r == nil || r.db == nil {
		return ErrNilDatabase
	}
	if ctx == nil {
		return errors.New("ledger transaction context is nil")
	}
	if tx == nil {
		return ErrNilTransaction
	}
	if transaction.ID == "" || transaction.IdempotencyKey == "" || transaction.Channel == "" {
		return errors.New("ledger transaction ID, idempotency key, and channel are required")
	}
	if len(transaction.Metadata) == 0 {
		transaction.Metadata = []byte(`{}`)
	}
	if transaction.CreatedAt.IsZero() {
		return errors.New("ledger transaction created at is required")
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO ledger_transactions (id, idempotency_key, channel, metadata, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		transaction.ID,
		transaction.IdempotencyKey,
		transaction.Channel,
		transaction.Metadata,
		transaction.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create ledger transaction %q: %w", transaction.ID, err)
	}
	return nil
}

func (r *MariaDBRepository) CreateOutboxEvent(ctx context.Context, tx *sql.Tx, event outbox.OutboxEventEntity) error {
	if r == nil || r.db == nil {
		return ErrNilDatabase
	}
	if ctx == nil {
		return errors.New("outbox event context is nil")
	}
	if tx == nil {
		return ErrNilTransaction
	}
	if event.ID == "" || event.EventType == "" || len(event.Payload) == 0 {
		return errors.New("outbox event ID, event type, and payload are required")
	}
	if event.Status == "" {
		event.Status = outbox.StatusPending
	}
	if event.Status != outbox.StatusPending {
		return fmt.Errorf("invalid outbox event status %q", event.Status)
	}
	if event.CreatedAt.IsZero() {
		return errors.New("outbox event created at is required")
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO outbox_events (id, event_type, payload, status, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		event.ID,
		event.EventType,
		event.Payload,
		outbox.StatusPending,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create outbox event %q: %w", event.ID, err)
	}
	return nil
}
