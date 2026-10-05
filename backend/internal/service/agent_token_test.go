package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// mockAgentTokenRepo 内存模拟仓储
type mockAgentTokenRepo struct {
	tokens    map[int64]*AgentToken
	byPrefix  map[string]*AgentToken
	auditLogs []*AgentAuditLog
	nextID    int64
}

func newMockAgentTokenRepo() *mockAgentTokenRepo {
	return &mockAgentTokenRepo{
		tokens:   make(map[int64]*AgentToken),
		byPrefix: make(map[string]*AgentToken),
		nextID:   1,
	}
}

func (m *mockAgentTokenRepo) Create(ctx context.Context, token *AgentToken) error {
	token.ID = m.nextID
	m.nextID++
	m.tokens[token.ID] = token
	m.byPrefix[token.TokenPrefix] = token
	return nil
}

func (m *mockAgentTokenRepo) GetByID(ctx context.Context, id int64) (*AgentToken, error) {
	if t, ok := m.tokens[id]; ok {
		return t, nil
	}
	return nil, ErrAgentTokenNotFound
}

func (m *mockAgentTokenRepo) GetByHash(ctx context.Context, tokenPrefix string) (*AgentToken, error) {
	if t, ok := m.byPrefix[tokenPrefix]; ok {
		return t, nil
	}
	return nil, ErrAgentTokenNotFound
}

func (m *mockAgentTokenRepo) List(ctx context.Context, filter *AgentTokenFilter) ([]*AgentToken, int64, error) {
	res := make([]*AgentToken, 0, len(m.tokens))
	for _, t := range m.tokens {
		if filter != nil && !filter.IncludeRevoked && t.RevokedAt != nil {
			continue
		}
		res = append(res, t)
	}
	return res, int64(len(res)), nil
}

func (m *mockAgentTokenRepo) Revoke(ctx context.Context, id int64, revokedAt time.Time) error {
	if t, ok := m.tokens[id]; ok {
		t.RevokedAt = &revokedAt
		return nil
	}
	return ErrAgentTokenNotFound
}

func (m *mockAgentTokenRepo) UpdateLastUsed(ctx context.Context, id int64, lastUsedAt time.Time) error {
	if t, ok := m.tokens[id]; ok {
		t.LastUsedAt = &lastUsedAt
		return nil
	}
	return ErrAgentTokenNotFound
}

func (m *mockAgentTokenRepo) InsertAuditLog(ctx context.Context, log *AgentAuditLog) error {
	log.ID = int64(len(m.auditLogs) + 1)
	m.auditLogs = append(m.auditLogs, log)
	return nil
}

func (m *mockAgentTokenRepo) ListAuditLogs(ctx context.Context, filter *AgentAuditLogFilter) ([]*AgentAuditLog, int64, error) {
	return m.auditLogs, int64(len(m.auditLogs)), nil
}

func TestAgentTokenService_CreateAndVerify(t *testing.T) {
	repo := newMockAgentTokenRepo()
	svc := NewAgentTokenService(repo)
	ctx := context.Background()

	// 1. 创建合法 Token
	scopes := []string{ScopeStatusRead, ScopeAccountsRead}
	plain, err := svc.CreateToken(ctx, "Test Agent", scopes, 1, nil)
	if err != nil {
		t.Fatalf("unexpected error creating token: %v", err)
	}

	if !strings.HasPrefix(plain.RawToken, AgentTokenPrefix) {
		t.Errorf("expected token prefix %s, got %s", AgentTokenPrefix, plain.RawToken)
	}

	// 2. 校验成功场景 (包含 status:read)
	token, err := svc.VerifyToken(ctx, plain.RawToken, ScopeStatusRead)
	if err != nil {
		t.Fatalf("unexpected verify error: %v", err)
	}
	if token.Name != "Test Agent" {
		t.Errorf("expected name 'Test Agent', got %s", token.Name)
	}

	// 3. 校验权限缺失场景 (请求 accounts:write，但只授予了 status:read 和 accounts:read)
	_, err = svc.VerifyToken(ctx, plain.RawToken, ScopeAccountsWrite)
	if err != ErrAgentForbidden {
		t.Errorf("expected ErrAgentForbidden, got %v", err)
	}

	// 4. 吊销 Token
	err = svc.RevokeToken(ctx, token.ID)
	if err != nil {
		t.Fatalf("failed to revoke token: %v", err)
	}

	// 5. 吊销后再次验证应被拦截
	_, err = svc.VerifyToken(ctx, plain.RawToken, ScopeStatusRead)
	if err != ErrAgentTokenRevoked {
		t.Errorf("expected ErrAgentTokenRevoked, got %v", err)
	}
}

func TestAgentTokenService_Expired(t *testing.T) {
	repo := newMockAgentTokenRepo()
	svc := NewAgentTokenService(repo)
	ctx := context.Background()

	// 创建已过期 Token
	expiredTime := time.Now().UTC().Add(-1 * time.Hour)
	plain, err := svc.CreateToken(ctx, "Expired Agent", []string{ScopeStatusRead}, 1, &expiredTime)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	_, err = svc.VerifyToken(ctx, plain.RawToken, ScopeStatusRead)
	if err != ErrAgentTokenExpired {
		t.Errorf("expected ErrAgentTokenExpired, got %v", err)
	}
}

func TestAgentTokenService_InvalidScope(t *testing.T) {
	repo := newMockAgentTokenRepo()
	svc := NewAgentTokenService(repo)
	ctx := context.Background()

	// 尝试授予非法的 superadmin 权限应被白名单拦截
	_, err := svc.CreateToken(ctx, "Hacker Agent", []string{"superadmin:all"}, 1, nil)
	if !errors.Is(err, ErrInvalidAgentScope) {
		t.Errorf("expected ErrInvalidAgentScope, got %v", err)
	}
}
