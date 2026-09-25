package outbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrNilDatabase    = errors.New("database connection is nil")
	ErrNilTransaction = errors.New("database transaction is nil")
	ErrEventNotFound  = errors.New("outbox event not found or already processed")
)

// Repository persists outbox events. CreateEvent must receive the same
// transaction as the ledger mutation so both changes commit or roll back together.
type Repository struct{}

func (r *Repository) CreateEvent(ctx context.Context, tx *sql.Tx, event OutboxEventEntity) error {
	if ctx == nil {
		return errors.New("outbox event context is nil")
	}
	if tx == nil {
		return ErrNilTransaction
	}
	if event.ID == "" || event.EventType == "" {
		return errors.New("outbox event ID and event type are required")
	}
	if event.Status == "" {
		event.Status = StatusPending
	}
	if event.Status != StatusPending {
		return fmt.Errorf("invalid outbox event status %q", event.Status)
	}
	if len(event.Payload) == 0 {
		return errors.New("outbox event payload is empty")
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
		StatusPending,
		event.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create outbox event %q: %w", event.ID, err)
	}
	return nil
}

func (r *Repository) GetPendingEvents(ctx context.Context, db *sql.DB, limit int) ([]OutboxEventEntity, error) {
	if ctx == nil {
		return nil, errors.New("outbox event context is nil")
	}
	if db == nil {
		return nil, ErrNilDatabase
	}
	if limit <= 0 {
		return nil, errors.New("outbox event limit must be greater than zero")
	}

	rows, err := db.QueryContext(ctx, `
		SELECT id, event_type, payload, status, created_at
		FROM outbox_events
		WHERE status = ?
		ORDER BY created_at ASC, id ASC
		LIMIT ?`, StatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending outbox events: %w", err)
	}
	defer rows.Close()

	events := make([]OutboxEventEntity, 0, limit)
	for rows.Next() {
		var event OutboxEventEntity
		if err := rows.Scan(&event.ID, &event.EventType, &event.Payload, &event.Status, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan pending outbox event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate pending outbox events: %w", err)
	}
	return events, nil
}

func (r *Repository) MarkEventProcessed(ctx context.Context, db *sql.DB, eventID string) error {
	if ctx == nil {
		return errors.New("outbox event context is nil")
	}
	if db == nil {
		return ErrNilDatabase
	}
	if eventID == "" {
		return errors.New("outbox event ID is empty")
	}

	result, err := db.ExecContext(ctx, `
		UPDATE outbox_events
		SET status = ?
		WHERE id = ? AND status = ?`, StatusProcessed, eventID, StatusPending)
	if err != nil {
		return fmt.Errorf("failed to mark outbox event %q processed: %w", eventID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to verify outbox event %q status: %w", eventID, err)
	}
	if rowsAffected != 1 {
		return fmt.Errorf("%w: %s", ErrEventNotFound, eventID)
	}
	return nil
}
