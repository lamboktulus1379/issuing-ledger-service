---
inclusion: always
---
# Clean Architecture Go Structure (Kiro Steering)

When generating code, you MUST place files into the following specific directories:

## 1. Domain Layer (domain/)
- Pure Go code only. No external dependencies, database imports, or struct tags (no JSON, no SQL/GORM).
- domain/model/: Pure Go structs representing business entities.
- domain/repository/: Interfaces ONLY. Defines the contract for data access using pure domain models.
- STRICT RULE: Do NOT place DTOs or DB Data Models in the domain layer.

## 2. Application Layer (usecase/)
- Contains the core business logic (e.g., issuing_usecase.go).
- usecase/dto/ (or inline): Define Commands and Queries here. 
- Depends only on Domain interfaces.

## 3. Infrastructure Layer (infrastructure/)
- Handles external I/O and concrete implementations of Domain interfaces.
- infrastructure/persistence/[feature_name]/: Contains the concrete DB implementation. Must be split into:
  - entities.go: Database structs containing SQL/DB tags.
  - mapper.go: Functions transforming DB entities to/from Domain models.
  - repository.go: The MariaDB queries using database/sql and *sql.Tx.
- infrastructure/cache/: Redis implementation logic.
- infrastructure/message/: Kafka publisher logic.

## 4. Presentation Layer (interfaces/)
- interfaces/http/: HTTP handlers.
- interfaces/http/dto/: Structs representing JSON HTTP requests/responses (contains `json` tags).
