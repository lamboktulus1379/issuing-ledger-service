package http

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"

	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/cache"
	"github.com/lamboktulus1379/issuing-ledger-service/usecase"
)

const (
	idempotencyTTL              = 24 * time.Hour
	maxRequestBytes             = 1 << 20
	transactionAuthorizedStatus = "AUTHORIZED"
	instrumentationName         = "issuing-ledger-service/interfaces/http"
)

var authorizationLatency = newAuthorizationLatency()

func newAuthorizationLatency() metric.Float64Histogram {
	histogram, err := otel.Meter(instrumentationName).Float64Histogram(
		"issuing.authorization.latency",
		metric.WithDescription("Authorization request latency in milliseconds."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil
	}
	return histogram
}

type IssuingUsecase interface {
	AuthorizePayment(ctx context.Context, req usecase.AuthorizeRequest) error
}

type IssuingHandler struct {
	idempotencyStore cache.IdempotencyStore
	issuingUsecase   IssuingUsecase
}

type AuthorizeRequestDTO struct {
	TransactionID        string `json:"transaction_id"`
	SourceAccountID      string `json:"source_account_id"`
	DestinationAccountID string `json:"destination_account_id"`
	Amount               int64  `json:"amount"`
	Currency             string `json:"currency"`
}

type authorizeResponse struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func NewIssuingHandler(idempotencyStore cache.IdempotencyStore, issuingUsecase IssuingUsecase) (*IssuingHandler, error) {
	if idempotencyStore == nil {
		return nil, errors.New("idempotency store is nil")
	}
	if issuingUsecase == nil {
		return nil, errors.New("issuing usecase is nil")
	}
	return &IssuingHandler{
		idempotencyStore: idempotencyStore,
		issuingUsecase:   issuingUsecase,
	}, nil
}

// HTTPHandler wraps the authorization endpoint with otelhttp. The wrapper
// extracts and propagates W3C traceparent/tracestate headers into the request
// context before HandleAuthorize creates its child spans.
func (h *IssuingHandler) HTTPHandler() http.Handler {
	return otelhttp.NewHandler(http.HandlerFunc(h.HandleAuthorize), "POST /authorize")
}

func (h *IssuingHandler) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	statusCode := http.StatusInternalServerError
	defer func() {
		if authorizationLatency != nil {
			authorizationLatency.Record(context.Background(), time.Since(startedAt).Seconds()*1000,
				metric.WithAttributes(
					attribute.Int("http.status_code", statusCode),
					attribute.Bool("error", statusCode >= http.StatusBadRequest),
				),
			)
		}
	}()

	if h == nil || h.idempotencyStore == nil || h.issuingUsecase == nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "service unavailable"})
		return
	}
	if r == nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "request is nil"})
		return
	}
	respond := func(status int, payload interface{}) {
		statusCode = status
		writeJSON(w, status, payload)
	}

	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		respond(http.StatusBadRequest, errorResponse{Error: "Idempotency-Key header is required"})
		return
	}

	ctx := r.Context()
	lockActive := false
	responseWritten := false
	defer func() {
		if recovered := recover(); recovered != nil {
			if lockActive {
				_ = h.idempotencyStore.ReleaseLock(context.WithoutCancel(ctx), key)
			}
			if !responseWritten {
				respond(http.StatusInternalServerError, errorResponse{Error: "internal server error"})
			}
			return
		}
	}()

	traceAttributes := []attribute.KeyValue{
		attribute.String("idempotency.key_hash", hashIdempotencyKey(key)),
		attribute.String("ledger.entry_type", "DEBIT_CREDIT"),
	}
	acquireCtx, acquireSpan := otel.Tracer(instrumentationName).Start(ctx, "Idempotency.AcquireLock", trace.WithAttributes(traceAttributes...))
	acquireErr := h.idempotencyStore.AcquireLock(acquireCtx, key, idempotencyTTL)
	if acquireErr != nil {
		acquireSpan.RecordError(acquireErr)
	}
	acquireSpan.End()
	if errors.Is(acquireErr, cache.ErrIdempotencyConflict) {
		result, err := h.idempotencyStore.GetResult(ctx, key)
		if err == nil && len(result) > 0 {
			statusCode = http.StatusOK
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(result)
			return
		}
		respond(http.StatusConflict, errorResponse{Error: "request is already in progress"})
		return
	}
	if acquireErr != nil {
		respond(http.StatusInternalServerError, errorResponse{Error: "failed to acquire idempotency lock"})
		return
	}
	lockActive = true

	request, err := decodeAuthorizeRequest(w, r)
	if err != nil {
		_ = h.idempotencyStore.ReleaseLock(context.WithoutCancel(ctx), key)
		lockActive = false
		respond(http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	usecaseCtx, usecaseSpan := otel.Tracer(instrumentationName).Start(ctx, "Usecase.AuthorizePayment", trace.WithAttributes(traceAttributes...))
	err = h.issuingUsecase.AuthorizePayment(usecaseCtx, usecase.AuthorizeRequest{
		TransactionID:        request.TransactionID,
		SourceAccountID:      request.SourceAccountID,
		DestinationAccountID: request.DestinationAccountID,
		Amount:               request.Amount,
		Currency:             request.Currency,
	})
	if err != nil {
		usecaseSpan.RecordError(err)
	}
	usecaseSpan.End()
	if err != nil {
		status := statusForAuthorizationError(err)
		_ = h.idempotencyStore.ReleaseLock(context.WithoutCancel(ctx), key)
		lockActive = false
		respond(status, errorResponse{Error: err.Error()})
		return
	}

	responseBytes, err := json.Marshal(authorizeResponse{
		TransactionID: request.TransactionID,
		Status:        transactionAuthorizedStatus,
	})
	if err != nil {
		_ = h.idempotencyStore.ReleaseLock(context.WithoutCancel(ctx), key)
		lockActive = false
		respond(http.StatusInternalServerError, errorResponse{Error: "failed to encode authorization response"})
		return
	}
	completeCtx, completeSpan := otel.Tracer(instrumentationName).Start(ctx, "Idempotency.Complete", trace.WithAttributes(traceAttributes...))
	err = h.idempotencyStore.Complete(completeCtx, key, responseBytes, idempotencyTTL)
	if err != nil {
		completeSpan.RecordError(err)
	}
	completeSpan.End()
	if err != nil {
		_ = h.idempotencyStore.ReleaseLock(context.WithoutCancel(ctx), key)
		lockActive = false
		respond(http.StatusInternalServerError, errorResponse{Error: "failed to complete idempotency request"})
		return
	}
	lockActive = false

	statusCode = http.StatusOK
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(responseBytes)
	responseWritten = true
}

func hashIdempotencyKey(key string) string {
	digest := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", digest[:])
}

func decodeAuthorizeRequest(w http.ResponseWriter, r *http.Request) (AuthorizeRequestDTO, error) {
	if r.Body == nil {
		return AuthorizeRequestDTO{}, errors.New("request body is required")
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
	decoder.DisallowUnknownFields()

	var request AuthorizeRequestDTO
	if err := decoder.Decode(&request); err != nil {
		if errors.Is(err, io.EOF) {
			return AuthorizeRequestDTO{}, errors.New("request body is required")
		}
		return AuthorizeRequestDTO{}, fmt.Errorf("invalid authorization request JSON: %w", err)
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return AuthorizeRequestDTO{}, errors.New("request body must contain exactly one JSON object")
	}
	return request, nil
}

func statusForAuthorizationError(err error) int {
	switch {
	case errors.Is(err, usecase.ErrInsufficientFunds):
		return http.StatusUnprocessableEntity
	case errors.Is(err, usecase.ErrAccountNotFound):
		return http.StatusNotFound
	case errors.Is(err, usecase.ErrInvalidAuthorization), errors.Is(err, usecase.ErrZeroSumViolation):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
