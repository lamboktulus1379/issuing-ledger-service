package ledger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mariadb"
)

var (
	integrationDB         *sql.DB
	integrationRepository *MariaDBRepository
	integrationSetupErr   error
	integrationContainer  *mariadb.MariaDBContainer
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	integrationContainer, integrationSetupErr = mariadb.Run(
		ctx,
		"mariadb:11.4",
		mariadb.WithDatabase("ledger_test"),
		mariadb.WithUsername("ledger_test"),
		mariadb.WithPassword("ledger_test_password"),
	)
	if integrationSetupErr == nil {
		var connectionString string
		connectionString, integrationSetupErr = integrationContainer.ConnectionString(ctx, "parseTime=true")
		if integrationSetupErr == nil {
			integrationDB, integrationSetupErr = sql.Open("mysql", connectionString)
		}
		if integrationSetupErr == nil {
			integrationDB.SetMaxOpenConns(10)
			integrationDB.SetMaxIdleConns(10)
			integrationSetupErr = integrationDB.PingContext(ctx)
		}
		if integrationSetupErr == nil {
			integrationSetupErr = createIntegrationSchema(ctx, integrationDB)
		}
		if integrationSetupErr == nil {
			integrationRepository, integrationSetupErr = NewMariaDBRepository(integrationDB)
		}
	}

	exitCode := m.Run()

	if integrationDB != nil {
		_ = integrationDB.Close()
	}
	if integrationContainer != nil {
		_ = integrationContainer.Terminate(ctx)
	}
	os.Exit(exitCode)
}

func createIntegrationSchema(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			balance DECIMAL(36,18) NOT NULL,
			currency VARCHAR(3) NOT NULL,
			updated_at DATETIME(6) NOT NULL
		) ENGINE=InnoDB`,
		`CREATE TABLE IF NOT EXISTS ledger_transactions (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			idempotency_key VARCHAR(255) NOT NULL UNIQUE,
			channel VARCHAR(50) NOT NULL,
			metadata JSON,
			created_at DATETIME(6) NOT NULL
		) ENGINE=InnoDB`,
		`CREATE TABLE IF NOT EXISTS ledger_entries (
			id VARCHAR(64) NOT NULL PRIMARY KEY,
			transaction_id VARCHAR(64) NOT NULL,
			account_id VARCHAR(64) NOT NULL,
			amount DECIMAL(36,18) NOT NULL,
			created_at DATETIME(6) NOT NULL,
			CONSTRAINT fk_ledger_entries_transaction
				FOREIGN KEY (transaction_id) REFERENCES ledger_transactions (id),
			CONSTRAINT fk_ledger_entries_account
				FOREIGN KEY (account_id) REFERENCES accounts (id)
		) ENGINE=InnoDB`,
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("failed to create integration schema: %w", err)
		}
	}
	return nil
}

func requireIntegrationDatabase(t *testing.T) {
	t.Helper()
	if integrationSetupErr != nil {
		t.Skipf("MariaDB integration environment unavailable: %v", integrationSetupErr)
	}
	require.NotNil(t, integrationDB)
	require.NotNil(t, integrationRepository)
}

func resetIntegrationData(t *testing.T, ctx context.Context) {
	t.Helper()
	_, err := integrationDB.ExecContext(ctx, "DELETE FROM ledger_entries")
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, "DELETE FROM accounts")
	require.NoError(t, err)
}

func TestExecuteInTransaction_Success(t *testing.T) {
	requireIntegrationDatabase(t)

	tests := []struct {
		name      string
		accountID string
		entryID   string
	}{
		{
			name:      "deducts balance and writes debit entry",
			accountID: "account-success-001",
			entryID:   "entry-success-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			// Arrange
			resetIntegrationData(t, ctx)
			_, err := integrationDB.ExecContext(ctx,
				"INSERT INTO accounts (id, balance, currency, updated_at) VALUES (?, ?, ?, ?)",
				tt.accountID, int64(1000), "USD", time.Now().UTC())
			require.NoError(t, err)
			_, err = integrationDB.ExecContext(ctx,
				"INSERT INTO ledger_transactions (id, idempotency_key, channel, metadata, created_at) VALUES (?, ?, ?, ?, ?)",
				"authorization-success-001", "idem-success-001", "TEST", "{}", time.Now().UTC())
			require.NoError(t, err)

			// Act
			err = integrationRepository.ExecuteInTransaction(ctx, func(tx *sql.Tx) error {
				account, err := integrationRepository.GetAccountForUpdate(ctx, tx, tt.accountID)
				if err != nil {
					return err
				}
				if !account.Balance.Equal(decimal.NewFromInt(1000)) {
					return fmt.Errorf("expected locked balance 1000, got %s", account.Balance)
				}
				if err := integrationRepository.UpdateAccountBalance(ctx, tx, tt.accountID, decimal.NewFromInt(500)); err != nil {
					return err
				}
				return integrationRepository.CreateLedgerEntry(ctx, tx, LedgerEntryEntity{
					ID:          tt.entryID,
					AccountID:   tt.accountID,
					Amount:      decimal.NewFromInt(-500),
					EntryType:   "DEBIT",
					ReferenceID: "authorization-success-001",
					CreatedAt:   time.Now().UTC(),
				})
			})

			// Assert
			require.NoError(t, err)
			var balance decimal.Decimal
			err = integrationDB.QueryRowContext(ctx,
				"SELECT balance FROM accounts WHERE id = ?", tt.accountID).Scan(&balance)
			require.NoError(t, err)
			assert.True(t, balance.Equal(decimal.NewFromInt(500)))

			var entryAmount decimal.Decimal
			var transactionID string
			err = integrationDB.QueryRowContext(ctx,
				"SELECT amount, transaction_id FROM ledger_entries WHERE id = ?", tt.entryID).
				Scan(&entryAmount, &transactionID)
			require.NoError(t, err)
			assert.True(t, entryAmount.Equal(decimal.NewFromInt(-500)))
			assert.Equal(t, "authorization-success-001", transactionID)
		})
	}
}

func TestExecuteInTransaction_RollbackOnError(t *testing.T) {
	requireIntegrationDatabase(t)

	tests := []struct {
		name      string
		accountID string
	}{
		{
			name:      "restores balance after callback error",
			accountID: "account-rollback-001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			deliberateErr := errors.New("deliberate downstream failure")

			// Arrange
			resetIntegrationData(t, ctx)
			_, err := integrationDB.ExecContext(ctx,
				"INSERT INTO accounts (id, balance, currency, updated_at) VALUES (?, ?, ?, ?)",
				tt.accountID, int64(1000), "USD", time.Now().UTC())
			require.NoError(t, err)

			// Act
			err = integrationRepository.ExecuteInTransaction(ctx, func(tx *sql.Tx) error {
				if _, err := integrationRepository.GetAccountForUpdate(ctx, tx, tt.accountID); err != nil {
					return err
				}
				if err := integrationRepository.UpdateAccountBalance(ctx, tx, tt.accountID, decimal.NewFromInt(500)); err != nil {
					return err
				}
				return deliberateErr
			})

			// Assert
			require.ErrorIs(t, err, deliberateErr)
			var balance decimal.Decimal
			err = integrationDB.QueryRowContext(ctx,
				"SELECT balance FROM accounts WHERE id = ?", tt.accountID).Scan(&balance)
			require.NoError(t, err)
			assert.True(t, balance.Equal(decimal.NewFromInt(1000)))
		})
	}
}
