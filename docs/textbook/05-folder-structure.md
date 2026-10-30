# 第 05 章　文件夹结构与 Go Package 设计

> 本章目标：理解项目每个目录的职责，掌握 Go 的 package 规则（导出、可见性、命名），并能自己设计一个合理的目录结构。

---

## 5.1 完整目录树（只列被 Git 跟踪的文件）

```text
magic-stream-movies/
├── .gitignore
├── .github/workflows/ci.yml              ← CI：后端测试 + 前端 lint/构建
├── docker-compose.yml                    ← 本地一键启动 MongoDB + 后端
│
├── Server/MagicStreamMoviesServer/       ← Go 后端（一个 module）
│   ├── .env.example                      ← 配置模板（真正的 .env 不提交）
│   ├── Dockerfile / .dockerignore
│   ├── go.mod / go.sum
│   ├── main.go                           ← package main：组装各层、启动 HTTP Server
│   ├── config/
│   │   └── config.go (+ _test)           ← 读取并校验环境变量
│   ├── database/
│   │   └── database_connection.go        ← 连接 MongoDB、创建索引
│   ├── models/
│   │   ├── movie_model.go                ← 数据库文档 + 请求/响应 DTO
│   │   └── user_model.go
│   ├── repository/                       ← 数据访问层：所有 MongoDB 查询
│   │   ├── repository.go                 ← ErrNotFound / ErrDuplicate
│   │   ├── user_repository.go
│   │   ├── movie_repository.go
│   │   ├── catalog_repository.go         ← genres + rankings
│   │   └── repository_test.go            ← 集成测试（需要真实 MongoDB）
│   ├── services/                         ← 业务层
│   │   ├── errors.go                     ← 业务错误
│   │   ├── auth_service.go
│   │   ├── movie_service.go
│   │   ├── openai_classifier.go          ← ReviewClassifier 的 OpenAI 实现
│   │   └── services_test.go
│   ├── controllers/                      ← HTTP 层
│   │   ├── response.go                   ← 错误映射、bindJSON
│   │   ├── user_controller.go            ← AuthController
│   │   └── movie_controller.go           ← MovieController
│   ├── middleware/
│   │   ├── auth.go                       ← Auth、RequireRole
│   │   └── security.go                   ← SecurityHeaders、BodyLimit、RateLimit
│   ├── routes/
│   │   ├── routes.go                     ← 路由组
│   │   └── routes_test.go                ← HTTP 级别的测试
│   ├── tokens/
│   │   └── tokens.go (+ _test)           ← 签发/校验 JWT
│   ├── validation/
│   │   └── validation.go                 ← 共享 validator + 自定义规则
│   ├── internal/memstore/
│   │   └── memstore.go                   ← 测试用的内存假实现
│   └── cmd/seed/
│       ├── main.go                       ← 种子数据命令
│       └── data/{genres,rankings,movies}.json
│
└── Client/magic-stream-client/           ← React 前端（一个 npm 项目）
    ├── .env.example
    ├── index.html
    ├── package.json / package-lock.json
    ├── vite.config.ts / tsconfig*.json / eslint.config.js
    ├── public/favicon.svg
    └── src/
        ├── main.tsx                      ← React 入口
        ├── App.tsx                       ← 页面路由表
        ├── index.css                     ← 全局样式与设计 Token
        ├── brand.ts                      ← 品牌常量
        ├── api/axiosConfig.ts            ← HTTP 客户端
        ├── context/                      ← 全局状态
        │   ├── AuthContext.ts
        │   └── AuthProvider.tsx
        ├── hook/                         ← 自定义 Hook
        │   ├── useAuth.tsx
        │   └── useAxiosPrivate.tsx
        ├── types/                        ← TypeScript 类型
        │   ├── movie.ts
        │   └── user.ts
        └── components/                   ← UI 组件（每个组件一个目录）
            ├── RequiredAuth.tsx          ← 路由守卫
            ├── auth/AuthShell.tsx
            ├── brand/{Icons,Logo}.tsx
            ├── header/Header.tsx
            ├── footer/Footer.tsx
            ├── home/Home.tsx
            ├── login/Login.tsx
            ├── register/Register.tsx
            ├── movies/Movies.tsx
            ├── movie/Movie.tsx
            ├── recommended/Recommended.tsx
            ├── review/Review.tsx
            └── stream/StreamMovie.tsx
```

---

## 5.2 Go Package 基础

### 5.2.1 What：什么是 package？

在 Go 中，**一个目录 = 一个 package**。同一目录下的所有 `.go` 文件必须声明相同的 `package` 名，它们共享同一个命名空间——可以直接互相调用函数，无需 import。

例如，`controllers/response.go`、`controllers/movie_controller.go` 和 `controllers/user_controller.go` 都以 `package controllers` 开头。因此 `response.go` 中定义的未导出函数：

