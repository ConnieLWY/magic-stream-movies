# 第 20 章　练习参考答案

> **请先独立完成练习，再看答案。** 很多题目没有唯一正确答案，这里给出的是参考思路。凡是涉及代码行为的答案，都是根据提交 `d56a038` 的实际源码推理得出的。

---

## 第 00 章

**关键概念**：理解、解释、重建、批判。审视代码能力决定了你能否发现 bug、做出正确的设计权衡，并在代码评审中贡献价值。

**0.1** `Server/MagicStreamMoviesServer/`：`cmd`、`config`、`controllers`、`database`、`internal`、`middleware`、`models`、`repository`、`routes`、`services`、`tokens`、`validation`。`Client/magic-stream-client/`：`public`、`src`（以及安装依赖后出现的 `node_modules`、构建后出现的 `dist`）。

**0.2** 不能。浏览器只与 Go 后端通信。原因：(1) 数据库凭证不能出现在浏览器中，否则任何用户都能拿到；(2) 业务规则（谁能看什么、谁能改什么）必须在服务器端强制执行；(3) MongoDB 协议不是 HTTP，浏览器本来也无法直接连接。

---

## 第 01 章

**关键概念**：访客、`USER`（注册产生，角色由服务器固定）、`ADMIN`（由种子脚本 `-admin` 参数提升）。项目只是嵌入 YouTube 预告片，服务器不存储也不传输视频。

**1.1** “用户先正常注册，注册接口总是创建 `USER`；然后由有数据库权限的人运行 `go run ./cmd/seed -admin <邮箱>` 把他提升为 `ADMIN`，他重新登录后拿到带新角色的 Token。注册时不能自选角色，是因为请求体由客户端完全控制——只要绕过前端直接调用 API，就能填任何值。所以注册请求绑定的是 `UserRegister` DTO，它根本没有 `role` 字段，角色由服务器写死。” 可以补充：如果需要在线管理，可以再做一个只有现有管理员能调用的“提升权限”接口。

**1.2** 公开：`GET /health`（内联函数）、`GET /movies`→`ListMovies`、`GET /genres`→`ListGenres`。认证：`POST /auth/register`→`Register`（限流）、`POST /auth/login`→`Login`（限流）、`POST /auth/refresh`→`Refresh`、`POST /auth/logout`→`Logout`。登录用户（经过 `Auth`）：`GET /movies/:imdb_id`→`GetMovie`、`GET /me/recommendations`→`Recommended`。管理员（经过 `Auth` + `RequireRole("ADMIN")`）：`POST /movies`→`CreateMovie`、`PATCH /movies/:imdb_id/review`→`UpdateReview`。所有路由还会经过 `main.go` 中的全局中间件（Logger、Recovery、CORS、SecurityHeaders、BodyLimit）。

**1.3** Cookie：第 1 步（`/auth/refresh`）、第 2 步（注册时 `Set-Cookie`）、第 3 步（登录时 `Set-Cookie`）、第 5 步（自动刷新）、第 6 步（刷新页面恢复）、第 7 步（登出）。`Authorization` Header：第 4 步（推荐）、第 5 步（原请求和重试）。

**1.4** 没有调用后端，是前端在已加载的 `movies` 数组上做 `filter`。10 万部电影时：首页要一次性下载所有电影（几十 MB JSON），渲染 10 万张卡片会让浏览器卡死。应改为后端分页 + 按类型查询（练习 19.18、19.20）。

---

## 第 02 章

**关键概念**：前后端分离带来 CORS 和认证复杂度。Go 禁止循环导入，因为包的初始化顺序必须是确定的有向无环图；本项目的分层结构让依赖始终向下（controllers → services → repository → models），`models`、`tokens`、`config` 等处于最底层，不依赖其他项目包。

**2.1** RT 在 HttpOnly Cookie（JS 读不到，XSS 偷不走；`Path=/auth` 让它只随认证请求发送）；AT 在内存（不会被浏览器自动发送，所以不受 CSRF 影响；刷新页面后用 RT 恢复）。localStorage 可以被任何在页面中运行的 JS 读取，一旦发生 XSS，两个 Token 都会被偷走，而且长期有效。

**2.2** 后端 API 基本不用改。主要差异在认证：App 没有浏览器的 Cookie 机制，通常把 RT 存在系统的安全存储（iOS Keychain / Android Keystore）中，并通过请求体或 Header 发送给 `/auth/refresh`。所以需要让 `AuthController.Refresh` 也支持从 Header/Body 读取 RT——`AuthService.Refresh` 本身只接收一个字符串，完全不用改，这正是分层的好处。CORS 对原生 App 无影响。

**2.4** 按现有分层：`models` 中加 `Review` 结构；`repository` 中加 `ReviewRepository`（依赖 `models`）；`services` 中加 `ReviewService`，并定义它需要的 `ReviewStore` 接口；`controllers` 中加 `ReviewController`（依赖 `services`、`middleware`）；`routes` 注册路由；`main` 负责组装。`repository` 和 `services` 都不能依赖 `controllers`，否则会形成循环。

**2.5** 编译报错：`import cycle not allowed`，并列出循环链 `services → repository → services`。正确做法是把共享的东西放在更底层的包里（本项目的 `repository.ErrNotFound` 就定义在 `repository` 中，由 `services` 引用）。

**2.6** `UserRepository.FavouriteGenreNames` 返回 `repository.ErrNotFound`；`MovieService.Recommended` 显式忽略这个错误，得到空的类型列表，然后直接返回 `[]models.Movie{}`（不再查询 movies 集合），最终是 **200 `[]`**。如果想让“用户已被删除”返回 401，可以在这里改为返回一个新的业务错误，并在 `controllers/response.go` 的映射表中登记。

---

## 第 03 章

**关键概念**：编译型语言先编译成机器码再运行（Go），解释型语言由解释器逐行执行。前端的开发工具（Vite、tsc、ESLint）都运行在 Node.js 上。

**3.3** 报错 `go: go.mod file not found in current directory or any parent directory`。Go 从当前目录向上查找 `go.mod`；`Server/` 和仓库根目录都没有。即使编译成功，程序也会因为在错误的工作目录下运行而找不到 `.env`（代码使用相对路径 `".env"`）。

**3.4** 完全不填 `OPENAI_API_KEY`：服务器根本不会启动，`config.Load()` 报告 `OPENAI_API_KEY is required`。填一个无效的占位值：除 `PATCH /movies/:imdb_id/review` 之外的接口都正常；这个接口在 `OpenAIClassifier.Classify` 中收到 OpenAI 的 401 错误，`MovieService.UpdateReview` 把它包装后返回，`respondError` 判断它不是业务错误，记录日志并返回 500 `internal server error`。电影不存在时则在调用 OpenAI 之前就返回 404。

---

## 第 04 章

**关键概念**：`go.mod` 声明 module path、Go 版本和依赖版本；`go.sum` 记录每个依赖的加密哈希，防止篡改。Vite 只暴露 `VITE_` 前缀的变量，防止误把服务器密钥打包进前端。

**4.1** `go get` 添加或升级指定的依赖；`go mod tidy` 根据代码中的实际 `import` 同步 `go.mod`/`go.sum`：添加缺失的、删除未使用的、修正 `// indirect` 标记。

**4.2** 不可以。前端代码会被下载到用户的浏览器中，任何人都能在 DevTools 或打包后的 JS 中看到它。需要密钥的调用必须由后端代理完成——本项目调用 OpenAI 就是这样做的。

**4.4** `go get` 之后 gin 被标为 `// indirect`（因为还没有代码 import 它）或者已在 require 中；写了 `import "github.com/gin-gonic/gin"` 再执行 `go mod tidy` 后，gin 会出现在第一个 `require` 块中且没有 `// indirect`，它自己的依赖出现在第二个块中并带有 `// indirect`。

**4.5** 服务器不会启动，打印类似：

```text
DATABASE_NAME is required
SECRET_KEY must be at least 32 bytes, got 5
```

`config.Load()` 用 `errors.Join` 收集所有问题后一起返回。如果遇到第一个就返回，你修好一项、重启、再看到下一项……配置项多的时候非常浪费时间。一次列全，一次改完。

**4.6** 能成功。golang-jwt 的 HMAC 签名不会拒绝空密钥（编写教材时查阅过 v5.3.1 的源码确认），签发和校验都用同一个空密钥，所以一切“正常”。危险在于：攻击者知道（或猜到）密钥为空，就能自己签发任意用户、任意角色的 Token。这就是 `config.Load()` 要求密钥至少 32 字节的原因——错误的配置应该让程序**启动失败**，而不是悄悄以不安全的方式运行。

---

## 第 05 章

**关键概念**：同一目录下的所有文件属于同一个包，共享命名空间。小写开头的字段对 `encoding/json` 和 mongo-driver 不可见。

