package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Agent Token 前缀与错误定义
const (
	AgentTokenPrefix = "sub2api_agt_"
)

var (
	ErrAgentTokenNotFound = infraerrors.NotFound("AGENT_TOKEN_NOT_FOUND", "agent token not found")
	ErrAgentTokenRevoked  = infraerrors.Unauthorized("AGENT_TOKEN_REVOKED", "agent token has been revoked")
	ErrAgentTokenExpired  = infraerrors.Unauthorized("AGENT_TOKEN_EXPIRED", "agent token has expired")
	ErrAgentTokenInvalid  = infraerrors.Unauthorized("AGENT_TOKEN_INVALID", "invalid agent token")
	ErrAgentForbidden     = infraerrors.Forbidden("AGENT_SCOPE_FORBIDDEN", "agent does not have required scope")
	ErrInvalidAgentScope  = infraerrors.BadRequest("INVALID_AGENT_SCOPE", "one or more scopes are invalid or restricted")
)

// Agent 细粒度权限 Scopes
const (
	ScopeStatusRead     = "status:read"
	ScopeAccountsRead   = "accounts:read"
	ScopeAccountsWrite  = "accounts:write"
	ScopeProxiesRead    = "proxies:read"
	ScopeProxiesWrite   = "proxies:write"
	ScopeUsersRead      = "users:read"
	ScopeUsersManage    = "users:manage"
	ScopeRedeemManage   = "redeem:manage"
	ScopeOpsTrigger     = "ops:trigger"
)

// AllowedAgentScopes 允许授予 Agent 的 Scope 白名单（严禁包含管理员密码修改、数据库重置、Token 签发等高危权限）
var AllowedAgentScopes = map[string]string{
	ScopeStatusRead:    "查看系统概览指标与健康探活状态",
	ScopeAccountsRead:  "查看上游账号列表与基本健康用量（脱敏凭证）",
	ScopeAccountsWrite: "修改上游账号调度开关与触发探活刷新",
	ScopeProxiesRead:   "查看代理节点列表、延迟与健康状态",
	ScopeProxiesWrite:  "启用或禁用特定代理节点",
	ScopeUsersRead:     "查看终端用户状态、余额与额度概况",
	ScopeUsersManage:   "封禁或恢复违规用户状态",
	ScopeRedeemManage:  "受控批量生成充值卡密与查询状态",
	ScopeOpsTrigger:    "触发受控日志清理预检与清理、刷新系统与路由缓存",
}

// AgentToken 实体（数据库中存储加盐哈希，绝不存明文）
type AgentToken struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	TokenPrefix     string     `json:"token_prefix"`
	TokenHash       string     `json:"-"`
	TokenSalt       string     `json:"-"`
	Scopes          []string   `json:"scopes"`
	CreatedByUserID int64      `json:"created_by_user_id"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// AgentTokenPlain 仅在签发时向管理员返回一次的明文载荷
type AgentTokenPlain struct {
	AgentToken
	RawToken string `json:"raw_token"`
}

// AgentAuditLog 记录 Agent 操作审计轨迹
type AgentAuditLog struct {
	ID             int64          `json:"id"`
	AgentTokenID   int64          `json:"agent_token_id"`
	AgentTokenName string         `json:"agent_token_name"`
	TaskGoal       string         `json:"task_goal"`
	IPHash         string         `json:"ip_hash"`
	Method         string         `json:"method"`
	Path           string         `json:"path"`
	ScopeRequired  string         `json:"scope_required"`
	RequestSummary map[string]any `json:"request_summary,omitempty"`
	StatusCode     int            `json:"status_code"`
	LatencyMs      int64          `json:"latency_ms"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// AgentTokenFilter 查询过滤条件
type AgentTokenFilter struct {
	IncludeRevoked bool
	Page           int
	PageSize       int
}

// AgentAuditLogFilter 审计日志过滤条件
type AgentAuditLogFilter struct {
	AgentTokenID *int64
	Page         int
	PageSize     int
}

// AgentTokenRepository 仓储接口
type AgentTokenRepository interface {
	Create(ctx context.Context, token *AgentToken) error
	GetByID(ctx context.Context, id int64) (*AgentToken, error)
	GetByHash(ctx context.Context, tokenHash string) (*AgentToken, error)
	List(ctx context.Context, filter *AgentTokenFilter) ([]*AgentToken, int64, error)
	Revoke(ctx context.Context, id int64, revokedAt time.Time) error
	UpdateLastUsed(ctx context.Context, id int64, lastUsedAt time.Time) error
	InsertAuditLog(ctx context.Context, log *AgentAuditLog) error
	ListAuditLogs(ctx context.Context, filter *AgentAuditLogFilter) ([]*AgentAuditLog, int64, error)
}

// AgentTokenService 服务
type AgentTokenService struct {
	repo AgentTokenRepository
}

func NewAgentTokenService(repo AgentTokenRepository) *AgentTokenService {
	return &AgentTokenService{repo: repo}
}

// ValidateScopes 检查请求授予的 Scopes 是否在白名单中
func ValidateScopes(scopes []string) error {
	if len(scopes) == 0 {
		return infraerrors.BadRequest("EMPTY_SCOPES", "at least one scope must be specified")
	}
	for _, s := range scopes {
		if _, ok := AllowedAgentScopes[s]; !ok {
			return ErrInvalidAgentScope.WithCause(fmt.Errorf("unrecognized or restricted scope: %s", s))
		}
	}
	return nil
}

