package manage

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ManageHandler 为外部 AI Agent 提供标准化、安全收敛的日常运维接口
type ManageHandler struct {
	accountRepo      service.AccountRepository
	proxyRepo        service.ProxyRepository
	userRepo         service.UserRepository
	redeemService    *service.RedeemService
	dashboardService *service.DashboardService
	tokenService     *service.AgentTokenService
	settingService   *service.SettingService
	redisClient      *redis.Client
	db               *sql.DB
}

func NewManageHandler(
	accountRepo service.AccountRepository,
	proxyRepo service.ProxyRepository,
	userRepo service.UserRepository,
	redeemService *service.RedeemService,
	dashboardService *service.DashboardService,
	tokenService *service.AgentTokenService,
	settingService *service.SettingService,
	redisClient *redis.Client,
	db *sql.DB,
) *ManageHandler {
	return &ManageHandler{
		accountRepo:      accountRepo,
		proxyRepo:        proxyRepo,
		userRepo:         userRepo,
		redeemService:    redeemService,
		dashboardService: dashboardService,
		tokenService:     tokenService,
		settingService:   settingService,
		redisClient:      redisClient,
		db:               db,
	}
}

// ---------------------------------------------------------
// 1. 状态感知层 (Telemetry / Read)
// ---------------------------------------------------------

// GetTelemetryOverview 获取系统全景监控大盘与待办告警
// GET /api/manage/telemetry/overview
func (h *ManageHandler) GetTelemetryOverview(c *gin.Context) {
	ctx := c.Request.Context()
	stats, err := h.dashboardService.GetDashboardStats(ctx)
	if err != nil {
		response.InternalError(c, "Failed to retrieve system telemetry")
		return
	}

	// 计算健康率
	accountHealthRate := 100.0
	if stats.TotalAccounts > 0 {
		accountHealthRate = float64(stats.NormalAccounts) / float64(stats.TotalAccounts) * 100.0
	}

	// 收集待办预警项（供 Agent 快速决定处置动作）
	pendingIssues := make([]string, 0)
	if stats.ErrorAccounts > 0 {
		pendingIssues = append(pendingIssues, fmt.Sprintf("%d 个上游账号处于错误状态", stats.ErrorAccounts))
	}
	if stats.RateLimitAccounts > 0 {
		pendingIssues = append(pendingIssues, fmt.Sprintf("%d 个上游账号触发速率限制冷却", stats.RateLimitAccounts))
	}
	if stats.OverloadAccounts > 0 {
		pendingIssues = append(pendingIssues, fmt.Sprintf("%d 个上游账号处于负载过载保护", stats.OverloadAccounts))
	}

	overview := gin.H{
		"system_time": time.Now().UTC(),
		"health_summary": gin.H{
			"account_health_rate": fmt.Sprintf("%.1f%%", accountHealthRate),
			"has_anomalies":       len(pendingIssues) > 0,
			"pending_issues":      pendingIssues,
		},
		"accounts": gin.H{
			"total":      stats.TotalAccounts,
			"normal":     stats.NormalAccounts,
			"error":      stats.ErrorAccounts,
			"rate_limit": stats.RateLimitAccounts,
			"overload":   stats.OverloadAccounts,
		},
		"users": gin.H{
			"total":     stats.TotalUsers,
			"active":    stats.ActiveUsers,
			"today_new": stats.TodayNewUsers,
		},
		"traffic_today": gin.H{
			"requests":     stats.TodayRequests,
			"total_tokens": stats.TodayTokens,
			"actual_cost":  stats.TodayActualCost,
			"rpm":          stats.Rpm,
			"tpm":          stats.Tpm,
		},
	}

	response.Success(c, overview)
}

