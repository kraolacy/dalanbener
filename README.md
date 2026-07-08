<div align="center">

# 大蓝本儿 · Sunshine-note 💙

**一个瀑布流式的男性社群兴趣分享社区**
主题：8·3 国际散帅节（International Sunshine Day）

散散帅 · 交交友 · 互帮互助

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
![React](https://img.shields.io/badge/React-18-61dafb.svg)
![Vite](https://img.shields.io/badge/Vite-6-646cff.svg)
![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)

</div>

---

## 🌞 这是什么

**大蓝本儿（Sunshine-note）** 是一个纯前端的、开源的社区 Demo，采用瀑布流笔记的形态（发帖 + 点赞收藏评论），
但内容聚焦 **男性社群的兴趣、生活与互助**：健身撸铁、机械键盘、钓鱼、篮球、数码、汽车、下厨房、露营、手冲咖啡、程序人生、球鞋、摄影……

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
| 🔐 **登录 / 注册** | 账号体系（用户名 + 密码 + 头像），未登录时发帖/点赞/评论会引导登录 |
| 😎 **个人主页** | 我的发布、我的收藏、点赞数、散帅节徽章、退出登录 |
| ➕ **发帖** | 封面 emoji、标题正文、圈子、话题标签、一键参加散帅节话题 |
| ❤️ **互动** | 点赞、收藏、评论，全部本地持久化 |

所有账号与互动数据通过浏览器 `localStorage` 持久化，**无需后端**，刷新不丢失。

> 🔒 **关于登录**：这是纯前端 Demo，账号与密码只保存在你本机浏览器的 `localStorage` 里，**没有真实的服务器和加密**，请勿使用真实密码。生产环境应接入真正的后端鉴权（见 Roadmap）。

## 🛠 技术栈

- **React 18** + **Vite 6**（纯前端 SPA，零后端）
- 状态管理：React Context + `localStorage`（`src/store.jsx`）
- 样式：原生 CSS（`src/styles.css`），移动端优先，响应式 masonry
- 无第三方 UI / 图表库，封面用 CSS 渐变 + emoji，**不依赖任何外链资源**，可离线运行

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
dalanshu/
├── index.html
├── vite.config.js
├── src/
│   ├── main.jsx            # 入口
│   ├── App.jsx             # 外壳：顶栏 / 分类 / 底部导航 / 弹层
│   ├── store.jsx           # 全局状态 + localStorage 持久化
│   ├── data.js             # 分类定义 + 种子内容
│   ├── styles.css          # 全局样式（蓝色主题）
│   └── components/
│       ├── Feed.jsx        # 瀑布流
│       ├── PostCard.jsx    # 笔记卡片
│       ├── PostDetail.jsx  # 详情弹层（点赞/收藏/评论）
│       ├── CreatePost.jsx  # 发帖弹层
│       ├── Festival.jsx    # 散帅节专题 + 倒计时
│       ├── Help.jsx        # 互助板
│       └── Me.jsx          # 个人主页
└── public/favicon.svg
```

## 🗺 Roadmap（欢迎 PR）

- [ ] 接入真实后端（帖子 / 用户 / 评论持久化）
- [ ] 用户注册登录与关注体系
- [ ] 图片上传（目前用 emoji + 渐变封面占位）
- [ ] 私信 / 搭子聊天
- [ ] 深色模式
- [ ] PWA 离线安装

## 🤝 贡献

欢迎提 Issue 和 PR：新的兴趣圈子、更好看的卡片、散帅节玩法都可以。

## 📄 License

[MIT](LICENSE) © 2026 —— 随便用，记得散个帅 🌞