**5.1** Go 的可见性是**包级别**的：首字母小写的标识符在整个包内（所有文件）都可见，在包外不可见。没有“文件私有”的概念。

**5.2** `internal/` 下的包只能被 `internal` 的父目录及其子目录中的代码导入。这是编译器强制的，用来防止外部项目依赖你的内部实现。

**5.3** **编译失败**。`services/auth_service.go` 的 `Register` 中引用了 `req.FirstName`，会报 `req.FirstName undefined (type models.UserRegister has no field or method FirstName, but does have field firstName)`。如果把引用也一起改掉，编译能通过，但 `firstName` 是未导出字段：`ShouldBindJSON` 不会给它赋值，validator 也会跳过它（`required,min=2` 失效）——注册的用户名字为空字符串，而且不会有任何报错。

**5.5** `bindJSON redeclared in this block`，并指出 `response.go` 中另一个声明的位置。同一个包的所有文件共享包级作用域，未导出的名字也不例外。

**5.6** `accessKey` 不能（小写，未导出字段）。`tokens.Hash` 可以（大写，已导出函数）。`tokens.Claims` 的 `UserID` 可以（导出类型的导出字段）。

---

## 第 06 章

**关键概念**：密码哈希需要“慢”，SHA-256 太快。`bson.M` 是无序 map，`bson.D` 是有序切片；排序、需要保证键顺序的命令必须用 `bson.D`。

**6.1** `^1.2.3`：允许 `>=1.2.3 <2.0.0`（不改变主版本号）。`~1.2.3`：允许 `>=1.2.3 <1.3.0`（只允许补丁版本）。对于 `0.x` 版本，`^0.2.3` 等价于 `>=0.2.3 <0.3.0`。

**6.2** Gin 用 `sync.Pool` 复用 `*gin.Context`。Handler 返回后，这个 Context 会被重置并分配给下一个请求。在 goroutine 中继续使用它，可能读到别的请求的数据，或者与新请求发生数据竞争。需要时使用 `cc := c.Copy()`。

**6.5**

```go
func classifyReview(ctx context.Context, apiKey, model, review string, labels []string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "Reply with exactly one of: " + strings.Join(labels, ", ")},
			{"role": "user", "content": review},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("openai: status %d", resp.StatusCode)
	}

	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("openai: empty choices")
	}
	answer := strings.TrimSpace(strings.Trim(out.Choices[0].Message.Content, ".\"'"))
	for _, l := range labels {
		if strings.EqualFold(l, answer) {
			return l, nil
		}
	}
	return "", fmt.Errorf("unexpected label %q", answer)
}
```

**6.6** 需要改的只有直接导入 mongo-driver 的包：
- 所有导入路径去掉 `/v2`（`go.mongodb.org/mongo-driver/mongo` 等）。
- `database`：`mongo.Connect(opts)` 改为 `mongo.Connect(ctx, opts)`。
- `models`、`repository`、`internal/memstore`：`bson.ObjectID` / `bson.NewObjectID()` / `bson.ObjectIDFromHex` 改为 `primitive` 包中的同名函数和类型。
- `repository`：`findAll` 的选项参数类型 `options.Lister[options.FindOptions]` 改为 v1 的 `*options.FindOptions`。
- `cmd/seed`：同样的导入路径和 `ObjectID` 修改。

`services` 和 `controllers` 不用改，因为它们根本不导入 mongo-driver：`services` 只认识 `models` 中的结构体和 `repository.ErrNotFound` 这样的错误值。这正是 repository 层存在的意义——把“用的是哪个数据库驱动”这件事封装起来。（`models` 中的 `bson.ObjectID` 是一个小小的泄漏：如果想让 `models` 也完全不依赖驱动，可以把 ID 存成字符串，在 repository 中转换。）

---

## 第 07 章

**关键概念**：反范式化 = 冗余存储数据以换取读取性能。本项目在 `users.favourite_genres`、`movies.genre`、`movies.ranking` 三处使用，副本只由服务器从字典集合中复制。`mongo.Connect` 不会立即建立连接，所以 `database.Connect` 紧接着调用 `Ping`。

**7.1** 先检查、后执行，两步之间状态可能被其他请求改变。本项目的做法：在 `email` 上创建唯一索引，注册时不先查询，直接 `InsertOne`；repository 把 `mongo.IsDuplicateKeyError` 翻译成 `ErrDuplicate`，service 转成 `ErrEmailTaken`，controller 返回 409。

**7.2** 嵌入：数据总是一起读取、子数据量小且有界、很少独立修改（电影的类型标签）。引用：子数据量大或无界（一部电影的所有用户评论）、需要独立查询或频繁修改、多处共享且需要强一致。

**7.3** 第一次运行日志中 `movies: 17 documents`；第二次同样是 17，`db.movies.countDocuments()` 也是 17。因为 `upsertAll` 按 `imdb_id` 执行 `ReplaceOne(..., SetUpsert(true))`：存在就替换，不存在才插入。

**7.4** 创建索引前 `winningPlan.stage` 为 `COLLSCAN`，`totalDocsExamined` 等于集合文档数；创建后为 `IXSCAN`（外层是 `FETCH`），`totalDocsExamined` 为 1。

**7.5**

```go
func Connect(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect to mongodb: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("ping mongodb: %w", err)
	}
	return client, nil
}

var ErrNotFound = errors.New("not found")

func translate(err error) error {
	if errors.Is(err, mongo.ErrNoDocuments) {
		return ErrNotFound
	}
	return err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.coll.FindOne(ctx, bson.M{"email": email}).Decode(&user); err != nil {
		return nil, translate(err)
	}
	return &user, nil
}
```

要点：错误用 `%w` 包装；`translate` 只转换认识的错误，其余原样返回；`FindOne(...).Decode` 在没有结果时返回 `mongo.ErrNoDocuments`。

**7.6** 正常版本立即退出并打印 `connect to mongodb: error parsing uri: scheme must be "mongodb" or "mongodb+srv"`——一眼就能看出是 URI 写错了。吞掉错误的版本中，`main` 拿到 `client == nil` 且 `err == nil`，接着调用 `client.Database(...)` 时崩溃：`panic: runtime error: invalid memory address or nil pointer dereference`。**报错没有告诉你真正的原因**，这就是吞掉错误的代价。

**7.7** 第一次：400，`{"error":"validation failed","details":["favourite_genre_ids: min=1"]}`。删掉 `min=1` 后注册成功；请求推荐得到 200 `[]`，因为 `MovieService.Recommended` 看到类型列表为空就直接返回空切片，根本不查询 movies。再删掉那个判断、并改用 `var names []string`：`genreNames` 是 nil 切片，被编码成 BSON 的 `null`，查询条件变成 `{"$in": null}`，MongoDB 报错 `$in needs an array`，接口返回 500。两层防御（校验 + 空列表短路）加上 `make([]string, 0, n)`，任何一层都足以避免这个问题。

**7.8** `MovieRepository.UpdateReview` 返回 `repository.ErrNotFound` → `MovieService.UpdateReview` 用 `errors.Is` 识别后返回 `services.ErrMovieNotFound` → `MovieController.UpdateReview` 调用 `respondError` → `errorStatus` 映射表把 `ErrMovieNotFound` 对应到 404。（实际上 service 在调用 LLM 之前已经用 `Get` 检查过一次，所以通常在那里就返回了；`UpdateReview` 中的检查用于处理“检查之后、更新之前电影被删除”的情况。）是 404 是因为查询本身成功执行了，只是没有文档匹配这个 `imdb_id`——这是客户端提供了不存在的资源 ID；500 用于服务器自身出错（例如数据库连接失败）。

---

## 第 08 章

**关键概念**：struct tag 由各个库通过反射读取；格式错误会静默失效。`ShouldBindJSON` 需要写入你的变量，所以要传指针。`omitempty` 让零值 `_id` 不被写入，由 MongoDB 生成。

**8.1** 客户端提交了服务器本不允许它设置的字段，而服务器把整个请求体直接映射到内部模型上。本项目中最敏感的字段是 `role`。本项目的做法：(1) 每个接口都有只包含允许字段的请求 DTO——注册用 `UserRegister`，其中没有 `Role`，由 `AuthService.Register` 设置为 `USER`；添加电影用 `MovieCreate`，其中没有 `Ranking`；(2) 数据库文档 `User` 的所有字段都是 `json:"-"`，即使误用也无法从 JSON 写入或读出；(3) 有回归测试（`TestRegisterAlwaysCreatesUserRole`、`TestRegisterIgnoresClientRoleAndSetsScopedCookie`）提交 `"role":"ADMIN"` 并断言结果是 `USER`。

**8.2** nil 切片编码为 `null`，空切片 `[]T{}` 编码为 `[]`。BSON 同理（默认情况下）。

**8.3** Go 的 interface 是**隐式**实现的：只要类型拥有接口要求的所有方法，就自动满足该接口，不需要声明。Java 需要显式写 `implements`。Go 的方式让你可以为别人写的类型定义接口（例如为 `*mongo.Collection` 定义一个只包含你用到的方法的接口，便于测试）。

