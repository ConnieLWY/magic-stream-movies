# 第 13a 章　前端：React + TypeScript

> 本章目标：理解前端如何启动、页面如何路由、全局状态如何共享，以及每个页面组件如何与后端交互。读完本章，你应该能从空白的 Vite 模板重建出 Reelm 前端。
>
> 本章是新增章节（原大纲中没有），因为本项目是全栈项目，前端是“Frontend integration”中必不可少的部分。

---

## 13a.1 SPA 是如何启动的

### 13a.1.1 `index.html`

```html
<!-- C/index.html（节选） -->
<html lang="en" data-bs-theme="dark">
  <head>
    <link href="https://fonts.googleapis.com/css2?family=Fraunces:...&family=Figtree:...&display=swap" rel="stylesheet" />
    <title>Reelm</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- 整个应用只有这**一个** HTML 文件。`<div id="root">` 是一个空容器，React 会把所有界面渲染到里面。
- `type="module"`：以 ES Module 方式加载。开发时 Vite 拦截这个请求，即时把 TSX 编译成浏览器能理解的 JS。
- `data-bs-theme="dark"`：Bootstrap 5.3 的**颜色模式**开关，让所有 Bootstrap 组件使用深色主题。

### 13a.1.2 `main.tsx`

```tsx
// C/src/main.tsx
createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/*" element={<App />} />
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  </StrictMode>,
);
```

| 部分 | 作用 |
|---|---|
| `document.getElementById("root")!` | `!` 是 TypeScript 的**非空断言**：告诉编译器“我保证这里不是 null”。 |
| `createRoot(...).render(...)` | React 18+ 的渲染入口。 |
| `<StrictMode>` | 开发模式下的额外检查（例如故意把 effect 执行两次）。**生产构建中无效果。** |
| `<AuthProvider>` | 提供全局认证状态。放在最外层，**所有**组件都能访问。 |
| `<BrowserRouter>` | 使用浏览器的 History API 实现路由（URL 像 `/login`，而不是 `/#/login`）。 |
| `<Route path="/*" element={<App />} />` | 所有路径都交给 `App`，`App` 内部再定义具体的子路由。`/*` 表示“匹配任何子路径”。 |

`import "bootstrap/dist/css/bootstrap.min.css"` 和 `import "./index.css"` 的**顺序**很重要：后引入的 `index.css` 能覆盖 Bootstrap 的默认样式。

---

## 13a.2 React 核心概念速成（结合本项目）

### 13a.2.1 组件与 Props

组件就是一个**返回 JSX 的函数**。Props 是父组件传给子组件的参数。

```tsx
// C/src/components/movies/Movies.tsx（节选）
interface MoviesProps {
  movies: MovieModel[];
  message?: string;                               // ? = 可选
  updateMovieReview?: (imdb_id: string) => void;  // 函数类型的 prop
}

function Movies({ movies, message, updateMovieReview }: MoviesProps) {   // 解构 props
  if (!movies || movies.length === 0) {
    return <div className="empty-state">...<p>{message || "No movies found."}</p></div>;
  }
  return (
    <div className="movie-grid">
      {movies.map((movie) => (
        <Movie key={movie.id} movie={movie} updateMovieReview={updateMovieReview} />
      ))}
    </div>
  );
}
```

- **JSX**：在 JS 中写类似 HTML 的语法，`{}` 中可以放任意 JS 表达式。`class` 要写成 `className`。
- **列表渲染**：`array.map(...)` 返回元素数组。**每个元素必须有唯一的 `key`**，React 用它判断哪些项新增、删除或移动。这里用 MongoDB 的 `id`，是稳定且唯一的。**不要用数组下标作为 key**（除非列表永不变化），否则排序或筛选时会出现状态错乱。（`MovieGridSkeleton` 中用了下标 `i` 作为 key，因为骨架屏列表是静态的，可以接受。）
- **条件渲染**：`if (...) return ...`，或者 JSX 中的 `{cond && <X/>}`、`{cond ? <A/> : <B/>}`。

### 13a.2.2 `useState`：组件的状态

```tsx
const [movies, setMovies] = useState<MovieModel[]>([]);
```

- 返回 `[当前值, 设置函数]`。调用 `setMovies(newValue)` 会**触发重新渲染**。
- `<MovieModel[]>` 是 TypeScript 泛型，指定状态的类型。
- **不要直接修改状态**（如 `movies.push(x)`），React 检测不到变化。必须创建新数组/对象。

**函数式更新**（`Register.tsx`）：

```tsx
const toggleGenre = (genre: Genre) => {
  setFavouriteGenres((prev) =>
    prev.some((g) => g.genre_id === genre.genre_id)
      ? prev.filter((g) => g.genre_id !== genre.genre_id)   // 已选 → 移除
      : [...prev, genre]                                    // 未选 → 添加（展开运算符创建新数组）
  );
};
```

当新状态依赖旧状态时，传入一个函数 `prev => next`，React 保证 `prev` 是最新值。

### 13a.2.3 `useEffect`：副作用

“副作用”指渲染之外的操作：网络请求、订阅、定时器、手动操作 DOM。

```tsx
// C/src/components/home/Home.tsx
useEffect(() => {
  const fetchMovies = async () => {
    setLoading(true);
    setMessage("");
    try {
      const response = await axiosClient.get<MovieModel[]>("/movies");
      setMovies(response.data ?? []);
      if (!response.data || response.data.length === 0) {
        setMessage("There are currently no movies available");
      }
    } catch (error) {
      console.error("Error fetching movies: ", error);
      setMessage("Error fetching movies");
    } finally {
      setLoading(false);
    }
  };
  fetchMovies();
}, []);   // ← 依赖数组为空：只在首次挂载后执行一次
```

| 依赖数组 | 执行时机 |
|---|---|
| 不写 | 每次渲染后 |
| `[]` | 首次挂载后一次 |
| `[a, b]` | 首次 + 每当 `a` 或 `b` 变化时 |

**为什么要在 effect 里再定义一个 async 函数？** `useEffect` 的回调**不能**是 async 函数，因为 async 函数返回 Promise，而 React 期望回调返回清理函数或 undefined。

**`??` 空值合并运算符**：`a ?? b`——`a` 为 `null` 或 `undefined` 时取 `b`。这里防御了后端返回 `null`（Go nil 切片）的情况。

**加载状态三件套**：`loading`（显示骨架屏）、`message`（空状态或错误提示）、`movies`（数据）。这是数据获取组件的标准模式。

### 13a.2.4 `useMemo`：派生数据的缓存

```tsx
// C/src/components/home/Home.tsx
// Prefer a title with a trailer for the hero slot
const featured = useMemo(
  () => movies.find((m) => m.youtube_id) ?? movies[0],
  [movies],
);

const genres = useMemo(() => {
  const byId = new Map<number, Genre>();
  movies.forEach((m) => m.genre?.forEach((g) => byId.set(g.genre_id, g)));   // Map 自动去重
  return [...byId.values()].sort((a, b) => a.genre_name.localeCompare(b.genre_name));
}, [movies]);

const visibleMovies =
  activeGenre === null
    ? movies
    : movies.filter((m) => m.genre?.some((g) => g.genre_id === activeGenre));
```

- `useMemo(fn, deps)`：只有 `deps` 变化时才重新计算 `fn`。
- `featured`、`genres` 是从 `movies` **派生**出来的，不需要单独的 state——**能计算出来的数据就不要存成 state**，否则要手动保持同步，容易出 bug。
- `visibleMovies` 没有用 `useMemo`：过滤一个小数组很便宜，不缓存也可以。
- `?.` 可选链：`m.genre?.forEach` 在 `genre` 为 undefined 时直接返回 undefined，不报错。
- `localeCompare`：按语言规则比较字符串，用于排序。

### 13a.2.5 `useRef`：不触发渲染的可变值

两种用途在本项目中都出现了：

1. **引用 DOM 元素**（`Review.tsx`）：
   ```tsx
   const revText = useRef<HTMLTextAreaElement>(null);
   <Form.Control as="textarea" ref={revText} defaultValue={movie.admin_review} />
   // 提交时读取：revText.current.value
   ```
   这是**非受控组件（Uncontrolled Component）**：输入框的值由 DOM 自己管理，React 只在需要时读取。对比 `Login.tsx` 中的**受控组件**（`value={email} onChange={e => setEmail(e.target.value)}`），值由 React state 管理。

   | | 受控 | 非受控 |
   |---|---|---|
   | 值存在哪 | React state | DOM |
   | 实时校验、联动 | 方便 | 不方便 |
   | 代码量 | 多 | 少 |

2. **保存跨渲染的可变值**（`useAxiosPrivate.tsx`）：见第 11 章 11.14.3 节。

### 13a.2.6 Context 与自定义 Hook

**问题**：`Header`、`Login`、`Review`、`Recommended`、`RequiredAuth` 都需要读取登录状态。如果通过 props 一层层传递（“prop drilling”），中间不需要它的组件也得接收并转发。

**Context** 让数据可以跨越组件层级直接被读取：

```ts
// C/src/context/AuthContext.ts
interface AuthContextValue {
  auth: User | null;
  setAuth: Dispatch<SetStateAction<User | null>>;   // 就是 useState 返回的 setter 的类型
  loading: boolean;
}

export const AuthContext = createContext<AuthContextValue>({
  auth: null,
  setAuth: () => {},     // 默认值：在 Provider 之外使用时的兜底
  loading: true,
});
```

```tsx
// C/src/context/AuthProvider.tsx
return (
  <AuthContext.Provider value={{ auth, setAuth, loading }}>
    {children}
  </AuthContext.Provider>
);
```

```ts
// C/src/hook/useAuth.tsx
function useAuth() {
  return useContext(AuthContext)
}
```

自定义 Hook `useAuth` 只是对 `useContext(AuthContext)` 的一层封装。好处：组件中写 `useAuth()` 比 `useContext(AuthContext)` 更简洁，而且以后如果要加逻辑（例如在 Provider 之外使用时抛出错误），只需改一处。

> **Hook 规则**：(1) 只在函数组件或自定义 Hook 的**顶层**调用 Hook，不能放在 `if`、循环里；(2) 自定义 Hook 必须以 `use` 开头。`eslint-plugin-react-hooks` 会自动检查这些规则。

**替代方案**：Redux Toolkit、Zustand、Jotai。对于本项目这种只有一个全局状态（auth）的应用，Context 完全够用。

---

## 13a.3 路由

### 13a.3.1 路由表

```tsx
// C/src/App.tsx
<Routes>
  <Route path="/" element={<Home updateMovieReview={updateMovieReview} />}></Route>
  <Route path="/register" element={<Register />}></Route>
  <Route path="/login" element={<Login />}></Route>
  <Route element={<RequiredAuth />}>                                   {/* 布局路由（无 path） */}
    <Route path="/recommended" element={<Recommended />}></Route>
    <Route path="/review/:imdb_id" element={<Review />}></Route>
    <Route path="/stream/:yt_id" element={<StreamMovie />}></Route>
  </Route>
