# 第 04 章　项目初始化

> 本章目标：从一个空文件夹开始，创建后端 Go module 和前端 Vite 项目，配置环境变量，并让两者都能启动。

---

## 4.1 创建仓库根目录

```bash
mkdir magic-stream-movies
cd magic-stream-movies
git init
mkdir -p Server/MagicStreamMoviesServer
mkdir -p Client
```

本项目采用 **Monorepo**（单仓库）结构：前端和后端放在同一个 Git 仓库的不同子目录里。

| 方案 | 优点 | 缺点 |
|---|---|---|
| Monorepo（本项目） | 一次提交可以同时修改前后端；方便学习 | 仓库变大；CI 需要区分子目录 |
| 多仓库（Polyrepo） | 权限、发布独立 | 跨仓库改动需要协调 |

---

## 4.2 初始化 Go module

### 4.2.1 什么是 Go module？

**What**：module（模块）是 Go 的依赖管理单位。一个 module 由一个 `go.mod` 文件定义，它声明了：
1. **module path**：这个模块的“名字”，也是导入其内部包时使用的前缀。
2. **Go 版本**。
3. **依赖列表**及其版本。

**Why**：在 module 出现之前（Go 1.11 以前），所有代码必须放在 `GOPATH` 目录下，没有版本管理，非常痛苦。

### 4.2.2 执行初始化

```bash
cd Server/MagicStreamMoviesServer
go mod init github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer
```

这会生成：

```go
// S/go.mod（初始状态）
module github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer

go 1.27.1
```

### 4.2.3 module path 的含义

项目里的每个 import 都以这个路径开头：

```go
// S/main.go（节选）
"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/config"
"github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/database"
```

拆解：`<module path>/<子目录名>`。`database` 子目录里的代码就是 `.../database` 包。

> **为什么 module path 要和仓库地址一致？** 在本地开发时，Go 根据 `go.mod` 所在目录解析 module 内部的包，路径写成什么都能编译。但如果别人想用 `go get github.com/ConnieLWY/magic-stream-movies/...` 下载你的代码，Go 会去这个地址取代码，并检查取到的 `go.mod` 里声明的 module path 是否与之相同——不一致就会失败。所以本项目的 module path 与 GitHub 仓库 `ConnieLWY/magic-stream-movies` 一一对应。对于确定不会被别人导入的应用程序，也可以使用简短的名字如 `magicstream`。

### 4.2.4 写第一个 `main.go`

```go
package main

import "github.com/gin-gonic/gin"

func main() {
	router := gin.Default()

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	router.Run(":8080")
}
```

逐行解释：
- `package main`：Go 规定，可执行程序的入口必须在名为 `main` 的包里，并且有一个 `func main()`。
- `import "github.com/gin-gonic/gin"`：导入 Gin 框架。
- `gin.Default()`：创建一个 `*gin.Engine`，并**默认挂载 Logger 和 Recovery 两个 Middleware**（Recovery 能在 Handler panic 时返回 500 而不是让整个程序崩溃）。
- `router.GET(path, handler)`：注册一个处理 `GET /health` 的 Handler。这种“健康检查”接口通常给负载均衡器和容器编排系统用来判断服务是否存活。
- `func(c *gin.Context)`：匿名函数。`*gin.Context` 是一个**指针**，指向本次请求的上下文对象。
- `c.JSON(200, gin.H{...})`：返回状态码 200 和 JSON。`gin.H` 是 `map[string]any` 的简写。
- `router.Run(":8080")`：监听 8080 端口，阻塞运行。这是最简单的写法；完整项目改用 `http.Server`，以便设置超时和优雅关闭（第 17 章）。

### 4.2.5 下载依赖

```bash
go get github.com/gin-gonic/gin
go run .
```

`go get` 会做三件事：
1. 下载包到模块缓存（`$GOPATH/pkg/mod`）。
2. 在 `go.mod` 的 `require` 中添加依赖。
3. 在 `go.sum` 中记录每个依赖的**哈希值**（防止依赖被篡改）。

### 4.2.6 直接依赖与间接依赖

```go
// S/go.mod（节选）
require (
	github.com/gin-contrib/cors v1.7.9
	github.com/gin-gonic/gin v1.12.0
	github.com/go-playground/validator/v10 v10.30.5
	...
)

require (
	github.com/bytedance/sonic v1.15.5 // indirect
	...
)
```

`go.mod` 中有两个 `require` 块：
- **第一个块**：项目代码直接 `import` 的包（直接依赖）。
- **第二个块**：带 `// indirect` 标记，是直接依赖自己需要的包（间接依赖）。

这两个块由 `go mod tidy` 自动维护：

```bash
go mod tidy
```