**8.4**

```go
type UserRegister struct {
	FirstName         string `json:"first_name" validate:"required,min=2,max=100"`
	LastName          string `json:"last_name" validate:"required,min=2,max=100"`
	Email             string `json:"email" validate:"required,email,max=254"`
	Password          string `json:"password" validate:"required,min=8,max=72"`
	FavouriteGenreIDs []int  `json:"favourite_genre_ids" validate:"required,min=1,max=20,unique,dive,min=1"`
}

func (s *AuthService) Register(ctx context.Context, req models.UserRegister) (*Session, error) {
	genres, err := s.genres.GenresByIDs(ctx, req.FavouriteGenreIDs)
	if err != nil {
		return nil, err
	}
	if len(genres) != len(req.FavouriteGenreIDs) {
		return nil, ErrUnknownGenre
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	user := &models.User{
		FirstName: strings.TrimSpace(req.FirstName), LastName: strings.TrimSpace(req.LastName),
		Email: NormalizeEmail(req.Email), PasswordHash: string(hash),
		Role: models.RoleUser, CreatedAt: now, UpdatedAt: now, FavouriteGenres: genres,
	}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, ErrEmailTaken
		}
		return nil, err
	}
	return s.startSession(ctx, user)
}
```

对照源码检查：你有没有处理超过 72 字节的密码？有没有用唯一索引而不是“先查询再插入”？

**8.5** 载荷字段：`uid`、`role`（来自 `tokens.Claims` 的 json tag），以及 `iss`、`sub`、`exp`、`iat`、`jti`（来自 `RegisteredClaims`）。`jti` 是每个 Token 的随机 ID（第 11 章）。载荷中没有姓名和邮箱。

**8.6** `go vet` 报告 `struct field tag ... not compatible with reflect.StructTag.Get: bad syntax for struct tag value`。`reflect.StructTag.Get("json")` 遇到格式错误返回空字符串，`encoding/json` 于是使用字段名，JSON 中的键变为 **`Title`**。（`bson` 在前面且格式正确，仍然能被读到，所以数据库读写不受影响。）前端读 `movie.title` 得到 `undefined`。第二部分：响应变成 `{"password":"$2a$10$..."}`——只有那一个字段出现，其他仍是 `json:"-"`。这说明 `json:"-"` 是一道有效的保险，也说明为什么不能把数据库文档直接返回给前端。

**8.7** **不会**有编译错误——TypeScript 类型只是你对数据形状的声明，不会在运行时检查。运行时 `movie.poster_path` 为 `undefined`：`<img src>` 没有有效地址，海报全部显示为空，Hero 背景变成 `url(undefined)`。

---

## 第 09 章

**关键概念**：方法值 `movies.ListMovies` 在注册路由时求值一次（绑定接收者 `movies`），得到的函数在每个请求时被调用。`c.JSON` 只写响应，不终止函数。路由组把中间件限定在组内，保护范围只取决于路由写在哪个组，与调用顺序无关，也不会影响 404。

**9.1** `ShouldBindJSON` 只返回错误，由你决定如何响应；`BindJSON` 出错时自动调用 `c.AbortWithError(400, err)`，以 400 中断请求（响应体为空，错误被记录在 `c.Errors` 中），你无法再写入自定义格式的错误响应。

**9.2** 401：未认证（没有或无效的凭证）；403：已认证但无权限。后果：本项目的 axios 拦截器收到 401 会调用 `/auth/refresh` 再重试原请求。如果权限不足也返回 401，前端会白白刷新一次（旧 Refresh Token 作废、签发新的），重试后仍是 401，于是拦截器认为会话失效，可能清空登录状态把用户踢回登录页——用户只是点了一个没权限的按钮，却被登出了。

**9.3** 依赖由外部传入，而不是在内部创建或从全局读取。本项目中，`main` 依次创建 repository → service → controller，每一层都通过构造函数（`NewAuthService(users, catalog, tm)`、`NewMovieController(movieService, validate, ...)`）接收下一层；services 依赖的是接口，所以测试中可以传入 `internal/memstore` 的内存实现。

**9.4** 参考 9.2.1 节的代码。验证可以直接复用 `routes_test.go` 的 `TestAuthorization` 写法：用 `httptest.NewRecorder` 和 `router.ServeHTTP`，不需要真的启动服务器。

**9.5** 删掉 `return` 后，`respondError` 写了 500，函数继续执行 `c.JSON(http.StatusOK, movies)`（`movies` 是 nil）。Gin 打印 `[WARNING] Headers were already written. Wanted to override status code 500 with 200`，客户端收到 500，响应体是 `{"error":"internal server error"}null`。测试：在 `memstore.Movies` 上加 `Err error` 字段，`List` 中 `if s.Err != nil { return nil, s.Err }`；测试断言状态码为 500 **且** `json.Unmarshal(body)` 成功——拼接的响应体会让 `Unmarshal` 失败，从而发现漏掉的 `return`。

**9.6**

