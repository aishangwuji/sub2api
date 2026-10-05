package admin

import (
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AgentTokenHandler struct {
	tokenService *service.AgentTokenService
}

func NewAgentTokenHandler(tokenService *service.AgentTokenService) *AgentTokenHandler {
	return &AgentTokenHandler{tokenService: tokenService}
}

// List 列出所有签发的 Agent Tokens
// GET /api/v1/admin/agent-tokens
func (h *AgentTokenHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	includeRevoked := c.Query("include_revoked") == "true"

	filter := &service.AgentTokenFilter{
		IncludeRevoked: includeRevoked,
		Page:           page,
		PageSize:       pageSize,
	}

	tokens, total, err := h.tokenService.ListTokens(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, "Failed to list agent tokens")
		return
	}

	response.Paginated(c, tokens, total, page, pageSize)
}

type CreateAgentTokenRequest struct {
	Name          string   `json:"name" binding:"required"`
	Scopes        []string `json:"scopes" binding:"required"`
	ExpiresInDays int      `json:"expires_in_days"`
}

// Create 生成新的 Agent Token（返回包含 RawToken 的单次结果）
// POST /api/v1/admin/agent-tokens
func (h *AgentTokenHandler) Create(c *gin.Context) {
	var req CreateAgentTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request payload: "+err.Error())
		return
	}

	userID := getAdminIDFromContext(c)
	var expiresAt *time.Time
	if req.ExpiresInDays > 0 {
		exp := time.Now().UTC().AddDate(0, 0, req.ExpiresInDays)
		expiresAt = &exp
	}

	plainToken, err := h.tokenService.CreateToken(c.Request.Context(), req.Name, req.Scopes, userID, expiresAt)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, plainToken)
}

// Revoke 吊销 Agent Token
// POST /api/v1/admin/agent-tokens/:id/revoke
func (h *AgentTokenHandler) Revoke(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid token ID")
		return
	}

	if err := h.tokenService.RevokeToken(c.Request.Context(), id); err != nil {
		if err == service.ErrAgentTokenNotFound {
			response.NotFound(c, "Agent token not found or already revoked")
			return
		}
		response.InternalError(c, "Failed to revoke agent token")
		return
	}

	response.Success(c, gin.H{"revoked": true, "id": id})
}

// GetScopes 获取当前系统支持授予的所有 Agent Scopes 字典
// GET /api/v1/admin/agent-tokens/scopes
func (h *AgentTokenHandler) GetScopes(c *gin.Context) {
	type ScopeItem struct {
		Scope       string `json:"scope"`
		Description string `json:"description"`
	}
	scopes := make([]ScopeItem, 0, len(service.AllowedAgentScopes))
	for s, desc := range service.AllowedAgentScopes {
		scopes = append(scopes, ScopeItem{
			Scope:       s,
			Description: desc,
		})
	}
	response.Success(c, scopes)
}

// GetSkillDoc 获取 Agent Skill 规约内容（仅限管理员查阅与复制）
// GET /api/v1/admin/agent-tokens/skill-doc
func (h *AgentTokenHandler) GetSkillDoc(c *gin.Context) {
	doc := h.tokenService.GetSkillDocument()
	response.Success(c, gin.H{"content": doc})
}

