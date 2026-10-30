# 第 16 章　安全

> 本章目标：从攻击者的角度审视这个项目——有哪些资产需要保护、可能受到哪些攻击、每一种攻击由哪段代码防御——并学会把这些做法迁移到你自己的项目中。

---

## 16.1 威胁模型与防护总览

**威胁模型（Threat Model）**：先回答三个问题——**保护什么**（资产）、**防谁**（攻击者能做什么）、**怎么防**（控制措施）。

| 资产 | 如果泄露或被滥用 |
|---|---|
| 用户密码 | 用户在其他网站的账户也可能被盗（密码复用） |
| Refresh / Access Token | 攻击者冒充用户 |
| 管理员权限 | 篡改影评和评级，影响所有用户的推荐 |
| JWT 签名密钥 | 攻击者伪造任意用户、任意角色的 Token |
| OpenAI API Key | 攻击者消耗你的额度，产生账单 |
| 服务器资源 | 拒绝服务（耗尽内存、连接、CPU） |

| 威胁 | 防御措施 | 代码位置 | 小节 |
|---|---|---|---|
| 密钥泄露 | `.env` 不提交、启动时校验强度、泄露后轮换 | `.gitignore`、`config.Load` | 16.2 |
| 越权访问（普通用户调用管理员接口、注册时自选角色） | 路由组 + `RequireRole`、请求 DTO 不含 `role` | `routes.go`、`models.UserRegister` | 16.3 |
| 密码被暴力破解 / 用户枚举 | bcrypt、统一错误信息、恒定耗时、限流 | `AuthService.Login`、`RateLimit` | 16.4、16.6 |
| Token 被盗用 | 短 AT、RT 轮换、只存 RT 哈希、HttpOnly + SameSite + Path | `tokens`、`AuthService.Refresh`、`setRefreshCookie` | 16.4 |
| Token 伪造 | 强密钥、限定算法和签发者、要求 `exp` | `tokens.parse` | 16.4 |
| 信息泄露（日志、错误响应） | 不记录凭证、500 只返回概述 | `database.Connect`、`respondError` | 16.5 |
| 资源耗尽 | 请求体上限、字段长度上限、HTTP 超时、限流 | `BodyLimit`、`http.Server` | 16.6、16.7、16.10 |
| 注入（NoSQL、Prompt） | 强类型绑定、LLM 输出白名单 | DTO、`matchRanking` | 16.7、16.9 |
| 浏览器端攻击（XSS、点击劫持、URL 注入） | React 自动转义、安全响应头、校验 YouTube ID | 前端、`SecurityHeaders` | 16.10、16.11 |
| 有漏洞的依赖 | `govulncheck`、`npm audit`、CI | — | 16.12 |

本章按这张表逐项展开。

---

## 16.2 密钥管理

### 16.2.1 密钥只存在于部署环境中

- 仓库中只提交 `S/.env.example` 和 `C/.env.example`：只有变量名、说明和无害的示例值。
- 根目录的 `.gitignore` 忽略所有 `.env` 和 `.env.*`，但用 `!.env.example` 放行模板（第 04 章 4.6 节）。
- `Dockerfile` 旁边的 `.dockerignore` 同样排除 `.env`：构建镜像时它不会被复制进镜像层。镜像经常被推送到镜像仓库，任何能拉取镜像的人都能解开每一层查看文件。
- 生产环境中由部署平台注入环境变量（第 17 章）。

> **注意**：只把 `.env` 写进 `.gitignore` 是不够的。`.gitignore` 只对**未被跟踪**的文件生效；如果 `.env` 已经被 Git 跟踪，之后对它的修改仍然会出现在 `git status` 中，并会被 `git commit -a` 提交。这种情况下需要执行 `git rm --cached <路径>`，从索引中移除但保留本地文件。

### 16.2.2 启动时校验密钥强度