```go
var ErrReviewTooShort = errors.New("review is too short")   // services/errors.go

var errorStatus = map[error]int{
	// ...
	services.ErrReviewTooShort: http.StatusUnprocessableEntity,
}

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

新增业务错误只需要两步：在 services 中定义，在映射表中登记。

**9.7** `router.Use(Auth)` 在注册任何路由之前调用，所以之后注册的**所有**路由（包括 `/auth/login`、`/auth/register`、`/movies`）都带上了 Auth。登录请求没有 Token，被拒绝：`401 {"error":"missing bearer token"}`。用户永远无法登录，首页也加载不出电影。不存在的路径同样返回 401：`engine.Use` 会重建 404 处理链并把 Auth 加进去（编写教材时查阅过 Gin v1.12.0 源码确认），请求在到达 404 处理之前就被拦下了。

**9.8** `404 page not found`（Gin 默认的纯文本 404）。全局中间件只有 Logger、Recovery、CORS、SecurityHeaders、BodyLimit，它们都会放行；Auth 只挂在路由组上，不在 404 处理链中。如果在 `main.go` 中用 `router.Use` 注册一个总是返回 401 的中间件，结果就会变成 401——全局中间件会进入 404 处理链。

**9.9** 有。`/health` 在 `routes.Register` 中注册，而 `routes.Register` 在 `router.Use(... cors.New(...) ...)` 之后调用，所以它的处理链包含 CORS 中间件。如果移到 `router.Use` 之前注册，它的处理链就只有 Handler 本身：响应中**没有** `Access-Control-Allow-Origin`，也没有安全响应头和日志。这再次说明：`router.Use` 只影响之后注册的路由，全局中间件必须在注册任何路由之前添加。

---

## 第 10 章

**关键概念**：点号路径作用于数组时，MongoDB 检查数组中的任意元素。用户 ID 来自已验证的 JWT，不可伪造。`v, ok := x.(T)` 在类型不符时返回 `ok=false`，`v := x.(T)` 则 panic。

**10.1** 在提示词中明确约束输出；对输出做规范化（trim、去标点、忽略大小写）；**只接受白名单中的值**，不在白名单中就报错（本项目返回 502）或重试；保存的是白名单中的标准值而不是原始输出；设置超时；记录原始输出以便排查。还可以考虑模型的结构化输出功能（JSON Schema）。

**10.2** 用户输入中包含试图改变模型行为的指令。本项目的防御：输出白名单（`matchRanking`）、模型输出只用于选择评级（最小权限）、影评长度上限 2000 字符、接口只对管理员开放。其他常见做法：把指令与用户输入分开（system / user 消息）、对输入做标记或转义。

**10.3** `limit(0)` 表示不限制数量。本项目中 `RECOMMENDED_MOVIE_LIMIT` 由 `config.positiveInt` 解析：不是正整数时 `config.Load()` 返回错误，服务器不会启动，所以运行中的服务器永远不会把 0 传给 `SetLimit`。

**10.4**

```go
func (r *UserRepository) FavouriteGenreNames(ctx context.Context, id string) ([]string, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, ErrNotFound
	}
	var doc struct {
		FavouriteGenres []models.Genre `bson:"favourite_genres"`
	}
	opts := options.FindOne().SetProjection(bson.M{"favourite_genres.genre_name": 1})
	if err := r.coll.FindOne(ctx, bson.M{"_id": oid}, opts).Decode(&doc); err != nil {
		return nil, translate(err)
	}
	names := make([]string, 0, len(doc.FavouriteGenres))
	for _, g := range doc.FavouriteGenres {
		names = append(names, g.GenreName)
	}
	return names, nil
}
```

注意不合法的 ID（不是 24 位十六进制）也当作“找不到”，而不是 500——它只可能来自一个伪造或损坏的 Token。

**10.5**

```go
func TestMatchRanking(t *testing.T) {
	rankings := []models.Ranking{
		{RankingValue: 1, RankingName: "Excellent"}, {RankingValue: 2, RankingName: "Good"},
		{RankingValue: models.NotRankedValue, RankingName: "Not_Ranked"},
	}
	cases := []struct {
		answer string
		want   int // 0 表示不应匹配
	}{
		{"Excellent", 1}, {" excellent.\n", 1}, {"**Good**", 2},
		{"Masterpiece", 0}, {"Not_Ranked", 0}, {"I think it's Good", 0},
	}
	for _, tc := range cases {
		got, ok := matchRanking(tc.answer, rankings)
		if (tc.want == 0) == ok || (ok && got.RankingValue != tc.want) {
			t.Errorf("matchRanking(%q) = %+v, %v", tc.answer, got, ok)
		}
	}
}
```

`matchRanking` 是未导出函数，所以测试文件要写 `package services`（而不是 `package services_test`）。

**10.6** `Recommended` 的实现见 10.2.3 节。假 `MovieStore` 可以参考 `internal/memstore` 的 `Movies.Recommend`：用 `slices.ContainsFunc` 过滤、`slices.SortStableFunc` 按 `RankingValue` 排序、截断到 limit。(c) 的验证方法：让假实现的 `Recommend` 记录调用次数，断言为 0。

**10.7** 已有电影文档中嵌入的是副本：它们仍显示 `Excellent`，`ranking_value` 仍为 1，排序不受影响。新的影评评级：候选列表中是 `Superb`，模型返回 `Superb` 时被映射为 `{1, "Superb"}`；如果模型（受影评中常见措辞影响）仍然返回 `Excellent`，`matchRanking` 找不到它，接口返回 502。结果是同一个评级在不同电影上显示两种名称——这就是反范式化的数据一致性代价。修复：同时批量更新 `db.movies.updateMany({"ranking.ranking_value": 1}, {$set: {"ranking.ranking_name": "Superb"}})`（按数值匹配比按名称更可靠）。

**10.8** `abc`：服务器不启动，打印 `RECOMMENDED_MOVIE_LIMIT must be a positive integer, got "abc"`。`0`：同样拒绝启动（`n <= 0`）。“启动失败”让错误在部署时就暴露，并明确指出哪个配置有问题；“悄悄使用 0”会让推荐接口返回所有匹配的电影，可能要过很久才有人发现，而且很难联想到是配置写错了。

**10.9** `"Good."`：`matchRanking` 去掉句号后忽略大小写匹配到 `{2, "Good"}`；返回 200，数据库中存入 `ranking: {ranking_value: 2, ranking_name: "Good"}`（标准名称，不带句号）。`"Masterpiece"`：不在白名单中，service 返回 `ErrUnrecognizedRanking`，controller 返回 502 `review classifier returned an unknown ranking`；**数据库没有任何变化**，影评和评级都保持原样。

---

## 第 11 章

**关键概念**：认证是“你是谁”，授权是“你能做什么”。AT 短是因为无法撤销；RT 长是为了体验，但要查库确认、并轮换以限制被盗后的有效期。HttpOnly 防 XSS 读取，Secure 防窃听，SameSite 控制跨站发送（防 CSRF），Path 限制 Cookie 发往哪些接口。密钥作为参数传入，是因为包级变量会在 `main()` 加载 `.env` 之前初始化，而且参数让依赖关系和读取顺序显式可见。

**11.1** 纯 JWT 无法撤销。方案：(1) AT 设短有效期 + RT 存数据库并可撤销（本项目）；要让某用户所有会话失效，删除其 RT 哈希即可，现有 AT 最多 15 分钟后失效；(2) 在用户文档中维护 `token_version`，写入 JWT，验证时比对（每次都要查库或缓存）；(3) 维护 AT 的 `jti` 黑名单（通常放在 Redis 中，本项目的 Token 已经带有 `jti`）。

**11.2** 不是加密的，只是 Base64URL 编码，任何人都能解码。可以放：用户 ID、角色、过期时间等非敏感的标识信息。不能放：密码、个人隐私数据、任何不想让持有者看到的内容。本项目甚至没有放姓名和邮箱。

**11.3** 见第 11 章 11.7.3 节的表格。

**11.4** 轮换：每次用 RT 换取 AT 时同时签发新 RT 并作废旧 RT。重用检测：如果收到一个已被作废的 RT，说明可能被盗用，立即撤销该用户（或该会话链）的全部 RT。本项目实现了轮换（用比较并交换保证并发安全），重用检测见练习 11.8。

**11.5** HS256 对称（同一密钥签名和验证），简单快速，适合单一服务。RS256/ES256 非对称（私钥签名、公钥验证），适合由认证服务签发、多个服务验证，验证方无需持有能伪造 Token 的私钥。

**11.6** 参考 11.5 节和 11.11 节的代码以及 `S/tokens/tokens_test.go`。容易漏掉的点：`WithExpirationRequired`（否则没有 `exp` 的 Token 永不过期）；随机 `jti`（否则同一秒签发的两个 RT 相同）；构造 `alg: none` 的测试 Token 时要用 `jwt.UnsafeAllowNoneSignatureType` 作为“密钥”。

**11.7** 实现见 11.9.4 节。测试：

```go
func TestRotateIsAtomic(t *testing.T) {
	db := testDB(t)
	users := repository.NewUserRepository(db)
	u := &models.User{Email: "a@example.com"}
	_ = users.Create(ctx, u)
	_ = users.SetRefreshTokenHash(ctx, u.ID.Hex(), "old")

	var wg sync.WaitGroup
	var ok atomic.Int32
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if users.RotateRefreshTokenHash(ctx, u.ID.Hex(), "old", fmt.Sprint("new", i)) == nil {
				ok.Add(1)
			}
		}()
	}
	wg.Wait()
	if ok.Load() != 1 {
		t.Errorf("%d rotations succeeded, want exactly 1", ok.Load())
	}
}
```

改成“先读后写”之后，测试通常会失败（多个 goroutine 都读到了 `old`，都比较通过，都执行了更新），而且失败是**偶发**的——有时恰好只有一个成功。这正是竞态 bug 难以发现的原因；用 `go test -race -count=20` 多跑几次更容易暴露。

**11.8** 在 `AuthService.Refresh` 中，`RotateRefreshTokenHash` 返回 `ErrNotFound` 时，先调用 `s.users.SetRefreshTokenHash(ctx, user.ID.Hex(), "")`，再返回 `ErrInvalidRefreshToken`。测试：注册得到 RT1 → 刷新得到 RT2 → 用 RT1 刷新（失败）→ 断言用 RT2 刷新**也**失败。注意它的副作用：前端如果因为并发刷新而偶尔用到旧 RT（例如去重逻辑有漏洞），用户会被直接登出——所以重用检测通常配合一个很短的“宽限期”。

**11.10** 开发模式下 StrictMode 让 `AuthProvider` 的 effect 执行两次，两个 `/auth/refresh` 请求带着同一个 RT1 几乎同时发出。先到达服务器的成功（数据库中的哈希变为 hash(RT2)，并 `Set-Cookie: RT2`）；后到达的在 `RotateRefreshTokenHash` 中匹配不到，返回 401 **并且通过 `setRefreshCookie(c, "", -1)` 删除 Cookie**。如果这个失败响应在成功响应之后被浏览器处理，Cookie 就被删除了。第二次 effect 的结果（`active` 为 true 的那个）如果是失败的，`auth` 被设为 null。最终：开发时每次刷新页面都可能被登出，而且 Cookie 也没了。

**11.11** 依次是：`recommendations` → 401；`refresh` → 200；`recommendations` → 200。（如果 Network 面板显示了预检请求，每个跨域请求前还可能有一个 `OPTIONS`，取决于预检缓存。）

**11.12** 每个用户在数据库中只存一个 `refresh_token_hash`。手机登录（或刷新）时覆盖了它，电脑上的 RT 就失效了。电脑上的 AT 到期（最多 15 分钟）后，自动刷新返回 401，用户被登出。这是“单会话”设计的结果，可能不是期望的行为。支持多设备：每个会话一条记录（练习 19.21）。

---

## 第 12 章

**关键概念**：`Next()` 执行剩余的链并返回；`Abort()` 阻止后续函数执行，但不会结束当前函数，所以要 `return`。CORS 保护用户（防止恶意网站利用用户的浏览器），由浏览器执行，curl 不受限。`AllowCredentials` 需要前端设置 `withCredentials: true`。

**12.1** 请求从外到内依次经过每个 Middleware 的 `c.Next()` 之前的代码，到达 Handler；然后从内到外依次执行 `c.Next()` 之后的代码。像剥洋葱一样进去再出来。

**12.2** 浏览器在发送“非简单请求”之前发送的 `OPTIONS` 请求，询问服务器是否允许。触发条件：方法不是 GET/HEAD/POST；或者有自定义请求头（如 `Authorization`）；或者 `Content-Type` 不是 `text/plain`、`multipart/form-data`、`application/x-www-form-urlencoded` 之一（例如 `application/json`）。

**12.3** 规范规定：携带凭证的请求，`Access-Control-Allow-Origin` 必须是具体的源。否则任何网站都能以用户身份读取数据，同源策略就形同虚设。

**12.4** 参考第 11 章 11.12–11.13 节以及 `routes_test.go` 的 `TestAuthorization`。

**12.6** 参考 12.6.3 节。测试中可以直接构造一个只有限流中间件和一个空 Handler 的 `gin.Engine`，用 `httptest` 发请求。删掉锁之后，`-race` 会报告 `WARNING: DATA RACE`，并指出两个 goroutine 同时读写 `visitors` 的位置；不开 `-race` 时也可能直接崩溃：`fatal error: concurrent map writes`。

**12.7** `c.JSON` 写入了 401 和 `{"error":"missing bearer token"}`，函数返回；由于没有 Abort，Gin 继续执行处理链中的下一个函数。
- `GET /me/recommendations`：执行到 `MovieController.Recommended`，`middleware.UserID(c)` 返回空字符串 → `FavouriteGenreNames("")` 因为 ID 不合法返回 `ErrNotFound` → service 当作没有喜爱类型 → `c.JSON(200, [])`。客户端收到**状态码 401**（第一次写入的状态码生效），响应体是 `{"error":"missing bearer token"}[]`，日志中出现 `Headers were already written` 警告。
- `POST /movies`：下一个函数是 `RequireRole`，`Role(c)` 为空，`AbortWithStatusJSON(403, ...)` 中止了处理链——Handler 没有执行。客户端收到状态码 401，响应体是两段 JSON 拼接。

这说明：只有 Abort 才能真正阻止后续处理。这里是 `RequireRole` 的“默认拒绝”设计挽救了管理员接口。

**12.8** 请求的 Origin `http://localhost:5173` 与白名单中的 `http://localhost:5173/` 不相等。gin-contrib/cors 对不在白名单中的 Origin 返回 **403** 且不带 CORS 响应头。浏览器控制台显示：`Access to XMLHttpRequest at 'http://localhost:8080/movies' from origin 'http://localhost:5173' has been blocked by CORS policy: No 'Access-Control-Allow-Origin' header is present on the requested resource.`

