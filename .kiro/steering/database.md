---
inclusion: fileMatch
fileMatch: "infrastructure/persistence/**/*.go"
---
# MariaDB / MySQL Database Rules

When writing code in the persistence layer:
1. Always pass `context.Context` to every database execution.
2. Use `*sql.Tx` for all financial updates to ensure ACID compliance.
3. Use `SELECT ... FOR UPDATE` to lock rows before applying ledger entry deductions.
4. Verify that `SUM(amount) == 0` in Go logic before calling `tx.Commit()`.
