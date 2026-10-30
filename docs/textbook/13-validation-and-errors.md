# 第 13 章　校验与错误处理

> 本章目标：理解“永远不要信任客户端输入”的原则，掌握 validator 的用法，理解 Go 独特的错误处理哲学，并学会设计一致、安全的错误响应。

---

## 13.1 为什么必须在服务器端校验？

**原则：所有来自客户端的数据都是不可信的。**

前端的校验（例如 `Register.tsx` 中检查密码长度）只是为了**用户体验**——让用户更快地看到错误提示。但攻击者完全可以绕过前端，直接用 curl 发送任意数据。所以：

| 层次 | 作用 | 本项目中的例子 |
|---|---|---|
| 前端校验 | 即时反馈，减少无效请求 | `Register.tsx` 的 `handleSubmit`：姓名 ≥2、密码 ≥8、两次密码一致、至少选一个类型 |
| HTML 原生校验 | 浏览器内置 | `<Form.Control type="email" required minLength={8}>` |
| **后端格式校验** | **安全防线** | `bindJSON` 中的 `validate.Struct(req)` |
| **后端业务校验** | **安全防线** | 类型 ID 必须存在、密码不超过 72 字节、LLM 回答必须在白名单中 |
| 数据库约束 | 最后一道防线 | `users.email`、`movies.imdb_id` 唯一索引 |

> `Register.tsx` 中的注释写着 “Mirror the server-side validation in models.UserRegister”——前端规则是后端规则的**镜像**，但不能**替代**后端规则。后端规则改了（例如密码最少位数），前端也要同步修改，否则用户会遇到“前端说可以、后端说不行”的情况。

---

## 13.2 校验流程的两个阶段

两个阶段都在 `controllers/response.go` 的 `bindJSON` 中完成：

```go
func bindJSON(c *gin.Context, v *validator.Validate, dst any) bool {
	// 阶段 1：解析（Parsing）—— JSON 格式对吗？类型对吗？
	if err := c.ShouldBindJSON(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return false
	}
	// 阶段 2：校验（Validation）—— 值符合规则吗？
	if err := v.Struct(dst); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed", "details": validation.Describe(err)})
		return false
	}
	return true
}
```

以 `POST /movies`（`MovieCreate`）为例：

| 输入 | 阶段 1 结果 | 阶段 2 结果 |
|---|---|---|
| `{not json` | ❌ 语法错误 | — |
| `{"title": 123}` | ❌ 类型错误（number 不能赋给 string） | — |
| 超过 1 MB 的请求体 | ❌ `BodyLimit` 让读取失败 | — |
| `{"title": "A"}` | ✅ | ❌ `title: min=2`，以及其他 required 字段缺失 |
| 完整合法数据 | ✅ | ✅ |
| `{"title": "Ok", "ranking": {...}, ...}` | ✅ **未知字段被静默忽略** | 取决于其他字段 |

最后一行很重要：`MovieCreate` 中没有 `ranking` 字段，客户端提交的评级在阶段 1 就被丢弃了。**DTO 只包含允许的字段**，是防御批量赋值的第一道关卡（第 08 章 8.4）。

第三种业务校验（例如“类型 ID 必须存在”）需要查询数据库，放在 service 中完成，失败时返回业务错误（`ErrUnknownGenre` → 400）。

> **关于 `ShouldBindJSON` 内部的校验**：Gin 的绑定器在解析后会自动执行 `binding:"..."` tag 的校验。本项目的 model 中只有 `validate:"..."` tag，没有 `binding` tag，所以阶段 1 只做解析，阶段 2 由共享的 `validate` 实例完成。两种方式选一种并保持一致即可。

---

## 13.3 validator 规则速查（本项目用到的全部）

