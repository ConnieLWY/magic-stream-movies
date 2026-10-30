# 第 01 章　项目概览

> 本章目标：准确描述这个项目**做什么**、**不做什么**，以及用户在界面上能完成哪些操作。

---

## 1.1 一句话描述

**MagicStream Movies** 是一个电影浏览与推荐网站：用户注册时选择喜爱的电影类型，登录后可以浏览电影、观看 YouTube 预告片、查看个性化推荐；管理员可以撰写影评，系统调用 OpenAI 自动判断影评的情感倾向，并据此给电影打上评级标签。

前端界面的品牌名是 **Reelm**（定义在 `C/src/brand.ts`），后端和仓库名是 MagicStream。两者指的是同一个项目。

```ts
// C/src/brand.ts —— 品牌名称的唯一来源
export const BRAND = {
  name: "Reelm",
  tagline: "Every story, one spell away.",
  description:
    "Trailers, curated reviews and picks tuned to the genres you love.",
};
```

> **为什么把品牌名放在一个单独的文件里？** 这叫“单一事实来源”（Single Source of Truth）。Header、Footer、登录页都会显示品牌名，如果每处都硬编码 `"Reelm"`，改名时就得改很多地方，还容易漏掉。

---

## 1.2 用户角色

| 角色 | 如何产生 | 能做什么 |
|---|---|---|
| 访客（未登录） | — | 浏览首页电影列表、按类型筛选、注册、登录 |
| 普通用户 `USER` | 通过注册页注册（角色由服务器固定设为 `USER`） | 访客的一切 + 看预告片、查看推荐、查看电影详情和影评 |
| 管理员 `ADMIN` | 先正常注册，再运行 `go run ./cmd/seed -admin <邮箱>` 提升 | 普通用户的一切 + 添加电影、撰写/修改影评（触发 AI 评级） |

---

## 1.3 功能清单（逐项对照源码）

### 1.3.1 已实现的功能

| # | 功能 | 前端页面 | 后端接口 | 权限 |
|---|---|---|---|---|
| F1 | 浏览全部电影 | `/`（`Home.tsx`） | `GET /movies` | 公开 |
| F2 | 按类型筛选 | `/`（前端本地筛选） | 无（前端从已加载的电影中提取类型） | 公开 |
| F3 | 获取全部类型 | `/register` | `GET /genres` | 公开 |
| F4 | 注册（成功后直接登录） | `/register` | `POST /auth/register` | 公开，限流 |
| F5 | 登录 | `/login` | `POST /auth/login` | 公开，限流 |
| F6 | 登出 | Header 中的“Sign out” | `POST /auth/logout` | 靠 Cookie 识别用户 |
| F7 | 刷新会话 | 页面加载时 + 401 时自动触发 | `POST /auth/refresh` | 靠 Cookie |
| F8 | 看预告片 | `/stream/:yt_id`（`StreamMovie.tsx`） | 无（直接嵌入 YouTube） | 登录（前端守卫） |
| F9 | 个性化推荐 | `/recommended` | `GET /me/recommendations` | 登录 |
| F10 | 电影详情 | `/review/:imdb_id` | `GET /movies/:imdb_id` | 登录 |
| F11 | 写影评 + AI 评级 | `/review/:imdb_id`（仅 ADMIN 看到表单） | `PATCH /movies/:imdb_id/review` | ADMIN |
| F12 | 添加电影 | 无前端页面（用 curl / Postman） | `POST /movies` | ADMIN |
| F13 | 健康检查 | 无 | `GET /health` | 公开 |
| F14 | 初始化数据库 | 无 | `go run ./cmd/seed` 命令行程序 | 本地运行 |

### 1.3.2 没有实现的功能（不要误以为存在）

- **真正的视频流媒体**：项目名叫 “Stream”，但实际只是用 `<iframe>` 嵌入 YouTube 预告片。服务器不存储、不转码、不传输任何视频文件。
- **用户资料编辑**、**修改喜爱类型**、**搜索**、**分页**、**评分/收藏**。
- **管理员后台页面**：添加电影只能通过 API 工具调用。

这些都列在第 21 章的扩展建议中，可以作为练手项目。

---

## 1.4 一次典型的用户旅程

下面用文字描述一个普通用户第一次使用网站的完整过程。后续章节会把每一步拆开讲解。

