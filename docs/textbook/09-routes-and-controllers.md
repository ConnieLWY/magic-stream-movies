# 第 09 章　路由与 Controller

> 本章目标：理解 HTTP 路由和路由组的工作原理，掌握本项目“Controller 结构体 + 方法值”的依赖注入模式，并逐个读懂所有 Controller。

---

## 9.1 HTTP 快速回顾

一个 HTTP 请求由这几部分组成：

```http
PATCH /movies/tt0111161/review HTTP/1.1          ← 方法 + 路径
Host: localhost:8080                             ← Header
Authorization: Bearer eyJhbGciOi...
Content-Type: application/json

{"admin_review": "A masterpiece."}               ← Body
```

响应：

```http
HTTP/1.1 200 OK                                   ← 状态码
Content-Type: application/json; charset=utf-8

{"admin_review":"A masterpiece.","ranking":{"ranking_value":1,"ranking_name":"Excellent"}}
```

### 本项目使用的 HTTP 方法

| 方法 | 语义 | 本项目中的例子 |
|---|---|---|
| `GET` | 读取，不修改数据 | `/movies`、`/movies/:imdb_id`、`/me/recommendations` |
| `POST` | 创建资源或执行动作 | `/movies`（创建）、`/auth/login`（动作） |
| `PATCH` | 部分更新 | `/movies/:imdb_id/review` |

### 本项目使用的状态码

| 状态码 | 含义 | 何时返回 |
|---|---|---|
| 200 OK | 成功 | 大多数成功响应 |
| 201 Created | 已创建 | 注册、添加电影 |
| 400 Bad Request | 请求格式或内容有误 | JSON 解析失败、校验失败、类型不存在 |
| 401 Unauthorized | 未认证（没登录或凭证无效） | 缺少或无效的 Access Token、登录失败、Refresh Token 无效 |
| 403 Forbidden | 已认证但没有权限 | 普通用户调用管理员接口 |
| 404 Not Found | 资源不存在 | 电影不存在、未知路径 |
| 409 Conflict | 与现有数据冲突 | 邮箱已注册、`imdb_id` 已存在 |
| 429 Too Many Requests | 请求太频繁 | 登录/注册超过限流 |
| 500 Internal Server Error | 服务器内部错误 | 数据库故障等未预期的错误 |
| 502 Bad Gateway | 上游服务返回了不可用的结果 | LLM 的回答不是合法的评级名称 |

> **401 vs 403**：401 表示“我不知道你是谁”（凭证缺失或无效），403 表示“我知道你是谁，但你没有权限”。这个区分不只是语义问题：前端的 axios 拦截器在收到 **401** 时会尝试刷新 Token 并重试（第 13a 章）。如果权限不足也返回 401，前端会白白刷新一次 Token，重试后再收到 401，最后误以为会话失效，把用户踢回登录页。所以 `RequireRole` 返回的是 **403**。

---

## 9.2 路由：URL → Handler

### 9.2.1 完整路由表

```go
// S/routes/routes.go
// Register sets up every route. Middleware is attached to route groups, so what is protected depends only on which group a route is in,
// not on call order, and unknown paths get 404 rather than 401.
func Register(router *gin.Engine, tm *tokens.Manager, auth *controllers.AuthController, movies *controllers.MovieController) {
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Login and register are rate limited per IP (10 per minute on average, bursts of 10) to slow down password guessing.
	// refresh and logout need a valid refresh token and are not limited (every page load calls refresh).
	authLimit := middleware.RateLimit(rate.Every(6*time.Second), 10)
	authGroup := router.Group("/auth")
	authGroup.POST("/register", authLimit, auth.Register)
	authGroup.POST("/login", authLimit, auth.Login)
	authGroup.POST("/refresh", auth.Refresh)
	authGroup.POST("/logout", auth.Logout)

	// Public
	router.GET("/movies", movies.ListMovies)
	router.GET("/genres", movies.ListGenres)

	// Signed-in users
	protected := router.Group("/", middleware.Auth(tm))
	protected.GET("/movies/:imdb_id", movies.GetMovie)
	protected.GET("/me/recommendations", movies.Recommended)

	// Admins only
	admin := protected.Group("/", middleware.RequireRole(models.RoleAdmin))
	admin.POST("/movies", movies.CreateMovie)
	admin.PATCH("/movies/:imdb_id/review", movies.UpdateReview)
}
```