| Tag | 含义 | 使用位置 |
|---|---|---|
| `required` | 不能是零值（字符串非空、数字非 0、切片/指针非 nil） | 多处 |
| `min=N` / `max=N` | 字符串：字符数；数字：值；切片：元素个数 | `Title`、`FirstName`、`Password`、`GenreIDs` |
| `email` | 合法邮箱格式 | `UserRegister.Email`、`UserLogin.Email` |
| `url` | 合法绝对 URL | `PosterPath` |
| `unique` | 切片中的元素不能重复 | `FavouriteGenreIDs`、`GenreIDs` |
| `dive` | 进入切片/map，对每个元素应用**之后**的规则（或元素自身的 struct 规则） | `Movie.Genre`、`GenreIDs` |
| `imdb_id` | 自定义：`tt` + 7–10 位数字 | `ImdbID` |
| `youtube_id` | 自定义：11 个字符，只含字母、数字、`_`、`-` | `YouTubeID` |

### 13.3.1 自定义规则

```go
// S/validation/validation.go
var (
	imdbIDPattern    = regexp.MustCompile(`^tt\d{7,10}$`)
	youTubeIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
)

func New() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	// Use JSON field names (first_name) in error messages instead of Go field names (FirstName).
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			return f.Name
		}
		return name
	})
	mustRegister(v, "imdb_id", imdbIDPattern)
	mustRegister(v, "youtube_id", youTubeIDPattern)
	return v
}

func mustRegister(v *validator.Validate, tag string, pattern *regexp.Regexp) {
	err := v.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		return pattern.MatchString(fl.Field().String())
	})
	if err != nil {
		panic(err)
	}
}
```

- **正则表达式只编译一次**：`regexp.MustCompile` 写在包级变量中，程序启动时编译。如果写在校验函数里，每次校验都要重新编译。`MustCompile` 在正则写错时 panic——正则是写死在代码里的，写错是程序员错误，启动时立即暴露比运行时返回错误更好。
- **`mustRegister` 中的 panic** 也是同样的道理：注册失败（例如 tag 名为空）只可能是代码写错了。Go 标准库也遵循这个命名惯例：`Must` 开头的函数在失败时 panic。
- **`^` 和 `$`**：锚定开头和结尾。没有它们，`"xxtt1234567yy"` 也能匹配 `tt\d{7}`。
- **为什么校验 YouTube ID？** 这个值会被前端拼进 `<iframe src=...>` 的 URL。限制成 11 个安全字符，就不可能注入路径、查询参数或其他 URL 片段。前端在路由参数上也做了同样的检查（第 13a 章）。
- **`WithRequiredStructEnabled`**：validator v10 的推荐选项，让 `required` 作用于 struct 字段时有意义（检查是否为零值），也是 v11 的默认行为。

### 13.3.2 几个容易踩的坑

**坑 1：`required` 对切片只检查非 nil**

```go
GenreIDs []int `validate:"required,min=1,max=20,unique,dive,min=1"`
```

- JSON 中没有该字段 → nil → `required` 失败。
- JSON 中是 `[]` → 空切片（非 nil）→ `required` **通过**！所以还需要 `min=1`。

**坑 2：`dive` 前后的规则作用于不同对象**

同一个 tag 中出现了两次 `min=1`：`dive` 之前的 `min=1` 要求**切片**至少有 1 个元素；`dive` 之后的 `min=1` 要求**每个元素**（一个 int）≥ 1。

**坑 3：`required` 对整数，0 视为缺失**

`Genre.GenreID int validate:"required,min=1"` → `genre_id: 0` 会校验失败。在本项目中类型 ID 从 1 开始，这正是我们想要的。但如果你的业务允许 0，就不能用 `required`。

**坑 4：`max` 数的是字符，不是字节**

`Password string validate:"max=72"`：40 个 `é` 是 40 个字符、80 个字节，能通过校验。bcrypt 关心的是字节，所以 `AuthService.Register` 再用 `len(req.Password)`（字节数）检查一次（第 11 章 11.3.4）。

**坑 5：格式校验 ≠ 权限控制**

如果 `User` 上有 `Role string validate:"oneof=ADMIN USER"`，它只保证角色是这两个值之一——`ADMIN` 也是合法值。“谁能成为管理员”不是格式问题，而是权限问题。所以注册 DTO 中**根本没有** `Role` 字段，由服务器决定（第 08 章 8.4）。

### 13.3.3 validator 的错误信息

`validator.ValidationErrors` 的默认 `Error()` 字符串形如：

