# 大蓝书 · 后端 API 交接文档（前端专用）

> 适用分支：`feature/high-concurrency-opt`（相对 `main` 的演进版）
> 后端技术栈：Go + Gin + GORM（SQLite / MySQL 可插拔）+ Redis（可选缓存）
> 文档日期：2026-07-09

---

## 一、当前可用 API 清单

基础路径：`/api`（前端 `api.js` 已用相对路径，同源部署无需改）。
鉴权：除 `/health`、`/register`、`/login`、`/posts`、`/helps` 外，其余接口需在请求头带 `Authorization: Bearer <token>`。

| 方法 | 路径 | 鉴权 | 说明 | 成功响应 |
| --- | --- | --- | --- | --- |
| GET | `/api/health` | 否 | 健康检查 | `{ "ok": true, "name": "dalanshu", "version": 1 }` |
| POST | `/api/register` | 否 | 注册 | `{ "token": "...", "user": { name, avatar, bio } }` |
| POST | `/api/login` | 否 | 登录 | 同上 |
| GET | `/api/me` | 是 | 当前用户 | `{ name, avatar, bio }` |
| GET | `/api/posts` | 否（可选登录） | 帖子流 | 见下方「列表响应双模式」 |
| POST | `/api/posts` | 是 | 发帖 | `PostOut`（含 id/liked/likeCount 等） |
| POST | `/api/posts/:id/comments` | 是 | 评论 | `PostOut`（含新评论） |
| POST | `/api/posts/:id/like` | 是 | 点赞切换 | `PostOut`（liked 翻转） |
| POST | `/api/posts/:id/collect` | 是 | 收藏切换 | `PostOut`（collected 翻转） |
| GET | `/api/helps` | 否 | 互助流 | 同帖子流双模式 |
| POST | `/api/helps` | 是 | 发互助 | `HelpOut` |

### 列表响应双模式（兼容现有前端）

前端 `api.js` 当前直接 `req('/api/posts')` 并把返回值当数组用，**无需改动**即可工作：

- **默认（不带 `cursor`/`limit` 参数）**：返回**纯数组** `PostOut[]` / `HelpOut[]`，行为与旧版完全一致。
- **显式分页**：带 `?limit=20&cursor=<游标>` 时，返回信封对象：
  ```json
  { "items": [ ... ], "next": "<base64游标或空串>" }
  ```
  `next` 为空串表示已到末页。前端未来要做无限滚动时，按此模式逐页拉取即可。

### PostOut / HelpOut 字段（与前端契约逐字段一致）

```ts
interface PostOut {
  id: string; cat: string; author: string; avatar: string;
  title: string; body: string; cover: string | null;
  tags: string[]; festival: boolean; tall: boolean;
  likeCount: number; collectCount: number;
  liked: boolean; collected: boolean;   // 仅登录用户有真实状态，匿名恒为 false
  comments: { author: string; avatar: string; text: string }[];
  createdAt: number;                     // 毫秒时间戳
}
interface HelpOut {
  id: string; type: 'need' | 'offer'; author: string; avatar: string;
  title: string; body: string; city: string; reward: string;
  ts: string; createdAt: number;
}
interface UserOut { name: string; avatar: string; bio: string; }
```

### 错误响应（统一格式）

所有失败均返回 **HTTP 状态码 + `{ "error": "<中文提示>" }`**：

| 场景 | 状态码 | error |
| --- | --- | --- |
| 请求体 JSON 格式错误 | 400 | 请求格式错误 |
| 用户名 < 2 字 / 密码 < 4 位 | 400 | 用户名至少 2 个字符 / 密码至少 4 位 |
| 标题或正文为空 | 400 | 标题和正文不能为空 |
| 互助标题或说明为空 | 400 | 标题和说明不能为空 |
| 评论为空 | 400 | 评论不能为空 |
| 非法游标（分页篡改） | 400 | 非法游标 |
| 未登录 / token 失效 | 401 | 请先登录 |
| 账号不存在（token 对应用户已删） | 401 | 账号不存在 |
| 登录密码错误 | 401 | 密码不对 |
| 帖子不存在 | 404 | 帖子不存在 |
| 用户名已注册 | 409 | 这个用户名已被注册 |
| 触发限流（如开启） | 429 | 请求太频繁了，歇会儿再来 |
| 服务内部错误 | 500 | 服务器开小差了 |

