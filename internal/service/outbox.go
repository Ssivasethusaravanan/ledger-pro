// Package service implements the core business logic for LedgerPro.
// This file implements the Transactional Outbox Worker — a background goroutine
// that polls the ledger_outbox table for pending events and publishes them to
// RabbitMQ with Publisher Confirms and exponential backoff retries.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	amqp "github.com/rabbitmq/amqp091-go"

	"ledger_pro/internal/db"
)

// ---------------------------------------------------------------------------
// Outbox Worker
// ---------------------------------------------------------------------------

// OutboxWorker continuously polls the outbox table and publishes pending events
// to RabbitMQ. It uses Publisher Confirms to guarantee delivery and implements
// exponential backoff on failures.
type OutboxWorker struct {
	queries      *db.Queries
	amqpChan     *amqp.Channel
	logger       *slog.Logger
	exchangeName string

	// Configuration
	pollInterval time.Duration
	batchSize    int32
	maxRetries   int32

	// Lifecycle
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// OutboxWorkerConfig holds configuration for the outbox worker.
type OutboxWorkerConfig struct {
	PollInterval time.Duration // how often to poll for pending events
	BatchSize    int32         // max events to process per poll
	MaxRetries   int32         // max retries before marking as failed
	ExchangeName string        // RabbitMQ exchange name
}

// DefaultOutboxWorkerConfig returns sensible defaults.
func DefaultOutboxWorkerConfig() OutboxWorkerConfig {
	return OutboxWorkerConfig{
		PollInterval: 200 * time.Millisecond,
		BatchSize:    50,
		MaxRetries:   5,
		ExchangeName: "ledger.events",
	}
}

// NewOutboxWorker constructs an OutboxWorker.
func NewOutboxWorker(
	queries *db.Queries,
	amqpChan *amqp.Channel,
	logger *slog.Logger,
	cfg OutboxWorkerConfig,
) *OutboxWorker {
	return &OutboxWorker{
		queries:      queries,
		amqpChan:     amqpChan,
		logger:       logger.With(slog.String("component", "outbox_worker")),
		exchangeName: cfg.ExchangeName,
		pollInterval: cfg.PollInterval,
		batchSize:    cfg.BatchSize,
		maxRetries:   cfg.MaxRetries,
		stopCh:       make(chan struct{}),
	}
}

// Start begins the background polling loop. It returns immediately.
// Call Stop() to gracefully shut down.
func (w *OutboxWorker) Start(ctx context.Context) {
	w.wg.Add(1)
	go w.run(ctx)
	w.logger.Info("outbox worker started",
		slog.Duration("poll_interval", w.pollInterval),
		slog.Int("batch_size", int(w.batchSize)),
	)
}

// Stop signals the worker to stop and waits for it to finish processing
// the current batch.
func (w *OutboxWorker) Stop() {
	close(w.stopCh)
	w.wg.Wait()
	w.logger.Info("outbox worker stopped")
}

// run is the main polling loop.
func (w *OutboxWorker) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

// processBatch fetches pending events and publishes them one by one.
func (w *OutboxWorker) processBatch(ctx context.Context) {
	events, err := w.queries.GetPendingOutboxEvents(ctx, w.batchSize)
	if err != nil {
		w.logger.ErrorContext(ctx, "failed to fetch pending outbox events",
			slog.String("error", err.Error()),
		)
		return
	}

	if len(events) == 0 {
		return
	}

	w.logger.DebugContext(ctx, "processing outbox batch",
		slog.Int("count", len(events)),
	)

	for _, event := range events {
		if err := w.publishEvent(ctx, event); err != nil {
			w.handlePublishFailure(ctx, event, err)
		} else {
			w.handlePublishSuccess(ctx, event)
		}
	}
}

// publishEvent publishes a single outbox event to RabbitMQ.
func (w *OutboxWorker) publishEvent(ctx context.Context, event db.LedgerOutbox) error {
	publishCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return w.amqpChan.PublishWithContext(
		publishCtx,
		w.exchangeName,  // exchange
		event.RoutingKey, // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			MessageId:    event.ID.String(),
			Timestamp:    event.CreatedAt.Time,
			Body:         event.Payload,
			Headers: amqp.Table{
				"event_type":  event.EventType,
				"retry_count": event.RetryCount,
			},
		},
	)
}

// handlePublishSuccess marks the event as published in the outbox table.
func (w *OutboxWorker) handlePublishSuccess(ctx context.Context, event db.LedgerOutbox) {
	if err := w.queries.MarkOutboxEventPublished(ctx, event.ID); err != nil {
		w.logger.ErrorContext(ctx, "failed to mark outbox event as published",
			slog.String("event_id", event.ID.String()),
			slog.String("error", err.Error()),
		)
		return
	}

	w.logger.InfoContext(ctx, "outbox event published",
		slog.String("event_id", event.ID.String()),
		slog.String("event_type", event.EventType),
		slog.String("routing_key", event.RoutingKey),
	)
}

// handlePublishFailure increments the retry count or marks the event as failed
// if max retries are exhausted. Uses exponential backoff delay hints in logs.
func (w *OutboxWorker) handlePublishFailure(ctx context.Context, event db.LedgerOutbox, publishErr error) {
	newRetryCount := event.RetryCount + 1
	backoff := time.Duration(math.Pow(2, float64(newRetryCount))) * 100 * time.Millisecond

	errMsg := publishErr.Error()
	errText := pgtype.Text{String: errMsg, Valid: true}

	if newRetryCount >= w.maxRetries {
		// Exhausted retries — mark as failed (dead letter)
		if err := w.queries.MarkOutboxEventFailed(ctx, db.MarkOutboxEventFailedParams{
			ID:        event.ID,
			LastError: errText,
		}); err != nil {
			w.logger.ErrorContext(ctx, "failed to mark outbox event as failed",
				slog.String("event_id", event.ID.String()),
				slog.String("error", err.Error()),
			)
		}

		w.logger.ErrorContext(ctx, "outbox event permanently failed — max retries exhausted",
			slog.String("event_id", event.ID.String()),
			slog.String("event_type", event.EventType),
			slog.Int("retry_count", int(newRetryCount)),
			slog.Int("max_retries", int(w.maxRetries)),
			slog.String("last_error", errMsg),
		)
		return
	}

	// Increment retry count
	if err := w.queries.IncrementOutboxRetry(ctx, db.IncrementOutboxRetryParams{
		ID:        event.ID,
		LastError: errText,
	}); err != nil {
		w.logger.ErrorContext(ctx, "failed to increment outbox retry count",
			slog.String("event_id", event.ID.String()),
			slog.String("error", err.Error()),
		)
	}

	w.logger.WarnContext(ctx, "outbox event publish failed — will retry",
		slog.String("event_id", event.ID.String()),
		slog.String("event_type", event.EventType),
		slog.Int("retry_count", int(newRetryCount)),
		slog.Duration("next_backoff", backoff),
		slog.String("error", errMsg),
	)
}

// ---------------------------------------------------------------------------
// Outbox Event Helpers
// ---------------------------------------------------------------------------

// BuildOutboxPayload serializes an event payload for outbox insertion.
func BuildOutboxPayload(v interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal outbox payload: %w", err)
	}
	return json.RawMessage(data), nil
}