</Routes>
```

| 路径 | 组件 | 是否需要登录 | 调用的 API |
|---|---|---|---|
| `/` | `Home` | 否 | `GET /movies` |
| `/register` | `Register` | 否 | `GET /genres`、`POST /auth/register` |
| `/login` | `Login` | 否 | `POST /auth/login` |
| `/recommended` | `Recommended` | 是 | `GET /me/recommendations` |
| `/review/:imdb_id` | `Review` | 是 | `GET /movies/:imdb_id`、`PATCH /movies/:imdb_id/review`（仅管理员） |
| `/stream/:yt_id` | `StreamMovie` | 是 | 无（YouTube） |

**布局路由（Layout Route）**：`<Route element={<RequiredAuth />}>` 没有 `path`，它包裹的子路由在匹配时，会先渲染 `RequiredAuth`，再由 `RequiredAuth` 中的 `<Outlet />` 渲染具体的子页面。这样“需要登录”的逻辑只写一次。

### 13a.3.2 路由守卫：`RequiredAuth`

```tsx
// C/src/components/RequiredAuth.tsx
function RequiredAuth() {
  const { auth, loading } = useAuth()
  const location = useLocation()

  // Wait for the session restore (refresh cookie -> access token) before deciding
  if (loading) {
    return (
      <div className="d-flex justify-content-center py-5" role="status">
        <Spinner animation="border" />
        <span className="visually-hidden">Restoring your session...</span>
      </div>
    )
  }

  // Not logged in: send to /login and remember where the user was going,
  // so Login can redirect back via location.state.from
  if (!auth) {
    return <Navigate to="/login" state={{ from: location }} replace />
  }

  return <Outlet />
}
```

- `loading` 为真时显示一个加载指示器。如果去掉这个判断，已登录用户刷新 `/recommended` 时，会在会话恢复完成之前被重定向到登录页。返回 `null`（空白页）也能避免错误重定向，但用户会以为页面坏了；加载指示器让用户知道“正在处理”。
- `role="status"` 和 `visually-hidden` 文字：屏幕阅读器会读出“Restoring your session...”，视力正常的用户只看到转圈。
- `<Navigate replace>`：用 `replace` 而不是 push，这样用户在登录页按“后退”不会回到一个又把他踢回登录页的地址。
- `state={{ from: location }}`：把“原本要去的地方”通过路由状态传给登录页。

登录成功后跳回原页面：

```tsx
// C/src/components/login/Login.tsx
const from = location.state?.from?.pathname || "/";
// ...
navigate(from, { replace: true });
```

### 13a.3.3 路径参数与编程式导航

```tsx
const { imdb_id } = useParams();     // /review/tt0111161 → "tt0111161"
const navigate = useNavigate();
navigate(-1);                        // 后退一页（Review、StreamMovie 的“Back”按钮）
navigate(`/review/${imdb_id}`);      // App.updateMovieReview
```

### 13a.3.4 `Link` vs `NavLink` vs `<a>`

- `<a href="/login">`：浏览器会**整页刷新**，React 状态（包括内存中的 AT）全部丢失。
- `<Link to="/login">`：由 React Router 处理，**不刷新页面**。
- `<NavLink>`：`Link` + 当前路由匹配时自动加 `active` 类，用于导航栏高亮。`end` 属性（`Header.tsx`）表示精确匹配，否则 `/` 会匹配所有路径。

---

## 13a.4 HTTP 层

两个 axios 实例的分工（细节见第 11 章）：

| 实例 | 定义位置 | 用于 | 特点 |
|---|---|---|---|
| `axiosClient` | `api/axiosConfig.ts` | 公开接口：`/movies`、`/genres`、`/auth/*` | 模块级单例；`withCredentials: true` |
| `useAxiosPrivate()` | `hook/useAxiosPrivate.tsx` | 受保护接口 | 每个使用它的组件各自创建一个实例；自动加 Token；401 自动刷新重试 |

```ts
// C/src/api/axiosConfig.ts
const apiUrl = import.meta.env.VITE_API_BASE_URL

export const axiosClient = axios.create({
    baseURL: apiUrl,
    headers: {'Content-Type': 'application/json'},
    withCredentials: true // sends the HttpOnly refresh_token cookie
})
```

`withCredentials: true` 告诉浏览器：跨域请求时**携带并接受 Cookie**。必须与后端的 `AllowCredentials: true` 配合。缺少任意一方，`/auth/login` 设置的 Cookie 都不会被保存，`/auth/refresh` 也不会携带 Cookie。

`useAxiosPrivate` 中创建实例时没有设置 `Content-Type`，但 axios 在发送普通对象作为 body 时会自动设置为 `application/json`，所以 `PATCH` 请求也能正常工作。

---

## 13a.5 页面组件逐个讲解

### 13a.5.1 `Header`

- 根据 `auth` 是否存在显示“用户头像 + Sign out”或“Sign in + Join free”。
- `auth.first_name?.charAt(0).toUpperCase()` 生成头像首字母。
- `Navbar.Brand as={Link}`：react-bootstrap 的 **`as` 属性**（多态组件）——用 `Link` 的行为渲染 `Navbar.Brand` 的样式。
- “For you” 链接对所有人可见；未登录用户点击后会被 `RequiredAuth` 重定向到登录页。

### 13a.5.2 `App` 中的登出

```tsx
// C/src/App.tsx
const handleLogout = async () => {
  try {
    // Server identifies the user from the refresh_token cookie and clears it
    await axiosClient.post("/auth/logout");
  } catch (error) {
    console.error("Error logging out:", error);
  } finally {
    setAuth(null);          // 无论服务器是否成功，前端都清除状态
  }
};
```

`finally` 保证即使网络出错，用户在界面上也会被登出。这是**乐观的本地登出**。

### 13a.5.3 `Home`

- 顶部 Hero：显示 `featured`（优先选择有预告片的电影），背景图使用 `style={{ backgroundImage: \`url(${featured.poster_path})\` }}`——外层 `{}` 是 JSX 表达式，内层 `{}` 是 JS 对象。
- 类型筛选：完全在前端完成（无 API 调用），`aria-pressed` 表达按钮的选中状态（无障碍）。
- 影评按钮：`updateMovieReview` 由 `App` 传入，**只在登录后显示**（电影详情页需要登录）。文案按角色区分：管理员看到带笔形图标的“Edit review”，普通用户看到“Read review”：

```tsx
// C/src/components/home/Home.tsx（Hero 节选）
const { auth } = useAuth();
const isAdmin = auth?.role === "ADMIN";
// ...
{updateMovieReview && auth && (
  <button type="button" className="btn btn-ghost btn-lg" onClick={() => updateMovieReview(featured.imdb_id)}>
    {isAdmin ? (
      <>
        <PenIcon size={16} /> Edit review
      </>
    ) : (
      "Read review"
    )}
  </button>
)}
```

- `{条件 && <JSX>}`：条件为假时什么都不渲染。这是 React 中最常见的条件渲染写法。
- `<>...</>` 是 **Fragment**：把多个元素包在一起，但不会在 DOM 中多出一层标签。
- 按钮文案要与用户能做的事情一致：给只能看的人显示一支“笔”，会让人以为可以编辑，点进去却只能看——这是一个小小的可用性陷阱。

### 13a.5.4 `Movie`

- 有 `youtube_id` 时，海报是一个 `<Link to="/stream/:id">`；否则是普通 `<div>`。
- `aria-label={\`Play trailer: ${movie.title}\`}` 为屏幕阅读器提供描述。
- 海报 `<img alt="" loading="lazy">`：`alt=""` 表示装饰性图片（标题已经在下方文字中）；`loading="lazy"` 让浏览器延迟加载屏幕外的图片。
- 卡片上的影评按钮与 Hero 中的规则相同：登录后才显示，文案按角色区分，`aria-label` 也随之变化（“Edit review: …” / “Read review: …”）。

### 13a.5.5 `Register`

- 7 个 state：5 个表单字段 + `favouriteGenres` + `genres`（可选项），外加 `error`、`loading`。
- 挂载时请求 `/genres` 填充可选项。
- 提交前做前端校验（与后端规则镜像）。
- 提交的请求体只包含用户能决定的字段，类型只发送 ID：
  ```json
  {"first_name":"Ada","last_name":"Lovelace","email":"ada@example.com",
   "password":"secret123","favourite_genre_ids":[2,5]}
  ```
  `favouriteGenres.map((g) => g.genre_id)` 把选中的类型对象转换成 ID 数组。没有 `role` 字段——角色由服务器决定，前端发不发都一样，干脆不发。
- 注册成功后直接登录：后端返回与登录相同的 `UserResponse`（并设置 RT Cookie），前端 `setAuth(response.data)` 后跳转首页，用户不需要再输入一次密码：
  ```tsx
  // Registering also signs the user in: the response matches /auth/login
  const response = await axiosClient.post<User>('/auth/register', { ... });
  setAuth(response.data);
  navigate('/', { replace: true });
  ```
- `<Form.Control isInvalid={...}>` + `<Form.Control.Feedback type="invalid">`：Bootstrap 的校验样式。

### 13a.5.6 `Login`

- 受控表单（`value` + `onChange`）。
- `SubmitEvent<HTMLFormElement>`：React 19 类型定义中的表单提交事件类型（旧代码和大多数教程中写作 `FormEvent`）。
- `e.preventDefault()`：阻止浏览器默认的表单提交（会整页刷新）。
- 成功后 `setAuth(response.data)`——后端的 `UserResponse` 与前端的 `User` 类型完全对应。
- 失败时按状态码给出提示：429 提示稍后再试，其他一律“邮箱或密码错误”（13.5.2 节）。
- 不把响应打印到控制台：响应中有 Access Token。

### 13a.5.7 `Recommended`

```tsx
// C/src/components/recommended/Recommended.tsx
const axiosPrivate = useAxiosPrivate();
useEffect(() => {
  const fetchRecommendedMovies = async () => { /* get('/me/recommendations') */ };
  fetchRecommendedMovies();
}, [axiosPrivate]);
```

依赖数组中有 `axiosPrivate`。因为 `useAxiosPrivate` 用 `useMemo(..., [])` 保证实例稳定，这个 effect 只会执行一次。如果实例不稳定，就会陷入“渲染 → 新实例 → effect 重跑 → setState → 渲染……”的**无限循环**。

页头显示的喜爱类型来自 `auth.favourite_genres`（登录响应中的数据），而推荐结果是服务器根据数据库中的数据计算的。两者通常一致。

### 13a.5.8 `Review`

- 挂载时 `GET /movies/:imdb_id`。
- `isAdmin = auth?.role === 'ADMIN'`：管理员看到表单，其他人看到只读的引用块或 “No review yet.”。
- **前端的 `isAdmin` 只控制显示**。真正的权限检查在后端的 `RequireRole("ADMIN")`。
- 提交成功后，用响应中的 `admin_review` 和完整的 `ranking`（名称 + 数值）更新本地状态（第 10 章 10.3.8）。
- 错误处理按状态码区分：

```tsx
// C/src/components/review/Review.tsx（handleSubmit 节选）
} catch (err) {
    // 401 is handled by useAxiosPrivate (refresh, or sign out if that fails)
    if (axios.isAxiosError(err) && err.response?.status === 403) {
        setError('Only admins can update reviews.');
    } else if (axios.isAxiosError(err) && err.response?.status === 502) {
        setError('Could not rank this review automatically. Please try again.');
    } else {
        setError('Error updating review');
    }
}
```

  - **401** 由 `useAxiosPrivate` 的拦截器统一处理（刷新 → 重试；刷新失败则清空登录状态，`RequiredAuth` 会把用户带到登录页），组件不需要重复处理。
  - **403** 表示“已登录但没权限”，刷新 Token 也没用，直接提示。正因为后端区分了 401 和 403，拦截器才不会把一个权限问题误当作会话过期、白白刷新一次 Token 再把用户踢出去。
  - **502** 表示 LLM 的回答无法识别，影评没有保存，提示用户重试。

### 13a.5.9 `StreamMovie`

```tsx
// C/src/components/stream/StreamMovie.tsx
// YouTube video IDs are exactly 11 characters from this set
const YOUTUBE_ID = /^[A-Za-z0-9_-]{11}$/;

