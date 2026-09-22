---
inclusion: always
---
# Go Engineering Standards & Stack (Kiro Steering)

## 1. Core Stack & Infrastructure
- Language: Go 1.26.
- Architecture: Strict 4-layer Clean Architecture.
- Database: MariaDB 11.4+ or MySQL 8.0+ (InnoDB). NEVER use SQL Server or PostgreSQL.
- Caching: Redis (via `go-redis/v9`).
- Observability: OpenTelemetry (OTLP).

## 2. Go Coding Rules
- Financial Precision: NEVER use `float64` for money. You MUST use `[github.com/shopspring/decimal](https://github.com/shopspring/decimal)`.
- Database Transactions: Use pessimistic locking via `SELECT ... FOR UPDATE`.
- Redis Conventions: Keys MUST follow the format: `[Entity]:[Identifier]:[Context]`.