```text
Key: 'UserRegister.Password' Error:Field validation for 'Password' failed on the 'min' tag
```

它暴露了 **Go 内部的 struct 名和字段名**，而不是 API 使用的 JSON 字段名，对 API 使用者也不友好。所以本项目用 `validation.Describe` 转换：

```go
// S/validation/validation.go
// Describe turns validation errors into client-friendly details (field and rule names only, no internal structure).
func Describe(err error) []string {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return nil
	}
	details := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		rule := fe.Tag()
		if fe.Param() != "" {
			rule += "=" + fe.Param()
		}
		details = append(details, fe.Field()+": "+rule)
	}
	return details
}
```

配合 `RegisterTagNameFunc`，`fe.Field()` 返回的就是 JSON 字段名。响应示例：

```json
{"error": "validation failed", "details": ["email: email", "password: min=8"]}
```

- `errors.As(err, &verrs)`：判断错误是不是（或者包装了）`validator.ValidationErrors` 类型，如果是，就把它赋给 `verrs`。
- `fe.Tag()` 是规则名（`min`），`fe.Param()` 是规则参数（`8`）。

---

## 13.4 Go 的错误处理哲学

### 13.4.1 What：错误是值

Go 没有 `try/catch` 异常机制。错误是一个普通的返回值，类型是内置的 `error` 接口：

```go
type error interface {
	Error() string
}
```

调用者**必须显式检查**：

```go
result, err := doSomething()
if err != nil {
	// 处理错误
	return
}
// 使用 result
```

### 13.4.2 Why
- **显式**：看代码就知道哪里可能出错、如何处理。不会有“某个深层函数抛出异常，跳过了中间 5 层代码”的意外。
- **代价**：代码啰嗦，满屏的 `if err != nil`。

### 13.4.3 `panic` 何时使用？
`panic` 用于**程序员错误**或**无法恢复的状态**（如 nil 指针、数组越界、写死在代码里的正则表达式有语法错误），不应该用于正常的业务错误（如“用户不存在”）。本项目中只有 `validation.mustRegister` 主动 panic（见 13.3.1）。

### 13.4.4 错误处理的几种做法

| 做法 | 代码 | 本项目中的例子 |
|---|---|---|
| 原样返回给调用者 | `return nil, err` | `findAll`、`MovieRepository.UpdateReview` |
| 包装后返回（添加上下文） | `return nil, fmt.Errorf("look up genres: %w", err)` | services 中几乎所有错误 |
| 翻译成另一个错误 | `if errors.Is(err, mongo.ErrNoDocuments) { return ErrNotFound }` | `repository.translate`、service 中把 `ErrNotFound` 变成 `ErrMovieNotFound` |
| 转换为 HTTP 响应 | `respondError(c, err); return` | 所有 Controller 方法 |
| 有意忽略（并注释原因） | `_ = client.Disconnect(...)` | `database.Connect` 中 Ping 失败后的清理；`Login` 中的假 bcrypt 比对 |
| 吞掉（最糟） | `if err != nil { return nil }` | 本项目中没有——它会让调用方在 nil 上继续执行，最后在毫不相关的地方崩溃（第 07 章调试挑战 7.6） |

**`_ =` 显式忽略**：只写 `client.Disconnect(...)` 不接收返回值，编译器不会报错，但读代码的人不知道你是忘了还是有意的。写成 `_ = ...` 明确表示“我知道它会返回错误，我选择忽略”，静态检查工具（如 `errcheck`）也会认可这种写法。

### 13.4.5 错误包装与判断（Go 1.13+）

```go
// 包装：%w 保留原始错误
err := fmt.Errorf("find user %s: %w", id, mongo.ErrNoDocuments)
fmt.Println(err)  // find user abc: mongo: no documents in result

// 判断：沿着包装链查找
errors.Is(err, mongo.ErrNoDocuments)   // true
err == mongo.ErrNoDocuments            // false！

// 提取特定类型
var ves validator.ValidationErrors
errors.As(err, &ves)
```