```go
// S/config/config.go（节选）
// MinSecretLength is the minimum JWT signing key length in bytes. An HS256 key should be at least as long as the hash output (32 bytes).
const MinSecretLength = 32

if n := len(cfg.SecretKey); n > 0 && n < MinSecretLength {
	errs = append(errs, fmt.Errorf("SECRET_KEY must be at least %d bytes, got %d", MinSecretLength, n))
}
// ... SECRET_REFRESH_KEY 同理 ...
if len(cfg.SecretKey) > 0 && string(cfg.SecretKey) == string(cfg.SecretRefreshKey) {
	errs = append(errs, errors.New("SECRET_KEY and SECRET_REFRESH_KEY must be different"))
}
```

- 缺失、太短、两个密钥相同，服务器都**拒绝启动**。golang-jwt 不会拒绝空密钥或短密钥（第 06 章 6.2.5），所以这道检查必须由我们自己做。
- 生成方法：`openssl rand -base64 32`（32 字节随机数，Base64 编码后 44 个字符）。人想出来的“密钥”（如 `mysecret123`）熵太低，可以被离线暴力破解：攻击者拿到任何一个合法的 JWT，就可以在自己的电脑上不断尝试密钥，直到签名对得上。

### 16.2.3 如果密钥曾经被提交到 Git

即使之后删除文件并提交，密钥仍然存在于 Git 历史中，`git log -p` 就能看到；如果仓库推送过，它可能已经被克隆、fork、被爬虫扫描到（公开仓库中的 API Key 通常几分钟内就会被自动化工具发现并滥用）。

**处理步骤（按顺序）**：

**第 1 步：立即轮换（最重要，不可跳过）**

| 密钥 | 操作 |
|---|---|
| OpenAI API Key | 在 OpenAI 控制台**删除**旧 Key，创建新 Key，并设置用量上限 |
| 数据库密码 | 修改该数据库用户的密码，或删除该用户另建一个 |
| `SECRET_KEY` / `SECRET_REFRESH_KEY` | 生成新值。更换后所有现有 Token 都会失效，用户需要重新登录——这正是我们想要的 |

**轮换之后，旧密钥就算被看到也没有用了。** 下面的步骤是“清理”，但不能代替轮换。

**第 2 步：停止跟踪**

```bash
git rm --cached Server/MagicStreamMoviesServer/.env Client/magic-stream-client/.env
git add .gitignore
git commit -m "Stop tracking .env files"
```

**第 3 步（可选）：从历史中清除**

```bash
# 需要安装 git-filter-repo：brew install git-filter-repo
git filter-repo --path Server/MagicStreamMoviesServer/.env --invert-paths
git push --force   # 改写历史，所有协作者需要重新克隆
```

⚠️ 改写历史是破坏性操作。对于个人学习项目可以这样做；对于多人协作的仓库，需要事先协调。而且 GitHub 上的旧提交可能仍被缓存或被 fork 保留——**这再次说明了第 1 步轮换的必要性。**

**第 4 步：预防**

```bash
brew install gitleaks
gitleaks detect --source .        # 扫描整个 Git 历史
```

还可以把 gitleaks 加入 pre-commit hook 或 CI，或者开启 GitHub 的 Secret Scanning。

---

## 16.3 访问控制

OWASP Top 10（2021 版）排名第一的风险是 **A01 Broken Access Control（失效的访问控制）**：用户能做他本不该被允许做的事。

### 16.3.1 授权集中在路由表

```go
// S/routes/routes.go（节选）
protected := router.Group("/", middleware.Auth(tm))
protected.GET("/movies/:imdb_id", movies.GetMovie)
protected.GET("/me/recommendations", movies.Recommended)

admin := protected.Group("/", middleware.RequireRole(models.RoleAdmin))
admin.POST("/movies", movies.CreateMovie)
admin.PATCH("/movies/:imdb_id/review", movies.UpdateReview)
```

- **新增的管理员接口只要放进 `admin` 组就自动受到保护**。如果授权检查分散在各个 Handler 中（`if role != "ADMIN" { ... }`），新写的 Handler 很容易忘记加。
- **默认拒绝**：`RequireRole` 在角色缺失时也返回 403，即使它被错误地挂在没有 `Auth` 的组上，结果也是拒绝而不是放行。
- **401 与 403 分开**：403 不会触发前端的 Token 刷新（第 09 章 9.1）。
- **测试锁定**：`TestAuthorization` 的表格列出了每个接口在不同身份下的状态码（第 15 章 15.5.1）。