### 9.2.2 路由组（Route Group）

`router.Group(prefix, middlewares...)` 返回一个 `*gin.RouterGroup`。在组上注册的路由，路径会加上组的前缀，处理链会在前面加上组的中间件：

| 路由 | 处理链（不含 `main.go` 中的全局中间件） |
|---|---|
| `POST /auth/login` | `RateLimit` → `auth.Login` |
| `POST /auth/refresh` | `auth.Refresh` |
| `GET /movies` | `movies.ListMovies` |
| `GET /movies/:imdb_id` | `Auth` → `movies.GetMovie` |
| `POST /movies` | `Auth` → `RequireRole(ADMIN)` → `movies.CreateMovie` |

- **嵌套组**：`admin` 是在 `protected` 上创建的，所以继承了 `Auth`，再追加 `RequireRole`。先认证、再检查角色的顺序由结构保证。
- **单条路由的中间件**：`authGroup.POST("/login", authLimit, auth.Login)`——处理函数参数是可变参数，可以在 Handler 之前放任意个中间件。这里只给登录和注册加限流。
- **同一路径、不同方法**：`GET /movies`（公开）和 `POST /movies`（管理员）是两条独立的路由，各自有自己的处理链。

### 9.2.3 为什么不用 `router.Use(Auth)`？

`router.Use(m)` 把中间件加到**引擎的全局列表**中。Gin 在注册每条路由时，会把**当时**的全局列表复制一份，与 Handler 合并成该路由的处理链。这带来两个副作用：

1. **结果取决于调用顺序**：`Use(Auth)` 之前注册的路由不需要登录，之后注册的都需要。把两段注册代码调换一下位置，登录接口本身就需要 Token 了——用户永远无法登录。
2. **影响 404**：`engine.Use` 每次调用都会重建 404/405 的处理链，把全局中间件加进去。于是访问一个不存在的路径时，请求会先被 Auth 拦下，返回 **401** 而不是 404，让调用方误以为是认证问题。

路由组把中间件限定在组内，这两个问题都不存在：一条路由需要什么权限，只看它写在哪个组里；不存在的路径不属于任何组，直接得到 404。`routes_test.go` 中的 `TestAuthorization` 用表格测试锁定了这些行为。

> **全局中间件放哪里？** 对**所有**请求都应该生效的中间件（日志、panic 恢复、CORS、安全头、请求体大小限制）在 `main.go` 中用 `router.Use` 注册，并且**在注册任何路由之前**调用。第 12 章详解。

### 9.2.4 路径参数

`/movies/:imdb_id` 中的 `:imdb_id` 是**路径参数**，匹配一个路径段。Handler 中用 `c.Param("imdb_id")` 读取。

| 请求 | 匹配的路由 | `c.Param("imdb_id")` |
|---|---|---|
| `GET /movies` | `GET /movies` | —— |
| `GET /movies/tt0111161` | `GET /movies/:imdb_id` | `"tt0111161"` |
| `PATCH /movies/tt0111161/review` | `PATCH /movies/:imdb_id/review` | `"tt0111161"` |
| `GET /movies/tt0111161/extra` | 无 → 404 | —— |

路径参数不会是空字符串（空段不匹配这条路由），所以 Controller 中不需要检查 `== ""`。参数格式不对（例如 `/movies/abc`）时，查询自然找不到，返回 404。

### 9.2.5 RESTful URL 设计

URL 用**名词**表示资源，用 **HTTP 方法**表示动作：

| 操作 | URL | 说明 |
|---|---|---|
| 列出电影 | `GET /movies` | 集合 |
| 查看一部电影 | `GET /movies/:imdb_id` | 集合中的一个成员 |
| 添加电影 | `POST /movies` | 向集合中添加 |
| 更新影评 | `PATCH /movies/:imdb_id/review` | 子资源：一部电影的影评 |
| 我的推荐 | `GET /me/recommendations` | `me` 指代当前登录用户，不需要在 URL 中写用户 ID |
| 登录、刷新、登出 | `POST /auth/login` 等 | 动作型接口，统一放在 `/auth` 下 |

