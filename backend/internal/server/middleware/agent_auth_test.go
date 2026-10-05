package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type mockAgentTokenRepo struct {
	tokens   map[int64]*service.AgentToken
	byPrefix map[string]*service.AgentToken
	logs     []*service.AgentAuditLog
	nextID   int64
}

func newMockAgentTokenRepo() *mockAgentTokenRepo {
	return &mockAgentTokenRepo{
		tokens:   make(map[int64]*service.AgentToken),
		byPrefix: make(map[string]*service.AgentToken),
		nextID:   1,
	}
}

func (m *mockAgentTokenRepo) Create(ctx context.Context, token *service.AgentToken) error {
	token.ID = m.nextID
	m.nextID++
	m.tokens[token.ID] = token
	m.byPrefix[token.TokenPrefix] = token
	return nil
}

func (m *mockAgentTokenRepo) GetByID(ctx context.Context, id int64) (*service.AgentToken, error) {
	if t, ok := m.tokens[id]; ok {
		return t, nil
	}
	return nil, service.ErrAgentTokenNotFound
}

func (m *mockAgentTokenRepo) GetByHash(ctx context.Context, tokenPrefix string) (*service.AgentToken, error) {
	if t, ok := m.byPrefix[tokenPrefix]; ok {
		return t, nil
	}
	return nil, service.ErrAgentTokenNotFound
}

func (m *mockAgentTokenRepo) List(ctx context.Context, filter *service.AgentTokenFilter) ([]*service.AgentToken, int64, error) {
	res := make([]*service.AgentToken, 0, len(m.tokens))
	for _, t := range m.tokens {
		res = append(res, t)
	}
	return res, int64(len(res)), nil
}

func (m *mockAgentTokenRepo) Revoke(ctx context.Context, id int64, revokedAt time.Time) error {
	if t, ok := m.tokens[id]; ok {
		t.RevokedAt = &revokedAt
		return nil
	}
	return service.ErrAgentTokenNotFound
}

func (m *mockAgentTokenRepo) UpdateLastUsed(ctx context.Context, id int64, lastUsedAt time.Time) error {
	return nil
}

func (m *mockAgentTokenRepo) InsertAuditLog(ctx context.Context, log *service.AgentAuditLog) error {
	m.logs = append(m.logs, log)
	return nil
}

func (m *mockAgentTokenRepo) ListAuditLogs(ctx context.Context, filter *service.AgentAuditLogFilter) ([]*service.AgentAuditLog, int64, error) {
	return m.logs, int64(len(m.logs)), nil
}

func setupTestRouter(tokenSvc *service.AgentTokenService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	authMiddleware := NewAgentAuthMiddleware(tokenSvc)

	r.GET("/api/manage/test-read", authMiddleware(service.ScopeStatusRead), func(c *gin.Context) {
		tokenName := c.GetString(ContextKeyAgentTokenName)
		taskGoal := c.GetString(ContextKeyAgentTaskGoal)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "agent": tokenName, "goal": taskGoal})
	})

	return r
}

func TestAgentAuthMiddleware_NoToken(t *testing.T) {
	repo := newMockAgentTokenRepo()
	svc := service.NewAgentTokenService(repo)
	router := setupTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/manage/test-read", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAgentAuthMiddleware_ValidTokenWithGoal(t *testing.T) {
	repo := newMockAgentTokenRepo()
	svc := service.NewAgentTokenService(repo)
	plain, err := svc.CreateToken(context.Background(), "AutoOps Agent", []string{service.ScopeStatusRead}, 1, nil)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	router := setupTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/manage/test-read", nil)
	req.Header.Set("Authorization", "Bearer "+plain.RawToken)
	req.Header.Set(HeaderAgentTaskGoal, "Nightly routine health check")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAgentAuthMiddleware_ForbiddenScope(t *testing.T) {
	repo := newMockAgentTokenRepo()
	svc := service.NewAgentTokenService(repo)
	// 仅授予 accounts:read，未授予 status:read
	plain, err := svc.CreateToken(context.Background(), "Accounts Only Agent", []string{service.ScopeAccountsRead}, 1, nil)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	router := setupTestRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/manage/test-read", nil)
	req.Header.Set("Authorization", "Bearer "+plain.RawToken)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
