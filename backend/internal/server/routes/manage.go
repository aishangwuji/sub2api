package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler/manage"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// RegisterManageRoutes 注册面向 AI Agent 的标准化运维接口（收敛于 /api/manage/*）
func RegisterManageRoutes(
	r *gin.Engine,
	h *manage.ManageHandler,
	agentAuth middleware.AgentAuthMiddleware,
) {
	manageGroup := r.Group("/api/manage")
	{
		// 1. 状态感知层 (Telemetry / Read)
		manageGroup.GET("/telemetry/overview", agentAuth(service.ScopeStatusRead), h.GetTelemetryOverview)
		manageGroup.GET("/accounts", agentAuth(service.ScopeAccountsRead), h.ListAccounts)
		manageGroup.GET("/accounts/:id", agentAuth(service.ScopeAccountsRead), h.GetAccount)
		manageGroup.GET("/proxies", agentAuth(service.ScopeProxiesRead), h.ListProxies)
		manageGroup.GET("/users", agentAuth(service.ScopeUsersRead), h.ListUsers)
		manageGroup.GET("/redeem-codes", agentAuth(service.ScopeRedeemManage), h.ListRedeemCodes)
		manageGroup.GET("/audit-logs", agentAuth(service.ScopeStatusRead), h.ListAuditLogs)

		// 2. 实体流转层 (Lifecycle / Write)
		manageGroup.PATCH("/accounts/:id/schedulable", agentAuth(service.ScopeAccountsWrite), h.SetAccountSchedulable)
		manageGroup.POST("/accounts/:id/refresh", agentAuth(service.ScopeAccountsWrite), h.RefreshAccount)
		manageGroup.PATCH("/proxies/:id/status", agentAuth(service.ScopeProxiesWrite), h.SetProxyStatus)
		manageGroup.PATCH("/users/:id/status", agentAuth(service.ScopeUsersManage), h.SetUserStatus)
		manageGroup.POST("/redeem-codes/generate", agentAuth(service.ScopeRedeemManage), h.GenerateRedeemCodes)

		// 3. 运维动作层 (Ops / Actions)
		opsGroup := manageGroup.Group("/ops")
		{
			opsGroup.POST("/batch/accounts/schedulable", agentAuth(service.ScopeAccountsWrite), h.BatchAccountsSchedulable)
			opsGroup.POST("/cache/flush", agentAuth(service.ScopeOpsTrigger), h.FlushCache)
			opsGroup.POST("/tasks/cleanup-logs", agentAuth(service.ScopeOpsTrigger), h.CleanupLogsTask)
		}
	}
}