对比“动词式”URL（`/addmovie`、`/updatereview`、`/recommendedmovies`）：RESTful 风格的 URL 更可预测——知道了 `GET /movies/:id`，很自然能猜到 `PATCH /movies/:id/...` 的存在；文档、权限规则和缓存策略也可以按资源组织。把认证接口放在 `/auth` 前缀下还有一个实际好处：Refresh Token Cookie 可以设置 `Path=/auth`，只随这几个请求发送（第 11 章）。

---

## 9.3 核心模式：Controller 结构体 + 方法值

### 9.3.1 What

```go
// S/controllers/movie_controller.go
type MovieController struct {
	movies   *services.MovieService
	validate *validator.Validate
	timeout  time.Duration
	// reviewTimeout is longer than a normal request because writing a review waits for the LLM.
	reviewTimeout time.Duration
}

func NewMovieController(movies *services.MovieService, v *validator.Validate, timeout, reviewTimeout time.Duration) *MovieController {
	return &MovieController{movies: movies, validate: v, timeout: timeout, reviewTimeout: reviewTimeout}
}

// ListMovies handles GET /movies.
func (h *MovieController) ListMovies(c *gin.Context) { ... }
```

注册路由时：

```go
router.GET("/movies", movies.ListMovies)
```

- `movies.ListMovies` 是一个**方法值**（method value）：它把方法和接收者 `movies` 绑定在一起，得到一个签名为 `func(*gin.Context)` 的普通函数，正好满足 `gin.HandlerFunc`。
- 每次请求时，Gin 调用这个函数，函数内部通过 `h` 访问依赖。

### 9.3.2 Why：这就是依赖注入

**Dependency Injection（依赖注入）**：一个组件需要的依赖由**外部传入**，而不是自己创建或从全局变量读取。依赖链完全由 `main` 组装：

```go
// S/main.go（节选）
movieService := services.NewMovieService(repository.NewMovieRepository(db), catalogRepo, userRepo,
	classifier, cfg.RecommendedMovieLimit, cfg.OpenAITimeout)
movieController := controllers.NewMovieController(movieService, validate,
	cfg.RequestTimeout, cfg.RequestTimeout+cfg.OpenAITimeout)
routes.Register(router, tokenManager, authController, movieController)
```

**如果不这样做**，常见的替代写法是全局变量：

```go
var Client *mongo.Client   // 全局变量

func GetMovies(c *gin.Context) {
	coll := Client.Database(os.Getenv("DATABASE_NAME")).Collection("movies")
	// ...
}
```

全局变量的问题：
- **测试困难**：无法为不同测试传入不同的（假的）依赖。本项目的 `routes_test.go` 用内存假实现组装出一个完整的应用，就是靠构造函数注入。
- **隐式依赖**：看函数签名看不出它依赖数据库、配置、超时。
- **初始化顺序问题**：如果某处在 `Client` 被赋值前就用了它，就会 panic。

### 9.3.3 另一种写法：Handler 工厂函数 + 闭包

```go
func ListMovies(svc *services.MovieService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 闭包“记住”了外层的 svc
	}
}

router.GET("/movies", ListMovies(movieService))
```

工厂函数在启动时执行一次，返回的内部函数在每次请求时执行。内部函数引用了外层变量 `svc`——这就是**闭包（Closure）**。

| | 结构体 + 方法值（本项目） | 工厂函数 + 闭包 |
|---|---|---|
| 依赖多时 | 构造一次，所有方法共享 | 每个工厂函数都要列出参数 |
| 相关 Handler 的组织 | 天然分组（`AuthController`、`MovieController`） | 只是一组独立函数 |
| 适合 | 依赖较多、Handler 较多 | 只有一两个依赖的小型 Handler |

两种都是依赖注入，选择哪种主要看依赖的数量。本项目的 `/health` 是一个直接写在路由表里的匿名函数，因为它没有任何依赖。

---

## 9.4 `gin.Context` 常用方法（本项目用到的全部）

