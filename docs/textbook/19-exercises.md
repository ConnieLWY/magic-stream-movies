# 第 19 章　综合练习与毕业项目

> 每章末尾的练习针对单个主题；本章的练习**跨越多个章节**，检验你是否能把知识融会贯通。答案见 [第 20 章 · 第 19 章答案](20-exercise-solutions.md#第-19-章)。
>
> 难度：⭐ 基础 · ⭐⭐ 中等 · ⭐⭐⭐ 困难

---

## 19.1 综合理解题

**19.1 ⭐** 不看任何资料，画出系统整体架构图，标注：浏览器中存储了什么、每条箭头上传输的是什么、哪些请求需要 AT、哪些需要 RT Cookie。

**19.2 ⭐** 填写下表（不看代码）：

| 接口 | 方法 | 需要 AT？ | 需要 RT Cookie？ | 需要 ADMIN？ | 限流？ | 读写哪些集合 |
|---|---|---|---|---|---|---|
| `/auth/register` | | | | | | |
| `/auth/login` | | | | | | |
| `/auth/logout` | | | | | | |
| `/auth/refresh` | | | | | | |
| `/movies` | GET | | | | | |
| `/movies` | POST | | | | | |
| `/genres` | | | | | | |
| `/movies/:imdb_id` | | | | | | |
| `/movies/:imdb_id/review` | | | | | | |
| `/me/recommendations` | | | | | | |

**19.3 ⭐⭐** 解释“用户在 A 电脑登录 → 在 B 手机登录 → 回到 A 电脑操作”时，A 电脑会发生什么。涉及哪些函数？这种行为是 bug 还是设计选择？

**19.4 ⭐⭐** `ranking_value` 这个字段同时影响了哪两个业务流程？如果把评级约定改为“数值越大越好”，需要修改哪些代码和数据？（提示：别忘了 `NotRankedValue`。）

**19.5 ⭐⭐** 列出项目中所有“服务器信任了客户端”的地方，并判断每一处是否安全。

**19.6 ⭐⭐⭐** 假设项目要从 MongoDB 迁移到 PostgreSQL。按包列出需要修改的地方，并设计 `users`、`movies`、`genres`、`rankings` 的表结构（包括 `movie_genres`、`user_favourite_genres` 关联表）。哪些查询会变得更简单？哪些会变得更复杂？

---

## 19.2 模拟面试

请用口头（或写下来）的方式回答，每题控制在 2–3 分钟。

**19.7 ⭐** “请介绍一下这个项目。”（要求：一句话定位、技术栈、你认为最有技术含量的部分、一个你做过的设计权衡以及为什么这样选。）

**19.8 ⭐⭐** “你的认证方案为什么这样设计？如果 XSS 攻击成功了，攻击者能拿到什么？”

**19.9 ⭐⭐** “前端怎么处理 Token 过期？并发请求同时过期怎么办？”

**19.10 ⭐⭐** “你的项目中如何调用 LLM？如何保证它的输出可靠？”

**19.11 ⭐⭐⭐** “如果这个服务的用户量增长 100 倍，你会先优化什么？”

---

## 19.3 调试场景

每个场景描述一个症状。请写出：(1) 你的假设；(2) 验证方法；(3) 根本原因；(4) 修复方案。

**19.12 ⭐** 新同事克隆仓库后运行 `go run .`，看到一串 `... is required` 后程序退出。但他确认 `.env` 文件存在。

**19.13 ⭐⭐** 部署到生产后，用户反映“Chrome 上一切正常，但在 iPhone Safari 上每次刷新页面都要重新登录”。前端在 `reelm.vercel.app`，后端在 `magicstream.onrender.com`，后端配置了 `COOKIE_SAMESITE=None`。如果改回默认的 `Lax`，又会怎样？

**19.14 ⭐⭐** 管理员反映：换了一个新的 `OPENAI_MODEL` 之后，发布影评时大约一半的请求返回 502 “Could not rank this review automatically”。

**19.15 ⭐⭐** 一位新同事加了一个接口之后，运维报告：服务器日志中偶尔出现 `[WARNING] Headers were already written. Wanted to override status code 500 with 200`。

**19.16 ⭐⭐⭐** 一个用户报告：“我注册时选了 Comedy，但推荐页一直显示 No recommendations yet。” 你在数据库中查到他的 `favourite_genres` 是 `[]`。但前端会阻止空选择，后端也要求 `min=1`。这条数据可能是怎么来的？还有哪些用户可能受影响？怎么处理？

**19.17 ⭐⭐⭐** 部署后，日志平台的告警显示有人在大量调用 `PATCH /movies/:imdb_id/review`，OpenAI 账单激增。但你确认只有你一个人是管理员。可能的原因有哪些？按可能性排序并给出排查步骤。

---

## 19.4 编码挑战

**19.18 ⭐** 为 `GET /movies` 增加按类型过滤的查询参数：`GET /movies?genre=Drama`。没有参数时返回全部。

**19.19 ⭐⭐** 实现 `PATCH /me/genres`，允许已登录用户修改喜爱的类型。要求：
- 只能修改自己的（ID 来自 Token）；
- 至少一个类型，且每个类型必须存在于 `genres` 集合中；
- 返回更新后的 `UserResponse`（但不需要新的 Token）；
- 前端提供修改页面，修改成功后更新 `auth` 状态，推荐页立即反映变化。

**19.20 ⭐⭐** 实现分页：`GET /movies?page=1&limit=20`，响应格式改为 `{"items": [...], "page": 1, "limit": 20, "total": 137}`。同时修改前端，并处理旧格式的兼容问题。

**19.21 ⭐⭐⭐** 实现“多设备登录”：每次登录创建一条 `sessions` 文档（`session_id`、`user_id`、`refresh_token_hash`、`user_agent`、`created_at`、`expires_at`），RT 的载荷中包含 `session_id`。刷新时只轮换对应的 session；登出只删除当前 session；新增 `POST /logout-all`。

**19.22 ⭐⭐⭐** 为 `OpenAIClassifier` 本身写测试：用 `httptest.NewServer` 启动一个模拟 OpenAI Chat Completions API 的本地服务器（返回预设的 JSON），让 langchaingo 的客户端连接它（提示：`openai.WithBaseURL`）。覆盖“正常返回”“服务器返回 500”“响应超过超时时间”三种情况。

---

## 19.5 毕业项目

选择一个完成。完成标准：代码可运行、有测试、有 README、通过第 16 章的安全清单。

### 项目 A：MagicStream 2.0
按第 18 章从零重建本项目，并在此基础上完成：
- 第 21 章扩展建议中的至少 3 项（其中至少 1 项需要修改前后端）；
- 第 16 章安全清单中尚未勾选的两项（`govulncheck` / `gitleaks` 进入 CI；部署到一个有 HTTPS 的环境）；
- 前端测试（Vitest）+ 至少一个 Playwright E2E 测试；
- 结构化日志（`log/slog`）和请求 ID。

### 项目 B：同构迁移
用另一种技术栈重写后端，保持 API 完全兼容（前端不需要任何修改）：
- 选项 1：Go 标准库 `net/http`（Go 1.22+ 路由）+ `database/sql` + PostgreSQL；
- 选项 2：Node.js + Express/Fastify + MongoDB；
- 选项 3：Python + FastAPI + MongoDB。

写一份对比报告：代码量、依赖数量、性能（用 `hey` 或 `wrk` 压测 `/movies`）、开发体验。

### 项目 C：用户评论系统
在现有项目基础上新增：
- 普通用户可以给电影打分（1–5）和写短评；
- 每个用户对每部电影只能评一次（唯一复合索引）；
- 电影详情页显示平均分和最新评论（聚合管道 `$group`、`$avg`）；
- 推荐算法改为：喜爱类型 + 平均分排序；
- 评论内容用 LLM 做内容审核（带白名单结果校验）。

---

## 19.6 自我评估量表

完成本教材后，对照下表给自己打分（1 = 完全不会，5 = 能教别人）：

| 能力 | 1 | 2 | 3 | 4 | 5 |
|---|---|---|---|---|---|
| 解释系统架构与数据流 | | | | | |
| 从零创建 Go module 和 Gin 服务 | | | | | |
| 使用 mongo-driver v2 完成 CRUD | | | | | |
| 解释并实现 JWT + Refresh Token 轮换 | | | | | |
| 编写和理解 Gin Middleware | | | | | |
| 编写 Go 单元测试和 HTTP 测试 | | | | | |
| 使用 React Hooks、Context、Router | | | | | |
| 实现 axios 拦截器自动刷新 | | | | | |
| 识别安全风险并设计防御 | | | | | |
| 容器化与部署 | | | | | |

所有项目都达到 4 分以上，你就达成了本教材的最终目标。
