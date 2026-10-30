# 第 08 章　Model 与类型

> 本章目标：掌握 Go 的 struct 和 struct tag，理解一个 struct 如何同时服务于 JSON、BSON 和校验三种用途，以及前端 TypeScript 类型如何与之对应。

---

## 8.1 Go struct 基础

### What
`struct` 是 Go 中把多个字段组合在一起的**复合类型**，类似其他语言中的“类”（但没有继承）。

```go
type Genre struct {
	GenreID   int
	GenreName string
}
```

### 创建与使用
```go
g := Genre{GenreID: 1, GenreName: "Drama"}   // 字面量
fmt.Println(g.GenreName)                    // 访问字段

var empty Genre                             // 零值：GenreID=0, GenreName=""
p := &Genre{GenreID: 2}                     // 取地址，得到 *Genre（指针）
p.GenreName = "Comedy"                      // Go 自动解引用，无需写 (*p).GenreName
```

**零值（zero value）** 是 Go 的重要概念：任何变量声明后都有一个确定的初始值——数字是 `0`，字符串是 `""`，指针、切片、map 是 `nil`，struct 是所有字段都取零值。这意味着 Go 中不存在“未初始化的变量”。

---

## 8.2 Struct Tag：一个字段，三种身份

### What
struct tag 是写在字段后面、用反引号包围的**元数据字符串**。Go 编译器本身不理解它，而是由各个库通过反射读取。

```go
// S/models/movie_model.go
type Genre struct {
	GenreID   int    `bson:"genre_id" json:"genre_id" validate:"required,min=1"`
	GenreName string `bson:"genre_name" json:"genre_name" validate:"required,min=2,max=100"`
}
```

拆解 `GenreName` 的 tag：

| Key | Value | 读取者 | 含义 |
|---|---|---|---|
| `bson` | `genre_name` | mongo-driver | 存入 MongoDB 时字段名为 `genre_name` |
| `json` | `genre_name` | `encoding/json`、Gin | HTTP JSON 中字段名为 `genre_name` |
| `validate` | `required,min=2,max=100` | validator | 必填，长度 2–100 |

### Why
Go 的命名惯例是 `PascalCase`（`GenreName`），而 JSON/数据库的惯例常常是 `snake_case`（`genre_name`）。Tag 让两边各自遵守惯例。

### 如果不写 tag 会怎样？
- `encoding/json`：使用字段名原样 `"GenreName"`（解析时大小写不敏感）。
- mongo-driver：使用字段名的**全小写** `"genrename"`。
- 前后端字段名就对不上了。

### 语法注意事项
- tag 中各项用**空格**分隔，`key:"value"` 中冒号两边**不能有空格**。
- 写错格式（如 `json: "x"`）不会编译报错，只会**静默失效**。`go vet` 能检查出来（它的 `structtag` 检查）。

---

## 8.3 电影模型逐字段讲解

```go
// S/models/movie_model.go
// NotRankedValue is the ranking_value for "not ranked yet". Recommendations sort by ranking_value ascending, so it must be the largest value.
const NotRankedValue = 999

type Ranking struct {
	RankingValue int    `bson:"ranking_value" json:"ranking_value" validate:"required"`
	RankingName  string `bson:"ranking_name" json:"ranking_name" validate:"required"`
}

type Movie struct {
	ID          bson.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	ImdbID      string        `bson:"imdb_id" json:"imdb_id" validate:"required,imdb_id"`
	Title       string        `bson:"title" json:"title" validate:"required,min=2,max=500"`
	PosterPath  string        `bson:"poster_path" json:"poster_path" validate:"required,url,max=2048"`
	YouTubeID   string        `bson:"youtube_id" json:"youtube_id" validate:"required,youtube_id"`
	Genre       []Genre       `bson:"genre" json:"genre" validate:"required,min=1,dive"`
	AdminReview string        `bson:"admin_review" json:"admin_review"`
	Ranking     Ranking       `bson:"ranking" json:"ranking"`
}
```