| 方法 | 作用 | 示例位置 |
|---|---|---|
| `c.Param(name)` | 读路径参数 | `GetMovie`、`UpdateReview` |
| `c.ShouldBindJSON(&v)` | 把请求体 JSON 解析到 `v`；失败返回 error | `bindJSON` |
| `c.JSON(code, obj)` | 写 JSON 响应 | 到处 |
| `c.AbortWithStatusJSON(code, obj)` | 写 JSON 响应并中止后续处理链 | `Auth`、`RequireRole`、`RateLimit` |
| `c.Cookie(name)` | 读 Cookie | `AuthController.Refresh`、`Logout` |
| `c.Writer` | 底层 `http.ResponseWriter` | `setRefreshCookie`、`SecurityHeaders` |
| `c.GetHeader(k)` | 读请求头 | `Auth` |
| `c.Request.Context()` | 本次请求的 `context.Context` | 每个 Controller 方法 |
| `c.Set(k, v)` / `c.GetString(k)` | 在本次请求的处理链中传值 | `Auth` 写入，`middleware.UserID(c)` 读取 |
| `c.ClientIP()` | 客户端 IP（考虑可信代理） | `RateLimit` |
| `c.FullPath()` | 匹配到的路由模板，如 `/movies/:imdb_id` | `respondError` 的日志 |
| `c.Next()` | 执行处理链中的下一个处理函数 | 所有中间件 |

`gin.H` 是 `map[string]any` 的类型别名，用来快速构造 JSON 对象：`gin.H{"error": "x"}` → `{"error":"x"}`。

### `ShouldBindJSON` vs `BindJSON`
- `ShouldBindJSON`：出错时**只返回 error**，由你决定如何响应。✅ 本项目使用。
- `BindJSON`：出错时**自动**以 400 中止请求，你无法自定义错误格式。

---

## 9.5 Controller 的公共部分：`response.go`

每个 Controller 方法都要做两件重复的事：解析和校验请求体、把错误转换成 HTTP 响应。这两件事集中在 `controllers/response.go` 中。

### 9.5.1 `bindJSON`：解析 + 校验

```go
// S/controllers/response.go
// bindJSON parses and validates the request body. On failure it has already written a 400 response and the caller just returns.
func bindJSON(c *gin.Context, v *validator.Validate, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return false
	}
	if err := v.Struct(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed", "details": validation.Describe(err)})
		return false
	}
	return true
}
```

调用方：

```go
var req models.MovieCreate
if !bindJSON(c, h.validate, &req) {
	return
}
```

- `if err := ...; err != nil { }` 是 Go 的 **if 初始化语句**：`err` 的作用域只在这个 `if` 块内。
- 返回 `bool` 而不是 `error`：失败时响应已经写好了，调用方唯一要做的就是 `return`。这让“忘记写 return”的机会只剩一处，而且很显眼。
- `dst any`：`any` 是 `interface{}` 的别名，任何类型都可以传入。`ShouldBindJSON` 和 `v.Struct` 内部都通过反射处理它。

### 9.5.2 `respondError`：错误 → 状态码

```go
// S/controllers/response.go
// Business errors mapped to HTTP status codes. Any error not listed here is a 500.
var errorStatus = map[error]int{
	services.ErrInvalidCredentials:  http.StatusUnauthorized,
	services.ErrInvalidRefreshToken: http.StatusUnauthorized,
	services.ErrEmailTaken:          http.StatusConflict,
	services.ErrMovieExists:         http.StatusConflict,
	services.ErrMovieNotFound:       http.StatusNotFound,
	services.ErrPasswordTooLong:     http.StatusBadRequest,
	services.ErrUnknownGenre:        http.StatusBadRequest,
	services.ErrUnrecognizedRanking: http.StatusBadGateway,
}

// respondError writes an error response in the shared format: {"error": "..."}.
// Internal errors are logged in full; the client only sees a generic message.
func respondError(c *gin.Context, err error) {
	for target, status := range errorStatus {
		if errors.Is(err, target) {
			c.JSON(status, gin.H{"error": target.Error()})
			return
		}
	}
	log.Printf("internal error on %s %s: %v", c.Request.Method, c.FullPath(), err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
```

- **services 不知道 HTTP**：它只返回 `ErrMovieNotFound` 这样的业务错误。“找不到对应 404”是 HTTP 层的知识，所以映射表放在 controllers 里。
- **`errors.Is` 而不是 `==`**：service 可能用 `fmt.Errorf("...: %w", err)` 包装过错误，`errors.Is` 会沿着包装链查找。
- **业务错误的消息可以给客户端看**（例如 `email is already registered`），所以响应中写的是 `target.Error()`。
- **其他错误一律 500**，完整的错误只写进服务器日志，客户端只看到 `internal server error`。数据库错误信息中可能包含集合名、查询条件、内部地址，不应该暴露给外部（第 13 章 13.6）。

