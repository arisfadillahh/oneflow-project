package httpapi

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5"
)

func (s *Server) insertAuditLog(ctx context.Context, action, entityType, entityID string, metadata map[string]any) error {
	body, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, actor_role, action, entity_type, entity_id, metadata, organization_id)
		VALUES (NULLIF($1, '')::uuid, $2, $3, $4, NULLIF($5, '')::uuid, $6::jsonb, NULLIF($7, '')::uuid)
	`, s.agentID(ctx), s.role(ctx), action, entityType, entityID, string(body), s.organizationID(ctx))
	return err
}

func (s *Server) insertAuditLogTx(ctx context.Context, tx pgx.Tx, action, entityType, entityID string, metadata map[string]any) error {
	body, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, actor_role, action, entity_type, entity_id, metadata, organization_id)
		VALUES (NULLIF($1, '')::uuid, $2, $3, $4, NULLIF($5, '')::uuid, $6::jsonb, NULLIF($7, '')::uuid)
	`, s.agentID(ctx), s.role(ctx), action, entityType, entityID, string(body), s.organizationID(ctx))
	return err
}