### 16.3.2 身份只来自服务器验证过的凭证

| 场景 | 身份来源 | 如果改为信任请求参数 |
|---|---|---|
| 推荐 | `middleware.UserID(c)`（来自 JWT） | 传别人的 `user_id` 就能看别人的数据 |
| 登出 | Cookie 中的 RT | 传别人的 `user_id` 就能让别人登出 |
| 角色 | JWT 中的 `role`（签发时从数据库读取） | 直接修改请求就能成为管理员 |

### 16.3.3 批量赋值：请求 DTO 只包含允许的字段

注册请求绑定到 `UserRegister`，它**没有** `Role` 字段；添加电影绑定到 `MovieCreate`，它**没有** `Ranking` 字段。客户端提交的多余字段在解析时被丢弃，服务器写入的值（角色 `USER`、评级“未评级”）由 service 决定（第 08 章 8.4）。注意：在 `User` 上加 `validate:"oneof=ADMIN USER"` 并不能解决这个问题——那只校验**格式**，`ADMIN` 也是合法的格式。

**管理员如何产生？** 只能由有数据库权限的人运行 `go run ./cmd/seed -admin <邮箱>`。这是一个**带外**（out-of-band）操作：HTTP API 中不存在任何提升权限的入口，自然也就无法被远程利用。

---

## 16.4 认证与会话

详细设计见第 11 章，这里汇总每个攻击场景对应的防御：

| 攻击 | 防御 |
|---|---|
| 数据库泄露 → 获取密码 | bcrypt（加盐、慢哈希） |
| 数据库泄露 → 获取 Token | AT 不存数据库；RT 只存 SHA-256 哈希 |
| 通过错误信息判断邮箱是否注册 | 两种登录失败返回同一个错误 |
| 通过响应时间判断邮箱是否注册 | 邮箱不存在时也做一次 bcrypt 比对（`dummyHash`） |
| XSS 脚本读取 Token | AT 只在 JS 内存中；RT 在 HttpOnly Cookie 中 |
| CSRF 利用 Cookie | AT 不在 Cookie 中；RT Cookie 是 `SameSite=Lax`，跨站 POST 不携带 |
| RT 被盗后长期使用 | 每次刷新都轮换；旧 RT 立即失效；7 天后过期 |
| 两个请求并发使用同一个 RT | 比较并交换（`RotateRefreshTokenHash`），只有一个成功 |
| 登出后 RT 继续使用 | 登出时删除数据库中的哈希 |
| 伪造 Token（`alg: none`、算法混淆、其他系统签发的 Token） | `WithValidMethods`、`WithIssuer`、强密钥 |
| 永不过期的 Token | `WithExpirationRequired` |
| RT 被当作 AT 使用 | 两个不同的签名密钥 |
| RT 出现在不需要它的请求中 | Cookie `Path=/auth` |

**已知的权衡**：
- AT 签发后无法撤销，最多 15 分钟内仍然有效（无状态的代价）。
- 注册接口的 409 会暴露邮箱是否已注册（第 11 章 11.3.4）。

---

## 16.5 日志与错误信息

### 16.5.1 日志中不出现凭证

日志通常会被：发送到日志平台、出现在 CI 输出中、被复制到 issue 和聊天记录里。**日志不是安全的存储位置。** 本项目的约定：

- **不记录连接字符串**：`database.Connect` 的注释写明了 “Never log the URI: it may contain a password”。连接失败时返回的错误只包含驱动给出的原因（例如 `error parsing uri: scheme must be...`）。
- **不记录 Token 和密码**：调试时如果需要，只记录“是否存在”（`hasStoredHash=true`），而不是值本身（第 15 章 15.8.3）。
- **前端不打印响应**：登录响应中有 Access Token，不能 `console.log` 出来。

如果确实需要确认连接到了哪里，可以只打印主机名：

```go
if u, err := url.Parse(uri); err == nil {
	log.Printf("connecting to MongoDB host %s", u.Host)   // 不含用户名密码
}
```

### 16.5.2 错误响应不暴露内部信息

