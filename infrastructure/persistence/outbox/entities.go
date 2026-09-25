package outbox

import "time"

const (
	StatusPending   = "PENDING"
	StatusProcessed = "PROCESSED"
)

type OutboxEventEntity struct {
	ID        string    `db:"id"`
	EventType string    `db:"event_type"`
	Payload   []byte    `db:"payload"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
}
