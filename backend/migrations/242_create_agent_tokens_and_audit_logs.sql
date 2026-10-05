-- Agent 自动化运维与管理体系凭证与审计日志表
CREATE TABLE IF NOT EXISTS agent_tokens (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(64) NOT NULL,
    token_prefix VARCHAR(24) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    token_salt VARCHAR(32) NOT NULL,
    scopes TEXT[] NOT NULL DEFAULT '{}',
    created_by_user_id BIGINT NOT NULL,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_tokens_hash ON agent_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_agent_tokens_revoked_at ON agent_tokens(revoked_at);

CREATE TABLE IF NOT EXISTS agent_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    agent_token_id BIGINT NOT NULL,
    agent_token_name VARCHAR(64) NOT NULL,
    task_goal VARCHAR(255) NOT NULL DEFAULT '',
    ip_hash VARCHAR(64) NOT NULL,
    method VARCHAR(16) NOT NULL,
    path VARCHAR(512) NOT NULL,
    scope_required VARCHAR(64) NOT NULL,
    request_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    status_code INT NOT NULL,
    latency_ms BIGINT NOT NULL,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_audit_logs_token_created ON agent_audit_logs(agent_token_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_agent_audit_logs_created_at ON agent_audit_logs(created_at DESC);