// AccountSummaryVO 脱敏后的账号概要（绝不泄露凭证、密钥与内部Token）
type AccountSummaryVO struct {
	ID               int64      `json:"id"`
	Name             string     `json:"name"`
	Platform         string     `json:"platform"`
	Type             string     `json:"type"`
	Schedulable      bool       `json:"schedulable"`
	Status           string     `json:"status"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	RateLimited      bool       `json:"rate_limited"`
	RateLimitResetAt *time.Time `json:"rate_limit_reset_at,omitempty"`
	Overloaded       bool       `json:"overloaded"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ListAccounts 分页筛选上游账号
// GET /api/manage/accounts
func (h *ManageHandler) ListAccounts(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	platform := strings.TrimSpace(c.Query("platform"))
	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))
	schedulableStr := strings.TrimSpace(c.Query("schedulable"))

	accounts, paginationResult, err := h.accountRepo.ListWithFilters(
		c.Request.Context(),
		pagination.PaginationParams{Page: page, PageSize: pageSize},
		platform,
		"", // accountType
		status,
		search,
		0,  // groupID
		"", // privacyMode
	)
	if err != nil {
		response.InternalError(c, "Failed to query accounts")
		return
	}

	now := time.Now().UTC()
	// 转换为安全脱敏 VO
	results := make([]AccountSummaryVO, 0, len(accounts))
	for _, a := range accounts {
		if schedulableStr != "" {
			reqSched := schedulableStr == "true"
			if a.Schedulable != reqSched {
				continue
			}
		}

		isRateLimited := a.RateLimitResetAt != nil && a.RateLimitResetAt.After(now)
		isOverloaded := a.OverloadUntil != nil && a.OverloadUntil.After(now)

		results = append(results, AccountSummaryVO{
			ID:               a.ID,
			Name:             a.Name,
			Platform:         a.Platform,
			Type:             a.Type,
			Schedulable:      a.Schedulable,
			Status:           a.Status,
			ErrorMessage:     a.ErrorMessage,
			RateLimited:      isRateLimited,
			RateLimitResetAt: a.RateLimitResetAt,
			Overloaded:       isOverloaded,
			CreatedAt:        a.CreatedAt,
			UpdatedAt:        a.UpdatedAt,
		})
	}

	total := int64(len(results))
	if paginationResult != nil {
		total = paginationResult.Total
	}

	response.Paginated(c, results, total, page, pageSize)
}

// GetAccount 查询单个账号详情与健康状态（脱敏）
// GET /api/manage/accounts/:id
func (h *ManageHandler) GetAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	a, err := h.accountRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == service.ErrAccountNotFound {
			response.NotFound(c, "Account not found")
			return
		}
		response.InternalError(c, "Failed to get account details")
		return
	}

	now := time.Now().UTC()
	isRateLimited := a.RateLimitResetAt != nil && a.RateLimitResetAt.After(now)
	isOverloaded := a.OverloadUntil != nil && a.OverloadUntil.After(now)

	res := AccountSummaryVO{
		ID:               a.ID,
		Name:             a.Name,
		Platform:         a.Platform,
		Type:             a.Type,
		Schedulable:      a.Schedulable,
		Status:           a.Status,
		ErrorMessage:     a.ErrorMessage,
		RateLimited:      isRateLimited,
		RateLimitResetAt: a.RateLimitResetAt,
		Overloaded:       isOverloaded,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}

	response.Success(c, res)
}

