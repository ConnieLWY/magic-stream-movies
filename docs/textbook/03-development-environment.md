# 第 03 章　开发环境搭建

> 本章目标：在一台全新的电脑上，安装运行本项目所需的全部工具，并验证它们能正常工作。

---

## 3.1 需要安装什么？

| 工具 | 本项目使用的版本 | 作用 | 验证命令 |
|---|---|---|---|
| Go | 1.27.1（见 `S/go.mod` 第 3 行） | 编译和运行后端 | `go version` |
| Node.js | 24.x（编写教材时为 v24.19.0） | 运行 Vite、npm | `node -v` |
| npm | 随 Node.js 安装 | 安装前端依赖 | `npm -v` |
| MongoDB | 任意现代版本，或 MongoDB Atlas 云服务 | 数据库 | `mongosh --version` |
| Git | 任意 | 版本控制 | `git --version` |
| 代码编辑器 | VS Code（推荐） | 写代码 | — |
| HTTP 客户端 | `curl`（系统自带）或 Postman/Bruno | 手动测试 API | `curl --version` |
| OpenAI API Key | — | 仅 AI 评级功能需要 | — |

---

## 3.2 安装 Go

### What
Go 是 Google 设计的一门**静态类型、编译型**语言。“编译型”意味着代码会先被编译成一个独立的二进制可执行文件，再运行。

### Why（为什么本项目选择 Go）
- **编译成单个二进制文件**：部署时只需复制一个文件，不需要在服务器上装运行时。
- **原生并发**：每个 HTTP 请求在独立的 goroutine（轻量级线程）中处理，性能好。
- **标准库强大**：`net/http`、`context`、`encoding/json` 都是内置的。
- **语法简单**：关键字只有 25 个，适合学习。

### How
1. 访问 <https://go.dev/dl/> 下载对应系统的安装包。
2. macOS 也可以用 Homebrew：`brew install go`。
3. 验证：

```bash
go version
# 期望输出类似：go version go1.27.1 darwin/arm64
```

4. 了解两个重要的环境变量：

```bash
go env GOPATH   # 第三方包下载缓存的位置，默认 ~/go
go env GOPROXY  # 下载包时使用的代理，默认 https://proxy.golang.org,direct
```

> **常见错误**：中国大陆网络环境下 `go get` 可能很慢或超时，可以设置 `go env -w GOPROXY=https://goproxy.cn,direct`。

### 推荐的 VS Code 扩展
- **Go**（官方，`golang.go`）：提供补全、跳转、`gopls` 语言服务器、保存时自动格式化。

---

## 3.3 安装 Node.js

### What
Node.js 让 JavaScript 能在浏览器之外运行。前端代码最终跑在浏览器里，但**开发工具**（Vite、TypeScript 编译器、ESLint）都是用 Node.js 运行的。

### How
- 推荐使用版本管理器（可以随时切换版本）：
  - macOS/Linux：`nvm`（`nvm install 24 && nvm use 24`）或 `fnm`。
  - Windows：`nvm-windows` 或 `fnm`。
- 验证：

```bash
node -v   # v24.x.x
npm -v
```

> **为什么要用版本管理器？** 不同项目可能需要不同版本的 Node。直接安装官方包的话，切换版本很麻烦。

---

## 3.4 准备 MongoDB

你有两种选择：

### 选项 A：本地安装（适合离线学习）
macOS：
```bash
brew tap mongodb/brew
brew install mongodb-community
brew services start mongodb-community
mongosh   # 进入交互式 shell，能看到 test> 提示符即成功
```
本地连接字符串通常是：`mongodb://localhost:27017`

### 选项 B：Docker（推荐，干净可删除）
```bash
docker run -d --name mongo -p 27017:27017 mongo:8
```

### 选项 C：MongoDB Atlas（云端免费集群）
1. 在 <https://www.mongodb.com/atlas> 注册并创建免费集群（M0）。
2. 创建数据库用户，设置网络访问白名单（学习时可临时允许你的 IP）。
3. 复制连接字符串，格式类似：
   `mongodb+srv://<用户名>:<密码>@<集群地址>/?retryWrites=true&w=majority`

> ⚠️ **安全提醒**：连接字符串里包含密码，属于**机密信息**。永远不要提交到 Git，不要贴在聊天记录或截图里。本仓库只提交 `.env.example` 模板，真正的 `.env` 被 `.gitignore` 排除（第 16 章）。

### 图形化工具（可选）
- **MongoDB Compass**：官方 GUI，可以浏览集合、导入 JSON、执行查询。强烈推荐初学者安装。

---

## 3.5 获取 OpenAI API Key（可选）

