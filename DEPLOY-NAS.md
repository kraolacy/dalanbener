# 在群晖 NAS 上自托管大蓝本儿（多人共享版）

部署后：所有人访问同一地址、注册到**同一个系统**、看到**同一批帖子**——真正的社区。
前端 + 后端 + SQLite 数据库全跑在你这一台 NAS 上，一个容器搞定。

## 前提
- 群晖套件中心安装 **Container Manager**（DS220+ 等 x86 机型支持）。
- 机器有 ≈1GB 空闲内存、几百 MB 磁盘。

## 步骤

### 1. 把项目放到 NAS
两种方式二选一：
- **File Station 上传**：新建文件夹（如 `/docker/dalanshu`），把本仓库内容传进去（至少要有：`Dockerfile`、`docker-compose.yml`、`package.json`、`package-lock.json`、`index.html`、`vite.config.js`、`src/`、`public/`、`server/`）。
- **命令行 git**（若开了 SSH）：`cd /volume1/docker && git clone https://github.com/kraolacy/dalanbener.git`

### 2. 改密钥（重要）
编辑 `docker-compose.yml`，把 `JWT_SECRET` 换成一长串随机字符（登录令牌签名用，泄露=别人能伪造登录）：
```yaml
    environment:
      - JWT_SECRET=你的一长串随机字符换掉这里
```

### 3. Container Manager 构建启动
1. 打开 **Container Manager → 项目 → 新增**
2. 项目名 `dalanshu`，路径选上面那个文件夹（里面有 `docker-compose.yml`）
3. 来源选「使用现有的 docker-compose.yml」→ 下一步 → 构建
4. 首次会拉 node 镜像 + 构建前端 + 装后端依赖，约 3–8 分钟

### 4. 访问
- 浏览器打开 **`http://<NAS内网IP>:8088`**（端口在 compose 里，想换改 `"8088:3000"` 左边的数字）
- 注册账号、发帖、点赞、评论——数据都进数据库，所有访客共享互通。

## 数据与备份
- 数据库文件在宿主机 **`<项目文件夹>/server/data/dalanshu.db`**，删容器不丢。
- 备份就把 `server/data/` 整个复制走即可。

## 让家人/朋友从外网访问
- **最简单**：群晖 **QuickConnect**（控制面板 → 外部访问）。
- **要独立域名 + HTTPS**：群晖 **DDNS** + **反向代理**（控制面板 → 登录门户 → 反向代理）+ Let's Encrypt 证书，把 `https://你的域名` 反代到 `localhost:8088`。
  > 国内公开对外服务，严格来说域名要 ICP 备案；家人/小范围用 QuickConnect 可绕开。

## 常见问题
- **构建卡在 npm**：NAS 网络访问 npm 慢，可在 Container Manager 给 Docker 配国内镜像源，或多等一会/重试。
- **端口冲突**：8088 被占就改 compose 里的左值（如 `9000:3000`）。
- **想重置数据**：停容器，删 `server/data/dalanshu.db*`，重启即重新写入种子。

## 本地开发/自测（可选）
```bash
# 后端（需 Node ≥ 22.5，用内置 node:sqlite）
cd server && npm install && node server.js      # 默认 :3000
# 另开一个终端跑前端 dev（连不上后端时自动用 localStorage 演示模式）
npm run dev
```