```go
// S/controllers/response.go
func respondError(c *gin.Context, err error) { ... }
func bindJSON(c *gin.Context, v *validator.Validate, dst any) bool { ... }
```

可以在另外两个文件中直接调用，不需要 import，也不需要导出。按“公共辅助代码一个文件、每类资源一个文件”拆分，是同一个包内组织代码的常见方式。

### 5.2.2 导出规则：大写字母开头 = 公开

Go 没有 `public`/`private` 关键字，而是用**首字母大小写**决定可见性：

| 标识符 | 可见性 | 本项目中的例子 |
|---|---|---|
| 大写开头 | **导出**（exported），其他包可访问 | `database.Connect`、`models.Movie`、`tokens.Manager`、`middleware.UserID` |
| 小写开头 | **未导出**，只能在本包内使用 | `tokens.Manager` 的 `accessKey` 字段、`middleware.ctxUserID`、`controllers.respondError` |

这个规则同样适用于 **struct 字段**。这一点非常关键：

```go
type User struct {
    FirstName string `json:"first_name"`  // 大写：encoding/json 能读写
    password  string                      // 小写：json、bson 库都看不见它！
}
```

JSON 和 BSON 库通过**反射（reflection）**读取字段，而反射只能访问导出字段。所以 `models` 中所有需要序列化的字段都必须大写开头。

> **为什么 `tokens.Manager` 的密钥字段是小写？** `accessKey` 和 `refreshKey` 是 JWT 签名密钥，只应在 `tokens` 包内部使用。其他包只能调用 `Generate`、`ValidateAccess` 等导出方法，从语言层面就无法读取密钥。同理，`middleware` 把 Context 的键名 `ctxUserID` 设为未导出，其他包只能通过 `middleware.UserID(c)` 读取——键名以后改了，调用方也不受影响。这是**最小暴露原则**的体现。

### 5.2.3 import 路径 vs package 名

```go
import "github.com/ConnieLWY/magic-stream-movies/Server/MagicStreamMoviesServer/database"
// 使用时：database.Connect(ctx, uri)
```

- **import 路径**：module path + 目录路径。
- **package 名**：文件里 `package xxx` 声明的名字，约定与目录名一致。
- 可以起别名：`jwt "github.com/golang-jwt/jwt/v5"`（`S/tokens/tokens.go`）。这里的别名其实和包名相同，作用是让读者一眼看出 `/v5` 这个路径对应的包叫 `jwt`。别名的另一个用途是解决同名冲突，例如同时导入两个都叫 `rand` 的包。原则是：**同一个包在全项目中使用统一的名字**，不要这里叫 `dotenv`、那里叫 `godotenv`。

### 5.2.4 Go 的命名惯例

| 规则 | 本项目中的例子 | 反例 |
|---|---|---|
| 包名小写、单个单词、无下划线 | `controllers`、`repository`、`tokens` | `token_utils`、`TokenUtils` |
| 文件名小写，可以用下划线 | `movie_controller.go` | `MovieController.go` |
| 局部变量用驼峰 `camelCase` | `genreNames`、`refreshToken` | `favourite_genres` |
| 缩写词保持全大写：`ID`、`URL`、`HTTP` | `UserID`、`ImdbID`、`FavouriteGenreIDs` | `UserId` |
| 包名不要与内容重复（调用时会带上包名） | `middleware.Auth`、`tokens.Hash` | `middleware.AuthMiddleware` |
| 不要用内置函数名当变量名 | `clone := *user` | `copy := *user`（遮蔽了内置的 `copy`） |

这些规则不影响程序运行，但 `go vet`、`staticcheck` 等工具会提示，团队协作时也会在 Code Review 中被指出。遵守惯例的代码，其他 Go 程序员读起来几乎不需要适应。

---

## 5.3 每个后端目录的职责

| 目录 | 职责 | 应该放什么 | 不应该放什么 |
|---|---|---|---|
| `main.go` | 组装（Composition Root） | 读取配置、创建依赖、注册路由、启动和关闭服务 | 业务逻辑 |
| `config/` | 配置 | 读取、校验环境变量 | 读取后的任何业务判断 |
| `database/` | 数据库基础设施 | 连接、创建索引 | 具体的业务查询 |
| `models/` | 数据结构 | struct 定义、校验规则、常量 | 访问数据库的方法 |
| `repository/` | 数据访问 | MongoDB 查询、驱动错误翻译 | 业务规则（例如“角色必须是 USER”） |
| `services/` | 业务逻辑 | 规则、流程、调用外部服务 | HTTP 状态码、Cookie、`gin.Context` |
| `controllers/` | HTTP 处理 | 解析请求、调用 service、写响应 | 数据库查询 |
| `middleware/` | 横切关注点 | 认证、角色检查、限流、安全头 | 具体业务 |
| `routes/` | URL 映射 | 路由表和路由组 | Handler 实现 |
| `tokens/`、`validation/` | 独立的工具组件 | 无状态、无 I/O 的计算 | 数据库访问 |
| `internal/` | 只给本 module 使用的代码 | 测试辅助 | 需要被其他项目导入的代码 |
| `cmd/<name>/` | 额外的可执行程序 | 每个子目录一个 `package main` | 被主程序导入的代码 |

