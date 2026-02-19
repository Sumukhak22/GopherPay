package audit

import "context"

type Repository interface {
	Log(ctx context.Context, entry *AuditLog) error
}