**12.9** 所有带 `Authorization` 头的请求都会在预检时被拒绝：推荐页、电影详情 / 影评页无法加载数据。不受影响：首页、注册、登录、登出、刷新会话（它们使用 `axiosClient`，不带 `Authorization`）、预告片页（不调用后端）。

**12.10** 没配置：`ClientIP()` 返回的是 Nginx 的地址，**所有用户共享同一个限流额度**——正常高峰期就会有大量用户收到 429。信任所有地址：`ClientIP()` 会采信请求中的 `X-Forwarded-For`，攻击者每次请求都伪造一个不同的 IP，就能完全绕过限流。正确做法是只填写真实代理的地址或网段。

---

## 第 13 章

**关键概念**：前端校验可以被绕过。`required`：string 非空、int 非 0、slice 非 nil。`errors.Is` 会沿着包装链查找，`==` 只比较最外层。

**13.1** Go 的设计者认为异常会让控制流变得隐式、难以推理。显式的错误返回迫使你在每个可能失败的地方做决定。啰嗦是代价，可以通过辅助函数、错误包装、统一的错误处理层来减轻。

**13.2** `%w` 包装错误，`errors.Is/As` 能沿着它找到原始错误；`%v` 只把错误转成字符串，原始错误的类型信息丢失。

**13.3** 应包含：一个通用的、对用户友好的描述，可选的请求 ID（方便对照日志）。不应包含：堆栈、SQL/查询语句、内部服务地址、第三方 API 的原始错误、文件路径。

**13.4** 参考 13.3.1 和 13.3.3 节。测试示例：

```go
func TestDescribeUsesJSONNames(t *testing.T) {
	v := validation.New()
	err := v.Struct(models.UserRegister{FirstName: "A"})
	details := validation.Describe(err)
	if !slices.Contains(details, "first_name: min=2") {
		t.Errorf("details = %v", details)
	}
}
```

**13.5**

```go
// validation.New 中
err := v.RegisterValidation("notblank", func(fl validator.FieldLevel) bool {
	return strings.TrimSpace(fl.Field().String()) != ""
})
if err != nil {
	panic(err)
}
// models.ReviewUpdate: validate:"required,notblank,max=2000"
```

（validator 在 `github.com/go-playground/validator/v10/non-standard/validators` 包中提供了现成的 `validators.NotBlank`，也可以用 `v.RegisterValidation("notblank", validators.NotBlank)` 注册它。）

**13.6** `ShouldBindJSON` 改为：`dec := json.NewDecoder(c.Request.Body); dec.DisallowUnknownFields(); err := dec.Decode(dst)`。向后兼容性：前端（或其他客户端）只要多发了一个字段——例如旧版本前端还在发送已经删除的 `role`——请求就会失败。所以严格模式通常只用于内部 API，或者需要配合 API 版本管理。

**13.7** (a) 400，`{"error":"validation failed","details":["password: max=72"]}`——100 个字符超过了 `max=72`，在校验阶段就被拒绝。(b) 400，`{"error":"password must be at most 72 bytes"}`——40 个字符通过了 `max=72`，但 80 个字节超过了 bcrypt 的上限，被 `AuthService.Register` 的字节检查拦下，返回业务错误 `ErrPasswordTooLong`。两者都是 400，但发生在不同的层：前者是格式校验，后者是业务规则（为了 bcrypt 能正常工作）。

**13.8** `{"error":"validation failed","details":["poster_path: url","youtube_id: youtube_id"]}`。如果其他必填字段也缺失，会有更多条目。

**13.9** 密码错误时，服务器返回 401，axios 抛出异常，执行跳到 `catch` 块，状态码不是 429，显示 `"Invalid email or password"`。连续输错：前 10 次都是 401（限流允许突发 10 次），第 11 次服务器返回 429，页面显示 “Too many attempts. Please wait a minute and try again.”；之后大约每 6 秒恢复一次尝试机会。

---

## 第 13a 章

**关键概念**：`useState` 保存会触发渲染的状态；`useEffect` 处理副作用；`useMemo` 缓存派生值；`useRef` 保存不触发渲染的可变值或 DOM 引用。`key` 让 React 正确识别列表项；用下标在增删排序时会错乱。前端守卫只是体验，不是安全。

**13a.1** 依赖数组决定 effect 何时重新执行。漏写依赖时，effect 中使用的是旧的值（“陈旧闭包”）；多写不稳定的依赖（如每次渲染新建的对象）会导致 effect 无限重复执行。

**13a.2** 为了把数据传给深层组件，中间每一层都要接收并转发 props。Context 让深层组件直接读取。缺点：Context 值变化时所有消费它的组件都会重新渲染；过度使用会让数据流向变得不清晰。

**13a.3** `<a href>` 让浏览器发起整页导航，所有 JS 状态丢失；`<Link to>` 由 React Router 在页面内切换。本项目中，用错会导致内存中的 AT 丢失，每次都要通过 `/auth/refresh` 恢复会话（并且会轮换 RT），体验变差。

**13a.4** 对照 13a.3.2 节的代码。容易漏掉的点：`<Navigate replace>`（否则按后退键会在受保护页和登录页之间来回跳）；`state={{ from: location }}`；加载指示器的无障碍文字。

**13a.5** 用 `react-player` 时，构建输出中会出现 `hls-*.js`、`dash.all.min-*.js` 等几百 KB 的独立 chunk，并有 “Some chunks are larger than 500 kB” 的警告；换成 `<iframe>` 后这些文件全部消失。

**13a.6** 参考 13a.5.3 节和 `Movie.tsx`：`useAuth()` 取出 `auth`，`{updateMovieReview && auth && (...)}` 控制显示，`isAdmin` 决定文案和图标。

**13a.7** 数据已经在了：`GET /movies/:imdb_id` 返回的 `movie.ranking` 包含 `ranking_value`。只需要在 `Review.tsx` 中渲染 `{movie.ranking?.ranking_name} ({movie.ranking?.ranking_value})`，不需要修改后端。（`999` 表示未评级，可以特殊处理成不显示数值。）

**13a.8** 每次渲染都创建新的 axios 实例。`Recommended` 中的 effect 依赖 `axiosPrivate`，于是每次渲染都会重新执行 → 发请求 → `setMovies`/`setLoading` → 再次渲染 → 新实例 → 再次执行……Network 面板中会看到 `/me/recommendations` 请求不断重复（无限循环），拦截器也被反复注册和移除。

**13a.9** 整页刷新，React 状态全部丢失。`AuthProvider` 重新挂载，调用 `/auth/refresh`；如果 RT Cookie 有效，会话被恢复（同时 RT 被轮换），用户**不会被登出**，但会多一次请求，页面闪一下。