| 字段 | 类型 | 要点 |
|---|---|---|
| `ID` | `bson.ObjectID` | 映射到 MongoDB 的 `_id`。`omitempty`：如果是零值（全 0 的 12 字节），插入时**不写这个字段**。本项目的 `Create` 方法会先赋值 `bson.NewObjectID()`，这样插入后立刻就知道新文档的 ID，可以直接返回给客户端。JSON 中名为 `id`。 |
| `ImdbID` | `string` | 业务主键，查询、更新都用它；有唯一索引。`imdb_id` 是自定义校验规则（`tt` + 7–10 位数字，第 13 章）。 |
| `PosterPath` | `string` | `url` 要求是合法的绝对 URL；`max=2048` 防止超长字符串。 |
| `YouTubeID` | `string` | `youtube_id` 自定义规则：恰好 11 个字符，只允许字母、数字、`_`、`-`。前端也用同样的规则检查路由参数。 |
| `Genre` | `[]Genre` | **切片**（动态数组），嵌套 struct。`min=1` 要求至少一个；`dive` 表示“进入切片，对每个元素执行 `Genre` 自己的校验规则”。 |
| `AdminReview` | `string` | 没有 `validate`，可以为空。 |
| `Ranking` | `Ranking` | 嵌套 struct（值类型，不是指针）。 |

### 常量代替魔法数字

`999` 在代码中出现在三个地方：给 LLM 的候选列表要排除它、新电影的初始评级是它、`matchRanking` 不接受它。把它定义成 `models.NotRankedValue`，并在注释中写明**为什么必须是最大值**，读代码的人就不需要猜测这个数字的含义；将来要改，也只改一处。

### 请求 DTO：`MovieCreate` 和 `ReviewUpdate`

`Movie` 是数据库文档，但添加电影的请求体并不是 `Movie`：

```go
// MovieCreate is the request DTO for adding a movie. Review and ranking come from the review flow; new movies always start as not ranked.
type MovieCreate struct {
	ImdbID     string `json:"imdb_id" validate:"required,imdb_id"`
	Title      string `json:"title" validate:"required,min=2,max=500"`
	PosterPath string `json:"poster_path" validate:"required,url,max=2048"`
	YouTubeID  string `json:"youtube_id" validate:"required,youtube_id"`
	GenreIDs   []int  `json:"genre_ids" validate:"required,min=1,max=20,unique,dive,min=1"`
}

type ReviewUpdate struct {
	AdminReview string `json:"admin_review" validate:"required,min=1,max=2000"`
}

type ReviewUpdateResponse struct {
	AdminReview string  `json:"admin_review"`
	Ranking     Ranking `json:"ranking"`
}
```

- `MovieCreate` **没有** `AdminReview` 和 `Ranking`：评级只能由“写影评 → LLM 分类”这条流程产生，不能在添加时随便填一个“Excellent”。
- 类型只提交 `genre_ids`，名称由服务器从 `genres` 集合中查出（第 07 章 7.2.2）。
- 校验规则读法：`required,min=1,max=20,unique,dive,min=1` = 必填、1 到 20 个、不能重复、**进入每个元素**后要求每个 ID ≥ 1。`dive` 之前的规则作用于切片本身，之后的作用于元素。
- `ReviewUpdate` 限制影评最长 2000 字符：这段文字会被发送给按 token 计费的 OpenAI。

### 关于 `required` 的一个陷阱

```go
RankingValue int `validate:"required"`
```

对于 `int`，`required` 的含义是“不是零值”。所以 **`ranking_value: 0` 会被判定为校验失败**。在本项目的约定中，评级值从 1 开始、999 表示未评级，因此不会遇到这个问题。但如果你设计的评级允许 0，就会掉进这个坑。

> **注意**：`Ranking` 作为嵌套 struct（非切片），validator 会**自动**递归校验其字段（`dive` 只用于切片和 map）。

---

## 8.4 用户模型：一个文档，三个 DTO

```go
// S/models/user_model.go
const (
	RoleAdmin = "ADMIN"
	RoleUser  = "USER"
)

// User is a document in the users collection. It stays inside the server: it is never bound from a request body or returned as a response.
// The user ID is MongoDB's _id (exposed as a hex string).
type User struct {
	ID               bson.ObjectID `bson:"_id,omitempty" json:"-"`
	FirstName        string        `bson:"first_name" json:"-"`
	LastName         string        `bson:"last_name" json:"-"`
	Email            string        `bson:"email" json:"-"`
	PasswordHash     string        `bson:"password" json:"-"`
	Role             string        `bson:"role" json:"-"`
	CreatedAt        time.Time     `bson:"created_at" json:"-"`
	UpdatedAt        time.Time     `bson:"updated_at" json:"-"`
	RefreshTokenHash string        `bson:"refresh_token_hash,omitempty" json:"-"`
	FavouriteGenres  []Genre       `bson:"favourite_genres" json:"-"`
}
```