`respondError` 只把**业务错误**（我们自己写的、专门给客户端看的消息）返回给客户端；其他错误一律返回 `internal server error`，完整信息只写进服务器日志（第 13 章 13.6）。校验错误的 `details` 也只包含 JSON 字段名和规则名，不包含 Go 的类型名。

---

## 16.6 暴力破解与限流

`POST /auth/login` 和 `POST /auth/register` 按客户端 IP 限流：平均每分钟 10 次，允许突发 10 次，超过返回 429（实现见第 12 章 12.6.3）。

- **为什么 bcrypt 的“慢”还不够？** bcrypt 让每次尝试花费几十毫秒，但攻击者可以并发发送请求；没有限流时，一台机器每秒仍然可以尝试几百次。
- **为什么注册也要限流？** 防止批量注册垃圾账户，也增加了通过 409 枚举邮箱的成本。
- **可信代理**：限流依赖 `c.ClientIP()`。默认不信任任何代理，客户端无法通过伪造 `X-Forwarded-For` 绕过；部署在反向代理后面时必须配置 `TRUSTED_PROXIES`（第 12 章理解检验 12.10）。

**进一步的措施**（适合更高安全要求的系统）：
- 按**账户**计数失败次数，连续失败 N 次后暂时锁定或要求验证码（按 IP 限流挡不住分布式攻击）。
- 在反向代理或 CDN 层（Nginx `limit_req`、Cloudflare）再做一层限流。
- 多实例部署时，用 Redis 等共享存储计数。

---

## 16.7 输入校验与资源限制

### 16.7.1 资源上限

| 资源 | 上限 | 如果没有 |
|---|---|---|
| 请求体 | 1 MB（`BodyLimit`） | 一个几 GB 的请求就能耗尽内存 |
| 字符串字段 | 姓名 100、标题 500、URL 2048、邮箱 254、影评 2000 | 数据库被塞满；影评被原样发给按 token 计费的 OpenAI |
| 列表字段 | 类型 ID 最多 20 个 | 一次查询带上万个 ID |
| 连接 | `http.Server` 的各项超时（16.10） | 慢速客户端长时间占用连接 |
| 请求处理时间 | 每个请求的 context 超时 | 下游卡住时 goroutine 无限堆积 |

### 16.7.2 NoSQL 注入

如果代码这样写：

```go
var body map[string]any
c.ShouldBindJSON(&body)
coll.FindOne(ctx, bson.M{"email": body["email"]})   // ⚠️ 危险
```

攻击者发送 `{"email": {"$ne": null}, ...}`，查询就变成了“邮箱不等于 null”，会匹配到第一个用户。

本项目把请求绑定到 **`UserLogin` struct，其 `Email` 字段是 `string` 类型**。如果客户端发送对象，JSON 解析会失败（返回 400）。**强类型绑定天然防御了操作符注入。** 再加上 `email` 校验规则，`Email` 只可能是一个格式合法的邮箱字符串。

---

## 16.8 密码策略

```go
Password string `json:"password" validate:"required,min=8,max=72"`
```

参考 NIST SP 800-63B：
- **最少 8 个字符**：长度是密码强度最重要的因素。
- **最多 72 字节**：bcrypt 的限制，超出的部分会被忽略（新版 `x/crypto` 直接报错）。本项目在校验之外再按字节检查一次（第 11 章 11.3.4）。
- **不强制复杂度规则**（必须包含大写字母、数字、符号）：研究表明它们效果有限，用户往往用 `Password1!` 这种可预测的方式满足规则。更有效的做法是检查密码是否在**已泄露密码列表**中（例如 Have I Been Pwned 的 k-匿名 API），这是一个很好的扩展练习。
- **登录时不检查最小长度**：密码规则可能随时间变严格，老用户在旧规则下设置的密码仍然应该能登录。

---

## 16.9 LLM 输出

LLM 的输出与用户输入一样不可信（第 10 章 10.3.5–10.3.6）：