### 为什么要分这么多目录？——如果不分会怎样？

假设所有代码都写在 `main.go` 里：
- 2000 行代码挤在一个文件中，找一个函数要翻很久。
- 无法单独测试某一部分。
- 多人同时修改时，Git 冲突不断。
- 没有包的边界，任何函数都能访问任何变量，代码耦合越来越严重。

### 替代的目录组织方式

| 方式 | 示例 | 适用场景 |
|---|---|---|
| **按技术层分**（本项目） | `controllers/`、`services/`、`repository/` | 中小项目、学习 |
| **按业务领域分** | `movie/`（含 handler、service、repo）、`user/` | 中大型项目，领域边界清晰 |
| Go 社区常见布局 | `cmd/server/main.go`、`internal/...` | 多个可执行程序；`internal/` 防止被外部导入（本项目用了其中的 `cmd/seed` 和 `internal/memstore`） |

按领域分的示例：

```text
internal/
├── movie/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   └── model.go
├── user/
│   └── ...
└── auth/
    ├── jwt.go
    └── middleware.go
```

好处是：改“电影”相关功能时，只需打开一个目录。

---

## 5.4 前端目录的职责

| 目录 | 职责 | 设计理由 |
|---|---|---|
| `api/` | 集中创建 axios 实例 | 基础 URL、`withCredentials` 只配置一次 |
| `context/` | React Context（全局状态） | 认证状态需要被多个页面共享 |
| `hook/` | 自定义 Hook | 复用“读取认证状态”“带 Token 发请求”的逻辑 |
| `types/` | TypeScript 接口 | 与后端 JSON 结构一一对应，多处复用 |
| `components/<name>/` | 每个组件一个目录 | 方便以后在同目录添加该组件专用的 CSS、测试文件 |

> **细节**：`AuthContext.ts` 和 `AuthProvider.tsx` 被拆成两个文件，是为了满足 `eslint-plugin-react-refresh` 的规则——一个 `.tsx` 文件如果同时导出组件和非组件（如 Context 对象），React 的热更新（Fast Refresh）就无法正常工作。

> **细节**：`hook/useAuth.tsx` 里没有 JSX，扩展名用 `.ts` 更准确，用 `.tsx` 也不会出错。

---

## 5.5 共享实例：validator 只创建一次

```go
// S/validation/validation.go
// New returns a validator with the custom rules registered. *validator.Validate is safe for concurrent use; create it once at startup and reuse it.
func New() *validator.Validate { ... }

// S/main.go
validate := validation.New()
authController := controllers.NewAuthController(authService, validate, ...)
movieController := controllers.NewMovieController(movieService, validate, ...)
```

**Why**：`validator.Validate` 内部会缓存每个 struct 的解析结果（哪些字段有哪些规则）。官方文档建议**只创建一次并复用**：每个请求都 `validator.New()` 会丢掉缓存，白白浪费 CPU；而且自定义规则（`imdb_id`、`youtube_id`）也得在每个新实例上重新注册。

**How**：在 `main` 中创建一次，通过构造函数传给需要它的 controller。

**为什么不用包级变量 `var validate = validator.New()`？**
- 包级变量在 `import` 时就被初始化，测试时无法替换。
- 如果某个函数里又写了 `validate := validator.New()`，局部变量会**遮蔽**（shadow）包级变量——读代码的人以为用的是共享实例，实际上不是。`go vet -vettool=$(which shadow)` 可以检查这类问题。

依赖注入让“用的是哪个实例”在代码中一目了然。

---

## 本章练习

> 答案见 [第 20 章](20-exercise-solutions.md#第-05-章)

**关键概念回顾**
- Go 中“一个目录 = 一个 package”意味着什么？
- 首字母大小写如何影响 struct 字段的 JSON 序列化？

**面试题 5.1** “Go 里怎么实现 private？包内私有和文件私有有区别吗？”

**面试题 5.2** “`internal/` 目录在 Go 中有什么特殊含义？”

**练习 5.3** 把 `models.UserRegister` 的 `FirstName` 字段在临时副本中改为 `firstName`（小写），运行 `go build ./...`。会出错吗？如果把引用处也一起改掉让编译通过，注册接口的行为会如何变化？

**练习 5.4** 用“按业务领域分”的方式，为本项目重新设计后端目录结构，画出目录树。

**调试挑战 5.5** 在 `controllers` 包中再新建一个文件，声明一个函数 `func bindJSON() {}`。编译会报什么错？为什么？

**理解检验 5.6** 在 `controllers` 包中能否读取 `tokens.Manager` 的 `accessKey` 字段？能否调用 `tokens.Hash`？`tokens.Claims` 中的 `UserID` 字段呢？
