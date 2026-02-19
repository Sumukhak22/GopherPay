package billing

import (
	"context"
	"database/sql"
)

type WalletRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)

	GetAccountForUpdate(ctx context.Context, tx *sql.Tx, accountID uint64) (*Account, error)

	UpdateAccountBalance(ctx context.Context, tx *sql.Tx, accountID uint64, newBalance int64) error

	InsertTransaction(ctx context.Context, tx *sql.Tx, txn *Transaction) (uint64, error)

	UpdateTransactionStatus(ctx context.Context, tx *sql.Tx, txnID uint64, status TransactionStatus, errMsg *string) error
}
