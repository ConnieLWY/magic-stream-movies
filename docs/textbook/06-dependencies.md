# 第 06 章　依赖详解

> 本章目标：理解项目使用的每一个第三方库——它解决什么问题、在项目哪里被使用、有哪些替代方案。

---

## 6.1 为什么要用第三方库？

**What**：第三方库（依赖）是别人写好、发布出来供你复用的代码。

**Why**：JWT 签名、bcrypt 哈希、MongoDB 协议这些东西，自己从零实现既费时又极易出错（尤其是安全相关的代码）。成熟的库经过了大量用户的检验。

**代价**：
- **供应链风险**：依赖可能被投毒或存在漏洞。`go.sum` 和 `package-lock.json` 通过锁定哈希值来缓解此风险。
- **升级成本**：大版本升级可能有破坏性变更（例如 mongo-driver v1 → v2，导入路径和 API 都变了）。
- **体积**：前端每多一个依赖，用户就要多下载一些 JS。

**原则**：能用标准库就用标准库；必须用第三方库时，选维护活跃、使用广泛的。

---

## 6.2 后端直接依赖

`go.mod` 第一个 `require` 块列出了 9 个直接依赖（见第 04 章 4.2.6 节）：

| 依赖 | 版本 | 被谁 import | 一句话作用 |
|---|---|---|---|
| `github.com/gin-gonic/gin` | v1.12.0 | main、routes、controllers、middleware | Web 框架 |
| `github.com/gin-contrib/cors` | v1.7.9 | main | CORS Middleware |
| `github.com/joho/godotenv` | v1.5.1 | main、cmd/seed | 读取 `.env` |
| `go.mongodb.org/mongo-driver/v2` | v2.9.2 | database、models、repository、cmd/seed | MongoDB 官方驱动 |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | tokens | JWT 签发与校验 |
| `golang.org/x/crypto`（`bcrypt`） | v0.58.0 | services | 密码哈希 |
| `github.com/go-playground/validator/v10` | v10.30.5 | validation、controllers | struct 校验 |
| `github.com/tmc/langchaingo` | v0.1.15 | services | 调用 OpenAI |
| `golang.org/x/time`（`rate`） | v0.16.0 | middleware、routes | 令牌桶限流 |

注意每个依赖只出现在少数几个包里：例如只有 `tokens` 知道 JWT 库，只有 `repository`（以及基础设施 `database`）知道 MongoDB 的查询 API。将来要换 JWT 库或数据库，影响范围一目了然。这是分层带来的另一个好处。

其余 30 多个（`bytedance/sonic`、`quic-go`、`pkoukk/tiktoken-go`……）都是这些库自己的依赖。

---

### 6.2.1 Gin

**What**：Go 最流行的 HTTP Web 框架之一，基于标准库 `net/http`，提供路由、Middleware 链、参数绑定、JSON 渲染。

**Why**：标准库 `net/http` 也能写 Web 服务，但需要自己处理很多细节。Gin 提供：
- 路径参数：`/movies/:imdb_id` → `c.Param("imdb_id")`。
- JSON 绑定：`c.ShouldBindJSON(&req)`。
- JSON 响应：`c.JSON(200, data)`。
- Middleware 链和路由组：`c.Next()`、`c.AbortWithStatusJSON()`、`router.Group()`。

**How it works（内部原理）**：
- 路由使用**基数树（radix tree）**存储，按路径前缀匹配，查找速度快。
- 每个请求从 `sync.Pool`（对象池）中取一个 `*gin.Context` 复用，减少内存分配。**这意味着：绝对不能在 Handler 返回后（例如在另起的 goroutine 中）继续使用 `c`**，因为它可能已被下一个请求重用。需要的话必须用 `c.Copy()`。

**项目中的用法**：`gin.New()`、`router.Use`、`router.Group`、`router.GET/POST/PATCH`、`c.JSON`、`c.Param`、`c.ShouldBindJSON`、`c.Set/c.GetString`、`c.Cookie`、`c.ClientIP`、`c.AbortWithStatusJSON`、`c.Next`。

**替代方案**

| 框架 | 特点 |
|---|---|
| 标准库 `net/http`（Go 1.22+ 支持 `GET /movies/{id}` 风格路由） | 零依赖；需要自己写 JSON 辅助函数 |
| Echo | 与 Gin 类似，API 更统一 |
| Fiber | 基于 fasthttp，性能更高但与 `net/http` 生态不兼容 |
| Chi | 轻量，完全兼容 `net/http` |

---

### 6.2.2 gin-contrib/cors

**What**：为 Gin 提供 CORS（跨源资源共享）支持的 Middleware。

