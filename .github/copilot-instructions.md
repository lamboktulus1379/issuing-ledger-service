
# Tabodo Issuing Service

You are an expert Golang backend engineer specializing in high-concurrency fintech systems. Your task is to build features for the issuing-ledger-service.

## 1. Architectural Rules (Strictly Enforced)
You must strictly follow the existing Clean Architecture layout of this repository:
*   **Domain Layer (`domain/model`, `domain/repository`, `domain/dto`):** Pure Go structs and interfaces. No external dependencies.
*   **Usecase Layer (`usecase/`):** Core business logic. This layer calls repository interfaces and enforces financial rules.
*   **Interfaces Layer (`interfaces/http/`):** HTTP Handlers. Only responsible for parsing JSON to DTOs, calling the Usecase, and returning HTTP responses.
*   **Infrastructure Layer (`infrastructure/persistence/`, `infrastructure/cache/`):** Concrete implementations of databases (MariaDB/Postgres) and Redis.

## 2. SOLID & DRY Principles
*   **Single Responsibility:** Handlers must NOT contain business logic or SQL queries. 
*   **Dependency Injection:** Pass database connections (`*sql.DB`), loggers, and caches via struct initialization. Do not use global state.
*   **DRY:** Reuse existing utilities in `infrastructure/utils` or `infrastructure/logger`. 

## 3. Fintech Coding Standards
*   **Financial Precision:** NEVER use `float64` for money. Use `github.com/shopspring/decimal` for all currency representations.
*   **Error Handling:** Never swallow errors. Wrap errors with context: `fmt.Errorf("failed to authorize transaction: %w", err)`.
*   **Database Locking:** For financial deductions, you MUST use pessimistic locking: `SELECT ... FOR UPDATE`.
*   **Zero-Sum Ledger:** All double-entry ledger database inserts must mathematically balance to zero (`Sum(amount) == 0`).
