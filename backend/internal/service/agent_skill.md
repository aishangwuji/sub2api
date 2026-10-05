---
name: sub2api-ops
description: Automate Sub2API gateway operations, account health monitoring, proxy status checks, user management, and safe maintenance tasks. Use when the user asks to check Sub2API health, inspect broken accounts, toggle upstream scheduling, check proxies, manage balance recharge codes, flush caches, or perform safe daily automated site administration.
---

# Sub2API Agent 自动化运维与管理规范 (Sub2API Ops Skill)

本文档专为具备自主决策能力的 **AI Agent（运维助手 / 巡检机器人）** 设计。通过调用标准的 `/api/manage/*` API，Agent 可以协助人类管理员对 Sub2API 系统执行自动化监控、状态流转与故障处置。

---

## 1. 基础配置与统一请求头

### 1.1 Base URL
- 生产环境：`https://sub.originagent.cn`（或环境变量 `SUB2API_BASE_URL`）
- 本地调试：`http://127.0.0.1:3005` 或 `http://localhost:8080`

### 1.2 认证与上下文请求头（必须）
Agent 发送的每一个 HTTP 请求都必须携带以下头部：

| 标头名 | 是否必须 | 说明与示例 |
|---|---|---|
| `Authorization` | 推荐 | `Bearer sub2api_agt_<32位随机凭证>` |
| `X-Agent-Token` | 可选 | 若无法设置 Authorization 标头，可作为备选：`sub2api_agt_xxx` |
| `X-Agent-Task-Goal` | **强烈建议** | 描述当前动作的任务意图，将被记录在系统不可篡改的审计轨迹中。例：`"Daily patrol: inspect and pause degraded openai accounts"` |
| `Content-Type` | 写请求必须 | `application/json` |

---

## 2. 渐进式工作流指引（Progressive Workflow）

为避免盲目拉取巨量数据导致消耗大量 LLM 上下文 Token，Agent 必须遵循以下 **“感知 -> 排查 -> 预检 -> 处置 -> 验证”** 递进原则：

```
┌─────────────────────────────────┐
│ 1. 状态感知 (Telemetry)         │
│ GET /api/manage/telemetry/overview
└────────────────┬────────────────┘
                 │
                 ▼ 是否存在 pending_issues？
        [ 否: 全局正常, 直接结束 ]
        [ 是: 锁定异常资源 ]
                 │
                 ▼
┌─────────────────────────────────┐
│ 2. 精确排查 (Filter)            │
│ GET /api/manage/accounts?status=error
└────────────────┬────────────────┘
                 │
                 ▼ 影响范围大于 1 还是批量操作？
┌─────────────────────────────────┐
│ 3. 预检模式 (Dry Run)           │
│ POST /api/manage/ops/batch/...  │ (dry_run: true)
└────────────────┬────────────────┘
                 │
                 ▼ 影响评估确认无误
┌─────────────────────────────────┐
│ 4. 正式处置 (Execute)           │
│ PATCH /api/manage/accounts/:id  │
└────────────────┬────────────────┘
                 │
                 ▼
┌─────────────────────────────────┐
│ 5. 审计留痕核验 (Verify)        │
│ GET /api/manage/audit-logs      │
└─────────────────────────────────┘
```

---

## 3. 端点速查表与接口规范

### 3.1 状态感知层 (Telemetry / Read)