**13a.10** 页面刚加载时 `auth` 是 null（会话恢复还在进行中），`RequiredAuth` 不再等待，立即重定向到 `/login`。恢复完成后 `auth` 有值了，但用户已经在登录页上——`Login` 组件不会在已登录时自动跳走，所以用户会看到登录表单，尽管实际上已经登录了（Header 中显示了名字）。

---

## 第 14 章

**关键概念**：登录时浏览器发两个请求：`OPTIONS` 预检（因为 `Content-Type: application/json`）和真正的 `POST`。推荐接口查两次数据库：先查用户的喜爱类型（投影只取类型名），再查匹配的电影；用户没有喜爱类型时只查一次。

**14.1** 参考答案的要点（按顺序）：表单提交事件与 `preventDefault` → React 状态 → axios 构造请求 → DNS/TCP（localhost）→ CORS 预检 → 请求到达 Gin，经过全局 Middleware 和限流 → JSON 绑定与校验 → service：邮箱小写化、数据库查询 → bcrypt 比对 → JWT 签名 → 只把 RT 的哈希写入数据库 → controller：Set-Cookie → JSON 响应 → 浏览器存储 Cookie → axios Promise 完成 → `setAuth` → Context 更新 → Header 重新渲染 → `navigate`。能讲到 Cookie 属性（HttpOnly、SameSite、Path）、AT 存放位置、为什么用 bcrypt、为什么数据库只存 RT 的哈希，就是一个很好的回答。

**14.3** 要点：`POST /auth/register` → RateLimit → `bindJSON(UserRegister)` → `AuthService.Register`：字节长度检查 → `GenresByIDs` 查询类型（数量不符 → 400）→ bcrypt 哈希 → `InsertOne`（唯一索引冲突 → 409）→ `startSession`：签发 Token、保存 RT 哈希 → 201 + Set-Cookie → 前端 `setAuth` → `navigate('/')`。

**14.4** 登录后：`refresh_token_hash` 出现（64 位十六进制），`updated_at` 更新为当前时间。刷新后：`refresh_token_hash` 变成另一个值，`updated_at` 再次更新。登出后：`refresh_token_hash` 字段消失（`$unset`），`updated_at` 更新。整个过程中数据库里从未出现过 Token 原文。

**14.5** OpenAI 的调用已经完成，**费用已经产生**。`UpdateOne` 失败，`respondError` 记录日志并返回 500，数据库中的电影保持原样——数据是一致的（没有出现“只更新了一半”的情况，因为 `$set` 在一次更新中同时设置两个字段），但钱花了却没有结果。管理员很可能会重试，再花一次钱。这类“外部调用成功、本地保存失败”的情况在实际系统中很常见；如果代价很高，可以把外部调用的结果先缓存起来（例如按影评内容的哈希），重试时直接复用。

---

## 第 15 章

**关键概念**：E2E、集成、HTTP、单元；越底层越快越稳定，所以数量最多。`Errorf` 继续，`Fatalf` 停止当前测试。`*gin.Engine` 实现了 `http.Handler`，`ServeHTTP` 可以直接调用。同包测试能访问未导出的标识符，`_test` 包只能使用公开 API。

**15.1** 把多个测试用例写成一个结构体切片（或 map），用循环 + `t.Run` 执行。好处：添加用例只需加一行；每个用例有名字，失败时一目了然；测试逻辑只写一遍。

**15.2** Mock：验证“调用了哪些方法、参数是什么”，容易与实现细节耦合。Fake：一个简化但可工作的实现（如用 map 实现的仓库），测试行为而不是调用细节。真实数据库：最接近真实，但慢、需要基础设施。本项目的做法：业务逻辑和 HTTP 层用 fake（`internal/memstore`），repository 用真实 MongoDB 做集成测试。

**15.3** 收集信息（日志、请求 ID、监控指标、受影响的用户/时间段）→ 在测试或预发环境中复现（用相同的数据和配置）→ 比较生产与开发环境的差异（配置、版本、数据量、网络拓扑、代理）→ 必要时增加有针对性的日志并发布 → 修复后加测试。

**15.4** `-cover` 只统计**被测包自身**的代码：`routes` 的测试实际执行了 controllers、middleware、services 的大量代码，但这些都不算进 `routes` 的覆盖率，controllers 和 middleware 因为没有自己的测试文件，显示为 0%。`-coverpkg=./...` 把所有包都纳入统计，每个测试包都会报告它对整个 module 的覆盖情况。所以看覆盖率时要清楚统计口径——0% 不一定代表“没被测试过”。

**15.5** 参考 `routes_test.go` 的 `TestRegisterIgnoresClientRoleAndSetsScopedCookie`。断言数据库中的角色，可以用 `users.FindByEmail` 读回来检查（fake 和真实仓库都可以）。

**15.6** 在 `memstore.Movies` 上加 `Err error` 字段，`List` 开头 `if s.Err != nil { return nil, s.Err }`。测试中设置 `Err = errors.New("connection refused: 10.0.0.5:27017")`，断言状态码 500、`json.Unmarshal` 成功、`error` 字段等于 `internal server error`、响应体中不包含 `10.0.0.5`。

**15.8** 示例计划：(1) 复现：登录后停留超过 15 分钟（或把 `AccessTokenTTL` 临时调短）再点击；(2) 观察：Network 中 `/auth/refresh` 的响应是什么、请求中是否带了 Cookie（Path、SameSite 是否匹配）；(3) 假设：Cookie 被浏览器拦截（前后端不同站点却用了 `SameSite=Lax`）；或者某个并发请求带着旧 RT 刷新失败，导致 `setAuth(null)`；或者在另一设备登录过（11.12）；(4) 验证：看 `/auth/refresh` 的状态码和请求的 Cookie 头；(5) 修复根本原因；(6) 加测试。

**15.9** `gin.New()` 不带 Logger 和 Recovery：测试输出更干净；而且 Recovery 不会把 panic 转成 500 掩盖问题——测试中出现 panic 会直接导致测试失败，更容易发现 bug。没有注册全局中间件意味着这些测试**不覆盖** CORS、安全响应头、请求体大小限制。可以再写一个小测试，用与 `main.go` 相同的方式注册全局中间件，专门断言这些响应头存在、超大请求体被拒绝；或者把全局中间件的注册提取成一个函数，让 `main` 和测试共用。

---

## 第 16 章

**关键概念**：删除历史无法撤回已经被看到/复制/缓存的密钥，只有轮换能让它们失效。`oneof` 校验的是格式，不是权限。强类型 struct 让操作符对象无法被解析为 string 字段。

**16.1** 立即轮换/吊销该密钥（在 AWS 控制台禁用并删除 Access Key）→ 检查 CloudTrail 等审计日志，确认是否已被滥用 → 从仓库中移除并加入 `.gitignore`，视情况改写历史 → 通知安全负责人 → 复盘并加入 gitleaks 等预防措施。顺序很重要：**先吊销**。

**16.2** A01 Broken Access Control（失效的访问控制）。本项目的防御：授权集中在路由组（`admin` 组挂 `RequireRole`），新增管理员接口不会忘记检查；角色由服务器决定，注册 DTO 中没有 `role`；用户身份只来自服务器验证过的 Token 或 Cookie，不来自请求参数；`TestAuthorization` 用表格锁定每个接口在不同身份下的状态码。

**16.3** React 在渲染 `{value}` 时会对字符串进行 HTML 转义。仍可能受攻击的情况：使用 `dangerouslySetInnerHTML`；把用户输入放进 `href`（如 `javascript:` URL）；第三方脚本被篡改；直接操作 DOM（`innerHTML`）。

**16.4** 如果输出中有发现，它会列出提交哈希、文件、规则名（例如 `generic-api-key`）。对每一个发现：确认它是不是真实的密钥；如果是，不管它是否还在当前代码中，都要轮换。

**16.5** 响应头中应该有 `X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Referrer-Policy: no-referrer`、`Content-Security-Policy: ...`。2 MB 请求体：400 `{"error":"invalid request body"}`——`BodyLimit` 让 `ShouldBindJSON` 读取时出错。可以用 `python3 -c "print('{\"email\":\"' + 'a'*2000000 + '\"}')" > big.json` 生成文件，再 `curl --data-binary @big.json ...`。

**16.6** 在 `users` 文档（或单独的集合 / Redis）中记录 `failed_logins` 和 `locked_until`；登录失败时 `$inc`，成功时清零；`locked_until` 在未来时直接返回 429。可以被利用：攻击者故意用错误密码登录别人的账户 5 次，就能让真正的用户无法登录（拒绝服务）。缓解：锁定时间短且递增；锁定期间允许通过邮件链接解锁；结合 IP 维度（同一 IP 对多个账户失败才锁定）；用验证码代替硬锁定。