function StreamMovie() {
  const { yt_id: youTubeId } = useParams();
  const navigate = useNavigate();

  if (!youTubeId || !YOUTUBE_ID.test(youTubeId)) {
    return (
      <Container className="section">
        <div className="empty-state">This trailer link is not valid.</div>
      </Container>
    );
  }

  return (
    <Container className="theater">
      {/* ... Back 按钮 ... */}
      <div className="theater-stage ratio ratio-16x9">
        {/* Plain YouTube embed: no player library needed for a single provider */}
        <iframe
          src={`https://www.youtube-nocookie.com/embed/${youTubeId}?autoplay=1&rel=0`}
          title="Trailer"
          allow="autoplay; encrypted-media; picture-in-picture; fullscreen"
          allowFullScreen
          referrerPolicy="strict-origin-when-cross-origin"
        />
      </div>
    </Container>
  );
}
```

- **解构时重命名**：`const { yt_id: youTubeId } = useParams()`——路由参数叫 `yt_id`，在组件内部用符合 JS 命名习惯的 `youTubeId`。
- **校验路由参数**：`yt_id` 来自 URL，用户可以随意修改。它会被拼接进 `<iframe src>`，所以先用与后端 `youtube_id` 规则相同的正则检查：11 个字符，只允许字母、数字、`_`、`-`。这样就不可能注入额外的路径或查询参数。`RegExp.test` 返回布尔值。
- **为什么不用播放器库？** YouTube 的嵌入页面本身就是一个完整的播放器。只播放一个来源时，一个 `<iframe>` 就够了，零依赖（第 06 章 6.3.1）。
- **`youtube-nocookie.com`**：YouTube 的“隐私增强模式”，在用户点击播放之前不写入跟踪 Cookie。
- **`allow` 属性**：iframe 中的页面默认不能自动播放、不能全屏，需要父页面显式授权。
- `ratio ratio-16x9`：Bootstrap 工具类，保持 16:9 的宽高比，iframe 会自动撑满。
- `autoplay=1`：浏览器的自动播放策略通常要求视频**静音**或者用户先与页面交互过；用户是点击海报进入此页的，通常满足条件。
- **`title` 属性**：屏幕阅读器用它描述 iframe 的内容，是无障碍检查的必备项。

---

## 13a.6 样式系统

- **Bootstrap 5.3 + react-bootstrap**：提供栅格（`Row`、`Col`）、表单、按钮、导航栏等组件。
- **设计 Token**（`C/src/index.css` 开头）：用 CSS 自定义属性定义颜色、圆角、字体、阴影，例如 `--ink-950: #0b0910`、`--amber: #ffb547`、`--font-display: "Fraunces"`。然后把 Bootstrap 自己的变量（`--bs-body-bg`、`--bs-link-color` 等）映射到这些 Token 上，实现“换肤”。
- **自定义按钮变体**：react-bootstrap 的 `<Button variant="brand">` 会渲染 `class="btn btn-brand"`。Bootstrap 本身没有 `btn-brand`，它定义在 `index.css:178`，`btn-ghost` 同理（`index.css:198`）。
- **响应式网格**：`.movie-grid { grid-template-columns: repeat(auto-fill, minmax(165px, 1fr)); }`——CSS Grid 自动根据宽度决定列数，无需媒体查询。
- 全部样式集中在 `index.css` 中，在 `main.tsx` 里引入一次。组件不再各自导入 CSS 文件。