本项目中所有的错误比较都使用 `errors.Is`。service 用 `%w` 包装了错误（例如 `"rotate refresh token: %w"`），如果 controller 用 `==` 比较，任何被包装过的业务错误都会匹配失败，变成 500。

**包装时写什么？** 写“我当时在做什么”，而不是“出错了”：`"look up genres: %w"`、`"store refresh token: %w"`。层层包装之后，日志中的一行错误就是一条完整的调用路径：

```text
internal error on POST /auth/register: store refresh token: server selection error: ...
```

---

## 13.5 本项目的错误响应格式

### 13.5.1 统一的格式

| 情况 | 格式 | 状态码 |
|---|---|---|
| 请求体无法解析 | `{"error": "invalid request body"}` | 400 |
| 校验失败 | `{"error": "validation failed", "details": [...]}` | 400 |
| 业务错误 | `{"error": "<业务错误的消息>"}` | 由映射表决定（第 09 章 9.5.2） |
| 中间件拒绝 | `{"error": "missing bearer token"}` 等 | 401 / 403 / 429 |
| 内部错误 | `{"error": "internal server error"}` | 500 |

成功响应直接返回资源本身（电影、用户信息）或 `{"message": "logged out"}`。

**一个键名、一种结构**：前端只需要 `err.response?.data?.error` 就能拿到任何错误的描述。

**行业标准**：RFC 9457 “Problem Details for HTTP APIs” 定义了一种更完整的标准错误格式（`type`、`title`、`status`、`detail`），适合需要对外公开、供第三方调用的 API。

### 13.5.2 前端如何读取错误

```tsx
// C/src/components/register/Register.tsx（handleSubmit 节选）
} catch (err) {
    if (axios.isAxiosError(err) && err.response?.data?.error) {
        setError(err.response.data.error);         // 显示后端的 error 字段，例如 “email is already registered”
    } else {
        setError("Registration failed. Please try again.");
    }
}
```

```tsx
// C/src/components/login/Login.tsx（handleSubmit 节选）
try {
  const response = await axiosClient.post<User>("/auth/login", { email, password });
  // Access token stays in memory (auth state); refresh token was set as an HttpOnly cookie
  setAuth(response.data);
  navigate(from, { replace: true });
} catch (err) {
  if (axios.isAxiosError(err) && err.response?.status === 429) {
    setError("Too many attempts. Please wait a minute and try again.");
  } else {
    setError("Invalid email or password");
  }
}
```

- axios 对**非 2xx** 状态码会**抛出异常**，进入 `catch`。所以 `try` 块中只需要处理成功的情况——在 `try` 里检查 `response.data.error` 是没有意义的，成功响应中不会有 `error` 字段。
- 按状态码区分提示：429 告诉用户稍后再试，其他情况统一提示“邮箱或密码错误”（与后端一样，不区分是哪个错了）。
- **不要把响应打印到控制台**：登录响应中有 Access Token。`console.log(response.data)` 会把它留在开发者工具里，任何人看一眼屏幕或截图就能拿到，违背了“AT 只放在内存中”的设计。

---

## 13.6 内部错误不应直接暴露

```go
// S/controllers/response.go
log.Printf("internal error on %s %s: %v", c.Request.Method, c.FullPath(), err)
c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
```

**为什么不把 `err.Error()` 直接返回给客户端？** 内部错误信息可能包含数据库地址、集合名、查询条件、内部路径、第三方 API 的错误详情（甚至部分 API Key）。这些信息能帮助攻击者了解系统内部结构（OWASP 称为 “Security Misconfiguration / Information Disclosure”）。

**正确做法**：**详细错误写入服务器日志，客户端只收到一个通用描述。** 业务错误的消息是我们自己写的、专门给客户端看的，所以可以返回；其他错误一律只返回 `internal server error`。

**日志中用 `c.FullPath()` 而不是 `c.Request.URL.Path`**：前者是路由模板（`/movies/:imdb_id`），后者是实际路径（`/movies/tt0111161`）。按模板记录，方便在日志系统中按接口聚合统计。

---

## 13.7 如何系统性地避免“忘记 return”

