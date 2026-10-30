# MagicStream Movies（Reelm）全栈开发教材

> 一本基于**真实代码库**编写的自学教材：从一个空文件夹开始，用 Go + Gin + MongoDB + React + TypeScript 亲手重建整个项目。

本教材的所有内容都以仓库中的实际代码为准。每一处设计都会讲清楚**做了什么、为什么这样做、不这样做会怎样**，并给出可以动手验证的命令或测试。

---

## 如何阅读本教材

| 你的目标 | 推荐阅读路径 |
|---|---|
| 快速了解项目 | 00 → 01 → 02 → 14 |
| 只学后端（Go） | 03 → 04 → 05 → 06 → 07 → 08 → 09 → 10 → 11 → 12 → 13 → 15 |
| 只学前端（React） | 03 → 06（前端部分）→ 13a → 14 |
| **完整重建项目（推荐）** | 按编号顺序 00 → 21 全部阅读，然后完成 18 的重建步骤和 19 的综合练习 |

每章末尾都有「本章练习」，答案统一放在 [20-exercise-solutions.md](20-exercise-solutions.md)，请先自己做再看答案。

### 路径缩写约定

为了让代码引用简洁，本教材使用以下缩写：

| 缩写 | 实际路径 |
|---|---|
| `S/` | `Server/MagicStreamMoviesServer/` |
| `C/` | `Client/magic-stream-client/` |

例如 `S/services/auth_service.go` 表示 `Server/MagicStreamMoviesServer/services/auth_service.go`。

> 代码引用以**文件名 + 函数名**为准（例如“`AuthService.Refresh`”），不使用行号——行号会随着代码修改而偏移，函数名不会。代码清单中的注释与源码一致；为了讲解而添加的中文标注（如 `// ①`）只出现在教材中。

---

## 章节目录

| 编号 | 文件 | 主题 | 依赖章节 |
|---|---|---|---|
| 00 | [00-learning-roadmap.md](00-learning-roadmap.md) | 学习路线图与前置知识 | — |
| 01 | [01-project-overview.md](01-project-overview.md) | 项目概览：它做什么、不做什么 | 00 |
| 02 | [02-system-architecture.md](02-system-architecture.md) | 系统架构与模块依赖（含 Mermaid 图） | 01 |
| 03 | [03-development-environment.md](03-development-environment.md) | 开发环境搭建（Go、Node、MongoDB、编辑器） | 00 |
| 04 | [04-project-initialization.md](04-project-initialization.md) | 项目初始化：`go mod init` 与 `npm create vite` | 03 |
| 05 | [05-folder-structure.md](05-folder-structure.md) | 文件夹结构与 Go package 设计 | 04 |
| 06 | [06-dependencies.md](06-dependencies.md) | 依赖逐个讲解：Gin、mongo-driver、jwt、bcrypt、langchaingo、axios…… | 04 |
| 07 | [07-database-design.md](07-database-design.md) | MongoDB 数据库设计与连接 | 05, 06 |
| 08 | [08-models-and-types.md](08-models-and-types.md) | Model、struct tag、TypeScript 类型 | 07 |
| 09 | [09-routes-and-controllers.md](09-routes-and-controllers.md) | 路由与 Controller（Handler） | 08 |
| 10 | [10-services-and-business-logic.md](10-services-and-business-logic.md) | 业务逻辑（services 层）：推荐算法、AI 影评评级 | 09 |
| 11 | [11-authentication.md](11-authentication.md) | 认证：bcrypt、JWT、Refresh Token 轮换、Cookie | 09 |
| 12 | [12-middleware.md](12-middleware.md) | Middleware：CORS、认证、角色、限流、安全响应头 | 11 |
| 13 | [13-validation-and-errors.md](13-validation-and-errors.md) | 校验与错误处理 | 09 |
| 13a | [13a-frontend-react.md](13a-frontend-react.md) | 前端：React、Router、Context、axios 拦截器 | 11 |
| 14 | [14-end-to-end-request-flow.md](14-end-to-end-request-flow.md) | 端到端请求流程追踪 | 12, 13a |
| 15 | [15-testing-and-debugging.md](15-testing-and-debugging.md) | 测试与调试：单元测试、内存 fake、HTTP 测试、集成测试 | 14 |
| 16 | [16-security.md](16-security.md) | 安全：威胁模型与每一项防御措施 | 11, 12 |
| 17 | [17-deployment.md](17-deployment.md) | 构建与部署：优雅关闭、Docker、部署拓扑、CI | 16 |
| 18 | [18-rebuild-from-scratch.md](18-rebuild-from-scratch.md) | 从零重建：完整的分步清单 | 全部 |
| 19 | [19-exercises.md](19-exercises.md) | 综合练习与毕业项目 | 全部 |
| 20 | [20-exercise-solutions.md](20-exercise-solutions.md) | 所有练习的参考答案 | — |
| 21 | [21-known-issues-and-improvements.md](21-known-issues-and-improvements.md) | 设计决策、有意接受的权衡与进阶扩展 | 16 |

