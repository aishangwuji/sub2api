package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type agentTokenRepository struct {
	db *sql.DB
}

// NewAgentTokenRepository 创建 AgentToken 仓储
func NewAgentTokenRepository(db *sql.DB) service.AgentTokenRepository {
	return &agentTokenRepository{db: db}
}

func (r *agentTokenRepository) Create(ctx context.Context, token *service.AgentToken) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil agent token repository or db")
	}

	query := `
		INSERT INTO agent_tokens (
			name, token_prefix, token_hash, token_salt, scopes,
			created_by_user_id, expires_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	return r.db.QueryRowContext(
		ctx,
		query,
		token.Name,
		token.TokenPrefix,
		token.TokenHash,
		token.TokenSalt,
		pq.Array(token.Scopes),
		token.CreatedByUserID,
		token.ExpiresAt,
		token.CreatedAt,
		token.UpdatedAt,
	).Scan(&token.ID)
}

func (r *agentTokenRepository) GetByID(ctx context.Context, id int64) (*service.AgentToken, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil agent token repository or db")
	}

	query := `
		SELECT id, name, token_prefix, token_hash, token_salt, scopes,
		       created_by_user_id, last_used_at, expires_at, revoked_at,
		       created_at, updated_at
		FROM agent_tokens
		WHERE id = $1
	`

	token := &service.AgentToken{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&token.ID,
		&token.Name,
		&token.TokenPrefix,
		&token.TokenHash,
		&token.TokenSalt,
		pq.Array(&token.Scopes),
		&token.CreatedByUserID,
		&token.LastUsedAt,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrAgentTokenNotFound
		}
		return nil, err
	}
	return token, nil
}

func (r *agentTokenRepository) GetByHash(ctx context.Context, tokenPrefix string) (*service.AgentToken, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("nil agent token repository or db")
	}

	query := `
		SELECT id, name, token_prefix, token_hash, token_salt, scopes,
		       created_by_user_id, last_used_at, expires_at, revoked_at,
		       created_at, updated_at
		FROM agent_tokens
		WHERE token_prefix = $1
		ORDER BY id DESC
		LIMIT 1
	`

	token := &service.AgentToken{}
	err := r.db.QueryRowContext(ctx, query, tokenPrefix).Scan(
		&token.ID,
		&token.Name,
		&token.TokenPrefix,
		&token.TokenHash,
		&token.TokenSalt,
		pq.Array(&token.Scopes),
		&token.CreatedByUserID,
		&token.LastUsedAt,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
		&token.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, service.ErrAgentTokenNotFound
		}
		return nil, err
	}
	return token, nil
}

func (r *agentTokenRepository) List(ctx context.Context, filter *service.AgentTokenFilter) ([]*service.AgentToken, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("nil agent token repository or db")
	}

	page := 1
	pageSize := 20
	if filter != nil {
		if filter.Page > 0 {
			page = filter.Page
		}
		if filter.PageSize > 0 && filter.PageSize <= 100 {
			pageSize = filter.PageSize
		}
	}
	offset := (page - 1) * pageSize

	where := "WHERE 1=1"
	if filter != nil && !filter.IncludeRevoked {
		where += " AND revoked_at IS NULL"
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM agent_tokens " + where
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, name, token_prefix, token_hash, token_salt, scopes,
		       created_by_user_id, last_used_at, expires_at, revoked_at,
		       created_at, updated_at
		FROM agent_tokens
		%s
		ORDER BY id DESC
		LIMIT %d OFFSET %d
	`, where, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	tokens := make([]*service.AgentToken, 0, pageSize)
	for rows.Next() {
		t := &service.AgentToken{}
		if err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.TokenPrefix,
			&t.TokenHash,
			&t.TokenSalt,
			pq.Array(&t.Scopes),
			&t.CreatedByUserID,
			&t.LastUsedAt,
			&t.ExpiresAt,
			&t.RevokedAt,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		tokens = append(tokens, t)
	}

	return tokens, total, nil
}

func (r *agentTokenRepository) Revoke(ctx context.Context, id int64, revokedAt time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil agent token repository or db")
	}

	query := `
		UPDATE agent_tokens
		SET revoked_at = $2, updated_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`

	res, err := r.db.ExecContext(ctx, query, id, revokedAt)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return service.ErrAgentTokenNotFound
	}
	return nil
}

func (r *agentTokenRepository) UpdateLastUsed(ctx context.Context, id int64, lastUsedAt time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil agent token repository or db")
	}

	query := `UPDATE agent_tokens SET last_used_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, lastUsedAt)
	return err
}

func (r *agentTokenRepository) InsertAuditLog(ctx context.Context, log *service.AgentAuditLog) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("nil agent token repository or db")
	}

	summaryJSON := "{}"
	if len(log.RequestSummary) > 0 {
		if data, err := json.Marshal(log.RequestSummary); err == nil {
			summaryJSON = string(data)
		}
	}

	createdAt := log.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	query := `
		INSERT INTO agent_audit_logs (
			agent_token_id, agent_token_name, task_goal, ip_hash, method,
			path, scope_required, request_summary, status_code, latency_ms,
			error_message, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	return r.db.QueryRowContext(
		ctx,
		query,
		log.AgentTokenID,
		log.AgentTokenName,
		log.TaskGoal,
		log.IPHash,
		log.Method,
		log.Path,
		log.ScopeRequired,
		summaryJSON,
		log.StatusCode,
		log.LatencyMs,
		log.ErrorMessage,
		createdAt,
	).Scan(&log.ID)
}

func (r *agentTokenRepository) ListAuditLogs(ctx context.Context, filter *service.AgentAuditLogFilter) ([]*service.AgentAuditLog, int64, error) {
	if r == nil || r.db == nil {
		return nil, 0, fmt.Errorf("nil agent token repository or db")
	}

	page := 1
	pageSize := 20
	if filter != nil {
		if filter.Page > 0 {
			page = filter.Page
		}
		if filter.PageSize > 0 && filter.PageSize <= 100 {
			pageSize = filter.PageSize
		}
	}
	offset := (page - 1) * pageSize

	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if filter != nil && filter.AgentTokenID != nil {
		where += fmt.Sprintf(" AND agent_token_id = $%d", argIdx)
		args = append(args, *filter.AgentTokenID)
		argIdx++
	}

	var total int64
	countQuery := "SELECT COUNT(*) FROM agent_audit_logs " + where
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(`
		SELECT id, agent_token_id, agent_token_name, task_goal, ip_hash,
		       method, path, scope_required, request_summary, status_code,
		       latency_ms, error_message, created_at
		FROM agent_audit_logs
		%s
		ORDER BY id DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := make([]*service.AgentAuditLog, 0, pageSize)
	for rows.Next() {
		l := &service.AgentAuditLog{}
		var summaryJSON string
		if err := rows.Scan(
			&l.ID,
			&l.AgentTokenID,
			&l.AgentTokenName,
			&l.TaskGoal,
			&l.IPHash,
			&l.Method,
			&l.Path,
			&l.ScopeRequired,
			&summaryJSON,
			&l.StatusCode,
			&l.LatencyMs,
			&l.ErrorMessage,
			&l.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		if summaryJSON != "" && summaryJSON != "{}" {
			_ = json.Unmarshal([]byte(summaryJSON), &l.RequestSummary)
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}
