---
covered_paths:
  - backend/internal/service/baizhi_service.go
  - backend/internal/handler/admin/baizhi_handler.go
  - backend/internal/handler/gateway_baizhi.go
  - backend/internal/server/routes/admin.go
  - backend/internal/server/routes/gateway.go
  - frontend/src/api/admin/baizhi.ts
  - frontend/src/views/admin/BaizhiView.vue
last_verified_commit: ecb603bc7
last_verified: 2026-11-08
---

# 百智云（Baizhi Cloud）聚合网关与多账号分发子系统

## 核心概念

- **四类独立业务集成**：
  1. **网页抓取与结构化提取 (Web Scrape)**：上游 `https://web-scrape.app.baizhi.cloud`，涵盖单页抓取、指定字段提取、提取建议、批量多页提取与整理。
  2. **网络通用/AI搜索 (Web Search)**：上游 `https://websearch.app.baizhi.cloud`，涵盖通用网页搜索、AI综合搜索 (SSE流式)、近期新闻搜索、文搜图与以图搜图。
  3. **智能语音/实时对话 (Voice ASR/TTS)**：上游 HTTP/WSS `asrmodel.app.baizhi.cloud/openapi/v1`，涵盖 WebSocket 实时语音双工识别/合成、录音文件转写、分句导出与音色试听。
  4. **图像生成与编辑 (Image Generation)**：上游 `https://image-generate.app.baizhi.cloud/openapi/v1`，涵盖文生图、图生图、预签名对象直传、任务轮询与重试。

- **多账号与多 Key 负载均衡 (Key Pool)**：
  每个业务类型支持配置多个不同账号及其 API Key，网关层通过线程安全的原子计数器 (`atomic.Uint64`) 对启用的 Key 进行轮询 (Round-Robin) 分发或故障转移备选，实现调用配额打散与多账号协同。

- **透明鉴权代理与身份注入**：
  - 客户端通过携带 Sub2API 发放的统一 Bearer Token 访问网关路由。
  - 网关根据当前服务配置的 `auth_required` 校验调用方身份。
  - 转发到百智云上游服务前，自动将请求头中的 `Authorization` 剥离并替换为当前轮询选中的百智云真实 API Key (`Bearer <upstream_key>`)。

## 关键数据流与协议处理

```
客户端请求 (Bearer <Sub2API_Key>)
    │
    ▼ [/baizhi/scrape/* | /baizhi/search/* | /baizhi/voice/* | /baizhi/image/*]
[Sub2API GatewayBaizhiHandler]
    │  1. 路由与前缀剥离，还原上游子路径
    │  2. 身份校验 (可选 auth_required，依赖 User/Key 模型)
    │  3. KeyPool 轮询选择可用 upstream API Key
    │  4. 协议适配分支：
    ├── WebSocket 双向透传 (Voice /realtime): 劫持 HTTP 升级为 WS，双协程互推消息
    ├── SSE 流式推流 (Search /openapi/ai_search): 禁用响应缓冲，逐 chunk Flush
    ├── Multipart 原图/音频上传: 直接透传 Content-Type (含 boundary) 与原始 body 流
    └── 标准 HTTP REST: 透传 Header、Query 参数与 JSON/二进制响应
    ▼
百智云对应云端端点
```

## 配置存储与安全性保证

1. **存储设计**：
   - 使用系统现有的 `SettingService`，将全量配置以 JSON 字符串存储于 PostgreSQL `settings` 表中，键名为 `baizhi_cloud_config`。
   - 无需对核心数据库做 DDL 迁移，升级与回退零停机、零脏数据风险。
2. **密钥脱敏防泄露**：
   - 管理后台 API `GET /api/v1/admin/baizhi/config` 返回时，自动对所有百智云 API Key 执行保留首尾各 4 位的脱敏掩码（如 `sk-baizhi-****1234`）。
   - 管理后台保存配置 `POST /api/v1/admin/baizhi/config` 时，网关层比对现有存储，若某 Key 未变更且处于脱敏状态，则保留原真实密文，防止掩码字符意外覆盖有效 Key。
3. **连通性探测诊断**：
   - 接口 `POST /api/v1/admin/baizhi/test` 支持在管理界面中单键触发对应上游服务（如请求 `/openapi/v1/pricing`），实时回显网络延迟 (ms) 与连通状态。