---

## 源文件 → 章节映射表

这张表告诉你：每个源文件在哪一章被重点讲解。重建项目时，可以按此表逐文件对照。

### 后端（Go）

| 源文件 | 作用 | 主要章节 |
|---|---|---|
| `S/main.go` | 程序入口：加载配置、连接数据库、组装各层、注册全局中间件、启动与优雅关闭 | 02, 12, 17 |
| `S/go.mod` / `S/go.sum` | Go module 定义与依赖锁定 | 04, 06 |
| `S/.env.example` | 配置模板（真正的 `.env` 不提交） | 04, 16 |
| `S/config/config.go` | 读取并校验所有环境变量 | 04, 10, 16 |
| `S/database/database_connection.go` | `Connect`、`EnsureIndexes` | 07 |
| `S/models/movie_model.go` | `Genre`、`Ranking`、`Movie` 及请求/响应 DTO | 08 |
| `S/models/user_model.go` | `User`、`UserRegister`、`UserLogin`、`UserResponse`、角色常量 | 08 |
| `S/repository/*.go` | 数据访问：用户、电影、类型与评级；错误翻译 | 07, 10, 11 |
| `S/services/auth_service.go` | 注册、登录、刷新（轮换）、登出 | 11 |
| `S/services/movie_service.go` | 电影列表、添加电影、推荐、影评评级 | 10 |
| `S/services/openai_classifier.go` | `ReviewClassifier` 的 OpenAI 实现 | 06, 10 |
| `S/services/errors.go` | 业务错误 | 09, 13 |
| `S/controllers/response.go` | `bindJSON`、`respondError`、错误映射表 | 09, 13 |
| `S/controllers/user_controller.go` | `AuthController`：认证接口与 Cookie | 09, 11 |
| `S/controllers/movie_controller.go` | `MovieController`：电影、类型、推荐、影评 | 09, 10 |
| `S/middleware/auth.go` | `Auth`、`RequireRole`、`UserID`、`Role` | 11, 12 |
| `S/middleware/security.go` | `SecurityHeaders`、`BodyLimit`、`RateLimit` | 12, 16 |
| `S/routes/routes.go` | 路由表与路由组 | 09 |
| `S/tokens/tokens.go` | JWT 签发与校验、Token 哈希 | 11 |
| `S/validation/validation.go` | 共享 validator、自定义规则、错误描述 | 13 |
| `S/cmd/seed/` | 种子数据命令（数据嵌入二进制）、提升管理员 | 07 |
| `S/internal/memstore/memstore.go` | 测试用的内存假实现 | 15 |
| `S/**/*_test.go` | 单元测试、HTTP 测试、集成测试 | 15 |
| `S/Dockerfile`、`S/.dockerignore` | 容器镜像 | 17 |
| `docker-compose.yml`、`.github/workflows/ci.yml` | 本地编排、持续集成 | 17 |

### 前端（React + TypeScript）

