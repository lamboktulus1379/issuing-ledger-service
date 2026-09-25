package usecase

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	ledgerpersistence "github.com/lamboktulus1379/issuing-ledger-service/infrastructure/persistence/ledger"
	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/persistence/outbox"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel"
)

type Account = ledgerpersistence.AccountEntity
type LedgerEntry = ledgerpersistence.LedgerEntryEntity
type OutboxEvent = outbox.OutboxEventEntity

type LedgerRepository interface {
	ExecuteInTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error
	GetAccountForUpdate(ctx context.Context, tx *sql.Tx, accountID string) (*Account, error)
	UpdateAccountBalance(ctx context.Context, tx *sql.Tx, accountID string, balance decimal.Decimal) error
	CreateLedgerEntry(ctx context.Context, tx *sql.Tx, entry LedgerEntry) error
	CreateLedgerTransaction(ctx context.Context, tx *sql.Tx, transaction ledgerpersistence.LedgerTransactionEntity) error
	CreateOutboxEvent(ctx context.Context, tx *sql.Tx, event OutboxEvent) error
}

type IssuingUsecase struct {
	repository LedgerRepository
}

type AuthorizeRequest struct {
	TransactionID        string
	SourceAccountID      string
	DestinationAccountID string
	Amount               int64
	Currency             string
}

type TransactionAuthorized struct {
	TransactionID        string    `json:"transaction_id"`
	SourceAccountID      string    `json:"source_account_id"`
	DestinationAccountID string    `json:"destination_account_id"`
	Amount               int64     `json:"amount"`
	Currency             string    `json:"currency"`
	AuthorizedAt         time.Time `json:"authorized_at"`
}

var (
	ErrNilLedgerRepository  = errors.New("ledger repository is nil")
	ErrInsufficientFunds    = errors.New("source account has insufficient funds")
	ErrAccountNotFound      = errors.New("account not found")
	ErrInvalidAuthorization = errors.New("invalid authorization request")
	ErrZeroSumViolation     = errors.New("ledger entries do not balance to zero")
)

func NewIssuingUsecase(repository LedgerRepository) (*IssuingUsecase, error) {
	if repository == nil {
		return nil, ErrNilLedgerRepository
	}
	return &IssuingUsecase{repository: repository}, nil
}