### 为什么 `User` 的每个字段都是 `json:"-"`？

`json:"-"` 让字段**完全不参与** JSON 编码和解码。`User` 是数据库文档，其中有 `PasswordHash`、`RefreshTokenHash`、`Role` 这样的敏感字段，它们只应该在服务器内部流动：

- **不能从请求体绑定**：如果把请求体直接绑定到 `User`，客户端就可以提交本该由服务器决定的字段——例如在注册时提交 `"role":"ADMIN"` 把自己变成管理员。这就是经典的**批量赋值（Mass Assignment）**漏洞。
- **不能直接作为响应**：如果某个 Handler 不小心写了 `c.JSON(200, user)`，密码哈希就会发给前端。

全部标为 `json:"-"` 是一道**保险**：即使有人误用，结果也只是一个空对象 `{}`，而不是泄露数据。字段名 `PasswordHash`（而不是 `Password`）也在提醒读者：这里存的是哈希，不是明文。（`bson` tag 仍然是 `password`，与已有数据保持兼容。）

### 专用 DTO（Data Transfer Object）

每个接口的输入和输出都有自己的结构体，只包含该接口允许的字段：

```go
// UserRegister is the registration request DTO. It deliberately has no Role or other server-controlled fields; if a client sends them they are ignored.
// Favourite genres are submitted as a list of genre_id values; the server looks up their names in the genres collection.
type UserRegister struct {
	FirstName         string `json:"first_name" validate:"required,min=2,max=100"`
	LastName          string `json:"last_name" validate:"required,min=2,max=100"`
	Email             string `json:"email" validate:"required,email,max=254"`
	Password          string `json:"password" validate:"required,min=8,max=72"`
	FavouriteGenreIDs []int  `json:"favourite_genre_ids" validate:"required,min=1,max=20,unique,dive,min=1"`
}

type UserLogin struct {
	Email    string `json:"email" validate:"required,email,max=254"`
	Password string `json:"password" validate:"required,max=72"`
}

// UserResponse is the user data sent to the frontend: no password hash and no refresh token.
type UserResponse struct {
	UserID          string  `json:"user_id"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	Email           string  `json:"email"`
	Role            string  `json:"role"`
	AccessToken     string  `json:"access_token"`
	FavouriteGenres []Genre `json:"favourite_genres"`
}
```

| DTO | 方向 | 设计要点 |
|---|---|---|
| `UserRegister` | 请求 → 服务器 | **没有 `Role` 字段**。客户端即使提交 `"role":"ADMIN"`，`ShouldBindJSON` 找不到对应字段，直接丢弃。`AuthService.Register` 构造 `User` 时写死 `Role: models.RoleUser`。 |
| `UserLogin` | 请求 → 服务器 | 密码只检查 `max=72`、不检查 `min`：老用户可能是在旧规则（最少 6 位）下注册的，登录不能拒绝他们。 |
| `UserResponse` | 服务器 → 前端 | 不含密码哈希和 Refresh Token（后者放在 HttpOnly Cookie 里）。由 `AuthController.writeSession` 从 `User` 手动构造。 |

**手动复制字段是不是很啰嗦？** 是的，但这种啰嗦是有意为之：每一个对外暴露的字段，都是写代码的人**明确选择**的。新增一个数据库字段（比如 `last_login_ip`）时，它不会自动出现在任何响应中。安全领域把这叫做“默认拒绝”（deny by default）。

### 角色常量

`models.RoleAdmin`、`models.RoleUser` 让角色名只在一处定义。写成字符串字面量 `"ADMIN"` 的话，拼错成 `"ADMN"` 编译器发现不了；写成常量，拼错就是编译错误。

### `time.Time` 的序列化
- JSON：编码为 RFC 3339 字符串，如 `"2026-10-30T08:00:00Z"`。
- BSON：编码为 BSON Date 类型（毫秒精度的 UTC 时间）。

---

## 8.5 指针 vs 值：为什么函数参数常用 `*T`？

本项目中到处都是指针：`*mongo.Client`、`*gin.Context`、`*services.AuthService`、`&req`。

| | 值 `T` | 指针 `*T` |
|---|---|---|
| 传参时 | 复制整个 struct | 只复制 8 字节的地址 |
| 函数内修改 | 不影响调用方 | 影响调用方 |
| 可以为 nil | 否 | 是 |

**为什么 `c.ShouldBindJSON(&req)` 要传 `&req`？** 因为 `ShouldBindJSON` 需要**往 `req` 里写数据**。如果传值，函数修改的只是副本，调用方的 `req` 仍是零值。

同理：`cursor.All(ctx, &items)`、`Decode(&user)`，以及 `UserRepository.Create(ctx, user)` 接收 `*models.User`——它要把生成的 `ID` 回填给调用方。

**规则**：凡是“请函数帮我填充数据”的场景，都要传指针。

---

## 8.6 前端类型：与后端对应

```ts
// C/src/types/movie.ts
// Mirrors Server/MagicStreamMoviesServer/models/movie_model.go
export interface Genre {
  genre_id: number
  genre_name: string
}

