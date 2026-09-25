package message

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/persistence/outbox"
	"github.com/segmentio/kafka-go"
)

type OutboxRepository interface {
	GetPendingEvents(ctx context.Context, db *sql.DB, limit int) ([]outbox.OutboxEventEntity, error)
	MarkEventProcessed(ctx context.Context, db *sql.DB, eventID string) error
}

type KafkaWriter interface {
	WriteMessages(ctx context.Context, messages ...kafka.Message) error
}

type OutboxRelay struct {
	db           *sql.DB
	repository   OutboxRepository
	writer       KafkaWriter
	pollInterval time.Duration
	batchSize    int
}

func NewOutboxRelay(db *sql.DB, repository OutboxRepository, writer KafkaWriter, pollInterval time.Duration, batchSize int) (*OutboxRelay, error) {
	if db == nil {
		return nil, errors.New("outbox relay database is nil")
	}
	if repository == nil {
		return nil, errors.New("outbox relay repository is nil")
	}
	if writer == nil {
		return nil, errors.New("outbox relay Kafka writer is nil")
	}
	if pollInterval <= 0 {
		return nil, errors.New("outbox relay poll interval must be greater than zero")
	}
	if batchSize <= 0 {
		return nil, errors.New("outbox relay batch size must be greater than zero")
	}
	return &OutboxRelay{
		db:           db,
		repository:   repository,
		writer:       writer,
		pollInterval: pollInterval,
		batchSize:    batchSize,
	}, nil
}

// Start runs until ctx is canceled. Events are marked processed only after
// Kafka acknowledges them. If the process crashes after publishing but before
// the database update, the event is published again, giving the outbox relay
// at-least-once delivery semantics without reintroducing the dual-write gap.
func (r *OutboxRelay) Start(ctx context.Context) error {
	if r == nil || r.db == nil {
		return errors.New("outbox relay database is nil")
	}
	if r.repository == nil || r.writer == nil {
		return errors.New("outbox relay dependencies are nil")
	}
	if ctx == nil {
		return errors.New("outbox relay context is nil")
	}

	if err := r.processBatch(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	ticker := time.NewTicker(r.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := r.processBatch(ctx); err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil
				}
				return err
			}
		}
	}
}

func (r *OutboxRelay) processBatch(ctx context.Context) error {
	events, err := r.repository.GetPendingEvents(ctx, r.db, r.batchSize)
	if err != nil {
		return fmt.Errorf("failed to load outbox batch: %w", err)
	}
	for _, event := range events {
		message := kafka.Message{
			Topic: event.EventType,
			Key:   []byte(event.EventType),
			Value: event.Payload,
		}
		if err := r.writer.WriteMessages(ctx, message); err != nil {
			return fmt.Errorf("failed to publish outbox event %q: %w", event.ID, err)
		}
		if err := r.repository.MarkEventProcessed(ctx, r.db, event.ID); err != nil {
			return fmt.Errorf("failed to mark outbox event %q processed: %w", event.ID, err)
		}
	}
	return nil
}