它会：删除未使用的依赖；根据代码中的实际 `import` 区分直接和间接依赖；补全缺失的 `go.sum` 条目。只用 `go get` 逐个添加依赖而从不运行 `tidy`，最后会得到一个所有依赖都标着 `// indirect` 的 `go.mod`，读者无法从中看出项目真正用了什么。**养成每次增删 import 后都运行 `go mod tidy` 的习惯**（CI 中也可以检查 `go mod tidy` 之后文件是否有变化）。

---

## 4.3 初始化前端项目

### 4.3.1 用 Vite 创建 React + TypeScript 项目

```bash
cd ../../Client
npm create vite@latest magic-stream-client -- --template react-ts
cd magic-stream-client
npm install
npm run dev
```

**What is Vite?** Vite 是一个前端构建工具。开发时，它启动一个开发服务器，利用浏览器原生 ES Module 按需加载文件，并支持 HMR（Hot Module Replacement，热更新：改代码后页面局部刷新，不丢失状态）。打包时，它把所有代码压缩、合并成静态文件。

**为什么不用 Create React App？** CRA 已经被官方弃用，启动和构建都很慢。

### 4.3.2 生成的关键文件

| 文件 | 作用 |
|---|---|
| `package.json` | 项目元数据、npm 脚本、依赖列表 |
| `index.html` | 浏览器加载的第一个文件，里面有 `<div id="root">` 和对 `src/main.tsx` 的引用 |
| `vite.config.ts` | Vite 配置，本项目只启用了 React 插件 |
| `tsconfig.json` | TypeScript 配置的“入口”，引用了 `tsconfig.app.json`（应用代码）和 `tsconfig.node.json`（Vite 配置文件本身） |
| `eslint.config.js` | 代码规范检查 |
| `src/main.tsx` | React 应用的入口 |

### 4.3.3 npm 脚本解读

```json
// C/package.json:6-11
"scripts": {
  "dev": "vite",                 // 启动开发服务器（默认端口 5173）
  "build": "tsc -b && vite build", // 先类型检查，再打包到 dist/
  "lint": "eslint .",            // 检查代码规范
  "preview": "vite preview"      // 本地预览打包后的产物
}
```

注意 `build` 脚本中的 `tsc -b`：Vite 本身打包时**不做类型检查**（为了速度），所以要先用 TypeScript 编译器检查一遍类型。如果有类型错误，`&&` 后面的 `vite build` 就不会执行。

### 4.3.4 修改项目名

现有 `package.json` 中的 `"name": "reelm-client"` 与品牌名一致。这个字段对 `private: true` 的项目来说只是一个标识，不会影响运行。

### 4.3.5 安装本项目需要的运行时依赖

```bash
npm install axios bootstrap react-bootstrap react-router-dom
```

每个依赖的作用在第 06 章详细讲解。

---

## 4.4 环境变量：What / Why / How

### What
环境变量是操作系统提供给进程的键值对配置，例如 `PATH`、`HOME`。程序可以通过 `os.Getenv("KEY")`（Go）或 `import.meta.env.KEY`（Vite）读取。

### Why
**配置与代码分离**（这是 “Twelve-Factor App” 方法论的第三条原则）：
- 同一份代码，在开发、测试、生产环境连接不同的数据库。
- 密钥不写进代码，降低泄露风险。

### `.env` 文件是什么？
在本地开发时，每次都 `export` 一堆变量很麻烦。`.env` 是一个约定俗成的文件，由 `godotenv` 这样的库在程序启动时读取，并“模拟”成环境变量。

**`.env` 只应用于本地开发。** 在生产环境中，应由部署平台（Docker、Kubernetes、云服务的 Secret 管理）注入真实的环境变量。

---

## 4.5 后端环境变量一览

所有环境变量都在 `config.Load()`（`S/config/config.go`）中**一次性**读取和校验。其他包从不调用 `os.Getenv`，只接收 `*config.Config`。

| 变量名 | 示例值（虚构） | 必填 | 校验规则 / 默认值 |
|---|---|---|---|
| `MONGODB_URI` | `mongodb://localhost:27017` | 是 | 非空 |
| `DATABASE_NAME` | `magic-stream-movies` | 是 | 非空 |
| `SECRET_KEY` | 32 字节以上随机串 | 是 | 至少 32 字节 |
| `SECRET_REFRESH_KEY` | 另一段随机串 | 是 | 至少 32 字节，且不能与 `SECRET_KEY` 相同 |
| `OPENAI_API_KEY` | `sk-...`（请使用你自己的） | 是 | 非空 |
| `BASE_PROMPT_TEMPLATE` | 见下文 | 是 | 必须包含 `{rankings}` |
| `OPENAI_MODEL` | `gpt-4.1-mini` | 否 | 默认 `gpt-4.1-mini` |
| `ALLOWED_ORIGINS` | `http://localhost:5173` | 否 | 逗号分隔；默认 `http://localhost:5173` |
| `PORT` | `8080` | 否 | 默认 `8080` |
| `RECOMMENDED_MOVIE_LIMIT` | `5` | 否 | 正整数；默认 5 |
| `REQUEST_TIMEOUT` / `OPENAI_TIMEOUT` | `10s` / `30s` | 否 | Go 的 duration 格式 |
| `COOKIE_SECURE` / `COOKIE_SAMESITE` | `true` / `Lax` | 否 | `SameSite=None` 时必须 `Secure=true` |
| `TRUSTED_PROXIES` | `10.0.0.0/8` | 否 | 默认不信任任何代理 |

