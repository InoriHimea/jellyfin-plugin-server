# Jellyfin Plugin Server — 需求说明

## 项目概述

Jellyfin Plugin Server 是一个自托管的 Jellyfin 插件仓库代理与镜像服务器。
它作为 Jellyfin 客户端与上游插件仓库之间的中间层，将插件包本地缓存，提升访问速度并支持离线使用。

**技术栈**：Go · fasthttp · SQLite · 嵌入式前端 UI (embed)

**目标版本**：`v1.0.0`（正式发布）

---

## 核心功能需求

### F1 — 请求代理与本地缓存

- Jellyfin 客户端发来的插件仓库请求（manifest / 包文件）优先命中本地缓存
- 本地无缓存时，从已配置的上游仓库拉取，拉取后存入本地，再返回给客户端
- manifest.json 支持可配置 TTL（默认 24 小时），过期后下次请求触发后台刷新
- 支持手动强制刷新指定仓库或全部仓库
- 基于 ETag / Last-Modified 做上游条件请求，避免不必要的全量拉取

### F2 — Manifest 解析与插件包镜像

- 解析 Jellyfin 标准 manifest.json 格式（catalog，包含 versions 数组）
- 提取每个版本的 `sourceUrl` 和 `checksum`（MD5）
- 异步下载所有插件包到本地存储目录
- 下载完成后校验 checksum，不通过则删除并记录错误
- 对外提供的 manifest.json 中，将 `sourceUrl` 替换为本地服务地址
- 未下载完成的包，manifest 中保留原始上游链接作为降级（stale fallback）

### F3 — 并发控制与降级

- 同一包的并发下载请求使用 singleflight 合并，只下载一次
- 配置最大并发下载数（默认 4）
- 上游不可达时：
  - 已缓存 → 直接返回本地缓存（即使 TTL 已过期）
  - 未缓存 → 返回 503，附带错误信息

### F4 — 存储管理

- 本地存储路径可配置（默认 `./data/packages/`）
- 支持磁盘空间上限配置
- 旧版本清理策略：保留最新 N 个版本（可配置，默认 3），其余 LRU 淘汰
- 提供手动清理接口

### F5 — 上游仓库管理

- 支持配置多个上游仓库（名称 + URL + 启用状态 + 优先级）
- 内置默认仓库列表：
  - Jellyfin 官方仓库：`https://repo.jellyfin.org/releases/plugin/manifest.json`
  - Jellyseerr 插件
  - Intro Skipper 插件
  - Open Subtitles 插件
  - 其他高活跃度社区插件仓库
- 支持增删改上游仓库，立即生效
- 上游仓库可用性检测（连通性测试接口）

### F6 — 代理配置

- 支持配置 HTTP / HTTPS / SOCKS5 代理，用于访问上游
- 代理仅在上游请求时生效，本地服务不走代理
- 支持 NO_PROXY 配置（逗号分隔域名列表）

### F7 — 管理 Web UI

- 嵌入式单页应用（`embed` 打包进二进制，零外部依赖）
- 页面列表：
  - **仪表盘**：缓存命中率、已缓存包数、磁盘使用量、活跃上游数
  - **仓库管理**：增删改上游仓库，手动刷新，可用性测试
  - **包列表**：已缓存包浏览，支持搜索/过滤
  - **代理设置**：配置 HTTP/SOCKS5 代理
  - **系统设置**：存储路径、TTL、并发数、清理策略
  - **操作日志**：下载记录、错误日志

### F8 — 可观测性与运维

- `GET /health` — 健康检查端点（返回 200 / 503）
- `GET /api/status` — 详细状态（版本、缓存统计、上游状态）
- 结构化日志输出（JSON 格式可选）
- 管理 UI 可配置基础认证（用户名 + 密码）

---

## 非功能需求

| 指标 | 目标 |
|------|------|
| 响应延迟（缓存命中） | < 10ms |
| 并发请求 | ≥ 100 |
| 二进制大小 | < 20MB（含前端） |
| 依赖外部服务 | 无（SQLite 内嵌） |
| 部署方式 | 单二进制 / Docker |

---

## API 设计概览

```
# 插件仓库代理（Jellyfin 客户端直接对接）
GET  /plugins/manifest/{repo-id}         # 返回本地化后的 manifest
GET  /plugins/packages/{checksum}/{file} # 返回本地包文件

# 管理 API
GET  /api/status                         # 系统状态
GET  /api/repos                          # 获取上游仓库列表
POST /api/repos                          # 添加上游仓库
PUT  /api/repos/:id                      # 修改上游仓库
DEL  /api/repos/:id                      # 删除上游仓库
POST /api/repos/:id/refresh              # 手动刷新指定仓库
POST /api/repos/refresh-all              # 刷新所有仓库
POST /api/repos/:id/test                 # 测试上游连通性

GET  /api/packages                       # 已缓存包列表
DEL  /api/packages/:checksum             # 删除指定缓存包
POST /api/packages/cleanup               # 触发清理

GET  /api/config                         # 获取配置
PUT  /api/config                         # 更新配置

GET  /api/logs                           # 操作日志

GET  /health                             # 健康检查
```

---

## 数据模型（SQLite）

```sql
-- 上游仓库
repos (id, name, url, enabled, priority, last_fetched_at, etag, created_at)

-- 插件元数据（来自 manifest）
plugins (id, repo_id, name, description, overview, owner, category)

-- 插件版本
plugin_versions (id, plugin_id, version, changelog, target_abi, source_url,
                 checksum, timestamp, local_path, download_status, downloaded_at)

-- 操作日志
logs (id, level, message, detail, created_at)

-- 系统配置（KV）
config (key, value, updated_at)
```

---

## 版本计划

| 版本 | 里程碑 |
|------|--------|
| `v0.1.0` | 基础代理 + manifest 拉取与本地化 |
| `v0.2.0` | 包下载、校验、本地服务 |
| `v0.3.0` | 上游管理 + 代理设置 API |
| `v0.4.0` | 管理 Web UI（仪表盘 + 仓库管理） |
| `v0.5.0` | 存储管理 + 清理策略 |
| `v0.6.0` | 可观测性（健康检查、日志、监控） |
| `v1.0.0` | Docker 镜像 + 文档 + 正式发布 |