**Why**：前端 `localhost:5173` 访问后端 `localhost:8080`，浏览器会先发 `OPTIONS` 预检请求，询问服务器“允许这个来源吗？”。没有正确的响应头，浏览器就会拦截真正的请求。

**项目中的用法**：`S/main.go` 中的 `cors.New(cors.Config{...})`，第 12 章详解。

---

### 6.2.3 godotenv

**What**：读取 `.env` 文件，把其中的键值对设置为进程环境变量。

**How**：`godotenv.Load(".env")` 读取文件 → 逐行解析 `KEY=VALUE` → 调用 `os.Setenv`。**已经存在的环境变量不会被覆盖**（`Load` 的行为；`Overload` 才会覆盖）。这个特性很有用：在生产环境中，平台注入的真实环境变量优先。

**项目中的用法**：只在程序入口调用一次——`main.run()` 和 `cmd/seed` 的 `run()`：

```go
// S/main.go
if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
	return err
}
cfg, err := config.Load()
```

- `.env` 不存在是正常的（例如 Docker 中由平台直接注入环境变量），所以忽略 `os.ErrNotExist`；其他错误（例如文件格式错误）才返回。
- 加载之后，`config.Load()` 一次性读取所有变量。业务代码从不调用 `os.Getenv`。

**为什么只加载一次？** 如果在每个需要配置的函数里都 `godotenv.Load()`，就会在每个请求中重复读磁盘，而且配置散落各处，很难回答“这个程序到底需要哪些配置”。集中到启动阶段，还能在启动时就发现缺失的配置，而不是等到某个请求触发时才出错。

---

### 6.2.4 mongo-driver v2

**What**：MongoDB 官方 Go 驱动。v2 是 2025 年发布的大版本，相比 v1：
- 导入路径变为 `go.mongodb.org/mongo-driver/v2/...`。
- `mongo.Connect` 不再需要传入 `context`：`mongo.Connect(opts)`（见 `S/database/database_connection.go` 的 `Connect`）。
- `primitive.ObjectID` 移到了 `bson.ObjectID`（见 `S/models/movie_model.go` 的 `Movie.ID`）。
- `Find` 等方法的选项参数变成了 `options.Lister[...]` 接口（见 `S/repository/movie_repository.go` 的 `findAll`）。

> 网上很多教程仍是 v1 写法（`mongo.Connect(ctx, opts)`、`primitive.ObjectID`），照抄会编译失败。这是学习时需要特别注意的。

**常用类型**

| 类型 | 含义 | 项目中的例子 |
|---|---|---|
| `bson.M` | `map[string]any`，无序 | 过滤条件 `bson.M{"imdb_id": movieID}` |
| `bson.D` | 有序键值对切片 | 排序 `bson.D{{Key: "ranking.ranking_value", Value: 1}}` |
| `bson.A` | 数组 `[]any` | 本项目没有用到：解码时直接用 struct，比在 `bson.A` / `bson.D` 上做类型断言安全得多 |
| `bson.ObjectID` | MongoDB 的 12 字节唯一 ID | `Movie.ID`、`User.ID` |

**为什么排序要用 `bson.D` 而不用 `bson.M`？** Go 的 map 遍历顺序是随机的。如果要按多个字段排序（先按 A，再按 B），用 map 就无法保证顺序。

---

### 6.2.5 golang-jwt/jwt v5

**What**：Go 社区最常用的 JWT 库（原 `dgrijalva/jwt-go` 的后继维护版本）。

**项目中的用法**：`jwt.NewWithClaims`、`SignedString`、`ParseWithClaims`、`jwt.RegisteredClaims`、`jwt.NewNumericDate`，以及解析选项 `jwt.WithValidMethods`、`jwt.WithIssuer`、`jwt.WithExpirationRequired`、`jwt.WithTimeFunc`（全部在 `S/tokens/tokens.go`）。

**v5 的重要行为**（第 11 章会用到）：
- `ParseWithClaims` 会**自动校验 `exp`（过期时间）**，过期返回 `jwt.ErrTokenExpired`，不需要手动检查。
- 但如果 Token 里**根本没有** `exp`，默认会被当作“永不过期”而通过——所以要加 `jwt.WithExpirationRequired()`。
- 默认拒绝 `alg: none` 的 Token；`jwt.WithValidMethods` 进一步把可接受的算法限定为 HS256，而且这个检查发生在**验证签名之前**。
- HMAC 签名时**不会拒绝空密钥**。`[]byte("")` 也能签名成功——这就是 `config.Load()` 要求密钥至少 32 字节的原因（第 04 章理解检验 4.6）。

---

### 6.2.6 bcrypt（`golang.org/x/crypto/bcrypt`）

**What**：一种专为密码设计的哈希算法。

