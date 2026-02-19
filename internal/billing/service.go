package billing

import (
	"context"
	"errors"
	"fmt"
	"gopherpay/internal/audit"
	"log/slog"
)

var (
	ErrInvalidAmount     = errors.New("amount must be greater than zero")
	ErrSameAccount       = errors.New("cannot transfer to same account")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type Service struct {
	repo   WalletRepository //repository for wallet operations
	audit  audit.Repository //audit repository for logging transfer attempts
	logger *slog.Logger
}

func NewService(repo WalletRepository, auditRepo audit.Repository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		audit:  auditRepo,
		logger: logger,
	}
}

func (s *Service) markTransactionFailed(ctx context.Context, txnID uint64, message string) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback()

	_ = s.repo.UpdateTransactionStatus(ctx, tx, txnID, StatusFailed, &message)
	_ = tx.Commit()
}

func (s *Service) markTransactionSuccess(ctx context.Context, txnID uint64) {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return
	}
	defer tx.Rollback()

	_ = s.repo.UpdateTransactionStatus(ctx, tx, txnID, StatusSuccess, nil)
	_ = tx.Commit()
}
func (s *Service) logAudit(ctx context.Context, requestID, action, status, message string) {
	msg := message
	_ = s.audit.Log(ctx, &audit.AuditLog{
		RequestID: requestID,
		Action:    action,
		Status:    status,
		Message:   &msg,
	})
}

func (s *Service) Transfer(ctx context.Context, req TransferRequest) error {

	s.logger.Info("transfer started",
		"request_id", req.RequestID,
		"from", req.FromID,
		"to", req.ToID,
		"amount", req.Amount,
	)

	if req.Amount <= 0 {
		s.logAudit(ctx, req.RequestID, "TRANSFER", "FAILED", "invalid amount")
		return ErrInvalidAmount
	}

	if req.FromID == req.ToID {
		s.logAudit(ctx, req.RequestID, "TRANSFER", "FAILED", "self transfer not allowed")
		return ErrSameAccount
	}

	// -------------------------------------------------
	// STEP 1: Insert transaction as PENDING (NO SQL TX)
	// -------------------------------------------------

	pendingTxn := &Transaction{
		RequestID:     req.RequestID,
		FromAccountID: req.FromID,
		ToAccountID:   req.ToID,
		Amount:        req.Amount,
		Status:        StatusPending,
	}

	// We insert using normal DB connection (no tx yet)
	txInsert, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin insert tx failed: %w", err)
	}

	txnID, err := s.repo.InsertTransaction(ctx, txInsert, pendingTxn)
	if err != nil {
		txInsert.Rollback()
		return err
	}

	if err := txInsert.Commit(); err != nil {
		return err
	}

	// -------------------------------------------------
	// STEP 2: Begin SQL transaction for balance updates
	// -------------------------------------------------

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Deadlock prevention: consistent lock ordering
	firstID, secondID := req.FromID, req.ToID
	if req.FromID > req.ToID {
		firstID, secondID = req.ToID, req.FromID
	}

	acc1, err := s.repo.GetAccountForUpdate(ctx, tx, firstID)
	if err != nil {
		s.markTransactionFailed(ctx, txnID, "account fetch failed")
		return err
	}

	acc2, err := s.repo.GetAccountForUpdate(ctx, tx, secondID)
	if err != nil {
		s.markTransactionFailed(ctx, txnID, "account fetch failed")
		return err
	}

	var sender, receiver *Account
	if req.FromID == firstID {
		sender = acc1
		receiver = acc2
	} else {
		sender = acc2
		receiver = acc1
	}

	// -------------------------------------------------
	// STEP 3: Validate balance
	// -------------------------------------------------

	if sender.Balance < req.Amount {

		s.logger.Warn("transfer failed - insufficient funds",
			"request_id", req.RequestID,
		)
		s.logAudit(ctx, req.RequestID, "TRANSFER", "FAILED", "insufficient funds")

		s.markTransactionFailed(ctx, txnID, "insufficient funds")
		return ErrInsufficientFunds
	}

	// -------------------------------------------------
	// STEP 4: Update balances atomically
	// -------------------------------------------------

	newSenderBalance := sender.Balance - req.Amount
	newReceiverBalance := receiver.Balance + req.Amount

	if err := s.repo.UpdateAccountBalance(ctx, tx, sender.ID, newSenderBalance); err != nil {
		s.markTransactionFailed(ctx, txnID, "balance update failed")
		return err
	}

	if err := s.repo.UpdateAccountBalance(ctx, tx, receiver.ID, newReceiverBalance); err != nil {
		s.markTransactionFailed(ctx, txnID, "balance update failed")
		return err
	}

	if err := tx.Commit(); err != nil {
		s.markTransactionFailed(ctx, txnID, "commit failed")
		return err
	}

	// -------------------------------------------------
	// STEP 5: Mark SUCCESS (outside balance transaction)
	// -------------------------------------------------

	s.markTransactionSuccess(ctx, txnID)
	s.logAudit(ctx, req.RequestID, "TRANSFER", "SUCCESS", "transfer completed")

	s.logger.Info("transfer successful",
		"request_id", req.RequestID,
		"txn_id", txnID,
	)

	return nil
}
