package audit

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

func (r *MySQLRepository) Log(ctx context.Context, entry *AuditLog) error {

	query := `
		INSERT INTO audit_logs (request_id, action, status, message)
		VALUES (?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		entry.RequestID,
		entry.Action,
		entry.Status,
		entry.Message,
	)

	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	return nil
}