任何一项不合法，`config.Load()` 都会返回一个汇总了**所有**问题的错误，`main` 打印后退出——宁可启动失败，也不要带着空密钥或错误配置运行。

`BASE_PROMPT_TEMPLATE` 在本项目中的值（这不是机密，是业务配置）：

```text
Return a response using one of these words: {rankings}. The response should be a single word and should not contain any other text. The response should be based on the following review:
```

`{rankings}` 会在运行时被替换成数据库里的评级名称列表（第 10 章详解）。

---

## 4.6 创建 `.env` 与 `.env.example`

1. 仓库中提交了 `S/.env.example`，它只包含变量名、说明和无害的示例值：

```dotenv
# S/.env.example（节选）
MONGODB_URI=mongodb://localhost:27017
DATABASE_NAME=magic-stream-movies
# 两个不同的随机密钥，各至少 32 字节。生成方法：openssl rand -base64 32
SECRET_KEY=
SECRET_REFRESH_KEY=
OPENAI_API_KEY=
BASE_PROMPT_TEMPLATE="Return a response using one of these words: {rankings}. ..."
# PORT=8080
# ALLOWED_ORIGINS=http://localhost:5173
```

2. 复制一份为 `.env` 并填入真实值：`cp .env.example .env`
3. 仓库根目录的 `.gitignore` **确保 `.env` 不被提交**，同时放行模板：

```gitignore
# Environment files (commit the .env.example templates instead)
.env
.env.*
!.env.example
```

4. 生成强随机密钥（两个密钥分别生成）：

```bash
openssl rand -base64 32
```

> ⚠️ **如果 `.env` 曾经被提交过**：即使之后删除，它仍然存在于 Git 历史中。只要仓库推送过，就必须**立即轮换**（作废并重新生成）其中的所有密钥。详细步骤见第 16 章 16.2 节。

### 前端环境变量

```dotenv
# C/.env.example —— 复制为 C/.env
VITE_API_BASE_URL=http://localhost:8080
```

- Vite 规定：只有以 `VITE_` 开头的变量才会暴露给浏览器端代码。这是一种保护措施，防止你不小心把服务器密钥打包进前端。
- 读取方式：`import.meta.env.VITE_API_BASE_URL`（`C/src/api/axiosConfig.ts:4`）。
- **重要认知**：前端的任何“环境变量”最终都会以明文形式出现在打包后的 JS 文件里，用户可以在浏览器中看到。**前端永远不能存放密钥。** API 地址不是密钥，但为了统一规则，前端同样只提交 `.env.example`。

---

## 4.7 本章完成后的目录状态

```text
magic-stream-movies/
├── .gitignore
├── Server/
│   └── MagicStreamMoviesServer/
│       ├── .env            (不提交)
│       ├── .env.example
│       ├── go.mod
│       ├── go.sum
│       └── main.go         (只有 /health)
└── Client/
    └── magic-stream-client/
        ├── .env            (不提交)
        ├── index.html
        ├── package.json
        ├── vite.config.ts
        ├── tsconfig*.json
        └── src/
            └── main.tsx
```

---

## 本章练习

> 答案见 [第 20 章](20-exercise-solutions.md#第-04-章)

**关键概念回顾**
- `go.mod` 和 `go.sum` 分别记录什么？
- 为什么 Vite 只暴露 `VITE_` 开头的变量？

**面试题 4.1** “`go get` 和 `go mod tidy` 有什么区别？”

**面试题 4.2** “前端代码里可以放 API Key 吗？为什么？”

**练习 4.3** 在一个临时目录中执行 `go mod init example.com/hello`，写一个只有 `/hello` 路由的 Gin 服务器，运行并用 `curl` 访问。

**练习 4.4** 在临时项目中执行 `go get github.com/gin-gonic/gin`，观察 `go.mod`；然后执行 `go mod tidy`，再观察 `go.mod` 的变化。

**调试挑战 4.5** 把 `S/.env` 中的 `SECRET_KEY` 改成 `short`，并删掉 `DATABASE_NAME`（在临时副本中操作），启动服务器。会看到什么输出？为什么 `config.Load()` 要把所有错误一起报告，而不是遇到第一个就返回？

**理解检验 4.6** 如果 `config.Load()` 不检查密钥长度，而 `SECRET_KEY` 恰好是空字符串，登录还能成功吗？为什么这很危险？请沿着 `tokens.Manager` 的代码推理。