- **白名单**：只接受与已有评级名称匹配的回答，否则返回 502、不保存。
- **规范化后存储标准值**：存入数据库的是 `rankings` 集合中的名称，而不是 LLM 的原始文本。
- **最小权限**：LLM 的输出只用于选择一个评级，不会被执行、不会拼进查询。
- **成本控制**：先确认电影存在再调用；影评长度上限；调用超时；建议在 OpenAI 控制台设置用量上限。

---

## 16.10 HTTP 层加固

### 16.10.1 安全响应头

```go
// S/middleware/security.go
h.Set("X-Content-Type-Options", "nosniff")
h.Set("X-Frame-Options", "DENY")
h.Set("Referrer-Policy", "no-referrer")
h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
```

| 响应头 | 作用 |
|---|---|
| `X-Content-Type-Options: nosniff` | 禁止浏览器“猜测”内容类型。否则一个被当作 JSON 返回、但内容像 HTML 的响应，可能被浏览器当作网页执行 |
| `X-Frame-Options: DENY` | 禁止页面被嵌入 `<iframe>`，防御点击劫持 |
| `Referrer-Policy: no-referrer` | 不在后续请求中泄露当前 URL |
| `Content-Security-Policy: default-src 'none'; frame-ancestors 'none'` | 这是一个 JSON API，响应不应该加载任何资源、执行任何脚本，也不应该被嵌入。CSP 把这一点告诉浏览器 |

这些头是为 **API** 设计的。前端页面的 CSP 要复杂得多（需要允许自己的脚本、YouTube 的 iframe 等），应在托管前端的服务器或 CDN 上配置（第 17 章）。

**HSTS**（`Strict-Transport-Security`）告诉浏览器“以后只用 HTTPS 访问这个域名”。它应该在 HTTPS 终结的地方（反向代理、CDN）设置——如果在本地 HTTP 开发环境中发送它，浏览器会记住并在之后强制使用 HTTPS，反而造成麻烦。

### 16.10.2 HTTP Server 超时

```go
// S/main.go
server := &http.Server{
	Addr:              ":" + cfg.Port,
	Handler:           router,
	ReadHeaderTimeout: 5 * time.Second,
	ReadTimeout:       15 * time.Second,
	WriteTimeout:      cfg.RequestTimeout + cfg.OpenAITimeout + 5*time.Second,
	IdleTimeout:       60 * time.Second,
}
```

`router.Run(":8080")` 使用的是 Go 默认的 `http.Server`，**没有任何超时**。慢速客户端（**Slowloris 攻击**：每隔几秒发送一个字节的请求头，永远不发完）可以长时间占用连接，用很少的资源耗尽服务器的连接数。

| 超时 | 限制什么 |
|---|---|
| `ReadHeaderTimeout` | 读取请求头的时间——Slowloris 的主要防线 |
| `ReadTimeout` | 读取整个请求（含请求体）的时间 |
| `WriteTimeout` | 从读完请求头到写完响应的时间——必须大于最慢的接口（写影评要等 LLM），否则正常请求也会被切断 |
| `IdleTimeout` | keep-alive 连接空闲多久后关闭 |

---

## 16.11 前端安全

### XSS（跨站脚本）
React 在渲染 `{movie.title}`、`{movie.admin_review}` 时会**自动转义** HTML。即使影评中写了 `<script>alert(1)</script>`，页面上也只会显示这段文字，不会执行。本项目没有使用 `dangerouslySetInnerHTML`。

需要注意的地方：`style={{ backgroundImage: \`url(${featured.poster_path})\` }}`——`poster_path` 来自数据库，添加时经过了 `url` 校验，React 对 style 值不会执行脚本，风险很低。

### URL 注入
`StreamMovie` 把路由参数拼接进 `<iframe src>`。参数先经过 `/^[A-Za-z0-9_-]{11}$/` 校验，不可能注入路径、查询参数或 `javascript:` 之类的协议。后端对 `youtube_id` 使用同样的规则。

### Token 处理
AT 只保存在 React state 中，不写入 localStorage，不打印到控制台。

### 前端守卫只是用户体验
`RequiredAuth` 和 `isAdmin` 只决定显示什么。用户可以修改浏览器中的 JS 绕过它们，真正的检查永远在后端。

---

## 16.12 依赖与供应链

