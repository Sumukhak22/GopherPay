package http

import (
	"encoding/json"
	"net/http"

	"gopherpay/internal/billing"
	"gopherpay/internal/middleware"
	"gopherpay/internal/worker"
)

type TransferHandler struct {
	pool *worker.Pool
}

func NewTransferHandler(pool *worker.Pool) *TransferHandler {
	return &TransferHandler{pool: pool}
}

type transferRequestPayload struct {
	FromID uint64 `json:"from_id"`
	ToID   uint64 `json:"to_id"`
	Amount int64  `json:"amount"`
}

func (h *TransferHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	var payload transferRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	reqID := middleware.GetRequestID(r.Context())

	job := worker.TransferJob{
		Request: billing.TransferRequest{
			RequestID: reqID,
			FromID:    payload.FromID,
			ToID:      payload.ToID,
			Amount:    payload.Amount,
		},
	}

	if !h.pool.Submit(job) {
		http.Error(w, "too many requests", http.StatusTooManyRequests)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status":"pending"}`))
}
