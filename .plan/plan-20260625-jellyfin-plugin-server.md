# Jellyfin Plugin Server — 开发计划

**创建日期**：2026-06-25  
**最终目标版本**：`v1.0.0`  
**技术栈**：Go · fasthttp · SQLite (modernc) · embed UI

---

## Phase 0 — 项目骨架 `v0.0.1`

- [x] 确认技术方案与需求（requirement.md）
- [x] 初始化 Go Module（`go mod init`）
- [x] 搭建目录结构（`cmd/` `internal/` `web/` `data/`）
- [x] 配置 `.gitignore`、`Makefile`（build / run / lint / test）
- [x] 引入核心依赖：`fasthttp`、`modernc sqlite3`、`golang.org/x/sync`
- [x] 编写基础配置结构体（`internal/config/config.go`）
- [x] SQLite 初始化 + schema migration（`internal/db/`）
- [x] 基础日志模块（结构化 JSON / 文本可切换）
- [x] `/health` 端点上线，验证 fasthttp 启动正常

---

## Phase 1 — 基础代理与 Manifest 拉取 `v0.1.0`

### 1.1 上游 HTTP 客户端
- [x] 封装 upstream client（支持超时、重试、代理透传）
- [x] 支持 HTTP / HTTPS / SOCKS5 代理配置
- [x] ETag / Last-Modified 条件请求逻辑

### 1.2 Manifest 拉取与解析
- [x] 定义 Jellyfin manifest.json 数据结构（`internal/manifest/types.go`）
- [x] 拉取并解析上游 manifest（`internal/manifest/fetcher.go`）
- [x] 将 manifest 元数据写入 SQLite（repos / plugins / plugin_versions）
- [x] manifest TTL 检查（过期则异步刷新）

### 1.3 代理路由
- [x] `GET /plugins/manifest/{repo-id}` — 返回本地化 manifest
  - [x] sourceUrl 替换为本地地址
  - [x] 未下载完成的包保留原始 URL（降级）
- [x] 缓存命中直接返回，miss 则触发拉取

### 1.4 默认仓库内置
- [x] 内置 Jellyfin 官方仓库配置
- [x] 内置 4 个高活跃社区插件仓库（Intro Skipper、Open Subtitles 等）

---

## Phase 2 — 插件包下载与本地服务 `v0.2.0`

### 2.1 包下载引擎
- [x] 异步下载队列（`internal/downloader/`）
- [x] singleflight 合并并发下载（同一包只下载一次）
- [x] 最大并发下载数限制（可配置，默认 4）
- [x] 下载进度写入 SQLite（`download_status`）

### 2.2 包校验
- [x] 下载完成后校验 MD5 checksum
- [x] 校验失败：删除文件，标记错误，记录日志

### 2.3 本地包服务
- [x] `GET /plugins/packages/{checksum}/{filename}` — 提供本地文件下载
- [x] 支持 Range 请求（断点续传友好，fasthttp.ServeFile 原生支持）
- [x] 文件不存在时透传上游原始 URL（302 重定向降级）

### 2.4 存储管理基础
- [x] 本地存储路径可配置（`data/packages/`）
- [x] 磁盘使用量统计接口

---

## Phase 3 — 管理 API `v0.3.0`

### 3.1 仓库管理 API
- [x] `GET  /api/repos` — 列表
- [x] `POST /api/repos` — 新增上游仓库
- [x] `PUT  /api/repos/:id` — 修改
- [x] `DEL  /api/repos/:id` — 删除
- [x] `POST /api/repos/:id/refresh` — 手动刷新
- [x] `POST /api/repos/refresh-all` — 刷新全部
- [x] `POST /api/repos/:id/test` — 连通性测试

### 3.2 包管理 API
- [x] `GET  /api/packages` — 已缓存包列表（支持搜索）
- [x] `DEL  /api/packages/:checksum` — 删除指定缓存
- [ ] `POST /api/packages/cleanup` — 触发 LRU 清理（Phase 5）

### 3.3 配置 API
- [x] `GET  /api/config` — 读取当前配置
- [x] `PUT  /api/config` — 更新配置（TTL / 并发数 / 存储路径 / 代理）

### 3.4 日志 API
- [x] `GET /api/logs` — 分页查询操作日志

### 3.5 状态 API
- [x] `GET /api/status` — 磁盘使用、DB 状态、版本

### 3.6 管理 UI 基础认证
- [ ] 可选 HTTP Basic Auth（用户名 + 密码，空则不启用）