- **锁文件**：`go.sum` 和 `package-lock.json` 记录每个依赖的哈希，防止依赖在下载过程中被替换。
- **漏洞扫描**：
  ```bash
  go install golang.org/x/vuln/cmd/govulncheck@latest
  govulncheck ./...           # 只报告代码实际调用到的有漏洞函数，误报少
  npm audit                   # 前端依赖
  ```
- **减少依赖**：每个依赖都是潜在的攻击面。用 `<iframe>` 代替播放器库（第 06 章 6.3.1）也减少了供应链风险。
- **CI**：`.github/workflows/ci.yml` 在每次提交时运行 `go vet`、测试（含 `-race`）、`npm run lint` 和构建，可以在此基础上加入 `govulncheck` 和 `gitleaks`。
- **容器**：Docker 运行阶段使用 distroless 镜像并以非 root 用户运行（第 17 章），镜像中没有 shell 和包管理器，即使应用被攻破，攻击者能做的事情也很有限。

---

## 16.13 安全检查清单（用于你的重建项目）

本项目已经满足以下全部条目。重建项目时可以逐项对照：

- [x] `.env` 在 `.gitignore` 和 `.dockerignore` 中；提交 `.env.example`
- [x] 启动时校验所有必需配置，密钥长度 ≥ 32 且两个密钥不同
- [x] 日志中没有密码、Token、连接字符串
- [x] 注册使用专门的 DTO，`role` 由服务器设置
- [x] 所有管理员接口都在 `RequireRole("ADMIN")` 组中
- [x] 非管理员返回 403，未登录返回 401
- [x] JWT 校验限定算法、签发者，要求 `exp`
- [x] 数据库只存 RT 的哈希，不存 AT；RT 轮换是原子操作
- [x] `users.email`、`movies.imdb_id` 有唯一索引
- [x] 邮箱存储前统一小写
- [x] 登录、注册接口限流；配置可信代理
- [x] 500 错误不向客户端暴露内部信息
- [x] 请求体和字段长度有上限；HTTP Server 有超时
- [x] Cookie `HttpOnly`、`Secure`、`SameSite`、`Path`
- [x] LLM 输出必须在白名单内
- [x] 安全响应头
- [ ] 生产环境启用 HTTPS 和 HSTS（部署时配置，第 17 章）
- [ ] `govulncheck ./...`、`npm audit`、`gitleaks detect` 加入 CI

---

## 本章练习

> 答案见 [第 20 章](20-exercise-solutions.md#第-16-章)

**关键概念回顾**
- 密钥泄露后，为什么“轮换”比“从 Git 历史中删除”更重要？
- `oneof=ADMIN USER` 为什么不能防止权限提升？
- 强类型绑定是如何防止 NoSQL 注入的？

**面试题 16.1** “你发现同事把 AWS 密钥提交到了公开仓库，你会怎么做？按顺序说明。”

**面试题 16.2** “OWASP Top 10 的第一名是什么？你的项目是如何防御它的？”

**面试题 16.3** “React 是如何防止 XSS 的？什么情况下 React 应用仍然可能受到 XSS 攻击？”

**练习 16.4** 在本仓库运行 `gitleaks detect --source .`，阅读输出。如果它报告了历史提交中的密钥，按 16.2.3 节判断需要轮换哪些密钥。

**练习 16.5** 用 `curl -i http://localhost:8080/health` 检查安全响应头；再用 `curl` 发送一个 2 MB 的请求体到 `/auth/login`，观察响应。

**练习 16.6** 实现“按账户计数的登录失败锁定”：同一邮箱连续失败 5 次后，15 分钟内拒绝登录（返回 429）。思考：攻击者能否利用这个功能故意锁定别人的账户？如何缓解？

**调试挑战 16.7** 在你的重建项目中，故意用 `map[string]any` 绑定登录请求（如 16.7.2 节所示），然后用 `{"email": {"$ne": null}, "password": "x"}` 测试。查询会返回什么？（完成后务必改回 struct 绑定。）

**理解检验 16.8** 假设攻击者拿到了 `SECRET_KEY`，但没有拿到 `SECRET_REFRESH_KEY`。他能做什么、不能做什么？反过来呢？