#### `GET /api/manage/telemetry/overview`
- **所需 Scope**：`status:read`
- **功能**：获取全站健康摘要、账号健康率、异常告警待办列表以及今日流量大盘。
- **示例响应**：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "system_time": "2026-10-05T23:50:00Z",
    "health_summary": {
      "account_health_rate": "92.5%",
      "has_anomalies": true,
      "pending_issues": [
        "3 个上游账号处于错误状态",
        "1 个上游账号触发速率限制冷却"
      ]
    },
    "accounts": {
      "total": 40,
      "normal": 37,
      "error": 3,
      "rate_limit": 1,
      "overload": 0
    },
    "traffic_today": {
      "requests": 15420,
      "total_tokens": 1289000,
      "actual_cost": 42.15,
      "rpm": 45,
      "tpm": 12000
    }
  }
}
```

#### `GET /api/manage/accounts`
- **所需 Scope**：`accounts:read`
- **查询参数**：
  - `page`（默认 1）
  - `page_size`（默认 20，上限 100）
  - `platform`（如 `openai`, `anthropic`, `gemini`）
  - `status`（如 `error`, `normal`）
  - `schedulable`（`true` 或 `false`）
  - `search`（按名称模糊检索）
- **注意**：返回结果中绝不包含账号密码与 API Key 等敏感材料。

#### `GET /api/manage/accounts/:id`
- **所需 Scope**：`accounts:read`
- **功能**：查看单个账号健康详情、限流状态 (`rate_limited`) 与过载状态 (`overloaded`)。

#### `GET /api/manage/proxies`
- **所需 Scope**：`proxies:read`
- **查询参数**：`status`（`active` 或 `disabled`）、`protocol`、`page`、`page_size`

#### `GET /api/manage/users`
- **所需 Scope**：`users:read`
- **查询参数**：`status`、`search`、`page`、`page_size`
- **功能**：查询用户余额、激活状态与并发限制（密码与加密信息已被严格过滤）。

#### `GET /api/manage/redeem-codes`
- **所需 Scope**：`redeem:manage`
- **查询参数**：`type`、`status`（`used` / `unused`）、`page`、`page_size`

#### `GET /api/manage/audit-logs`
- **所需 Scope**：`status:read`
- **功能**：查验 Agent 自身最近的调用流水与执行结果。

---

### 3.2 实体流转层 (Lifecycle / Write)

#### `PATCH /api/manage/accounts/:id/schedulable`
- **所需 Scope**：`accounts:write`
- **说明**：启用或关闭指定上游账号的路由调度（幂等）。
- **请求体**：
```json
{
  "schedulable": false,
  "reason": "Upstream token quota exhausted"
}
```

#### `POST /api/manage/accounts/:id/refresh`
- **所需 Scope**：`accounts:write`
- **说明**：触发账号重置错误状态并探活刷新。

#### `PATCH /api/manage/proxies/:id/status`
- **所需 Scope**：`proxies:write`
- **请求体**：`{"status": "disabled"}` 或 `{"status": "active"}`

#### `PATCH /api/manage/users/:id/status`
- **所需 Scope**：`users:manage`
- **安全防线**：系统禁止 Agent 修改管理员角色的账户。
- **请求体**：`{"status": "disabled"}` 或 `{"status": "active"}`

#### `POST /api/manage/redeem-codes/generate`
- **所需 Scope**：`redeem:manage`
- **安全防线**：单次最多生成 20 张，面值上限 1000。
- **请求体**：
```json
{
  "count": 5,
  "type": "balance",
  "value": 10.0
}
```

---

### 3.3 运维动作层 (Ops / Actions)

#### `POST /api/manage/ops/batch/accounts/schedulable`
- **所需 Scope**：`accounts:write`
- **安全防线**：单次最多 50 个 ID。支持 `dry_run=true` 预检模式。
- **预检请求体**：
```json
{
  "account_ids": [101, 102, 103],
  "schedulable": false,
  "dry_run": true,
  "reason": "Bulk disabling due to 429 cluster incident"
}
```
- **预检返回**：
```json
{
  "code": 0,
  "data": {
    "dry_run": true,
    "total_checked": 3,
    "would_affect": 2,
    "preview": [
      { "id": 101, "name": "gpt4-01", "current_schedulable": true, "needs_change": true },
      { "id": 102, "name": "gpt4-02", "current_schedulable": false, "needs_change": false }
    ]
  }
}
```

#### `POST /api/manage/ops/cache/flush`
- **所需 Scope**：`ops:trigger`
- **请求体**：`{"target": "routes"}` 或 `{"target": "all"}`

#### `POST /api/manage/ops/tasks/cleanup-logs`
- **所需 Scope**：`ops:trigger`
- **安全防线**：保留期不能少于 7 天。支持 `dry_run=true` 查询将影响的条目数。
- **请求体**：`{"days": 30, "dry_run": true}`

---

## 4. 异常处理与安全准则

1. **遇 429 请求超频退避（Backoff）**：
   - 当 API 返回 `429 Too Many Requests` 时，说明触发限流保护，Agent 必须采用指数退避（等待 2s、4s、8s 后重试），切勿暴力重试。
2. **遇 403 权限不足拦截**：
   - 证明当前 Token 没有授予该接口所需的特定 Scope。Agent 不应试图越权，而应在会话中提示管理员在后台分配对应 Scope。
3. **人类确认准则（Human-in-the-Loop）**：
   - 任何涉及封禁活跃用户（`PATCH /api/manage/users/:id/status`）或批量下线 5 个以上核心账号的操作，Agent 应在最终执行前向人类明确列出目标清单并请求确认。
