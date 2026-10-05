package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	ContextKeyAgentToken     = "agent_token"
	ContextKeyAgentTokenID   = "agent_token_id"
	ContextKeyAgentTokenName = "agent_token_name"
	ContextKeyAgentTaskGoal  = "agent_task_goal"

	HeaderAgentToken    = "X-Agent-Token"
	HeaderAgentTaskGoal = "X-Agent-Task-Goal"

	agentMaxRequestBodyAuditBytes = 4096
)

// AgentAuthMiddleware 工厂函数类型定义
type AgentAuthMiddleware func(scope string) gin.HandlerFunc

// NewAgentAuthMiddleware 创建 Agent 鉴权与审计中间件
func NewAgentAuthMiddleware(tokenService *service.AgentTokenService) AgentAuthMiddleware {
	return func(requiredScope string) gin.HandlerFunc {
		return func(c *gin.Context) {
			start := time.Now()

			// 1. 提取 Token
			rawToken := ""
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
					rawToken = strings.TrimSpace(parts[1])
				}
			}
			if rawToken == "" {
				rawToken = strings.TrimSpace(c.GetHeader(HeaderAgentToken))
			}

			if rawToken == "" {
				AbortWithError(c, 401, "UNAUTHORIZED", "Agent authentication required (Bearer or X-Agent-Token)")
				return
			}

			// 2. 验证 Token 与 Scope
			token, err := tokenService.VerifyToken(c.Request.Context(), rawToken, requiredScope)
			if err != nil {
				if err == service.ErrAgentForbidden {
					AbortWithError(c, 403, "FORBIDDEN", "Agent token lacks required scope: "+requiredScope)
					return
				}
				if err == service.ErrAgentTokenRevoked {
					AbortWithError(c, 401, "TOKEN_REVOKED", "Agent token has been revoked")
					return
				}
				if err == service.ErrAgentTokenExpired {
					AbortWithError(c, 401, "TOKEN_EXPIRED", "Agent token has expired")
					return
				}
				AbortWithError(c, 401, "UNAUTHORIZED", "Invalid agent token")
				return
			}

			// 3. 上下文注入
			taskGoal := strings.TrimSpace(c.GetHeader(HeaderAgentTaskGoal))
			c.Set(ContextKeyAgentToken, token)
			c.Set(ContextKeyAgentTokenID, token.ID)
			c.Set(ContextKeyAgentTokenName, token.Name)
			c.Set(ContextKeyAgentTaskGoal, taskGoal)

			// 4. 捕获请求参数摘要（脱敏）
			summary := make(map[string]any)
			if len(c.Request.URL.Query()) > 0 {
				queryMap := make(map[string]any)
				for k, v := range c.Request.URL.Query() {
					if len(v) == 1 {
						queryMap[k] = v[0]
					} else {
						queryMap[k] = v
					}
				}
				summary["query"] = queryMap
			}

			// 如果是写操作且有请求体，读取并截断脱敏
			if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
				if c.Request.Body != nil {
					bodyBytes, _ := io.ReadAll(io.LimitReader(c.Request.Body, agentMaxRequestBodyAuditBytes))
					c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

					var bodyJSON map[string]any
					if err := json.Unmarshal(bodyBytes, &bodyJSON); err == nil {
						// 敏感字段擦除
						for k := range bodyJSON {
							lowerK := strings.ToLower(k)
							if strings.Contains(lowerK, "password") || strings.Contains(lowerK, "secret") || strings.Contains(lowerK, "token") {
								bodyJSON[k] = "******"
							}
						}
						summary["body"] = bodyJSON
					}
				}
			}

			// 5. 执行具体业务 Handler
			c.Next()

			// 6. 审计日志异步落库
			latencyMs := time.Since(start).Milliseconds()
			statusCode := c.Writer.Status()
			errMsg := ""
			if len(c.Errors) > 0 {
				errMsg = c.Errors.String()
			}

			auditLog := &service.AgentAuditLog{
				AgentTokenID:   token.ID,
				AgentTokenName: token.Name,
				TaskGoal:       taskGoal,
				IPHash:         service.HashIP(c.ClientIP()),
				Method:         c.Request.Method,
				Path:           c.Request.URL.Path,
				ScopeRequired:  requiredScope,
				RequestSummary: summary,
				StatusCode:     statusCode,
				LatencyMs:      latencyMs,
				ErrorMessage:   errMsg,
				CreatedAt:      time.Now().UTC(),
			}

			go func(log *service.AgentAuditLog) {
				_ = tokenService.RecordAuditLog(context.Background(), log)
			}(auditLog)
		}
	}
}

// GetAgentTokenFromContext 从 Context 获取 AgentToken
func GetAgentTokenFromContext(c *gin.Context) *service.AgentToken {
	if val, ok := c.Get(ContextKeyAgentToken); ok {
		if token, ok := val.(*service.AgentToken); ok {
			return token
		}
	}
	return nil
}
