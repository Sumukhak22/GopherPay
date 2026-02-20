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
			createdAt string
		)

		if err := rows.Scan(&txnID, &requestID, &fromID, &toID, &amount, &status, &errorMsg, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan transaction: %w", err)
		}

		// Step 2: Get account balances (at the time of query; ideally store snapshots)
		fromBalQuery := `SELECT balance FROM accounts WHERE id = ?`
		toBalQuery := `SELECT balance FROM accounts WHERE id = ?`

		var fromBal, toBal int64
		db.QueryRowContext(ctx, fromBalQuery, fromID).Scan(&fromBal)
		db.QueryRowContext(ctx, toBalQuery, toID).Scan(&toBal)

		// Step 3: Get audit logs for this request
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
		defer auditRows.Close()

		var audits []AuditEntry
		for auditRows.Next() {
			var (
				action    string
				auditStat string
				message   sql.NullString
				timestamp string
			)
			if err := auditRows.Scan(&action, &auditStat, &message, &timestamp); err != nil {
				return nil, fmt.Errorf("failed to scan audit log: %w", err)
			}
			audits = append(audits, AuditEntry{
				Action:    action,
				Status:    auditStat,
				Message:   nullStringToPtr(message),
				Timestamp: timestamp,
			})
		}

		details = append(details, TransactionDetail{
			TransactionID: txnID,
			RequestID:     requestID,
			FromAccountID: fromID,
			ToAccountID:   toID,
			Amount:        amount,
			Status:        status,
			ErrorMessage:  nullStringToPtr(errorMsg),
			FromBalance:   fromBal,
			ToBalance:     toBal,
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