---

## Phase 4 — 管理 Web UI `v0.4.0`

### 4.1 UI 工程搭建
- [x] 选定前端方案（React 18 + Vite + shadcn/ui + Tailwind v3）
- [x] `go:embed` 配置，将 `web/dist` 打包进二进制（`internal/handler/embed.go`）
- [x] 路由：`/` 及深路由返回 SPA，`/api/*` `/plugins/*` 走 Go handler

### 4.2 仪表盘页
- [x] 磁盘使用 / 已索引版本数 / 仓库数 / 服务状态 四卡片
- [x] 下载状态分布（pending / downloading / done / failed）

### 4.3 仓库管理页
- [x] 上游仓库列表（名称、URL、优先级、状态、最后刷新时间）
- [x] 新增 / 编辑 / 删除仓库（Dialog 表单）
- [x] 手动刷新按钮 + 刷新全部 + 连通性测试

### 4.4 包列表页
- [x] 已缓存插件列表（搜索 / 状态标签）
- [x] 单个包删除（仅已完成状态可删）

### 4.5 设置页
- [x] 代理配置（类型 / 地址 / 用户名密码 / NO_PROXY）
- [x] 存储配置（路径 / 磁盘上限 / 保留版本数）
- [x] 缓存配置（TTL / 并发下载数）
- [x] 管理认证配置（Basic Auth 开关 + 用户名密码）

### 4.6 操作日志页
- [x] 日志列表（level 彩色 badge / 消息 / 详情 / 时间）

---

## Phase 5 — 存储管理与清理 `v0.5.0`

- [x] LRU 版本清理（超出保留数量时删除最旧版本）
- [x] 磁盘空间上限检查（超出时拒绝新下载，告警日志）
- [x] 定时自动清理任务（可配置，默认每天凌晨 3 点）
- [x] 孤儿文件检测（文件存在但 DB 无记录，可一键清理）
- [x] 清理预览接口（返回将被删除的包列表，不真正删除）

---

## Phase 6 — 可观测性与稳定性 `v0.6.0`

- [x] 请求级别结构化日志（method / path / status / latency）
- [x] 缓存命中率统计持久化（每小时汇总写入 DB）
- [x] 上游健康检测（定时 ping，失败记录并告警日志）
- [x] Graceful shutdown（SIGTERM 等待进行中下载完成）
- [x] `/health` 增强：区分 healthy / degraded / unhealthy
- [x] Panic recovery 中间件（fasthttp）

---

## Phase 7 — 发布与文档 `v1.0.0`

- [x] Dockerfile（多阶段构建，最终镜像 < 30MB）
- [x] docker-compose.yml 示例
- [x] README.md（安装、配置、Jellyfin 对接说明）
- [x] 环境变量配置支持（`JPSERVER_PORT`、`JPSERVER_DATA_DIR` 等）
- [x] GitHub Actions CI（build / test / docker push）
- [x] 版本信息注入（`ldflags -X`，`/api/status` 返回 git commit）
- [ ] 打 `v1.0.0` tag，发布 GitHub Release（需 git remote）

---

## 版本语义说明

```
v0.0.x  — 骨架/脚手架，无功能
v0.1.x  — 基础代理可用（manifest 拉取 + 本地化）
v0.2.x  — 包下载与本地服务
v0.3.x  — 管理 API 完整
v0.4.x  — Web UI 可用
v0.5.x  — 存储管理稳定
v0.6.x  — 可观测性与稳定性
v1.0.0  — 正式发布，生产可用
```

---

## 目录结构规划

```
jellyfin-plugin-server/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/          # 配置加载与结构体
│   ├── db/              # SQLite schema + 迁移 + 查询
│   ├── manifest/        # manifest 解析 + 拉取 + 本地化
│   ├── downloader/      # 包下载引擎 + singleflight
│   ├── proxy/           # upstream HTTP 客户端
│   ├── handler/         # fasthttp handler（路由）
│   ├── api/             # 管理 API handler
│   ├── storage/         # 本地文件服务 + 清理
│   └── logger/          # 结构化日志
├── web/                 # 嵌入式前端 UI
│   ├── index.html
│   ├── app.js
│   └── style.css
├── data/                # 运行时数据（gitignore）
│   ├── packages/
│   └── jellyfin.db
├── requirement.md
├── Makefile
├── Dockerfile
├── docker-compose.yml
└── go.mod
```