`c.JSON` 只写响应，**不会**终止函数。在错误分支中写了响应却忘了 `return`，函数会继续执行，用无效的数据再写一次响应，甚至在 nil 值上 panic（第 09 章 9.6.1）。本项目用几种方式降低这种风险：

1. **把“写错误响应”收拢到少数几个函数中**：`bindJSON` 返回 `bool`，`respondError` 是唯一写 500 的地方。每个错误分支都是同样的两行：`respondError(c, err)` + `return`，一眼就能看出有没有漏。
2. **中间件使用 `AbortWithStatusJSON`**：写响应和中止处理链一步完成。
3. **让 service 返回 `error`、controller 统一转换**：业务代码中不出现 `c.JSON`，自然也就没有“写了响应忘了 return”的机会。
4. **写测试覆盖错误分支**（第 15 章）：断言错误响应的 JSON 能被正常解析——两段 JSON 拼接在一起的响应体无法解析，测试就会失败。

另一种更彻底的模式是让 Handler 的签名变成 `func(c *gin.Context) error`，由一个包装器统一把返回的错误写成响应。代价是要为每个 Handler 包一层适配器。

---

## 13.8 其他输入限制

| 风险 | 本项目的措施 |
|---|---|
| 超大请求体（攻击者发送 1 GB 的 JSON，耗尽内存） | `middleware.BodyLimit(1 << 20)` |
| 超长密码（bcrypt 只接受 ≤72 字节） | `max=72` + 按字节检查，返回 400 |
| 影评长度无限制（会被原样发送给按 token 计费的 OpenAI） | `ReviewUpdate.AdminReview` 的 `max=2000` |
| 邮箱大小写（`A@x.com` 与 `a@x.com` 被视为不同用户） | `NormalizeEmail` + 唯一索引 |
| 列表无上限（一次提交上万个类型 ID） | `max=20` |
| 字符串字段无上限 | 姓名 `max=100`、标题 `max=500`、URL `max=2048`、邮箱 `max=254` |
| 未知字段 | 被静默忽略（DTO 中不存在的字段不会生效）。如需严格拒绝，可以使用 `json.Decoder.DisallowUnknownFields()` |

**每个字符串字段都应该有长度上限**：没有上限的字段，就是一个可以写入任意大数据的入口——占用数据库空间、拖慢查询、在前端渲染时撑破页面。

---

## 本章练习

> 答案见 [第 20 章](20-exercise-solutions.md#第-13-章)

**关键概念回顾**
- 为什么前端校验不能替代后端校验？
- `required` 对 string、int、slice 分别意味着什么？
- `errors.Is` 和 `==` 比较错误有什么区别？

**面试题 13.1** “Go 为什么不用异常？你怎么看 `if err != nil` 的啰嗦？”

**面试题 13.2** “`%w` 和 `%v` 在 `fmt.Errorf` 中有什么区别？”

**面试题 13.3** “API 返回 500 时，错误响应中应该包含什么、不应该包含什么？”

**练习 13.4** 不看源码，实现 `validation.New`（含 `RegisterTagNameFunc` 和 `imdb_id` 规则）和 `validation.Describe`，并写测试断言：`{"first_name":"A"}` 的错误详情中包含 `first_name: min=2`。

**练习 13.5** 为 `ReviewUpdate` 增加一条自定义规则 `notblank`：影评不能全部由空白字符组成。写测试验证 `"   "` 被拒绝。

**练习 13.6** 在 `bindJSON` 中改用 `json.Decoder` + `DisallowUnknownFields()`，让包含未知字段的请求返回 400。思考：这对前端的向后兼容性有什么影响？

**调试挑战 13.7** 用 curl 向 `/auth/register` 分别发送：(a) 100 个字母组成的密码；(b) 40 个 `é` 组成的密码（其他字段合法）。你得到什么状态码和错误信息？沿着代码解释两者为什么不同。

**调试挑战 13.8** 以管理员身份向 `POST /movies` 发送 `"poster_path": "not-a-url"` 和 `"youtube_id": "abc"`，观察 `details` 字段的内容。

**理解检验 13.9** 在 `Login.tsx` 中，如果密码错误，代码会走到哪一行？如果连续输错 11 次呢？