// ListProxies 获取代理节点列表与健康状态
// GET /api/manage/proxies
func (h *ManageHandler) ListProxies(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	status := strings.TrimSpace(c.Query("status"))
	protocol := strings.TrimSpace(c.Query("protocol"))
	search := strings.TrimSpace(c.Query("search"))

	proxies, paginationResult, err := h.proxyRepo.ListWithFilters(
		c.Request.Context(),
		pagination.PaginationParams{Page: page, PageSize: pageSize},
		protocol,
		status,
		search,
	)
	if err != nil {
		response.InternalError(c, "Failed to query proxies")
		return
	}

	type ProxyVO struct {
		ID        int64     `json:"id"`
		Name      string    `json:"name"`
		Protocol  string    `json:"protocol"`
		Host      string    `json:"host"`
		Port      int       `json:"port"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}

	res := make([]ProxyVO, 0, len(proxies))
	for _, p := range proxies {
		res = append(res, ProxyVO{
			ID:        p.ID,
			Name:      p.Name,
			Protocol:  p.Protocol,
			Host:      p.Host,
			Port:      p.Port,
			Status:    p.Status,
			CreatedAt: p.CreatedAt,
			UpdatedAt: p.UpdatedAt,
		})
	}

	total := int64(len(res))
	if paginationResult != nil {
		total = paginationResult.Total
	}

	response.Paginated(c, res, total, page, pageSize)
}

// ListUsers 分页查询终端用户
// GET /api/manage/users
func (h *ManageHandler) ListUsers(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	search := strings.TrimSpace(c.Query("search"))
	status := strings.TrimSpace(c.Query("status"))

	users, paginationResult, err := h.userRepo.ListWithFilters(
		c.Request.Context(),
		pagination.PaginationParams{Page: page, PageSize: pageSize},
		service.UserListFilters{
			Search: search,
			Status: status,
		},
	)
	if err != nil {
		response.InternalError(c, "Failed to query users")
		return
	}

	type UserSummaryVO struct {
		ID             int64      `json:"id"`
		Email          string     `json:"email"`
		Role           string     `json:"role"`
		Status         string     `json:"status"`
		Balance        float64    `json:"balance"`
		Concurrency    int        `json:"concurrency"`
		TotalRecharged float64    `json:"total_recharged"`
		LastActiveAt   *time.Time `json:"last_active_at,omitempty"`
		CreatedAt      time.Time  `json:"created_at"`
	}

	res := make([]UserSummaryVO, 0, len(users))
	for _, u := range users {
		res = append(res, UserSummaryVO{
			ID:             u.ID,
			Email:          u.Email,
			Role:           u.Role,
			Status:         u.Status,
			Balance:        u.Balance,
			Concurrency:    u.Concurrency,
			TotalRecharged: u.TotalRecharged,
			LastActiveAt:   u.LastActiveAt,
			CreatedAt:      u.CreatedAt,
		})
	}

	total := int64(len(res))
	if paginationResult != nil {
		total = paginationResult.Total
	}

	response.Paginated(c, res, total, page, pageSize)
}

// ListRedeemCodes 分页查询卡密列表
// GET /api/manage/redeem-codes
func (h *ManageHandler) ListRedeemCodes(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	codeType := strings.TrimSpace(c.Query("type"))
	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))

	codes, paginationResult, err := h.redeemService.List(c.Request.Context(), pagination.PaginationParams{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		response.InternalError(c, "Failed to query redeem codes")
		return
	}

	// 简单过滤
	filtered := make([]service.RedeemCode, 0, len(codes))
	for _, item := range codes {
		if codeType != "" && item.Type != codeType {
			continue
		}
		if status != "" && item.Status != status {
			continue
		}
		if search != "" && !strings.Contains(item.Code, search) {
			continue
		}
		filtered = append(filtered, item)
	}

	total := int64(len(filtered))
	if paginationResult != nil {
		total = paginationResult.Total
	}

	response.Paginated(c, filtered, total, page, pageSize)
}

// ListAuditLogs 查询当前 Agent 的审计操作轨迹
// GET /api/manage/audit-logs
func (h *ManageHandler) ListAuditLogs(c *gin.Context) {
	token := middleware.GetAgentTokenFromContext(c)
	if token == nil {
		response.Unauthorized(c, "Agent token not found in context")
		return
	}

	page, pageSize := response.ParsePagination(c)
	logs, total, err := h.tokenService.ListAuditLogs(c.Request.Context(), &service.AgentAuditLogFilter{
		AgentTokenID: &token.ID,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		response.InternalError(c, "Failed to list audit logs")
		return
	}

	response.Paginated(c, logs, total, page, pageSize)
}

// ---------------------------------------------------------
// 2. 实体流转层 (Lifecycle / Write)
// ---------------------------------------------------------

type SetSchedulableRequest struct {
	Schedulable bool   `json:"schedulable"`
	Reason      string `json:"reason"`
}

// SetAccountSchedulable 调整单个账号调度状态（幂等）
// PATCH /api/manage/accounts/:id/schedulable
func (h *ManageHandler) SetAccountSchedulable(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	var req SetSchedulableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request payload")
		return
	}

	ctx := c.Request.Context()
	account, err := h.accountRepo.GetByID(ctx, id)
	if err != nil {
		if err == service.ErrAccountNotFound {
			response.NotFound(c, "Account not found")
			return
		}
		response.InternalError(c, "Failed to fetch account")
		return
	}

	// 幂等更新
	if err := h.accountRepo.SetSchedulable(ctx, id, req.Schedulable); err != nil {
		response.InternalError(c, "Failed to update account schedulable status")
		return
	}

	response.Success(c, gin.H{
		"account_id":  id,
		"name":        account.Name,
		"schedulable": req.Schedulable,
		"reason":      req.Reason,
		"updated_at":  time.Now().UTC(),
	})
}

// RefreshAccount 触发账号探活刷新
// POST /api/manage/accounts/:id/refresh
func (h *ManageHandler) RefreshAccount(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid account ID")
		return
	}

	ctx := c.Request.Context()
	account, err := h.accountRepo.GetByID(ctx, id)
	if err != nil {
		if err == service.ErrAccountNotFound {
			response.NotFound(c, "Account not found")
			return
		}
		response.InternalError(c, "Failed to fetch account")
		return
	}

	// 清理账号的历史错误状态以支持重新探测
	_ = h.accountRepo.ClearError(ctx, id)
	_ = h.accountRepo.ClearRateLimit(ctx, id)

	response.Success(c, gin.H{
		"account_id":   id,
		"name":         account.Name,
		"action":       "refreshed",
		"refreshed_at": time.Now().UTC(),
	})
}

type SetStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// SetProxyStatus 启用/停用代理节点
// PATCH /api/manage/proxies/:id/status
func (h *ManageHandler) SetProxyStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid proxy ID")
		return
	}

	var req SetStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request payload")
		return
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status != "active" && status != "disabled" {
		response.BadRequest(c, "Status must be either 'active' or 'disabled'")
		return
	}

	ctx := c.Request.Context()
	proxy, err := h.proxyRepo.GetByID(ctx, id)
	if err != nil {
		if err == service.ErrProxyNotFound {
			response.NotFound(c, "Proxy not found")
			return
		}
		response.InternalError(c, "Failed to fetch proxy")
		return
	}

	proxy.Status = status
	if err := h.proxyRepo.Update(ctx, proxy); err != nil {
		response.InternalError(c, "Failed to update proxy status")
		return
	}

	response.Success(c, gin.H{
		"proxy_id":   id,
		"name":       proxy.Name,
		"status":     status,
		"updated_at": time.Now().UTC(),
	})
}

// SetUserStatus 封禁或解禁用户（安全红线：绝不允许封禁管理员）
// PATCH /api/manage/users/:id/status
func (h *ManageHandler) SetUserStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid user ID")
		return
	}

	var req SetStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request payload")
		return
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	if status != "active" && status != "disabled" {
		response.BadRequest(c, "Status must be either 'active' or 'disabled'")
		return
	}

	ctx := c.Request.Context()
	user, err := h.userRepo.GetByID(ctx, id)
	if err != nil {
		response.NotFound(c, "User not found")
		return
	}

	// 严密安全防线：禁止 Agent 操控管理员账号
	if user.Role == "admin" {
		response.Forbidden(c, "Cannot mutate status of administrator accounts")
		return
	}

	user.Status = status
	if err := h.userRepo.Update(ctx, user, service.UserUpdateFields{Status: true}); err != nil {
		response.InternalError(c, "Failed to update user status")
		return
	}

	response.Success(c, gin.H{
		"user_id":    id,
		"email":      user.Email,
		"status":     status,
		"updated_at": time.Now().UTC(),
	})
}

type GenerateRedeemCodesRequest struct {
	Count int     `json:"count" binding:"required"`
	Type  string  `json:"type" binding:"required"` // balance
	Value float64 `json:"value" binding:"required"`
}

// GenerateRedeemCodes 受控批量生成卡密（安全防线：单次上限 20 张）
// POST /api/manage/redeem-codes/generate
func (h *ManageHandler) GenerateRedeemCodes(c *gin.Context) {
	var req GenerateRedeemCodesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid payload: "+err.Error())
		return
	}

	if req.Count <= 0 || req.Count > 20 {
		response.BadRequest(c, "Count must be between 1 and 20")
		return
	}
	if req.Value <= 0 || req.Value > 1000 {
		response.BadRequest(c, "Value must be positive and not exceed 1000")
		return
	}
	if req.Type != "balance" {
		response.BadRequest(c, "Only 'balance' redeem codes can be generated via Agent API")
		return
	}

	codes, err := h.redeemService.GenerateCodes(c.Request.Context(), service.GenerateCodesRequest{
		Count: req.Count,
		Type:  req.Type,
		Value: req.Value,
	})
	if err != nil {
		response.InternalError(c, "Failed to generate redeem codes: "+err.Error())
		return
	}

	type GeneratedCodeItem struct {
		Code  string  `json:"code"`
		Type  string  `json:"type"`
		Value float64 `json:"value"`
	}

	resultList := make([]GeneratedCodeItem, 0, len(codes))
	for _, cd := range codes {
		resultList = append(resultList, GeneratedCodeItem{
			Code:  cd.Code,
			Type:  cd.Type,
			Value: cd.Value,
		})
	}

	response.Success(c, gin.H{
		"generated_count": len(resultList),
		"codes":           resultList,
	})
}

// ---------------------------------------------------------
// 3. 运维动作层 (Ops / Actions)
// ---------------------------------------------------------

type BatchAccountsSchedulableRequest struct {
	AccountIDs  []int64 `json:"account_ids" binding:"required"`
	Schedulable bool    `json:"schedulable"`
	DryRun      bool    `json:"dry_run"`
	Reason      string  `json:"reason"`
}

// BatchAccountsSchedulable 批量切换账号调度状态（设单次上限 50，支持 dryRun 预检）
// POST /api/manage/ops/batch/accounts/schedulable
func (h *ManageHandler) BatchAccountsSchedulable(c *gin.Context) {
	var req BatchAccountsSchedulableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request payload: "+err.Error())
		return
	}

	if len(req.AccountIDs) == 0 {
		response.BadRequest(c, "AccountIDs list cannot be empty")
		return
	}
	if len(req.AccountIDs) > 50 {
		response.BadRequest(c, "Batch operation exceeds maximum limit of 50 accounts")
		return
	}

	ctx := c.Request.Context()
	accounts, err := h.accountRepo.GetByIDs(ctx, req.AccountIDs)
	if err != nil {
		response.InternalError(c, "Failed to query target accounts")
		return
	}

	type DryRunItem struct {
		ID            int64  `json:"id"`
		Name          string `json:"name"`
		Platform      string `json:"platform"`
		CurrentStatus bool   `json:"current_schedulable"`
		TargetStatus  bool   `json:"target_schedulable"`
		NeedsChange   bool   `json:"needs_change"`
	}

	preview := make([]DryRunItem, 0, len(accounts))
	affectedCount := 0
	for _, acc := range accounts {
		needsChange := acc.Schedulable != req.Schedulable
		if needsChange {
			affectedCount++
		}
		preview = append(preview, DryRunItem{
			ID:            acc.ID,
			Name:          acc.Name,
			Platform:      acc.Platform,
			CurrentStatus: acc.Schedulable,
			TargetStatus:  req.Schedulable,
			NeedsChange:   needsChange,
		})
	}

	// 如果是预检模式，只返回预检分析
	if req.DryRun {
		response.Success(c, gin.H{
			"dry_run":        true,
			"total_checked":  len(accounts),
			"would_affect":   affectedCount,
			"target_status":  req.Schedulable,
			"preview":        preview,
		})
		return
	}

	// 正式执行更新
	updatedCount := 0
	for _, acc := range accounts {
		if acc.Schedulable != req.Schedulable {
			if err := h.accountRepo.SetSchedulable(ctx, acc.ID, req.Schedulable); err == nil {
				updatedCount++
			}
		}
	}

	response.Success(c, gin.H{
		"dry_run":        false,
		"updated_count":  updatedCount,
		"target_status":  req.Schedulable,
		"reason":         req.Reason,
		"executed_at":    time.Now().UTC(),
	})
}

type FlushCacheRequest struct {
	Target string `json:"target"` // "all" | "routes" | "settings"
}

// FlushCache 刷新特定系统与路由缓存
// POST /api/manage/ops/cache/flush
func (h *ManageHandler) FlushCache(c *gin.Context) {
	var req FlushCacheRequest
	_ = c.ShouldBindJSON(&req)
	target := strings.ToLower(strings.TrimSpace(req.Target))
	if target == "" {
		target = "all"
	}

	ctx := c.Request.Context()
	flushed := make([]string, 0)

	if target == "all" || target == "settings" {
		if h.redisClient != nil {
			// 清理系统通用缓存键
			keys, _ := h.redisClient.Keys(ctx, "setting:*").Result()
			if len(keys) > 0 {
				_ = h.redisClient.Del(ctx, keys...).Err()
			}
		}
		flushed = append(flushed, "settings_cache")
	}

	if target == "all" || target == "routes" {
		if h.redisClient != nil {
			keys, _ := h.redisClient.Keys(ctx, "route:*").Result()
			if len(keys) > 0 {
				_ = h.redisClient.Del(ctx, keys...).Err()
			}
		}
		flushed = append(flushed, "route_cache")
	}

	response.Success(c, gin.H{
		"flushed_targets": flushed,
		"executed_at":     time.Now().UTC(),
	})
}

type CleanupLogsRequest struct {
	Days   int  `json:"days"`
	DryRun bool `json:"dry_run"`
}

// CleanupLogsTask 受控清理过期的临时调用日志（支持 dryRun 预检）
// POST /api/manage/ops/tasks/cleanup-logs
func (h *ManageHandler) CleanupLogsTask(c *gin.Context) {
	var req CleanupLogsRequest
	_ = c.ShouldBindJSON(&req)
	days := req.Days
	if days <= 0 {
		days = 30 // 默认清理 30 天以前
	}
	if days < 7 {
		response.BadRequest(c, "Retention days cannot be less than 7 days")
		return
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	ctx := c.Request.Context()

	// 预检：查询截止时间之前的数据量
	var count int64
	countQuery := "SELECT COUNT(*) FROM usage_logs WHERE created_at < $1"
	if err := h.db.QueryRowContext(ctx, countQuery, cutoff).Scan(&count); err != nil {
		response.InternalError(c, "Failed to count expired logs")
		return
	}

	if req.DryRun {
		response.Success(c, gin.H{
			"dry_run":        true,
			"cutoff_time":    cutoff,
			"retention_days": days,
			"estimated_rows": count,
		})
		return
	}

	// 正式受控清理（单次批量上限 10000 行，避免长事务锁表）
	deleteQuery := `
		DELETE FROM usage_logs
		WHERE id IN (
			SELECT id FROM usage_logs WHERE created_at < $1 LIMIT 10000
		)
	`
	res, err := h.db.ExecContext(ctx, deleteQuery, cutoff)
	if err != nil {
		response.InternalError(c, "Failed to cleanup logs: "+err.Error())
		return
	}
	deleted, _ := res.RowsAffected()

	response.Success(c, gin.H{
		"dry_run":        false,
		"cutoff_time":    cutoff,
		"retention_days": days,
		"deleted_rows":   deleted,
		"remaining_rows": count - deleted,
		"executed_at":    time.Now().UTC(),
	})
}