只有“管理员更新影评”功能需要调用 OpenAI。`OPENAI_API_KEY` 是启动时必填的配置（第 06 章 6.2.3），但如果你暂时不想付费，可以先填一个占位值（例如 `sk-placeholder`）：服务器能正常启动，其他所有功能都能使用，只有 `PATCH /movies/:imdb_id/review` 会因为 OpenAI 拒绝请求而返回 `500 {"error": "internal server error"}`（详细原因写在服务器日志里）。

1. 在 <https://platform.openai.com/> 注册并创建 API Key。
2. 设置用量上限，防止意外扣费。
3. 把 Key 写进本地 `.env`（**不要提交到 Git**）。

---

## 3.6 验证工具链（一键检查脚本）

在终端执行：

```bash
echo "== Go ==";      go version
echo "== Node ==";    node -v && npm -v
echo "== Git ==";     git --version
echo "== Mongo ==";   mongosh --eval 'db.runCommand({ ping: 1 })' --quiet
echo "== curl ==";    curl --version | head -1
```

如果 `mongosh` 那一行输出 `{ ok: 1 }`，说明 MongoDB 已经在运行。

---

## 3.7 运行现有项目（验证环境是否完整）

在开始重建之前，先让现有项目跑起来，确认环境没有问题。

### 后端
```bash
cd Server/MagicStreamMoviesServer
# 1. 准备 .env：复制模板后填写密钥（见第 04 章 4.6 节）
cp .env.example .env
# 2. 写入种子数据（类型、评级、示例电影）并创建索引，可重复执行
go run ./cmd/seed
# 3. 运行
go run .
```
期望看到类似日志：
```text
[GIN-debug] POST   /auth/register            --> ...AuthController.Register-fm (2 handlers)
[GIN-debug] GET    /movies                   --> ...MovieController.ListMovies-fm (1 handlers)
...
2026/10/30 16:00:00 listening on :8080
```
另开一个终端：
```bash
curl http://localhost:8080/health
# {"status":"ok"}
```

> 如果启动时立即退出并打印 `MONGODB_URI is required` 之类的信息，说明 `.env` 缺失或有配置没填。`config.Load()` 会**一次性列出所有**有问题的配置项。

### 前端
```bash
cd Client/magic-stream-client
npm install
npm run dev
# VITE vX.Y.Z  ready in ... ms
# ➜  Local:   http://localhost:5173/
```
浏览器打开 <http://localhost:5173/>。

> 如果首页显示 “There are currently no movies available”，说明数据库里还没有数据——回到后端目录运行 `go run ./cmd/seed`。

---

## 3.8 常见环境问题排查

| 现象 | 可能原因 | 解决方法 |
|---|---|---|
| `go: command not found` | PATH 未包含 Go | 重开终端；检查 `/usr/local/go/bin` 是否在 PATH |
| `go.mod requires go >= 1.27.1` | 本地 Go 版本太旧 | 升级 Go，或启用 toolchain 自动下载（Go 1.21+ 默认会尝试） |
| 后端启动后立刻退出：`MONGODB_URI is required` 等 | `.env` 缺失或不在当前目录 | 必须在 `S/` 目录下运行 `go run .`，因为代码用相对路径 `.env` 加载 |
| `SECRET_KEY must be at least 32 bytes` | 密钥太短 | 用 `openssl rand -base64 32` 生成 |
| `connect to mongodb: ...` | URI 格式错误 | 检查 `mongodb://` 或 `mongodb+srv://` 前缀 |
| `ping mongodb: ...` | MongoDB 未启动或网络不通 | 启动 MongoDB；Atlas 检查 IP 白名单 |
| 浏览器控制台出现 CORS 错误 | 前端地址不在 `ALLOWED_ORIGINS` 中 | 第 12 章 |
| `npm install` 很慢 | 网络问题 | 设置镜像 `npm config set registry https://registry.npmmirror.com` |

---

## 本章练习

> 答案见 [第 20 章](20-exercise-solutions.md#第-03-章)

**关键概念回顾**
- “编译型语言”和“解释型语言”的区别是什么？Go 属于哪一种？
- 为什么前端项目也需要安装 Node.js？

**练习 3.1** 完成 3.6 节的检查脚本，截图保存结果。

**练习 3.2** 用 Docker 启动一个 MongoDB，然后用 `mongosh` 连接并执行 `show dbs`。

**调试挑战 3.3** 在 `Server/` 目录（而不是 `Server/MagicStreamMoviesServer/`）下执行 `go run ./MagicStreamMoviesServer`。观察报错，并解释为什么。

**理解检验 3.4** 如果没有 OpenAI Key，哪些 API 能正常工作，哪些不能？请根据源码回答。