**16.7** 查询变为 `{email: {$ne: null}}`，返回集合中第一个有邮箱的用户。但接下来 `bcrypt.CompareHashAndPassword` 在 Go 代码中比对密码（不在数据库查询中），`"x"` 不匹配，返回 401——这个场景下密码校验挡住了攻击。但如果有“按邮箱发送重置链接”之类只依赖查询结果的功能，就会被利用。教训：不要依赖“后面还有一道检查”，从源头使用强类型。

**16.8** 只泄露 `SECRET_KEY`：攻击者可以签发任意用户、任意角色（包括 ADMIN）的 AT，而且可以自己设置很远的 `exp`——能完全冒充任何人，直到你更换密钥。他无法签发有效的 RT，但也不需要。只泄露 `SECRET_REFRESH_KEY`：攻击者能伪造签名有效的 RT，但 `RotateRefreshTokenHash` 要求它的哈希与数据库中存储的值**完全一致**，伪造的 RT 无法通过——这正是“与数据库比对”带来的纵深防御。即使同时拿到了数据库，数据库中也只有哈希，无法反推出一个可用的 RT。

---

## 第 17 章

**关键概念**：`VITE_` 变量在构建时被替换为字面量。SPA 回退让所有未知路径返回 `index.html`，否则直接访问或刷新子路由会 404。`log.Fatal` 会调用 `os.Exit`，跳过所有 `defer`；`run()` 返回 error，让清理代码（断开数据库）在退出前执行。

**17.1** 构建环境与运行环境分离：最终镜像小、启动快、攻击面小（没有编译器、shell）；构建依赖不会泄露到生产镜像中；可以更好地利用层缓存。

**17.2** 收到停止信号后，停止接受新连接、等待进行中的请求完成、释放资源再退出。Kubernetes 滚动更新、扩缩容时会频繁发送 `SIGTERM`，没有优雅关闭会导致每次部署都有一部分用户请求失败。

**17.3** RT Cookie 成为第三方 Cookie，必须 `SameSite=None; Secure`，并可能被 Safari 等浏览器拦截，还失去了 SameSite 对 CSRF 的防护。解决：同站子域（`app.x.com` / `api.x.com`，可以用 `SameSite=Lax`）；反向代理实现同源；或者改用不依赖 Cookie 的方案（风险更高）。

**17.4** curl 仍然拿到了完整的 200 响应。日志顺序：`shutting down...` → （约 5 秒后）`[GIN] 200 | 5.0s | ... GET "/movies"` → 进程退出。`Shutdown` 先关闭监听（新连接会被拒绝），再等待这个进行中的请求完成，然后 `run()` 返回、`defer` 断开数据库。

**17.5** `web` 服务的 build args 中设置 `VITE_API_BASE_URL: http://localhost:8080`，端口映射 `"3000:80"`；`server` 的 `environment` 中加 `ALLOWED_ORIGINS: http://localhost:3000`。localhost:3000 和 localhost:8080 是同站的，`SameSite=Lax` 可以正常工作。

**17.6** 需要修改：`vite.config.ts` 的代理；`VITE_API_BASE_URL=/api`；Cookie 的 Path——浏览器看到的刷新地址是 `/api/auth/refresh`，而后端设置的 Cookie Path 是 `/auth`，浏览器不会把它发给 `/api/auth/...`。可以把 `refreshCookiePath` 改为可配置并设成 `/api/auth`，或者让后端所有路由都挂在 `/api` 下（代理就不需要去掉前缀）。同源之后 CORS 不再需要，`SameSite` 可以改成 `Strict`。

**17.7** Nginx 在磁盘上找不到 `/recommended` 文件，返回 **404 Not Found** 页面，React 应用根本没有加载。

**17.8** Docker 按层缓存。依赖（`go.mod`/`go.sum`）变化很少，源代码变化频繁。先复制依赖文件并下载，这一层可以在后续构建中被复用；如果先 `COPY . .`，任何源代码改动都会让依赖下载层失效，每次构建都要重新下载所有依赖。

---

## 第 18 章

自检问题参考答案：

| 里程碑 | 答案 |
|---|---|
| M1 | `gin.Default()` = `gin.New()` + Logger + Recovery。`router.Use` 只影响之后注册的路由，写在后面会让已注册的路由缺少中间件。 |
| M2 | 没有。`Connect` 只创建客户端并在后台启动连接池，第一次操作（如 Ping）时才真正连接。`CreateMany` 是幂等的，索引已存在且定义相同时什么也不做。 |
| M3 | 防止数据库文档被误用为请求或响应（批量赋值、泄露密码哈希）。角色由服务器决定，DTO 中没有它，客户端提交了也会被丢弃。 |
| M4 | 让 service 不依赖数据库驱动；换数据库时只改 repository；也能区分“找不到”（404）和“数据库故障”（500）。 |
| M5 | 包级变量在 `main()` 加载 `.env` 之前初始化，会读到空值；参数让依赖显式、可测试。`iat`/`exp` 精确到秒，没有 `jti` 时同一秒签发的两个 RT 完全相同，轮换就失效了。 |
| M6 | “先查后插”有竞态，两个并发请求可能都通过检查；唯一索引在数据库内部原子地保证不重复。`dummyHash` 让“邮箱不存在”和“密码错误”耗时相同，防止通过响应时间枚举邮箱。 |
| M7 | 401：未认证；403：已认证但没有权限。前端收到 401 会刷新 Token 并重试，权限问题返回 401 会导致白白刷新、甚至把用户登出。 |
| M8 | MongoDB 对数组字段的点号路径会检查每个元素，任意一个元素的 `genre_name` 在 `$in` 列表中即匹配。空列表短路既是业务规则，也避免了把 nil（BSON null）传给 `$in`。 |
| M9 | 先读再写时，两个并发请求可能都读到同一个旧哈希、都通过比较、都签发新 RT，一个 RT 被用了两次。把比较放进 `UpdateOne` 的过滤条件，由数据库原子地完成。 |
| M10 | 返回 502，数据库不变。OpenAI 按次收费，先做便宜的检查，避免为不存在的电影付费。 |
| M11 | 不配置时，部署在代理后面的所有请求 `ClientIP` 都是代理地址，所有用户共享额度；信任所有地址又会让客户端可以伪造 IP。`log.Fatal` 调用 `os.Exit`，进程立即结束，已注册的 `defer` 不会执行。 |
| M12 | async 函数返回 Promise，而 React 期望 effect 回调返回清理函数或 undefined。 |
| M13 | 服务器每次刷新都轮换 RT，并发的第二个请求会因 RT 已作废而失败；StrictMode 在开发时让 effect 执行两次，必然产生并发刷新。 |
| M14 | `useMemo` 让实例稳定，避免依赖它的 effect 无限循环；`useRef` 让拦截器读到最新 Token 而无需重新注册。只有 401 意味着 Token 有问题，刷新才有意义。 |
| M15 | 最终镜像小、攻击面小、构建缓存更有效。CI 中的 MongoDB 让 repository 集成测试在每次提交时都运行，而不是被跳过。 |

---

## 第 19 章

**19.1** 参考第 02 章 2.2 节和第 11 章 11.2 节的两张图。

**19.2**

| 接口 | 方法 | AT | RT Cookie | ADMIN | 限流 | 集合 |
|---|---|---|---|---|---|---|
| `/auth/register` | POST | 否 | 设置 | 否 | 是 | genres（读）、users（写） |
| `/auth/login` | POST | 否 | 设置 | 否 | 是 | users（读、写） |
| `/auth/logout` | POST | 否 | 读取并清除 | 否 | 否 | users（写） |
| `/auth/refresh` | POST | 否 | 读取并轮换 | 否 | 否 | users（读、写） |
| `/movies` | GET | 否 | 否 | 否 | 否 | movies（读） |
| `/movies` | POST | 是 | 否 | 是 | 否 | genres、rankings（读）、movies（写） |
| `/genres` | GET | 否 | 否 | 否 | 否 | genres（读） |
| `/movies/:imdb_id` | GET | 是 | 否 | 否 | 否 | movies（读） |
| `/movies/:imdb_id/review` | PATCH | 是 | 否 | 是 | 否 | movies（读、写）、rankings（读） + OpenAI |
| `/me/recommendations` | GET | 是 | 否 | 否 | 否 | users（读）、movies（读） |

**19.3** 与 11.12 相同：B 手机登录时 `AuthService.Login` → `startSession` → `SetRefreshTokenHash` 覆盖了数据库中的哈希。A 电脑的 AT 仍可用到过期；之后 `AuthService.Refresh` 中 `RotateRefreshTokenHash` 匹配不到 A 的旧哈希，返回 401 并清除 Cookie，A 被登出。这是单会话设计的副作用；是否是 bug 取决于产品需求（练习 19.21）。