1. **打开首页** `http://localhost:5173/`
   - `AuthProvider` 自动调用 `POST /auth/refresh` 尝试恢复会话 → 没有 Cookie → 401 → `auth = null`。
   - `Home` 调用 `GET /movies` → 显示电影网格和顶部 Hero（优先选择有预告片的电影）。
2. **点击 “Join free” 注册**
   - `Register` 调用 `GET /genres` 获取类型列表 → 用户选择喜爱类型。
   - 提交 → `POST /auth/register` → 后端校验、确认类型存在、哈希密码、写入 `users` 集合（角色固定为 `USER`）→ 直接签发 Token，用户已处于登录状态，跳转首页。
   - Refresh Token 写入 **HttpOnly Cookie**；Access Token 放在响应体中，前端只保存在**内存**（React state）。
3. **下次登录**
   - `POST /auth/login` → 后端比对密码 → 生成 Access Token 和 Refresh Token，返回方式与注册相同。
4. **访问 “For you”**
   - `Recommended` 用 `useAxiosPrivate` 发送 `GET /me/recommendations`，自动带上 `Authorization: Bearer <token>`。
   - 后端从 Token 中取出用户 ID → 查用户喜爱类型 → 查匹配的电影 → 按评级排序 → 返回前 5 部。
5. **15 分钟后**再点击某部电影
   - Access Token 已过期 → 后端返回 401 → axios 拦截器自动调用 `/auth/refresh` → 拿到新 Token（旧的 Refresh Token 同时作废） → **自动重试**原请求。用户完全无感知。
6. **刷新浏览器**
   - 内存中的 Access Token 消失 → `AuthProvider` 再次调用 `/auth/refresh` → 凭 Cookie 恢复登录状态。
7. **点击 “Sign out”**
   - `POST /auth/logout` → 后端删除数据库中的 Refresh Token 哈希，并让 Cookie 过期 → 前端 `setAuth(null)`。

---

## 1.5 管理员旅程：AI 影评评级

这是项目中最有特色的功能：

1. 管理员打开 `/review/tt0111161`（假设的 IMDb ID）。
2. 页面检测 `auth.role === 'ADMIN'`，显示文本框。
3. 管理员输入影评，例如 “A masterpiece of hope and friendship.”
4. 前端发送 `PATCH /movies/tt0111161/review`，body 为 `{"admin_review": "..."}`。
5. 后端：
   - `RequireRole("ADMIN")` 确认调用者是管理员，否则返回 403。
   - 先确认电影存在（不存在就不调用收费的 OpenAI）。
   - 从 `rankings` 集合读取所有评级名称（种子数据中是 `Excellent, Good, Okay, Bad, Terrible`）。
   - 把它们填入 `.env` 中的 `BASE_PROMPT_TEMPLATE`，拼接影评，发给 OpenAI（最多等待 30 秒）。
   - OpenAI 返回一个单词，如 `Excellent`。后端去掉空白和标点后，**只接受与已有评级名称匹配的回答**，否则返回 502 且不保存。
   - 更新电影文档，并在响应中返回完整的评级（名称 + 数值）。
6. 前端显示新的评级标签。

---

## 1.6 项目规模

| 部分 | 文件数 | 代码行数（约，含空行） |
|---|---|---|
| 后端 Go（不含测试） | 23 个 `.go` 文件 | 约 2100 行 |
| 后端测试 | 6 个 `_test.go` 文件 | 约 700 行 |
| 前端 TS/TSX（`src/` 下） | 24 个源文件 | 约 1400 行 + 944 行 CSS |

这是一个“小而完整”的项目：足够小，可以完全理解；又足够完整，覆盖了真实项目的大部分关键技术点（认证、授权、分层架构、数据库、第三方 API、测试、容器化与 CI）。

---

## 本章练习

> 答案见 [第 20 章](20-exercise-solutions.md#第-01-章)

**关键概念回顾**
- 项目中有哪三种用户身份？分别如何产生？
- 为什么说本项目“不是真正的流媒体服务”？

**面试题 1.1** 面试官问：“你的项目里管理员是怎么创建的？为什么不能在注册时选择？” 请结合代码回答。

**练习 1.2** 打开 `S/routes/routes.go`，自己整理一张 API 列表（方法、路径、Handler 名、经过哪些中间件），与 1.3.1 节的表格对照。

**练习 1.3** 在 1.4 节的用户旅程中，哪几步涉及 Cookie？哪几步涉及 `Authorization` Header？

**理解检验 1.4** 首页的“按类型筛选”调用了后端接口吗？如果电影数量增加到 10 万部，这种实现会有什么问题？