### 9.5.3 统一的错误格式

所有错误响应都是：

```json
{"error": "movie not found"}
```

校验失败时多一个 `details` 数组。中间件（`Auth`、`RequireRole`、`RateLimit`）也使用同样的 `"error"` 键。前端只需要一种方式读取错误：`err.response?.data?.error`。

---

## 9.6 逐个讲解 Controller

### 9.6.1 `ListMovies` —— 查询全部电影

```go
// S/controllers/movie_controller.go
// ListMovies handles GET /movies.
func (h *MovieController) ListMovies(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)   // ① 超时
	defer cancel()

	movies, err := h.movies.List(ctx)                                     // ② 调用 service
	if err != nil {
		respondError(c, err)                                              // ③ 错误 → 响应
		return                                                            // ④ 立即返回
	}
	c.JSON(http.StatusOK, movies)                                         // ⑤ 成功
}
```

几乎所有 Controller 方法都是这五步。**④ 的 `return` 至关重要**：`c.JSON` 只是写入响应，**不会**终止函数。如果漏掉 `return`，函数会继续往下执行，用无效的数据再写一次响应——Gin 会打印 `[WARNING] Headers were already written`，客户端收到的状态码是第一次写的，响应体却是两段 JSON 拼接在一起，前端无法解析；更糟的情况是继续使用 nil 值导致 panic。这是 Go Web 开发中最常见的 bug 之一，`go vet` 无法发现它，只能靠代码结构（每个错误分支都以 `return` 结尾）和测试来防止。

**示例**
```bash
curl http://localhost:8080/movies
# [{"id":"6650...","imdb_id":"tt0060196","title":"The Good, The Bad, and the Ugly",...}]
```

### 9.6.2 `GetMovie` —— 查询单部电影

```go
// GetMovie handles GET /movies/:imdb_id.
func (h *MovieController) GetMovie(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	movie, err := h.movies.Get(ctx, c.Param("imdb_id"))
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, movie)
}
```

错误如何变成状态码：
- 电影不存在：repository 返回 `ErrNotFound` → service 转换为 `ErrMovieNotFound` → **404**。
- 数据库故障：repository 原样返回驱动错误 → service 原样返回 → `respondError` 不认识 → **500**，并记录日志。

区分这两种情况很重要：如果把所有错误都当作 404，数据库宕机时用户只会看到“电影不存在”，监控也不会报警。

### 9.6.3 `CreateMovie` —— 添加电影

```go
// CreateMovie handles POST /movies (admins only).
func (h *MovieController) CreateMovie(c *gin.Context) {
	var req models.MovieCreate
	if !bindJSON(c, h.validate, &req) {             // ① 解析 + 校验
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	movie, err := h.movies.Create(ctx, req)         // ② 查类型、设初始评级、插入
	if err != nil {
		respondError(c, err)                        // 类型不存在 400；imdb_id 重复 409
		return
	}
	c.JSON(http.StatusCreated, movie)               // ③ 返回完整的新电影，含生成的 id
}
```

- 这个方法**没有检查角色**——它也不需要。路由把它放在 `admin` 组里，请求到达这里时 `RequireRole` 已经确认过了。授权规则集中在路由表，Controller 只关心“怎么做”。
- 返回的是完整的 `Movie`（包括服务器填写的类型名称、“未评级”评级和新生成的 `id`），客户端不需要再查一次。

### 9.6.4 `UpdateReview`、`Recommended`、`ListGenres`

结构与上面相同。值得注意的差别：
- `UpdateReview` 使用 `h.reviewTimeout`（普通超时 + OpenAI 超时），因为要等待 LLM。
- `Recommended` 用 `middleware.UserID(c)` 读取当前用户的 ID——这个值来自已验证的 Token，而不是请求参数，用户无法查看别人的推荐。

具体的业务逻辑在第 10 章。

### 9.6.5 `AuthController.Register` —— 注册

