package audit
 
import "context"
 
type Repository interface {
    Log(ctx context.Context, entry *AuditLog) error
}
 
func (r *MySQLRepository) GetRecentAuditLogs(ctx context.Context) ([]AuditLog, error) {
 
    query := `
        SELECT id, request_id, action, status, message, created_at
        FROM audit_logs
        ORDER BY created_at DESC
        LIMIT 50
    `
 
    rows, err := r.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
 
    var logs []AuditLog
 
    for rows.Next() {
        var logEntry AuditLog
        if err := rows.Scan(
            &logEntry.ID,
            &logEntry.RequestID,
            &logEntry.Action,
            &logEntry.Status,
            &logEntry.Message,
            &logEntry.CreatedAt,
        ); err != nil {
            return nil, err
        }
        logs = append(logs, logEntry)
    }
 
    return logs, nil
}
 