// HashToken 计算加盐哈希 SHA-256(rawToken + salt)
func HashToken(rawToken, salt string) string {
	h := sha256.New()
	h.Write([]byte(rawToken + salt))
	return hex.EncodeToString(h.Sum(nil))
}

// HashIP 计算 IP 的 SHA-256 哈希用于隐私保护审计
func HashIP(ip string) string {
	if ip == "" {
		return ""
	}
	h := sha256.New()
	h.Write([]byte(ip))
	return hex.EncodeToString(h.Sum(nil))
}

// GenerateRawToken 生成安全的随机 Token 字符串
func GenerateRawToken() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return AgentTokenPrefix + hex.EncodeToString(bytes), nil
}

// GenerateSalt 生成随机盐值
func GenerateSalt() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// CreateToken 创建新的 Agent Token（返回包含 RawToken 的单次明文结构体）
func (s *AgentTokenService) CreateToken(ctx context.Context, name string, scopes []string, createdByUserID int64, expiresAt *time.Time) (*AgentTokenPlain, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, infraerrors.BadRequest("INVALID_TOKEN_NAME", "token name is required")
	}
	if err := ValidateScopes(scopes); err != nil {
		return nil, err
	}

	rawToken, err := GenerateRawToken()
	if err != nil {
		return nil, infraerrors.InternalServer("TOKEN_GEN_FAILED", "failed to generate secure token").WithCause(err)
	}

	salt, err := GenerateSalt()
	if err != nil {
		return nil, infraerrors.InternalServer("SALT_GEN_FAILED", "failed to generate salt").WithCause(err)
	}

	tokenHash := HashToken(rawToken, salt)
	tokenPrefix := rawToken[:min(len(rawToken), 18)] // 包含前缀 + 部分 hex

	now := time.Now().UTC()
	token := &AgentToken{
		Name:            name,
		TokenPrefix:     tokenPrefix,
		TokenHash:       tokenHash,
		TokenSalt:       salt,
		Scopes:          scopes,
		CreatedByUserID: createdByUserID,
		ExpiresAt:       expiresAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.repo.Create(ctx, token); err != nil {
		return nil, err
	}

	return &AgentTokenPlain{
		AgentToken: *token,
		RawToken:   rawToken,
	}, nil
}

// VerifyToken 验证传入的 RawToken，若有效返回 Token 实体并异步记录最后活跃时间
func (s *AgentTokenService) VerifyToken(ctx context.Context, rawToken string, requiredScope string) (*AgentToken, error) {
	rawToken = strings.TrimSpace(rawToken)
	if !strings.HasPrefix(rawToken, AgentTokenPrefix) {
		return nil, ErrAgentTokenInvalid
	}

	// 查库时，通过快速哈希匹配。因为我们存储了 salt，在数据库层我们需要按前缀或通过查询比对。
	// 但更好的方式是：如果支持按 TokenPrefix 过滤，或者直接将 salt 纳入索引，或者通过 token_prefix 索引缩小范围。
	// 这里我们通过 TokenPrefix 快速查出候选记录比对，也可以全表单记录匹配。
	// 由于 Token 格式统一为 sub2api_agt_<48位hex>，我们提取前18位 prefix 查询候选记录。
	prefix := rawToken[:min(len(rawToken), 18)]
	tokens, err := s.repo.GetByHash(ctx, prefix) // repo 层将实现 prefix 匹配与 salt 计算比对
	if err != nil {
		return nil, err
	}
	token := tokens
	if token == nil {
		return nil, ErrAgentTokenInvalid
	}

	// 验证 hash 匹配
	expectedHash := HashToken(rawToken, token.TokenSalt)
	if token.TokenHash != expectedHash {
		return nil, ErrAgentTokenInvalid
	}

	// 验证是否已吊销
	if token.RevokedAt != nil {
		return nil, ErrAgentTokenRevoked
	}

	// 验证是否过期
	now := time.Now().UTC()
	if token.ExpiresAt != nil && token.ExpiresAt.Before(now) {
		return nil, ErrAgentTokenExpired
	}

	// 验证 Scope
	if requiredScope != "" {
		hasScope := false
		for _, sc := range token.Scopes {
			if sc == requiredScope {
				hasScope = true
				break
			}
		}
		if !hasScope {
			return nil, ErrAgentForbidden
		}
	}

	// 异步更新最后使用时间
	go func(id int64) {
		_ = s.repo.UpdateLastUsed(context.Background(), id, time.Now().UTC())
	}(token.ID)

	return token, nil
}

// RevokeToken 吊销 Token
func (s *AgentTokenService) RevokeToken(ctx context.Context, id int64) error {
	return s.repo.Revoke(ctx, id, time.Now().UTC())
}

// ListTokens 列出 Token 列表
func (s *AgentTokenService) ListTokens(ctx context.Context, filter *AgentTokenFilter) ([]*AgentToken, int64, error) {
	return s.repo.List(ctx, filter)
}

// RecordAuditLog 记录审计日志
func (s *AgentTokenService) RecordAuditLog(ctx context.Context, log *AgentAuditLog) error {
	if log == nil {
		return nil
	}
	return s.repo.InsertAuditLog(ctx, log)
}

// ListAuditLogs 列出审计日志
func (s *AgentTokenService) ListAuditLogs(ctx context.Context, filter *AgentAuditLogFilter) ([]*AgentAuditLog, int64, error) {
	return s.repo.ListAuditLogs(ctx, filter)
}