---

## 13a.7 从零写前端的顺序

1. `npm create vite@latest`，选 `react-ts`。
2. 安装依赖；在 `main.tsx` 中引入 Bootstrap CSS。
3. 写 `types/movie.ts`、`types/user.ts`（对照后端 model）。
4. 写 `api/axiosConfig.ts`（先只有 `axiosClient`）。
5. 写 `Home`，调用 `/movies`，能显示列表即可。
6. 加入 `BrowserRouter`、`App` 路由表、`Header`。
7. 写 `Register`、`Login`（先把 `auth` 存在 `App` 的 state 中）。
8. 抽出 `AuthContext`、`AuthProvider`、`useAuth`。
9. 写 `RequiredAuth`，保护 `/recommended`。
10. 写 `useAxiosPrivate`（先只加请求拦截器）→ `Recommended`。
11. 在后端实现 `/auth/refresh` 后，加入 `refreshSession`、`AuthProvider` 中的会话恢复、响应拦截器。
12. 写 `Review`、`StreamMovie`。
13. 最后打磨样式。

---

## 本章练习

> 答案见 [第 20 章](20-exercise-solutions.md#第-13a-章)

**关键概念回顾**
- `useState`、`useEffect`、`useMemo`、`useRef` 各自的作用？
- 为什么列表渲染需要 `key`？为什么不推荐用下标？
- 受控组件与非受控组件的区别？
- 前端路由守卫能保证安全吗？

**面试题 13a.1** “`useEffect` 的依赖数组有什么作用？漏写依赖会怎样？”

**面试题 13a.2** “什么是 prop drilling？Context 如何解决它？Context 有什么缺点？”

**面试题 13a.3** “`<a href>` 和 `<Link to>` 有什么区别？在你的项目中用错会导致什么后果？”

**练习 13a.4** 不看源码，在你的重建项目中写出 `RequiredAuth`：会话恢复中显示 `Spinner`，未登录时重定向到 `/login` 并记住原本要去的页面。

**练习 13a.5** 在重建项目中先用 `react-player` 实现预告片页面，运行 `npm run build` 记录产物大小；再换成 `<iframe>`，再次构建，比较两次的输出。

**练习 13a.6** 不看源码，实现电影卡片上的影评按钮：未登录不显示；管理员显示带图标的“Edit review”；普通用户显示“Read review”。

**练习 13a.7** 给 `Review` 页面增加一个功能：普通用户看到的只读影评下方，显示评级的数值（例如 “Excellent (1)”）。数据从哪里来？需要修改后端吗？

**调试挑战 13a.8** 把 `useAxiosPrivate` 中的 `useMemo(() => axios.create(...), [])` 改成直接 `const axiosAuth = axios.create(...)`，打开推荐页，观察 Network 面板。解释发生了什么。

**调试挑战 13a.9** 把 `Header` 中的 `<Link to="/login">` 改成 `<a href="/login">`。登录后点击它，会发生什么？会被登出吗？为什么？（提示：考虑 `AuthProvider` 的会话恢复。）

**理解检验 13a.10** 在 `RequiredAuth` 中删除 `if (loading) { ... }` 这个分支。已登录用户直接在地址栏输入 `/recommended` 并回车，会看到什么？