func (u *IssuingUsecase) AuthorizePayment(ctx context.Context, req AuthorizeRequest) (err error) {
	if u == nil || u.repository == nil {
		return ErrNilLedgerRepository
	}
	if ctx == nil {
		return errors.New("authorization context is nil")
	}
	if err := validateAuthorizeRequest(req); err != nil {
		return err
	}

	tracer := otel.Tracer("issuing-usecase")
	ctx, span := tracer.Start(ctx, "AuthorizePayment")
	defer func() {
		if err != nil {
			span.RecordError(err)
		}
		span.End()
	}()

	debitAmount := -req.Amount
	creditAmount := req.Amount
	// This is the first zero-sum guard. The same invariant is checked again
	// immediately before persistence so future changes cannot bypass it.
	if debitAmount+creditAmount != 0 {
		return ErrZeroSumViolation
	}

	lockOrder := []string{req.SourceAccountID, req.DestinationAccountID}
	// Every authorization locks accounts in lexical order. Two concurrent
	// transfers therefore acquire overlapping locks in the same order, avoiding
	// the circular wait that causes database deadlocks.
	sort.Strings(lockOrder)

	return u.repository.ExecuteInTransaction(ctx, func(tx *sql.Tx) error {
		accounts := make(map[string]*Account, len(lockOrder))
		for _, accountID := range lockOrder {
			account, err := u.repository.GetAccountForUpdate(ctx, tx, accountID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("%w: %s", ErrAccountNotFound, accountID)
				}
				return fmt.Errorf("failed to lock account %q: %w", accountID, err)
			}
			if account == nil {
				return fmt.Errorf("failed to lock account %q: nil account", accountID)
			}
			accounts[accountID] = account
		}

		source := accounts[req.SourceAccountID]
		destination := accounts[req.DestinationAccountID]
		amount := decimal.NewFromInt(req.Amount)
		if source.Balance.LessThan(amount) {
			return fmt.Errorf("%w: account %q", ErrInsufficientFunds, req.SourceAccountID)
		}
		if source.Currency != destination.Currency {
			return fmt.Errorf("%w: account currencies differ", ErrInvalidAuthorization)
		}
		if req.Currency != "" && (source.Currency != req.Currency || destination.Currency != req.Currency) {
			return fmt.Errorf("%w: request currency does not match account currency", ErrInvalidAuthorization)
		}
		newSourceBalance := source.Balance.Sub(amount)
		newDestinationBalance := destination.Balance.Add(amount)
		if err := u.repository.CreateLedgerTransaction(ctx, tx, ledgerpersistence.LedgerTransactionEntity{
			ID:             req.TransactionID,
			IdempotencyKey: req.TransactionID,
			Channel:        "API",
			Metadata:       []byte(`{}`),
			CreatedAt:      time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("failed to create ledger transaction: %w", err)
		}
		if err := u.repository.UpdateAccountBalance(ctx, tx, source.ID, newSourceBalance); err != nil {
			return fmt.Errorf("failed to update source account balance: %w", err)
		}
		if err := u.repository.UpdateAccountBalance(ctx, tx, destination.ID, newDestinationBalance); err != nil {
			return fmt.Errorf("failed to update destination account balance: %w", err)
		}

		now := time.Now().UTC()
		debitEntry := LedgerEntry{
			ID:          uuid.NewString(),
			AccountID:   req.SourceAccountID,
			Amount:      decimal.NewFromInt(debitAmount),
			EntryType:   "DEBIT",
			ReferenceID: req.TransactionID,
			CreatedAt:   now,
		}
		creditEntry := LedgerEntry{
			ID:          uuid.NewString(),
			AccountID:   req.DestinationAccountID,
			Amount:      decimal.NewFromInt(creditAmount),
			EntryType:   "CREDIT",
			ReferenceID: req.TransactionID,
			CreatedAt:   now,
		}
		if !debitEntry.Amount.Add(creditEntry.Amount).IsZero() {
			return ErrZeroSumViolation
		}
		if err := u.repository.CreateLedgerEntry(ctx, tx, debitEntry); err != nil {
			return fmt.Errorf("failed to create debit ledger entry: %w", err)
		}
		if err := u.repository.CreateLedgerEntry(ctx, tx, creditEntry); err != nil {
			return fmt.Errorf("failed to create credit ledger entry: %w", err)
		}

		payload, err := json.Marshal(TransactionAuthorized{
			TransactionID:        req.TransactionID,
			SourceAccountID:      req.SourceAccountID,
			DestinationAccountID: req.DestinationAccountID,
			Amount:               req.Amount,
			Currency:             source.Currency,
			AuthorizedAt:         now,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal transaction authorized event: %w", err)
		}
		if err := u.repository.CreateOutboxEvent(ctx, tx, OutboxEvent{
			ID:        uuid.NewString(),
			EventType: "TransactionAuthorized",
			Payload:   payload,
			Status:    outbox.StatusPending,
			CreatedAt: now,
		}); err != nil {
			return fmt.Errorf("failed to create transaction authorized outbox event: %w", err)
		}
		return nil
	})
}

func validateAuthorizeRequest(req AuthorizeRequest) error {
	if req.TransactionID == "" || req.SourceAccountID == "" || req.DestinationAccountID == "" {
		return ErrInvalidAuthorization
	}
	if req.SourceAccountID == req.DestinationAccountID || req.Amount <= 0 {
		return ErrInvalidAuthorization
	}
	return nil
}