| 源文件 | 作用 | 主要章节 |
|---|---|---|
| `C/package.json` | npm 脚本与依赖 | 04, 06 |
| `C/vite.config.ts`、`C/tsconfig*.json`、`C/eslint.config.js` | 构建、类型检查、Lint 配置 | 04, 06 |
| `C/.env.example` | `VITE_API_BASE_URL` 模板 | 04, 13a |
| `C/index.html` | HTML 入口、字体、`data-bs-theme="dark"` | 13a |
| `C/src/main.tsx` | React 根节点：`AuthProvider` + `BrowserRouter` | 13a |
| `C/src/App.tsx` | 页面路由表、登出逻辑 | 13a, 14 |
| `C/src/api/axiosConfig.ts` | 公共 axios 实例、`refreshSession()`（并发去重） | 11, 13a |
| `C/src/hook/useAxiosPrivate.tsx` | 带 Bearer Token 和自动刷新的 axios 实例 | 11, 13a |
| `C/src/hook/useAuth.tsx` | 读取 `AuthContext` 的自定义 Hook | 13a |
| `C/src/context/AuthContext.ts`、`AuthProvider.tsx` | 全局认证状态、页面加载时恢复会话 | 11, 13a |
| `C/src/components/RequiredAuth.tsx` | 前端路由守卫 | 13a |
| `C/src/types/movie.ts`、`user.ts` | 与后端 JSON 对应的 TypeScript 类型 | 08 |
| `C/src/components/home/Home.tsx` | 首页：电影列表、类型筛选、Hero 区 | 13a |
| `C/src/components/movies/Movies.tsx`、`movie/Movie.tsx` | 电影网格与卡片 | 13a |
| `C/src/components/login/Login.tsx`、`register/Register.tsx` | 登录、注册表单 | 13a, 14 |
| `C/src/components/recommended/Recommended.tsx` | 个性化推荐页 | 10, 13a |
| `C/src/components/review/Review.tsx` | 电影详情 + 管理员影评（触发 AI 评级） | 10, 13a |
| `C/src/components/stream/StreamMovie.tsx` | YouTube 预告片播放（`<iframe>`，校验视频 ID） | 06, 13a |
| `C/src/components/header/`、`footer/`、`auth/`、`brand/` | 布局、品牌、图标 | 13a |
| `C/src/index.css`、`C/src/brand.ts` | 设计 Token（CSS 变量）与品牌名称 | 13a |

---

## 已实现 vs 未实现（一览）

| 功能 | 状态 |
|---|---|
| 用户注册（注册后直接登录）/ 登录 / 登出 | ✅ 已实现 |
| JWT Access Token（15 分钟）+ Refresh Token（7 天，HttpOnly Cookie，每次刷新轮换，数据库只存哈希） | ✅ 已实现 |
| 电影列表、单部电影详情、类型列表 | ✅ 已实现 |
| 基于用户喜爱类型的推荐 | ✅ 已实现 |
| 管理员写影评 → OpenAI 分类（白名单校验）→ 更新电影评级 | ✅ 已实现 |
| 添加电影 `POST /movies`（仅管理员） | ✅ 后端已实现，前端没有页面（用 API 工具调用） |
| YouTube 预告片播放 | ✅ 已实现（只是嵌入 YouTube，**不是**真正的视频流媒体服务） |
| 登录/注册限流、安全响应头、请求体上限、优雅关闭 | ✅ 已实现 |
| 后端自动化测试（单元、HTTP、集成） | ✅ 已实现 |
| Dockerfile / docker-compose / CI | ✅ 已实现 |
| 数据库种子数据与索引 | ✅ 已实现（`go run ./cmd/seed`；索引在启动时自动创建） |
| 前端自动化测试 | ❌ 未实现（第 15 章 15.7） |
| 修改个人资料、修改喜爱类型、分页、搜索、多设备会话 | ❌ 未实现（第 21 章的扩展建议） |

---

## 核对记录

- [x] 所有章节中的代码清单与当前源码一致（以文件名 + 函数名引用，不依赖行号）
- [x] 后端 `go build ./...`、`go vet ./...`、`go test -race ./...` 通过（含 MongoDB 集成测试）
- [x] 前端 `npm run lint`、`npm run build` 通过
- [x] 文档中没有出现任何真实密钥、URI 或 Token
- [x] 每章练习的参考答案已与当前代码同步