```go
// S/controllers/user_controller.go
// Register handles POST /auth/register. A successful registration also signs the user in and returns the same body as Login.
func (h *AuthController) Register(c *gin.Context) {
	var req models.UserRegister                     // 注册专用 DTO，不含 Role
	if !bindJSON(c, h.validate, &req) {
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), h.timeout)
	defer cancel()

	session, err := h.auth.Register(ctx, req)       // 校验类型、哈希密码、插入、签发 Token
	if err != nil {
		respondError(c, err)                        // 邮箱已注册 409；类型不存在 400
		return
	}
	h.writeSession(c, http.StatusCreated, session)  // Set-Cookie + UserResponse
}
```

- 注册成功即登录：返回与登录相同的 `UserResponse`，并设置 Refresh Token Cookie。前端不需要让用户再输入一次密码。
- `AuthService.Register` 内部做了什么（为什么不先查询邮箱是否存在、为什么要按字节检查密码长度）见第 07、11 章。

### 9.6.6 其余 Controller

- `Login`、`Refresh`、`Logout`、`writeSession`、`setRefreshCookie` → 第 11 章（认证）。
- `MovieService` 中的推荐和影评评级逻辑 → 第 10 章（业务逻辑）。

---

## 9.7 如何从零写一个 Controller（步骤模板）

以“`GET /genres`”为例：

1. **确定契约**：方法 `GET`，路径 `/genres`，公开，返回 `[]Genre`。
2. **repository**：`CatalogRepository.Genres(ctx)` 执行查询，返回 `[]models.Genre`（没有结果时返回空切片）。
3. **service**：`MovieService.Genres(ctx)` 直接转发。看起来多余，但保持了“controller 只和 service 打交道”的规则；将来要加缓存，就加在这里。
4. **controller**：
   1. 创建带超时的 Context，`defer cancel()`。
   2. 调用 service。
   3. **错误时 `respondError` 并 `return`**。
   4. 写成功响应。
5. **注册路由**：决定放在哪个组（这里是公开路由）。
6. **测试**：在 `routes_test.go` 的表格中加一行；手动验证用 `curl -i http://localhost:8080/genres`（`-i` 显示响应头和状态码）。

---

## 本章练习

> 答案见 [第 20 章](20-exercise-solutions.md#第-09-章)

**关键概念回顾**
- 什么是方法值？`movies.ListMovies` 在什么时候被求值，又在什么时候被调用？
- 为什么 `c.JSON` 之后通常要 `return`？
- 为什么中间件挂在路由组上，比用 `router.Use` 更可靠？

**面试题 9.1** “Gin 中 `ShouldBindJSON` 和 `BindJSON` 有什么区别？”

**面试题 9.2** “解释 401 和 403 的区别。在一个会自动刷新 Token 的前端里，用错会造成什么后果？”

**面试题 9.3** “什么是依赖注入？你的项目是如何实现的？”

**练习 9.4** 不看源码，在你的重建项目中用路由组写出 `routes.Register`，然后验证：(a) 公开路由不需要 Token；(b) 普通用户访问管理员接口得到 403；(c) 访问不存在的路径得到 404。

**练习 9.5** 在副本中删掉 `ListMovies` 错误分支里的 `return`，然后让 `MovieService.List` 返回一个错误（例如停掉 MongoDB），观察响应和服务器日志。再写一个测试（提示：给 `memstore.Movies` 加一个可以注入错误的字段），确保这种错误被测试发现。

**练习 9.6** 不看源码，写出 `respondError`。然后新增一个业务错误 `ErrReviewTooShort`，让它映射到 422 Unprocessable Entity。

**调试挑战 9.7** 在副本中把 `routes.Register` 的开头改为 `router.Use(middleware.Auth(tm))`，并把 `protected` 组改成普通的 `router`。然后尝试登录、访问 `/movies` 和一个不存在的路径。会发生什么？给出完整的解释。

**调试挑战 9.8** 执行 `curl -i http://localhost:8080/does-not-exist`，观察状态码和响应体。为什么是 404 而不是 401？如果在 `main.go` 中用 `router.Use` 注册一个总是返回 401 的中间件，结果会变吗？

**理解检验 9.9** `curl -i -H "Origin: http://localhost:5173" http://localhost:8080/health` 的响应中有 `Access-Control-Allow-Origin` 吗？如果把 `/health` 的注册移到 `main.go` 中 `router.Use(...)` 之前，结果会怎样？为什么？
