package billing

import "time"

type Account struct {
	ID        uint64
	Balance   int64 // stored in cents
	CreatedAt time.Time
	UpdatedAt time.Time
}

type TransactionStatus string

const (
	StatusPending TransactionStatus = "PENDING"
	StatusSuccess TransactionStatus = "SUCCESS"
	StatusFailed  TransactionStatus = "FAILED"
)

type Transaction struct {
	ID            uint64
	RequestID     string
	FromAccountID uint64
	ToAccountID   uint64
	Amount        int64
	Status        TransactionStatus
	ErrorMessage  *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type TransferRequest struct {
	RequestID string
	FromID    uint64
	ToID      uint64
	Amount    int64
}
