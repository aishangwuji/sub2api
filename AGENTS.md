# AGENTS · Sub2API 运维上线与协同开发手册

> **定位声明**：本文档旨在让任意 AI 智能体（Agent）或运维协作者在 `154.9.224.192 (hnw)` 服务器上，全面掌握 **Sub2API**（AI 订阅转换与 API 网关）的架构拓扑、配置基线、运维指令、故障排查与灾备规范。
>
> **线上站点**：[https://sub.originagent.cn](https://sub.originagent.cn)  
> **宿主机部署路径**：`/opt/sub2api`  
> **服务编排模式**：Docker Compose（主服务 + 独立 PostgreSQL 18 + 独立 Redis 8）  
> **服务器架构权威文档源**：`/opt/server-architecture/11-app-sub2api.md`、`00-README.md`、`07-operations-guide.md`  
> **代码仓库**：[https://github.com/aishangwuji/sub2api](https://github.com/aishangwuji/sub2api)（Upstream: [https://github.com/Wei-Shaw/sub2api](https://github.com/Wei-Shaw/sub2api)）

---

## 1. 架构速览与网络拓扑

### 1.1 流量分发与代理链路

```
客户端请求 (https://sub.originagent.cn)
    │
    ▼ (公网端口 443)
[Nginx Stream SNI 分流] (/etc/nginx/nginx.conf)
    │   preread $ssl_preread_server_name
    │   sub.originagent.cn -> upstream nginx_https
    ▼ (本地端口 127.0.0.1:8444)
[Nginx HTTP SSL 服务块] (/etc/nginx/conf.d/sub-ssl.conf)
    │   - 证书: /etc/letsencrypt/live/sub.originagent.cn/ (ECDSA)
    │   - 安全头: HSTS (max-age=31536000), nosniff, server_tokens off
    │   - 流式支持: proxy_buffering off, proxy_cache off, read_timeout 600s
    ▼ (本地环回反代)
[宿主机环回端口 127.0.0.1:3005]
    │
    ▼ (Docker 端口映射 127.0.0.1:3005 -> 8080)
[sub2api 容器] (Go 后端 + Vue 前端一体化服务)
    │
    ├── 内部桥接网络 sub2api-network (不对宿主机暴露端口) ──┐
    │                                                      │
    ▼                                                      ▼
[sub2api-postgres 容器]                                [sub2api-redis 容器]
(PostgreSQL 18-alpine, 内部:5432)                     (Redis 8-alpine, 内部:6379)
```

### 1.2 端口隔离与安全铁律

1. **宿主机端口严格收敛**：
   - 外部网络严禁直接监听，主服务容器仅映射至宿主机本地环回 `127.0.0.1:3005`。
   - `sub2api-postgres` 与 `sub2api-redis` **严禁映射任何宿主机端口**。
   - *特别警告*：宿主机上已运行 `postgresql@16-main`（监听 5432）以及 Docker `search_redis`（监听 `127.0.0.1:6379`），Sub2API 的专有存储必须完全在内部容器网桥 `sub2api-network` 内通信，切勿在宿主机发生端口冲突。
2. **Nginx 443 端口保护**：
   - 443 端口由 Nginx Stream SNI 分流独占，HTTP(S) 业务一律在本地 `8444 ssl` 监听，严禁在 `sub-ssl.conf` 中直接写 `listen 443 ssl`。
   - Nginx 配置更新必须走 `nginx -t` 检查后 `systemctl reload nginx`，**绝对严禁 `systemctl restart nginx`**（会导致全站流量与 Xray REALITY 隧道瞬断）。
3. **配置文件权限基线**：
   - 生产环境敏感配置文件 `/opt/sub2api/.env` 必须严格保持 `600` 权限（`chmod 600 /opt/sub2api/.env`），属主保持为 `root:root`。

---

## 2. 容器服务与资源限额基线

宿主机总内存为 **4GB**，承载了多个在线生产系统（北向行者、北行搜、文档站、BBS、SysOne 等）。为防止容器无节制消耗内存导致系统触发内核 OOM Killer，所有容器均配置了硬内存上限与参数调优：

| 容器服务名 | 基础镜像 | 内部端口 | 宿主机映射 | 内存限额 (`mem_limit`) | 资源优化与启动关键参数 |
|---|---|---|---|---|---|
| `sub2api` | `weishaw/sub2api:latest` | 8080 | `127.0.0.1:3005` | **512MB** | Go 运行时，健康检查 `wget -q http://localhost:8080/health` |
| `sub2api-postgres` | `postgres:18-alpine` | 5432 | 无（仅内部网桥） | **512MB** | `shared_buffers=128MB`, `max_connections=100`, `effective_cache_size=512MB`, `maintenance_work_mem=64MB` |
| `sub2api-redis` | `redis:8-alpine` | 6379 | 无（仅内部网桥） | **256MB** | `--save 60 1 --appendonly yes --appendfsync everysec`，密码认证 |

> **实测负载水平**：日常稳态下三容器内存总占用约为 **130MB ~ 150MB**（sub2api ~35MB, postgres ~75MB, redis ~32MB），资源控制处于极佳状态。

---

## 3. 部署目录与持久化规范

部署基准目录位于服务器 `/opt/sub2api`：

```
/opt/sub2api/
├── .env                  # 生产环境变量（权限 600，包含数据库密码、JWT/TOTP密钥等，绝不进版本库）
├── .env.example          # 环境变量模版参考
├── docker-compose.yml    # 容器编排文件
├── data/                 # 应用持久化目录 (挂载至容器 /app/data)
│   ├── .installed        # 系统初始化标识
│   ├── config.yaml       # 系统自动生成的运行时配置文件
│   ├── logs/             # 系统运行与审计日志
│   ├── model_pricing.json# 模型定价与计费字典缓存
│   ├── pages/            # 自定义静态页面
│   └── plugins/          # 插件扩展目录
├── postgres_data/        # PostgreSQL 18 独立持久化存储目录 (PGDATA)
└── redis_data/           # Redis 8 独立持久化数据目录 (dump.rdb + appendonlydir)
```

---

## 4. 核心配置与环境变量速查

在 `/opt/sub2api/.env` 中，核心生产环境变量定义如下：

```ini
# 基础服务
BIND_HOST=0.0.0.0
SERVER_PORT=8080
SERVER_MODE=release
TZ=Asia/Shanghai

# PostgreSQL 数据库配置 (容器内直连 postgres:5432)
POSTGRES_USER=sub2api
POSTGRES_PASSWORD=<自动生成的高强度伪随机密码>
POSTGRES_DB=sub2api
DATABASE_PORT=5432
DATABASE_MAX_OPEN_CONNS=20
DATABASE_MAX_IDLE_CONNS=5

# Redis 缓存配置 (容器内直连 redis:6379)
REDIS_PORT=6379
REDIS_PASSWORD=<自动生成的高强度伪随机密码>
REDIS_DB=0
REDIS_POOL_SIZE=20
REDIS_MIN_IDLE_CONNS=5

# 安全与身份认证凭证
JWT_SECRET=<64位伪随机字符>
JWT_EXPIRE_HOUR=24
TOTP_ENCRYPTION_KEY=<32位伪随机字符>

# 管理员初始化账号
ADMIN_EMAIL=admin@originagent.cn
ADMIN_PASSWORD=<管理员初始密码>

# 日志输出配置
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT_TO_STDOUT=true
LOG_ROTATION_COMPRESS=true
```

---

## 5. 日常运维与健康巡检（SOP）

所有运维操作均通过 SSH 连接服务器执行：`ssh hnw`。

### 5.1 服务状态与健康检查

```bash
# 1. 检查容器运行状态（健康检查状态应为 healthy）
cd /opt/sub2api && docker compose ps

# 2. 查看容器实时内存与 CPU 负载
docker stats --no-stream --format "table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}" | grep -E "NAME|sub2api"

# 3. 本地环回接口探活
curl -s http://127.0.0.1:3005/health
# 预期输出: {"status":"ok"}

# 4. 线上 HTTPS 入口检查
curl -sI https://sub.originagent.cn/
# 预期输出: HTTP/1.1 200 OK 并带有 Strict-Transport-Security 头
```

### 5.2 日志检索与实时监控

```bash
# 实时跟踪 Sub2API 应用日志
docker compose -f /opt/sub2api/docker-compose.yml logs -f sub2api

# 查看最近 100 行日志
docker compose -f /opt/sub2api/docker-compose.yml logs --tail=100 sub2api

# 检索错误与异常调用
docker compose -f /opt/sub2api/docker-compose.yml logs sub2api | grep -iE "error|fatal|panic"

# 查看 Nginx 访问与错误日志
tail -f /var/log/nginx/sub-ssl-access.log
tail -f /var/log/nginx/sub-ssl-error.log
```

### 5.3 服务重启与平滑更新

```bash
# 平滑重启全部容器
cd /opt/sub2api && docker compose restart

# 单独重启主服务（不重启数据库与缓存）
cd /opt/sub2api && docker compose restart sub2api

# 拉取最新镜像并平滑重建
cd /opt/sub2api
docker compose pull sub2api
docker compose up -d sub2api

# 停止与彻底拉起
cd /opt/sub2api
docker compose down
docker compose up -d
```

---

## 6. 数据备份与灾难恢复（DR）

### 6.1 PostgreSQL 数据库备份

```bash
# 创建备份目录（如不存在）
mkdir -p /opt/backups/sub2api

# 导出数据库逻辑全量备份（带时间戳）
docker exec sub2api-postgres pg_dump -U sub2api sub2api | gzip > /opt/backups/sub2api/sub2api_pg_$(date +%Y%m%d_%H%M%S).sql.gz

# 验证备份完整性
ls -lh /opt/backups/sub2api/
```

### 6.2 数据库恢复流程

```bash
# 注意：恢复前请先停止应用写入
cd /opt/sub2api && docker compose stop sub2api

# 从指定的 sql.gz 备份文件中还原数据库
gunzip -c /opt/backups/sub2api/sub2api_pg_YYYYMMDD_HHMMSS.sql.gz | docker exec -i sub2api-postgres psql -U sub2api -d sub2api

# 恢复完成后重新拉起服务
docker compose start sub2api
```

### 6.3 配置文件与 Redis 持久化备份

```bash
# 手动触发 Redis 产生 RDB 快照
docker exec sub2api-redis redis-cli BGSAVE

# 打包备份应用配置、.env 与持久化元数据
tar -czvf /opt/backups/sub2api/sub2api_config_$(date +%Y%m%d_%H%M%S).tar.gz \
  /opt/sub2api/.env \
  /opt/sub2api/docker-compose.yml \
  /opt/sub2api/data/config.yaml \
  /opt/sub2api/data/model_pricing.json
```

---

## 7. 踩坑记录与高频排障手册

### 7.1 流式对话输出被截断 / SSE 无法推送
- **原因**：Nginx 默认启用了响应缓冲（`proxy_buffering on`），导致大模型流式响应的 chunk 被暂存直至缓冲区满才输出给客户端，造成打字机效果卡顿甚至超时断连。
- **排查与修复**：检查 `/etc/nginx/conf.d/sub-ssl.conf`，确保配置了：
  ```nginx
  proxy_buffering off;
  proxy_cache off;
  proxy_connect_timeout 15s;
  proxy_send_timeout 600s;
  proxy_read_timeout 600s;
  ```

### 7.2 容器无法连接数据库（`connection refused`）
- **原因**：Sub2API 主容器如果使用 `127.0.0.1:5432` 尝试连接数据库，会访问主容器自身的 loopback 而非数据库容器；或者尝试连接宿主机的 PG16 从而认证失败。
- **排查与修复**：
  1. 确保 `.env` 或 `data/config.yaml` 中配置的数据库主机名必须是 Docker 网桥域名 `postgres`（端口 5432），Redis 主机名必须是 `redis`（端口 6379）。
  2. 运行 `docker compose ps` 确认 `sub2api-postgres` 处于 `Up (healthy)` 状态。

### 7.3 SSL 证书续期与 ACME Challenge 机制
- **证书管理**：Certbot 自动续期服务管理 `/etc/letsencrypt/live/sub.originagent.cn/`。
- **ACME 路由**：`/etc/nginx/conf.d/acme-sub.conf` 捕获 80 端口的 `/.well-known/acme-challenge/` 并映射到本地 `/var/www/certbot`。
- **测试续期**：
  ```bash
  certbot renew --dry-run --cert-name sub.originagent.cn
  ```

---

## 8. 代码提交与协同工作流规范

1. **版本库分支管理**：
   - 远程主分支：`main`
   - 上游仓库同步：通过 `git fetch upstream` 与 `git merge upstream/main` 进行功能同步。
2. **提交规范**：
   - 遵循规范的语义化 Commit 规范：`feat:`、`fix:`、`docs:`、`chore:`。
   - 严格审查提交变更，**严禁将包含真实密码、Token 或 API Key 的 `.env`、临时备份文件提交入库**。
3. **文件编码规范**：
   - 所有文本、Markdown 及配置文件统一使用 **UTF-8 无 BOM** 编码，行尾符保持 **LF**。