**Why 不用 SHA-256？** SHA-256 设计目标是“快”，GPU 每秒能算几十亿次，暴力破解很容易。bcrypt 故意设计得“慢”，并且有可调的**成本因子（cost）**。

**项目中的用法**：`bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)`（`DefaultCost` = 10）和 `bcrypt.CompareHashAndPassword`。第 11 章详解。

**限制**：bcrypt 只使用密码的前 72 个字节。较新版本的 `x/crypto` 在密码超过 72 字节时会返回错误（`bcrypt.ErrPasswordTooLong`），而不是静默截断。所以 `UserRegister.Password` 的校验规则是 `max=72`，`AuthService.Register` 还会再按**字节**检查一次（validator 的 `max` 数的是字符，一个中文或 `é` 占多个字节）。

**替代方案**：Argon2id（OWASP 当前首推）、scrypt。

---

### 6.2.7 go-playground/validator v10

**What**：通过 struct tag 声明校验规则，例如 `validate:"required,email"`。

**How**：用反射读取每个字段的 `validate` tag，逐条执行规则，收集所有失败项，返回 `validator.ValidationErrors`。

**项目中的用法**：`validation.New()` 创建一个共享实例，注册自定义规则 `imdb_id`、`youtube_id`，并让错误信息使用 JSON 字段名；`controllers.bindJSON` 中调用 `v.Struct(dst)`。第 13 章详解。

> **注意**：Gin 内置的 `binding:"required"` tag 底层也是这个库，但本项目没有使用 `binding` tag，而是手动调用 `validate.Struct`。这样校验规则和自定义规则只在一个地方配置，不依赖 Gin 内部的那个 validator 实例。

---

### 6.2.8 langchaingo

**What**：LangChain 的 Go 语言移植版，为多种 LLM（OpenAI、Anthropic、Ollama 等）提供统一接口。

**项目中的用法**（`S/services/openai_classifier.go`）：
```go
// 启动时创建一次
llm, err := openai.New(openai.WithToken(apiKey), openai.WithModel(model))

// 每次分类时调用；ctx 带有超时
prompt := strings.Replace(c.promptTemplate, "{rankings}", strings.Join(labels, ","), 1) + review
return llms.GenerateFromSinglePrompt(ctx, c.llm, prompt)
```

`OpenAIClassifier` 实现了 `services.ReviewClassifier` 接口。业务代码只依赖这个接口，不知道背后是 langchaingo、官方 SDK 还是测试用的假实现。

**评价**：本项目只做了一次简单的文本补全调用，引入整个 langchaingo（间接带来了 `tiktoken-go`、`dlclark/regexp2` 等依赖）有点“大材小用”。好在它被封装在一个文件里，替换成本很低（练习 6.5）。

**替代方案**

| 方案 | 优点 | 缺点 |
|---|---|---|
| langchaingo（本项目） | 切换 LLM 供应商方便 | 依赖多、抽象层厚 |
| OpenAI 官方 Go SDK（`github.com/openai/openai-go`） | 官方维护、功能完整 | 只支持 OpenAI |
| 直接用 `net/http` 调 REST API | 零依赖、完全可控 | 要自己处理 JSON、重试 |

### 6.2.9 golang.org/x/time/rate

**What**：Go 团队维护的扩展库中的**令牌桶（token bucket）**限流器。

**How**：桶里最多有 `burst` 个令牌，每隔一段时间补充一个。每个请求取走一个令牌；桶空了就拒绝。这样既允许短时间的突发，又限制了长期的平均速率。

**项目中的用法**（`S/middleware/security.go` 的 `RateLimit`、`S/routes/routes.go`）：
```go
// 平均每 6 秒一个令牌（每分钟 10 次），最多突发 10 次
authLimit := middleware.RateLimit(rate.Every(6*time.Second), 10)
```
每个客户端 IP 一个 `*rate.Limiter`，存在一个加锁的 map 里。第 12 章详解。

**替代方案**：Redis + 滑动窗口（多实例部署时需要共享计数）、反向代理（Nginx `limit_req`）或 API 网关层限流。

---

## 6.3 前端依赖

### 6.3.1 运行时依赖（`dependencies`）

| 依赖 | 版本范围 | 作用 | 使用位置 |
|---|---|---|---|
| `react` / `react-dom` | ^19.3.1 | UI 库 / 渲染到浏览器 DOM | 全部组件 |
| `react-router-dom` | ^7.18.5 | 客户端路由 | `main.tsx`、`App.tsx`、各页面 |
| `axios` | ^1.21.0 | HTTP 客户端 | `api/axiosConfig.ts`、`hook/useAxiosPrivate.tsx` |
| `bootstrap` | ^5.3.8 | CSS 框架 | `main.tsx` 引入 CSS |
| `react-bootstrap` | ^2.10.10 | Bootstrap 的 React 组件封装 | `Form`、`Button`、`Navbar`、`Alert`…… |

