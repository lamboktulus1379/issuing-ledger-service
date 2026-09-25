-- Create accounts table
CREATE TABLE accounts (
    id VARCHAR(36) PRIMARY KEY,
    corporate_client_id VARCHAR(36) NOT NULL,
    type VARCHAR(30) NOT NULL,
    currency VARCHAR(10) NOT NULL,
    balance DECIMAL(36, 18) NOT NULL DEFAULT 0.000000000000000000,
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

INSERT INTO accounts (id, corporate_client_id, type, currency, balance, status) VALUES
-- 1. The primary test account (matches your earlier curl payload)
('acc_123456', 'corp-client-1111', 'ISSUING_PREPAID', 'USD', 500.00, 'ACTIVE'),

-- 2. A high-balance corporate float account using standard UUIDs
('550e8400-e29b-41d4-a716-446655440000', 'corp-client-1111', 'CORPORATE_FLOAT', 'USD', 250000.00, 'ACTIVE'),

-- 3. An empty account to test "insufficient funds" Usecase logic
('660e8400-e29b-41d4-a716-446655441111', 'corp-client-2222', 'ISSUING_PREPAID', 'USD', 0.00, 'ACTIVE'),

-- 4. A suspended account to test "account status" rejection logic
('770e8400-e29b-41d4-a716-446655442222', 'corp-client-2222', 'ISSUING_CREDIT', 'USD', 150.75, 'SUSPENDED'),

-- 5. An alternate currency account to test currency mismatch validation
('880e8400-e29b-41d4-a716-446655443333', 'corp-client-3333', 'ISSUING_CHECKING', 'IDR', 15000000.00, 'ACTIVE');