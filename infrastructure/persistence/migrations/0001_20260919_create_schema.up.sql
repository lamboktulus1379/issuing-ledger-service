-- Create accounts table
CREATE TABLE accounts (
    id VARCHAR(36) PRIMARY KEY,
    corporate_client_id VARCHAR(36) NOT NULL,
    type VARCHAR(30) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    cached_balance DECIMAL(36, 18) NOT NULL DEFAULT 0.000000000000000000,
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_corporate_client (corporate_client_id)
) ENGINE=InnoDB;

-- Create ledger_transactions table
CREATE TABLE ledger_transactions (
    id VARCHAR(36) PRIMARY KEY,
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    channel VARCHAR(50) NOT NULL,
    metadata JSON,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

-- Create ledger_entries table
CREATE TABLE ledger_entries (
    id VARCHAR(36) PRIMARY KEY,
    transaction_id VARCHAR(36) NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    amount DECIMAL(36, 18) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_ledger_entries_transactions FOREIGN KEY (transaction_id) REFERENCES ledger_transactions(id),
    CONSTRAINT fk_ledger_entries_accounts FOREIGN KEY (account_id) REFERENCES accounts(id),
    INDEX idx_account_created (account_id, created_at)
) ENGINE=InnoDB;
