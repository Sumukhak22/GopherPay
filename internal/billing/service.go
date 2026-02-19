package billing

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

var (
	ErrInvalidAmount     = errors.New("amount must be greater than zero")
	ErrSameAccount       = errors.New("cannot transfer to same account")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type Service struct {
	repo   WalletRepository
	logger *slog.Logger
}

func NewService(repo WalletRepository, logger *slog.Logger) *Service {
	return &Service{
		repo:   repo,
		logger: logger,
	}
}

func (s *Service) Transfer(ctx context.Context, req TransferRequest) error {

	s.logger.Info("transfer started",
		"request_id", req.RequestID,
		"from", req.FromID,
		"to", req.ToID,
		"amount", req.Amount,
	)

	if req.Amount <= 0 {
		return ErrInvalidAmount
	}
	if req.FromID == req.ToID {
		return ErrSameAccount
	}

	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	defer tx.Rollback()

	txn := &Transaction{
		RequestID:     req.RequestID,
		FromAccountID: req.FromID,
		ToAccountID:   req.ToID,
		Amount:        req.Amount,
		Status:        StatusPending,
	}

	txnID, err := s.repo.InsertTransaction(ctx, tx, txn)
	if err != nil {
		return err
	}

	// Deadlock prevention: consistent lock order
	firstID, secondID := req.FromID, req.ToID
	if req.FromID > req.ToID {
		firstID, secondID = req.ToID, req.FromID
	}

	acc1, err := s.repo.GetAccountForUpdate(ctx, tx, firstID)
	if err != nil {
		return err
	}

	acc2, err := s.repo.GetAccountForUpdate(ctx, tx, secondID)
	if err != nil {
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

	if sender.Balance < req.Amount {
		msg := "insufficient balance"
		_ = s.repo.UpdateTransactionStatus(ctx, tx, txnID, StatusFailed, &msg)
		s.logger.Warn("transfer failed - insufficient funds",
			"request_id", req.RequestID,
		)
		return ErrInsufficientFunds
	}

	newSenderBalance := sender.Balance - req.Amount
	newReceiverBalance := receiver.Balance + req.Amount

	if err := s.repo.UpdateAccountBalance(ctx, tx, sender.ID, newSenderBalance); err != nil {
		return err
	}

	if err := s.repo.UpdateAccountBalance(ctx, tx, receiver.ID, newReceiverBalance); err != nil {
		return err
	}

	if err := s.repo.UpdateTransactionStatus(ctx, tx, txnID, StatusSuccess, nil); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}

	s.logger.Info("transfer successful",
		"request_id", req.RequestID,
		"txn_id", txnID,
	)

	return nil
}
