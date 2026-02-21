package reporting

import (
	"context"
	"database/sql"
	"fmt"
)

// TransactionDetail combines transaction, account, and audit data
type TransactionDetail struct {
	TransactionID uint64
	RequestID     string
	FromAccountID uint64
	ToAccountID   uint64
	Amount        int64
	Status        string
	ErrorMessage  *string
	FromBalance   int64
	ToBalance     int64
	CreatedAt     string
	AuditLogs     []AuditEntry
}

type Summary struct {
	TotalTransactions int64
	SuccessfulCount   int64
	FailedCount       int64
	SuccessRate       float64
	TotalSent         int64
	TotalReceived     int64
	NetChange         int64
	AccountCreatedAt  string
	LastActivityAt    string
}

type AuditEntry struct {
	Action    string
	Status    string
	Message   *string
	Timestamp string
}

// GetUserTransactionsWithAudit fetches transactions with account info and audit logs for a user
func GetUserTransactionsWithAudit(ctx context.Context, db *sql.DB, userID uint64) ([]TransactionDetail, error) {

	// Step 1: Get all transactions for user (as sender or receiver)
	txnQuery := `
        SELECT 
            t.id,
            t.request_id,
            t.from_account_id,
            t.to_account_id,
            t.amount,
            t.status,
            t.error_message,
			t.from_balance,
			t.to_balance,
            t.created_at
        FROM transactions t
        WHERE t.from_account_id = ? OR t.to_account_id = ?
        ORDER BY t.created_at DESC
    `

	rows, err := db.QueryContext(ctx, txnQuery, userID, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query transactions: %w", err)
	}
	defer rows.Close()

	var details []TransactionDetail

	for rows.Next() {
		var (
			txnID     uint64
			requestID string
			fromID    uint64
			toID      uint64
			amount    int64
			status    string
			errorMsg  sql.NullString
			fromBal   sql.NullInt64
			toBal     sql.NullInt64
			createdAt string
		)

		if err := rows.Scan(
			&txnID,
			&requestID,
			&fromID,
			&toID,
			&amount,
			&status,
			&errorMsg,
			&fromBal,
			&toBal,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		// ---- fetch audit logs ----
		auditQuery := `
			SELECT action, status, message, created_at
			FROM audit_logs
			WHERE request_id = ?
			ORDER BY created_at ASC
		`

		auditRows, err := db.QueryContext(ctx, auditQuery, requestID)
		if err != nil {
			return nil, fmt.Errorf("failed to query audit logs: %w", err)
		}

		var audits []AuditEntry
		for auditRows.Next() {
			var (
				action    string
				auditStat string
				message   sql.NullString
				timestamp string
			)

			if err := auditRows.Scan(&action, &auditStat, &message, &timestamp); err != nil {
				auditRows.Close()
				return nil, fmt.Errorf("failed to scan audit log: %w", err)
			}

			audits = append(audits, AuditEntry{
				Action:    action,
				Status:    auditStat,
				Message:   nullStringToPtr(message),
				Timestamp: timestamp,
			})
		}
		auditRows.Close()

		details = append(details, TransactionDetail{
			TransactionID: txnID,
			RequestID:     requestID,
			FromAccountID: fromID,
			ToAccountID:   toID,
			Amount:        amount,
			Status:        status,
			ErrorMessage:  nullStringToPtr(errorMsg),
			FromBalance:   fromBal.Int64,
			ToBalance:     toBal.Int64,
			CreatedAt:     createdAt,
			AuditLogs:     audits,
		})
	}

	return details, rows.Err()
}

// Helper to convert sql.NullString to *string
func nullStringToPtr(ns sql.NullString) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func GetUserSummary(ctx context.Context, db *sql.DB, userID uint64) (*Summary, error) {
	query := `
		SELECT 
			COUNT(*) as total_txns,
			SUM(CASE WHEN status = 'SUCCESS' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status = 'FAILED' THEN 1 ELSE 0 END) as failed_count,
			COALESCE(SUM(CASE WHEN from_account_id = ? THEN amount ELSE 0 END), 0) as total_sent,
			COALESCE(SUM(CASE WHEN to_account_id = ? THEN amount ELSE 0 END), 0) as total_received,
			MAX(t.created_at) as last_txn
		FROM transactions t
		WHERE t.from_account_id = ? OR t.to_account_id = ?
	`

	var (
		total, success, failed sql.NullInt64
		sent, received         sql.NullInt64
		lastTxn                sql.NullString
	)

	err := db.QueryRowContext(ctx, query, userID, userID, userID, userID).Scan(
		&total, &success, &failed, &sent, &received, &lastTxn,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query summary: %w", err)
	}

	totalVal := total.Int64
	successVal := success.Int64
	failedVal := failed.Int64
	sentVal := sent.Int64
	receivedVal := received.Int64

	successRate := 0.0
	if totalVal > 0 {
		successRate = (float64(successVal) / float64(totalVal)) * 100
	}

	// Get account creation time (use the first account involved in transactions)
	accountQuery := `
		SELECT MIN(created_at) FROM accounts 
		WHERE id IN (
			SELECT DISTINCT from_account_id FROM transactions WHERE from_account_id = ?
			UNION
			SELECT DISTINCT to_account_id FROM transactions WHERE to_account_id = ?
		)
	`

	var accountCreated sql.NullString
	db.QueryRowContext(ctx, accountQuery, userID, userID).Scan(&accountCreated)

	return &Summary{
		TotalTransactions: totalVal,
		SuccessfulCount:   successVal,
		FailedCount:       failedVal,
		SuccessRate:       successRate,
		TotalSent:         sentVal,
		TotalReceived:     receivedVal,
		NetChange:         receivedVal - sentVal,
		AccountCreatedAt:  accountCreated.String,
		LastActivityAt:    lastTxn.String,
	}, nil
}