前端 `api.js` 已有的 `if (!res.ok) throw new Error((data && data.error) || ...)` 逻辑**已天然兼容**此格式，无需改。

---

## 二、本次相对旧版的改动点（对前端的影响）

1. **目录与分层重构（后端内部）**：入口迁到 `server-go/cmd/server`，新增 `service` 业务层。对前端**零影响**（接口路径、字段、错误格式均未变）。
2. **统一错误/状态管理（C5）**：所有响应改由 `internal/resp` 包统一写出。**行为不变**，但保证：
   - 错误永远是 `{ "error": "..." }` 字段名，不再出现其他键名；
   - 修复了一个隐藏 bug：此前鉴权失败会**重复写两次响应**，现已用 `AbortWithStatusJSON` 正确终止。
3. **游标分页（C2）**：新增可选 `?cursor=&limit=` 分页能力。**默认行为完全兼容**（仍返回纯数组）；仅当显式传参才返回 `{items,next}` 信封。前端若不主动分页，完全无感。
4. **限流中间件（C3）**：当后端配置 `RATE_LIMIT>0` 时按客户端 IP 限速，超限返回 `429`。默认关闭，前端**无影响**；若运维开启，前端应友好提示 `error` 文案即可。
5. **读写分离（C1）**：后端配置了 MySQL 主从时自动将读请求走从库、写请求走主库；无副本时自动回落主库。**对前端透明**。

> 结论：**现有前端 `api.js` / `data.js` 无需任何修改即可对接本版后端**。已用 curl 逐接口验证中文存储、列表、点赞、评论、互助均正常返回正确数据。

---

## 三、后续规划（Roadmap，供前端提前对齐）

1. **前端接入游标分页**：当前前端是一次性拉全量。后续可改造 `api.posts()` 支持传入 `cursor/limit`，用 `{items,next}` 做无限滚动 / 下拉加载，降低首屏 payload。
2. **列表数据缓存与增量更新**：后端已预留 Redis 缓存（feed 列表），后续可加 `ETag`/`Last-Modified` 或版本号，前端做 stale-while-revalidate。
3. **限流的前端降级**：若运维开启 `RATE_LIMIT`，前端建议在收到 `429` 时做指数退避重试 + 轻提示，避免用户感知硬失败。
4. **写后一致性**：发帖/评论/点赞目前为「写主库 + 清缓存」，读从库可能有秒级延迟。前端如遇到「刚点赞数没变」，可乐观更新本地状态（已是 per-user 前端逻辑），不必等后端回查。
5. **错误码结构化（可选演进）**：当前错误以中文 `error` 文案为主。若后续要做多语言 / 精细化提示，可改为 `{ "code": "POST_NOT_FOUND", "message": "..." }`，前端按 `code` 映射文案。此变更会破坏当前契约，需前后端同步升级，暂不在本期。
6. **鉴权过期处理**：JWT 有效期 30 天。前端可在收到 `401` 时引导重新登录（当前已抛错，由调用方决定跳转）。

---

## 四、本地联调指引（给前端）

```bash
# 后端（SQLite 零配置）
cd server-go
DB_DRIVER=sqlite JWT_SECRET=devsecret go run ./cmd/server
# 默认监听 :8080，自动托管同目录前端（若设 STATIC_DIR）

# 前端（Node 代理 / 同源）
npm run dev
```

- 不启动后端时，`api.js` 会自动退回 `localStorage` 演示模式，页面仍可浏览。
- 验证中文：用 curl 发中文请求体可正常往返（注意 PowerShell 控制台对 UTF-8 显示有缺陷，勿被 `?` 误导，以 curl / 浏览器为准）。
