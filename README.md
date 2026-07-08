<div align="center">

# 大蓝本儿 · Sunshine-note 💙

**一个瀑布流式的男性社群兴趣分享社区**
主题：8·3 国际散帅节（International Sunshine Day）

散散帅 · 交交友 · 互帮互助

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
![React](https://img.shields.io/badge/React-18-61dafb.svg)
![Vite](https://img.shields.io/badge/Vite-6-646cff.svg)
![Node](https://img.shields.io/badge/Node-Express%20%2B%20SQLite-3c873a.svg)
![Style](https://img.shields.io/badge/UI-Cyberpunk%202077-fcee0a.svg)
![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)

🔗 **在线体验**：https://kraolacy.github.io/dalanbener/

</div>

---

## 🌞 这是什么

**大蓝本儿（Sunshine-note）** 是一个开源的社区应用，采用瀑布流笔记的形态（发帖 + 点赞收藏评论 + 关注 + 私信），界面是 **赛博朋克 2077** 风格。
内容聚焦 **男性社群的兴趣、生活与互助**：健身撸铁、机械键盘、钓鱼、篮球、数码、汽车、下厨房、露营、手冲咖啡、程序人生、球鞋、摄影……

它有两种运行方式：**① 纯前端演示版**（数据存浏览器、零部署、可离线 / GitHub Pages）；**② 自托管联网版**（带后端，多人注册共享、私信可用，可跑在自己的 NAS / 服务器）。前端会自动判断走哪种。

节日彩蛋是 **8 月 3 日「国际散帅节」**：

> 三月八号是女神节，八月三号是散帅节。
> 今天，放下手机，散步、散心、散个帅 —— 做一个阳光、松弛、爱交朋友的男人。
>
> （"散帅" ≈ Sunshine 的谐音梗，8·3 也是 3·8 妇女节的一个善意反转。）

> ℹ️ **说明**：这是一个轻松、包容的趣味项目，纯属自娱与技术练习，不针对、不冒犯任何群体。欢迎所有人来玩。

## ✨ 功能一览

| 模块 | 说明 |
| --- | --- |
| 🏠 **首页瀑布流** | 双列/多列 masonry 笔记流，16 个兴趣圈子分类筛选，关键词搜索 |
| 🌞 **散帅节专题** | 到 8·3 的实时倒计时、节日话题、精选散帅动态 |
| 🎉 **男生节专栏** | 6·20 男生节专栏，带节日倒计时介绍条 |
| ⚔️ **海克斯大乱斗专栏** | 英雄联盟大乱斗专区：上分心得、英雄强度榜、五黑招募 |
| 🤝 **散帅互助** | 搭子广场：求助 / 帮忙两类帖，搬家、修电脑、找搭子、约球局 |
| 🔐 **登录 / 注册** | 账号体系（用户名 + 密码 + 头像），未登录时互动会引导登录 |
| 😎 **个人主页** | 我的发布、我的收藏、点赞/关注/粉丝数、散帅节徽章、退出登录 |
| ➕ **发帖** | 封面 emoji **或上传图片**、标题正文、圈子、话题标签、一键参加散帅节话题 |
| ❤️ **互动** | 点赞、收藏、评论 |
| ➕ **关注** | 关注作者、「关注」信息流只看已关注的人、关注/粉丝数 |
| ✉️ **私信** | 一对一聊天、会话列表、未读提醒（联网版功能） |
| 🖼 **图片上传** | 发帖上传图片作封面（联网版存服务器，演示版存本地） |

**两种模式自动切换**：前端启动时探测后端——连得上就走后端 API（真账号、多人共享、私信可用）；连不上退回浏览器 `localStorage`（单机演示、数据存本地、可离线）。

> 🔒 **关于安全**：演示版账号只存在你本机浏览器（无加密），请勿用真实密码；联网版密码用 **bcrypt** 加密存服务器数据库。

## 🛠 技术栈

- **前端**：React 18 + Vite 6，单文件可离线打开（vite-plugin-singlefile）
- **后端**（可选）：Node + Express + 内置 `node:sqlite`（免原生编译）、bcrypt 密码加密、JWT 鉴权
- 状态：React Context，**双模式**（后端 API / `localStorage`），见 `src/store.jsx`、`src/api.js`
- 样式：原生 CSS（`src/styles.css`），**赛博朋克 2077** 风格，移动端优先，响应式 masonry
- 演示版不依赖任何外链资源，可离线运行

## 🚀 快速开始

```bash
# 1. 安装依赖
npm install

# 2. 启动开发服务器（默认 http://localhost:5173 ）
npm run dev

# 3. 生产构建
npm run build

# 4. 本地预览构建产物
npm run preview
```

需要 Node.js ≥ 18。

## 📁 项目结构

```
.
├── index.html · vite.config.js
├── src/                        # 前端
│   ├── main.jsx / App.jsx      # 入口 / 外壳（顶栏·分类·底部导航·弹层）
│   ├── store.jsx               # 全局状态（后端 API + localStorage 双模式）
│   ├── api.js                  # 后端 API 客户端
│   ├── data.js                 # 分类定义 + 种子内容
│   ├── styles.css              # 赛博朋克 2077 样式
│   └── components/             # Feed·PostCard·PostDetail·CreatePost·Festival
│                               # Help·Me·AuthModal·ColumnIntro·MessagesSheet
├── server/                     # 可选后端（Express + node:sqlite）
│   ├── server.js · db.js · seed.js · package.json
├── Dockerfile                  # 多阶段：构建前端 + 跑后端
├── docker-compose.yml          # 一键自托管
└── DEPLOY-NAS.md               # 群晖 NAS 部署图文
```

## 🖥 自托管多人共享版（后端）

默认纯前端是「单机演示」（数据存各自浏览器）。仓库同时提供了一套**可自托管的后端**（Express + 内置 `node:sqlite`），
部署后所有人注册到同一系统、看到同一批帖子，就是真正的社区。前端会自动探测后端：连得上走 API（共享），连不上退回 localStorage（离线演示）。

- 一键部署（含群晖 NAS 图文步骤）见 **[DEPLOY-NAS.md](DEPLOY-NAS.md)**
- 本质：`docker compose up` 起一个容器，Node 同时托管前端与 API，SQLite 落盘持久化。

## 🗺 Roadmap（欢迎 PR）

- [x] 接入真实后端（账号 / 帖子 / 评论 / 点赞持久化，可自托管，见 DEPLOY-NAS.md）
- [x] 注册登录 + **关注体系**（关注 / 粉丝 / 关注信息流）
- [x] **图片上传**（发帖上传封面图）
- [x] **私信 / 搭子聊天**（一对一、未读提醒）
- [ ] 第三方登录（微信 / QQ / 抖音，需各平台开发者认证）
- [ ] 深色 / 主题切换 · PWA 离线安装 · 通知推送

## 🤝 贡献

欢迎提 Issue 和 PR：新的兴趣圈子、更好看的卡片、散帅节玩法都可以。

## 📄 License

[MIT](LICENSE) © 2026 —— 随便用，记得散个帅 🌞