export interface Ranking {
  ranking_value: number
  ranking_name: string
}

export interface MovieModel {
  id: string
  imdb_id: string
  title: string
  poster_path: string
  youtube_id?: string
  genre?: Genre[]
  admin_review?: string
  ranking?: Ranking
}

// Response of PATCH /movies/:imdb_id/review
export interface ReviewUpdateResponse {
  admin_review: string
  ranking: Ranking
}
```

```ts
// C/src/types/user.ts
// Mirrors UserResponse in Server/MagicStreamMoviesServer/models/user_model.go
export interface User {
  user_id: string
  first_name: string
  last_name: string
  email: string
  role: string
  access_token: string
  favourite_genres: Genre[]
}
```

### 对照表

| Go 类型 | JSON | TypeScript |
|---|---|---|
| `int` | number | `number` |
| `string` | string | `string` |
| `bson.ObjectID` | 24 位十六进制字符串 | `string` |
| `[]Genre` | 数组或 `null`（nil 切片） | `Genre[]` |
| `time.Time` | RFC 3339 字符串 | `string` |
| 嵌套 struct | 对象 | `interface` |

### 要点
- **TypeScript 的类型只在编译期存在**。运行时 axios 收到什么就是什么，TS 不会校验。`axiosClient.get<MovieModel[]>("/movies")` 中的泛型只是“告诉编译器我相信返回的是这个类型”。
- `?` 表示可选。前端把 `youtube_id`、`genre` 等标为可选，是一种**防御性**写法：即使后端数据不完整，组件也会用 `movie.genre?.map(...)` 安全处理。
- 文件顶部的注释 “Mirrors …” 说明了**手动同步**的约定。后端改了字段，前端必须手动跟着改。

### 替代方案：自动生成类型
| 方案 | 说明 |
|---|---|
| 手动维护（本项目） | 简单，但容易不同步 |
| OpenAPI（Swagger）+ 代码生成 | 后端写 API 规范，自动生成 TS 类型和客户端 |
| `tygo` 等工具 | 直接从 Go struct 生成 TS interface |
| 运行时校验（`zod`） | 在前端校验收到的 JSON，类型和校验合二为一 |

### Go nil 切片 → JSON `null`
在 Go 中，`var movies []models.Movie` 声明的是 **nil 切片**，JSON 编码为 **`null`**，不是 `[]`。前端拿到 `null` 后调用 `.map` 就会报错。所以后端返回列表的地方都显式构造空切片：`findAll` 中的 `items := []T{}`、`MovieService.Recommended` 中的 `return []models.Movie{}, nil`。前端的 `setMovies(response.data ?? [])`（`Home.tsx`）则是另一层防御。两端都防御，是因为你无法保证另一端永远不出错。

---

## 8.7 Go Interface

### 标准库和第三方库中的 interface

| Interface | 被谁实现 | 项目中的使用 |
|---|---|---|
| `error` | 任何有 `Error() string` 方法的类型 | 所有函数的错误返回值 |
| `context.Context` | `c.Request.Context()`、`context.WithTimeout` 的返回值 | 所有数据库和 OpenAI 调用的第一个参数 |
| `jwt.Claims` | `*tokens.Claims`（通过嵌入 `jwt.RegisteredClaims` 获得方法） | `jwt.NewWithClaims`、`jwt.ParseWithClaims` |
| `llms.Model` | langchaingo 的 `*openai.LLM` | `OpenAIClassifier.llm` 字段 |

### 项目自己定义的 interface

```go
// S/services/movie_service.go
// ReviewClassifier assigns a review to one of the ranking names in labels. OpenAI implements it in production; tests use a fake.
type ReviewClassifier interface {
	Classify(ctx context.Context, review string, labels []string) (string, error)
}
```

`services.OpenAIClassifier` 和测试用的 `memstore.Classifier` 都有这个方法，所以都满足 `ReviewClassifier`。**Go 的 interface 是隐式实现的**：不需要写 `implements`，只要方法签名匹配，就自动满足。这是“鸭子类型”的静态版本。

Go 社区有一句常被引用的建议：**“接受接口，返回结构体”**（accept interfaces, return structs）。`NewMovieService` 的参数是接口（`MovieStore`、`ReviewClassifier`……），返回的是具体的 `*MovieService`。接口由**使用方**定义，并且只包含使用方真正需要的方法——`FavouriteGenreReader` 只有一个方法，因为 `MovieService` 只需要这一个。

### 嵌入（Embedding）
```go
// S/tokens/tokens.go
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims    // 没有字段名 → 嵌入
}
```
嵌入后，`RegisteredClaims` 的字段（`ExpiresAt`、`Issuer`……）和方法（`GetExpirationTime()` 等）都被“提升”到 `Claims` 上，可以直接 `claims.ExpiresAt` 访问。因此 `*Claims` 自动满足 `jwt.Claims` interface。

接口也可以嵌入接口：

```go
// S/services/movie_service.go
type CatalogStore interface {
	GenreLookup   // 包含 GenresByIDs
	Genres(ctx context.Context) ([]models.Genre, error)
	Rankings(ctx context.Context) ([]models.Ranking, error)
}
```

**JSON tag 决定载荷中的字段名**：`Claims` 的自定义字段带有 json tag，所以 JWT 载荷中是简短的 `uid`、`role`。载荷只放鉴权需要的信息；姓名、邮箱等由接口响应提供，不放进 Token（Token 会随每个请求发送，越小越好，而且载荷只是 Base64 编码、任何人都能解码）。第 11 章会展示解码后的 JWT。

---

## 本章练习

> 答案见 [第 20 章](20-exercise-solutions.md#第-08-章)

**关键概念回顾**
- struct tag 由谁读取？写错格式会怎样？
- 为什么 `ShouldBindJSON` 必须传指针？
- `omitempty` 在 `_id` 字段上起什么作用？

**面试题 8.1** “什么是 Mass Assignment 漏洞？你的项目是怎么避免的？”

**面试题 8.2** “Go 的 nil 切片和空切片在 JSON 编码时有什么区别？”

**面试题 8.3** “Go 是如何实现 interface 的？和 Java 有什么不同？”

**练习 8.4** 不看源码，写出 `UserRegister` DTO，以及从它构造 `models.User` 的 `AuthService.Register`（在你的重建项目中完成）。要求：角色由服务器决定、邮箱统一小写、类型名称从数据库查出。

**练习 8.5** 用 <https://jwt.io> 或 `echo <payload> | base64 -d` 解码一个本项目生成的 Access Token，列出载荷中所有字段名。

**调试挑战 8.6** 把 `Movie.Title` 的 tag 改成 `` `bson:"title" json: "title"` ``（冒号后多一个空格），运行 `go vet ./...`。然后调用 `GET /movies`，返回的 JSON 中标题字段叫什么？最后把 `User` 的某个字段从 `json:"-"` 改成 `json:"password"`，并在某个 Handler 中临时写 `c.JSON(200, user)`，观察会泄露什么。（验证后记得改回来。）

**理解检验 8.7** 如果后端把 `poster_path` 改名为 `poster_url`，前端会出现编译错误吗？运行时会发生什么？