**版本号中的 `^` 是什么意思？** `^1.21.0` 表示“兼容 1.21.0 的版本”，即 `>=1.21.0 <2.0.0`。`npm install` 可能安装更新的次版本。真正安装的精确版本记录在 `package-lock.json` 中——**这个文件必须提交到 Git**，以保证所有人安装的版本一致。

#### axios vs fetch

| | axios | 浏览器原生 `fetch` |
|---|---|---|
| 拦截器（interceptors） | ✅ 内置 | ❌ 需要自己封装 |
| 非 2xx 自动抛错 | ✅ | ❌（需要手动检查 `response.ok`） |
| 自动 JSON 解析 | ✅ | 需要 `await res.json()` |
| 体积 | 约 13 KB（gzip 后） | 0 |

本项目大量依赖**拦截器**实现自动刷新 Token，这是选择 axios 的主要理由。

#### 播放预告片为什么不用播放器库？

`StreamMovie.tsx` 直接渲染一个 `<iframe>`：

```tsx
<iframe
  src={`https://www.youtube-nocookie.com/embed/${youTubeId}?autoplay=1&rel=0`}
  title="Trailer"
  allow="autoplay; encrypted-media; picture-in-picture; fullscreen"
  allowFullScreen
/>
```

像 `react-player` 这样的通用播放器库支持 YouTube、Vimeo、HLS、DASH 等多种来源，会把这些播放器引擎一起打包（HLS 和 DASH 引擎压缩前就超过 1 MB）。本项目只播放 YouTube，YouTube 官方的嵌入页面已经是一个完整的播放器，所以一个 `<iframe>` 就够了，**零依赖**。`youtube-nocookie.com` 是 YouTube 的“隐私增强模式”域名，在用户点击播放之前不会写入跟踪 Cookie。

**原则**：引入依赖之前先问“平台本身能不能做到”。

### 6.3.2 开发依赖（`devDependencies`）

| 依赖 | 作用 |
|---|---|
| `vite` ^8.3.5 | 开发服务器与打包 |
| `@vitejs/plugin-react` | 让 Vite 支持 JSX 和 React Fast Refresh |
| `typescript` ~6.0.3 | 类型检查（`~` 表示只接受补丁版本更新：`>=6.0.3 <6.1.0`） |
| `@types/react`、`@types/react-dom`、`@types/node` | 类型定义 |
| `eslint` + `@eslint/js` + `typescript-eslint` | 代码规范 |
| `eslint-plugin-react-hooks` | 检查 Hook 规则（如依赖数组是否完整） |
| `eslint-plugin-react-refresh` | 检查 Fast Refresh 兼容性 |
| `globals` | 为 ESLint 提供浏览器全局变量列表 |

**为什么区分 `dependencies` 和 `devDependencies`？** 对于前端 SPA，两者最终都只在构建时使用，区别主要是语义上的。但对于 Node 后端或发布的 npm 包，`npm install --production` 只会安装 `dependencies`，可以减小部署体积。

---

## 6.4 如何安全地管理依赖

1. **锁文件必须提交**：`go.sum`、`package-lock.json`。
2. **定期检查漏洞**：
   ```bash
   # Go
   go install golang.org/x/vuln/cmd/govulncheck@latest
   govulncheck ./...
   # npm
   npm audit
   ```
3. **升级前读 CHANGELOG**，特别是大版本。
4. **清理未使用的依赖**：`go mod tidy`；前端可以用 `npx depcheck`。

---

## 本章练习

> 答案见 [第 20 章](20-exercise-solutions.md#第-06-章)

**关键概念回顾**
- 为什么密码不能用 SHA-256 哈希？
- `bson.M` 和 `bson.D` 的区别是什么？什么时候必须用 `bson.D`？

**面试题 6.1** “`^1.2.3` 和 `~1.2.3` 有什么区别？”

**面试题 6.2** “为什么不能在 goroutine 中直接使用 `*gin.Context`？”

**练习 6.3** 在 `S/` 目录运行 `grep -rh '"github.com\|"go.mongodb\|"golang.org' --include=*.go . | sort -u`，验证 6.2 节的直接依赖列表。

**练习 6.4** 运行 `govulncheck ./...` 和 `npm audit`，记录结果。

**练习 6.5** 写一个不依赖 langchaingo、只使用 `net/http` 调用 OpenAI Chat Completions API 的函数 `classifyReview(review string, labels []string) (string, error)`。

**理解检验 6.6** 如果把 mongo-driver 从 v2 换回 v1，需要改哪些包？为什么 `services`、`controllers` 完全不用改？
