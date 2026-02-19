package audit

import "time"

type AuditLog struct {
	ID        uint64
	RequestID string
	Action    string
	Status    string
	Message   *string
	CreatedAt time.Time
}
