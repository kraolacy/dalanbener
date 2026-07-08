# 大蓝本儿 · Go 后端（gin + gorm + sqlite/mysql 可插拔 + redis 可选）

配套前端见仓库根目录 `src/`，前端通过相对路径 `/api/*` 调用本服务；连不上时自动退回 `localStorage` 演示模式。

## 接口契约（与前端严格一致）

| 方法 | 路径 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| GET | `/api/health` | 否 | 存活探测 `{ok,name,version}` |
| POST | `/api/register` | 否 | 注册，返回 `{token,user}` |
| POST | `/api/login` | 否 | 登录，返回 `{token,user}` |
| GET | `/api/me` | 是 | 当前用户 |
| GET | `/api/posts` | 可选 | 帖子流 |
| POST | `/api/posts` | 是 | 发帖 |
| POST | `/api/posts/:id/comments` | 是 | 评论 |
| POST | `/api/posts/:id/like` | 是 | 点赞切换 |
| POST | `/api/posts/:id/collect` | 是 | 收藏切换 |
| GET | `/api/helps` | 否 | 互助流 |
| POST | `/api/helps` | 是 | 发互助 |

## 本地开发

默认 **零配置** 使用 SQLite（`DB_DRIVER=sqlite`），无需任何外部依赖即可跑通全部接口；需要高并发时切到 MySQL。

```bash
# 1. 直接启动（SQLite 默认，数据落在 ./data/app.db）
cd server-go
export JWT_SECRET='dev-secret-change-me'
go run .

# 2.（可选）使用 MySQL 获得更高并发能力
export DB_DRIVER=mysql
export MYSQL_DSN='root:密码@tcp(127.0.0.1:3306)/dalanshu?charset=utf8mb4'
export REDIS_ADDR='127.0.0.1:6379'      # 不设置则自动降级为直查数据库
go run .

# 3. 启动前端（另开终端，/api 代理到 Go）
npm install
npm run dev        # http://localhost:5173
```

> Redis 不可用时服务自动降级为直查数据库，其余功能不受影响。

## 数据迁移（SQLite → MySQL 平滑升级）

本地用 SQLite 跑了一段时间、积攒了用户与帖子后，可一键把存量数据迁到 MySQL：

```bash
# 默认：sqlite(本地库) -> mysql(由 MYSQL_DSN 指定)
DB_DRIVER=mysql MYSQL_DSN='...' SQLITE_PATH='./data/app.db' go run . migrate
# 或显式指定两端：
MIGRATE_FROM=sqlite MIGRATE_TO=mysql MYSQL_DSN='...' go run . migrate
```

迁移为**同构模型拷贝、幂等可执行**：目标表已非空则自动跳过，可反复运行而不重复写入。

## 生产 / NAS 自托管（一键）

```bash
docker compose up --build      # 含 mysql + redis + go；前端由 Go 同源托管
# 访问 http://<IP>:8088
```

数据持久化在宿主机 `./data/{mysql,redis}`，删容器不丢。上线前务必修改 `JWT_SECRET`。

## 配置（环境变量）

| 变量 | 默认 | 说明 |
| --- | --- | --- |
| `PORT` | `8080` | 服务端口 |
| `JWT_SECRET` | `dev-secret-change-me` | 令牌签名密钥 |
| `DB_DRIVER` | `sqlite` | 数据库驱动：`sqlite` 或 `mysql` |
| `MYSQL_DSN` | 空 | GORM MySQL DSN（`DB_DRIVER=mysql` 时必填） |
| `SQLITE_PATH` | `./data/app.db` | SQLite 文件路径（`DB_DRIVER=sqlite` 时） |
| `REDIS_ADDR` | `127.0.0.1:6379` | Redis 地址（留空则禁用缓存、直查数据库） |
| `REDIS_PASS` | 空 | Redis 密码 |
| `STATIC_DIR` | 空 | 前端构建产物目录；设置后由本服务同源托管 SPA |
| `GIN_MODE` | `release` | gin 模式 |
| `MIGRATE_FROM` / `MIGRATE_TO` | `sqlite` / `mysql` | `migrate` 子命令的源/目标驱动 |

## 高并发要点

- **DB 可插拔 + 连接池**：MySQL 模式启用 `MaxOpenConns=100` 等；SQLite 模式单连接规避锁竞争，高并发请切 MySQL。
- **批量聚合查询**：帖子流用 3 条 `GROUP BY` 聚合取代每帖 4 条查询，消除 N+1。
- **缓存防击穿**：Redis 缓存 feed，未命中时用 `singleflight` 合并并发请求，仅放行一次 DB 查询；Redis 不可达自动降级。
- **并发安全**：缓存中的基础帖子流为只读共享对象，`overlayLikes` 返回新切片，杜绝共享切片写竞争。
- **优雅关闭**：HTTP 超时 + 信号优雅停机，避免高并发下请求被粗暴中断。

## 目录结构

```
server-go/
├── main.go              # 启动：配置→DB→迁移→种子→Redis→路由；支持 migrate 子命令
├── internal/
│   ├── config/          # 环境变量（含 DB_DRIVER / 迁移配置）
│   ├── model/           # GORM 模型 + 对外 DTO
│   ├── db/              # 可插拔连接 + 连接池 + SQLite→MySQL 迁移
│   ├── seed/            # 首次启动种子（go:embed seed.json）
│   ├── cache/           # Redis 封装（feed 缓存，可降级）
│   ├── middleware/      # JWT 鉴权中间件
│   └── handler/         # 11 个接口 + 单测（含批量聚合/并发）
└── go.mod
```
