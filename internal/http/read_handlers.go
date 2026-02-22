package http

import (
	"context"
	"encoding/json"
	"gopherpay/internal/audit"
	"gopherpay/internal/billing"
	"net/http"
	"time"
)

type AccountsHandler struct {
	repo *billing.MySQLRepository
}

func NewAccountsHandler(repo *billing.MySQLRepository) *AccountsHandler {
	return &AccountsHandler{repo: repo}
}

func (h *AccountsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	accounts, err := h.repo.GetAllAccounts(ctx)
	if err != nil {
		http.Error(w, "failed to fetch accounts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

type TransactionsHandler struct {
	repo *billing.MySQLRepository
}

func NewTransactionsHandler(repo *billing.MySQLRepository) *TransactionsHandler {
	return &TransactionsHandler{repo: repo}
}

func (h *TransactionsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	txns, err := h.repo.GetRecentTransactions(ctx)
	if err != nil {
		http.Error(w, "failed to fetch transactions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(txns)
}

type AuditHandler struct {
	repo *audit.MySQLRepository
}

func NewAuditHandler(repo *audit.MySQLRepository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

func (h *AuditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	logs, err := h.repo.GetRecentAuditLogs(ctx)
	if err != nil {
		http.Error(w, "failed to fetch audit logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}
