package billing

import (
	"context"
	"database/sql"
	"fmt"
)

type MySQLRepository struct {
	db *sql.DB
}

func NewMySQLRepository(db *sql.DB) *MySQLRepository {
	return &MySQLRepository{db: db}
}

func (r *MySQLRepository) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return r.db.BeginTx(ctx, nil)
}

func (r *MySQLRepository) GetAccountForUpdate(ctx context.Context, tx *sql.Tx, accountID uint64) (*Account, error) {
	query := `
        SELECT id, balance, created_at, updated_at
        FROM accounts
        WHERE id = ?
        FOR UPDATE
    `

	row := tx.QueryRowContext(ctx, query, accountID)

	var acc Account
	err := row.Scan(&acc.ID, &acc.Balance, &acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account %d: %w", accountID, err)
	}

	return &acc, nil
}

func (r *MySQLRepository) GetAccountBalance(ctx context.Context, tx *sql.Tx, accountID uint64) (int64, error) {
	query := `
        SELECT balance FROM accounts WHERE id = ?
    `
	var bal int64
	row := tx.QueryRowContext(ctx, query, accountID)
	if err := row.Scan(&bal); err != nil {
		return 0, fmt.Errorf("failed to fetch balance for account %d: %w", accountID, err)
	}
	return bal, nil
}

func (r *MySQLRepository) UpdateAccountBalance(ctx context.Context, tx *sql.Tx, accountID uint64, newBalance int64) error {
	query := `
        UPDATE accounts
        SET balance = ?, updated_at = NOW()
        WHERE id = ?
    `

	_, err := tx.ExecContext(ctx, query, newBalance, accountID)
	if err != nil {
		return fmt.Errorf("failed to update balance for account %d: %w", accountID, err)
	}

	return nil
}

func (r *MySQLRepository) InsertTransaction(ctx context.Context, tx *sql.Tx, txn *Transaction) (uint64, error) {
	query := `
        INSERT INTO transactions (request_id, from_account_id, to_account_id, amount,
        status, from_balance, to_balance, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, NOW(), NOW())
    `

	result, err := tx.ExecContext(ctx, query,
		txn.RequestID,
		txn.FromAccountID,
		txn.ToAccountID,
		txn.Amount,
		txn.Status,
		txn.FromBalance,
		txn.ToBalance,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert transaction: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get inserted transaction id: %w", err)
	}

	return uint64(id), nil
}

func (r *MySQLRepository) UpdateTransactionStatus(ctx context.Context, tx *sql.Tx, txnID uint64, status TransactionStatus, errMsg *string) error {
	query := `
        UPDATE transactions
        SET status = ?, error_message = ?, updated_at = NOW()
        WHERE id = ?
    `

	_, err := tx.ExecContext(ctx, query, status, errMsg, txnID)
	if err != nil {
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return nil
}

func (r *MySQLRepository) GetAllAccounts(ctx context.Context) ([]Account, error) {

	query := `
        SELECT id, balance, created_at, updated_at
        FROM accounts
        ORDER BY id ASC
    `

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account

	for rows.Next() {
		var acc Account
		if err := rows.Scan(&acc.ID, &acc.Balance, &acc.CreatedAt, &acc.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}

	return accounts, nil
}

func (r *MySQLRepository) GetRecentTransactions(ctx context.Context) ([]Transaction, error) {

	query := `
        SELECT id, request_id, from_account_id, to_account_id,
               amount, status, error_message,
               from_balance, to_balance,
               created_at, updated_at
        FROM transactions
        ORDER BY created_at DESC
        LIMIT 50
    `

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txns []Transaction

	for rows.Next() {
		var txn Transaction
		if err := rows.Scan(
			&txn.ID,
			&txn.RequestID,
			&txn.FromAccountID,
			&txn.ToAccountID,
			&txn.Amount,
			&txn.Status,
			&txn.ErrorMessage,
			&txn.FromBalance,
			&txn.ToBalance,
			&txn.CreatedAt,
			&txn.UpdatedAt,
		); err != nil {
			return nil, err
		}
		txns = append(txns, txn)
	}

	return txns, nil
}