**19.4** (1) 推荐排序（`MovieRepository.Recommend` 中 `SetSort(... Value: 1)`）；(2) AI 评级（`UpdateReview` 构造候选列表时排除 `NotRankedValue`，`matchRanking` 返回对应的数值）。改为“越大越好”：排序改为 `-1`；`rankings` 数据重新编号；`NotRankedValue = 999` 会变成最好的，需要改为 0（或负数），并同步修改常量和注释（注释中写着“必须是最大值”）；已有电影文档中嵌入的 `ranking_value` 需要批量迁移；种子数据也要修改。

**19.5** 信任客户端的地方：`favourite_genre_ids` / `genre_ids`（✅ 只是 ID，服务器查询确认存在并复制名称）；`email`/`first_name` 等（✅ 有格式和长度校验）；`imdb_id` 路径参数（✅ 只用作查询条件，强类型 string）；`admin_review` 内容（✅ 长度上限 2000，只有管理员能提交，LLM 输出有白名单）；`Origin` 头（✅ 由 CORS 白名单判断）；`X-Forwarded-For`（✅ 默认不信任，只信任配置的代理）；`yt_id`（前端，✅ 用正则校验）。服务器**不**信任的：角色（DTO 中没有）、用户 ID（来自 Token）、评级（由服务器根据 LLM 回答 + 白名单决定）。

**19.6** 要点：只需要修改 `repository`（所有查询）、`database`（连接、迁移代替 `EnsureIndexes`）、`models`（struct tag 改为 `db:` 或使用 ORM；`bson.ObjectID` 改为字符串或整数 ID）和 `cmd/seed`；`services` 和 `controllers` 只要 repository 的方法签名和错误值不变，就不需要修改。表：`users(id, email UNIQUE, password_hash, role, refresh_token_hash, ...)`、`genres(id, name UNIQUE)`、`rankings(value PK, name)`、`movies(imdb_id PK, title, poster_path, youtube_id, admin_review, ranking_value FK)`、`movie_genres(imdb_id FK, genre_id FK, PK(imdb_id, genre_id))`、`user_favourite_genres(user_id FK, genre_id FK)`。变简单：唯一约束和外键直接保证类型存在、修改类型名只需改一处、推荐查询可以用一条 JOIN 完成。变复杂：读取一部电影及其类型需要 JOIN 或两次查询；Schema 变更需要迁移脚本；`RotateRefreshTokenHash` 改为 `UPDATE ... WHERE id = ? AND refresh_token_hash = ?` 并检查影响行数（思路完全相同）。

**19.7–19.11** 开放题，参考相应章节。19.11 的思路：先测量（第 15 章）；最可能的瓶颈是 `GET /movies` 返回全部数据（加分页、缓存）、写影评时的 OpenAI 延迟（异步处理）、bcrypt 的 CPU 开销（登录高峰）；后端是无状态的（JWT），可以水平扩展多个实例——但要把内存中的限流状态移到 Redis（第 12 章 12.6.3）；MongoDB 使用副本集。

**19.12** 假设：工作目录不对。`godotenv.Load()` 读取的是**当前工作目录**下的 `.env`；文件不存在时被静默忽略（为了支持 Docker），于是所有配置都是空的。他可能用 IDE 运行时工作目录被设置成了仓库根目录。验证：`pwd`；在 `S/` 目录下运行 `go run .` 对比。另一种可能：他只复制了 `.env.example` 却没有填写必填项——错误列表中会精确地指出哪些变量是空的。修复：在 `S/` 目录下运行，或在 IDE 中设置工作目录。

**19.13** Safari 的智能防跟踪（ITP）默认拦截第三方 Cookie。`vercel.app` 和 `onrender.com` 是不同的站点，`SameSite=None` 的 RT Cookie 属于第三方 Cookie，被拦截，所以刷新页面后 `/auth/refresh` 没有 Cookie。改回 `Lax`：跨站的 XHR POST 根本不会携带 `Lax` Cookie，**所有浏览器**都会出现同样的问题。修复：使用同站的自定义域名（`app.reelm.com` + `api.reelm.com`，可以用默认的 `Lax`），或者反向代理同源部署（第 17 章 17.5）。

**19.14** 假设：新模型的回答格式不同，`matchRanking` 无法识别。验证：临时在 `UpdateReview` 中记录 LLM 的原始回答（或者直接用相同的提示词调用新模型），观察它返回了什么——例如 `Rating: Excellent`、一句完整的话，或者带 Markdown 的 `**Excellent**`（最后这种已经能被处理）。根本原因：提示词对新模型的约束不够。修复：调整 `BASE_PROMPT_TEMPLATE`（更明确地要求“只输出列表中的一个词”）、使用模型的结构化输出功能，或者对模型做一次简单的回归测试再上线。注意白名单让这次事故只是“部分请求失败、可以重试”，而没有把错误的评级写进数据库、污染推荐结果。

**19.15** 新接口的某个错误分支调用了 `respondError`（写了 500）之后没有 `return`，又写了成功响应。日志中警告前后的请求日志行能直接定位是哪个接口。修复：补上 `return`。预防：Code Review 时检查每个 `respondError` 后面是否紧跟 `return`；为错误分支写测试并断言响应体是合法 JSON（第 13 章 13.7）。

**19.16** 可能的来源：(1) 校验规则加入之前注册的旧数据；(2) 有人直接修改了数据库；(3) 从别处导入的用户数据。注册接口目前无法产生这样的数据（`min=1` + 类型存在性检查）。受影响用户：`db.users.find({favourite_genres: {$size: 0}})`。推荐接口对空列表直接返回 `[]`，所以用户看到的是 “No recommendations yet” 而不是错误——这是空列表短路的价值。处理：给这些用户提供修改喜爱类型的入口（练习 19.19），或者写一次性的迁移脚本为他们设置默认类型。

**19.17** 按可能性排序：(1) `SECRET_KEY` 泄露（例如曾经被提交到 Git），有人自己签发了 ADMIN Token——检查请求中 Token 的 `uid` 是否对应真实存在的用户、`jti` 是否重复出现，并立即轮换密钥；(2) 你自己的管理员会话被盗用（Token 出现在日志、截图中，或者设备被他人使用）——登出所有设备（删除 RT 哈希），观察调用是否停止；(3) 数据库被人直接改过——查 `db.users.find({role: "ADMIN"})` 确认确实只有你一个管理员。另外，如果 OpenAI Key 本身泄露，账单激增可能根本不经过你的服务器——检查 OpenAI 控制台的用量来源。处理：轮换所有密钥、给影评接口也加上限流、在 OpenAI 控制台设置用量上限。

**19.18** 按分层分别修改：

```go
// repository
func (r *MovieRepository) List(ctx context.Context, genre string) ([]models.Movie, error) {
	filter := bson.M{}
	if genre != "" {
		filter["genre.genre_name"] = genre
	}
	return findAll[models.Movie](ctx, r.coll, filter)
}

// controller
movies, err := h.movies.List(ctx, c.Query("genre"))
```

service 只是把参数传下去；`memstore.Movies.List` 也要相应修改，并在 `routes_test.go` 中加一个用例。

**19.19–19.22** 是设计性较强的编码题，参考答案要点：
- 19.19：路由放进 `protected` 组；用 `middleware.UserID(c)`；请求 DTO `{"genre_ids": [1,2]}`（规则与 `UserRegister.FavouriteGenreIDs` 相同），service 用 `GenresByIDs` 取出完整对象并检查数量；repository 新增 `SetFavouriteGenres` 执行 `$set`；响应可以只返回新的 `favourite_genres`，前端调用后 `setAuth(prev => prev && ({...prev, favourite_genres: data.favourite_genres}))`。
- 20：`c.DefaultQuery("page", "1")` + `strconv.Atoi` + 范围校验；`SetSkip((page-1)*limit)`、`SetLimit(limit)`；`CountDocuments` 计算 total；前端 `Home` 读取 `response.data.items`。兼容：可以先新增 `/v2/movies` 或 `?paginate=1`，前端迁移完成后再切换。
- 21：`sessions` 集合；`tokens.Claims` 增加 `SessionID`（json tag `sid`）；`AuthService.Refresh` 按 `session_id` 做比较并交换；`Logout` 删除当前 session；`POST /auth/logout-all` 按用户 ID 删除所有 session；给 `expires_at` 创建 TTL 索引让 MongoDB 自动清理过期会话；`users.refresh_token_hash` 字段不再需要。
- 22：`httptest.NewServer` 的 Handler 中检查请求路径是 `/chat/completions`，返回 `{"choices":[{"message":{"role":"assistant","content":"Excellent"}}]}`；用 `openai.New(openai.WithToken("test"), openai.WithBaseURL(srv.URL), ...)` 创建客户端（需要给 `NewOpenAIClassifier` 增加一个可选的 base URL 参数或选项）；500 的情况断言 `Classify` 返回错误；超时的情况让 Handler `time.Sleep` 超过 ctx 的期限，断言错误满足 `errors.Is(err, context.DeadlineExceeded)`。
