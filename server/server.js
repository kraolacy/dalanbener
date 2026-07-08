const path = require('path')
const fs = require('fs')
const express = require('express')
const bcrypt = require('bcryptjs')
const jwt = require('jsonwebtoken')
const db = require('./db')

const PORT = process.env.PORT || 3000
const JWT_SECRET = process.env.JWT_SECRET || 'dev-secret-change-me'
const PUBLIC_DIR = process.env.PUBLIC_DIR || path.join(__dirname, 'public')
const DB_PATH = process.env.DB_PATH || path.join(__dirname, 'data', 'dalanshu.db')
const UPLOAD_DIR = path.join(path.dirname(DB_PATH), 'uploads')
fs.mkdirSync(UPLOAD_DIR, { recursive: true })

const app = express()
app.use(express.json({ limit: '8mb' })) // 放开体积以容纳图片 base64

// ---------- 预编译语句 ----------
const Q = {
  userByName: db.prepare('SELECT * FROM users WHERE username = ?'),
  userById: db.prepare('SELECT id, username, avatar, bio FROM users WHERE id = ?'),
  createUser: db.prepare('INSERT INTO users (username, password_hash, avatar, bio, created_at) VALUES (?,?,?,?,?)'),
  allPosts: db.prepare('SELECT * FROM posts ORDER BY created_at DESC'),
  postById: db.prepare('SELECT * FROM posts WHERE id = ?'),
  postByAuthor: db.prepare('SELECT avatar FROM posts WHERE author = ? LIMIT 1'),
  postCountByAuthor: db.prepare('SELECT COUNT(*) AS n FROM posts WHERE author = ?'),
  insertPost: db.prepare(`INSERT INTO posts (id,cat,author,avatar,title,body,cover,image,tags,festival,tall,base_likes,base_collects,created_at)
    VALUES (?,?,?,?,?,?,?,?,?,?,?,0,0,?)`),
  commentsOf: db.prepare('SELECT author, avatar, text FROM comments WHERE post_id = ? ORDER BY created_at ASC'),
  insertComment: db.prepare('INSERT INTO comments (post_id,author,avatar,text,created_at) VALUES (?,?,?,?,?)'),
  likeCount: db.prepare('SELECT COUNT(*) AS n FROM likes WHERE post_id = ?'),
  collectCount: db.prepare('SELECT COUNT(*) AS n FROM collects WHERE post_id = ?'),
  liked: db.prepare('SELECT 1 FROM likes WHERE user_id = ? AND post_id = ?'),
  collected: db.prepare('SELECT 1 FROM collects WHERE user_id = ? AND post_id = ?'),
  addLike: db.prepare('INSERT OR IGNORE INTO likes (user_id, post_id) VALUES (?,?)'),
  delLike: db.prepare('DELETE FROM likes WHERE user_id = ? AND post_id = ?'),
  addCollect: db.prepare('INSERT OR IGNORE INTO collects (user_id, post_id) VALUES (?,?)'),
  delCollect: db.prepare('DELETE FROM collects WHERE user_id = ? AND post_id = ?'),
  allHelps: db.prepare('SELECT * FROM helps ORDER BY created_at DESC'),
  insertHelp: db.prepare(`INSERT INTO helps (id,type,author,avatar,title,body,city,reward,ts,created_at) VALUES (?,?,?,?,?,?,?,?,?,?)`),
  // 关注
  followingOf: db.prepare('SELECT target FROM follows WHERE follower_id = ?'),
  followingCount: db.prepare('SELECT COUNT(*) AS n FROM follows WHERE follower_id = ?'),
  followerCount: db.prepare('SELECT COUNT(*) AS n FROM follows WHERE target = ?'),
  isFollowing: db.prepare('SELECT 1 FROM follows WHERE follower_id = ? AND target = ?'),
  follow: db.prepare('INSERT OR IGNORE INTO follows (follower_id, target) VALUES (?,?)'),
  unfollow: db.prepare('DELETE FROM follows WHERE follower_id = ? AND target = ?'),
  // 私信
  insertMsg: db.prepare('INSERT INTO messages (from_name,to_name,text,created_at) VALUES (?,?,?,?)'),
  msgsInvolving: db.prepare('SELECT * FROM messages WHERE from_name = ? OR to_name = ? ORDER BY created_at DESC'),
  thread: db.prepare(`SELECT from_name, text, created_at FROM messages
    WHERE (from_name=? AND to_name=?) OR (from_name=? AND to_name=?) ORDER BY created_at ASC`),
  markRead: db.prepare('UPDATE messages SET read = 1 WHERE from_name = ? AND to_name = ? AND read = 0'),
  unreadCount: db.prepare('SELECT COUNT(*) AS n FROM messages WHERE to_name = ? AND read = 0'),
}

// ---------- 辅助 ----------
function sign(user) { return jwt.sign({ id: user.id, name: user.username }, JWT_SECRET, { expiresIn: '30d' }) }
function avatarOf(name) {
  const u = Q.userByName.get(name); if (u) return u.avatar
  const p = Q.postByAuthor.get(name); return p ? p.avatar : '🙂'
}
// 当前用户完整信息（含关注/粉丝/未读私信数）
function meObject(username) {
  const u = Q.userByName.get(username); if (!u) return null
  return {
    name: u.username, avatar: u.avatar, bio: u.bio,
    following: Q.followingOf.all(u.id).map((r) => r.target),
    followers: Q.followerCount.get(u.username).n,
    unread: Q.unreadCount.get(u.username).n,
  }
}
function readUser(req) {
  const h = req.headers.authorization || ''
  const t = h.startsWith('Bearer ') ? h.slice(7) : null
  if (!t) return null
  try { return jwt.verify(t, JWT_SECRET) } catch { return null }
}
function auth(req, res, next) {
  const u = readUser(req)
  if (!u) return res.status(401).json({ error: '请先登录' })
  req.user = u; next()
}
function shapePost(row, userId) {
  return {
    id: row.id, cat: row.cat, author: row.author, avatar: row.avatar,
    title: row.title, body: row.body, cover: row.cover || null, image: row.image || null,
    tags: JSON.parse(row.tags || '[]'), festival: !!row.festival, tall: !!row.tall,
    likeCount: row.base_likes + Q.likeCount.get(row.id).n,
    collectCount: row.base_collects + Q.collectCount.get(row.id).n,
    liked: userId ? !!Q.liked.get(userId, row.id) : false,
    collected: userId ? !!Q.collected.get(userId, row.id) : false,
    comments: Q.commentsOf.all(row.id),
  }
}

// ---------- 账号 ----------
app.get('/api/health', (req, res) => res.json({ ok: true, name: 'dalanshu', version: 2 }))

app.post('/api/register', (req, res) => {
  const { username, password, avatar } = req.body || {}
  const name = (username || '').trim()
  if (name.length < 2) return res.status(400).json({ error: '用户名至少 2 个字符' })
  if ((password || '').length < 4) return res.status(400).json({ error: '密码至少 4 位' })
  if (Q.userByName.get(name)) return res.status(409).json({ error: '这个用户名已被注册' })
  const info = Q.createUser.run(name, bcrypt.hashSync(password, 10), avatar || '😎', '新来的散帅，请多关照 🌞', Date.now())
  const user = Q.userById.get(Number(info.lastInsertRowid))
  res.json({ token: sign(user), user: meObject(name) })
})

app.post('/api/login', (req, res) => {
  const { username, password } = req.body || {}
  const row = Q.userByName.get((username || '').trim())
  if (!row) return res.status(404).json({ error: '用户不存在，去注册一个吧' })
  if (!bcrypt.compareSync(password || '', row.password_hash)) return res.status(401).json({ error: '密码不对' })
  res.json({ token: sign(row), user: meObject(row.username) })
})

app.get('/api/me', auth, (req, res) => {
  const u = Q.userById.get(req.user.id)
  if (!u) return res.status(401).json({ error: '账号不存在' })
  res.json(meObject(u.username))
})

// ---------- 图片上传 ----------
app.post('/api/upload', auth, (req, res) => {
  const m = /^data:image\/(png|jpe?g|gif|webp);base64,(.+)$/.exec((req.body && req.body.dataUrl) || '')
  if (!m) return res.status(400).json({ error: '仅支持 png/jpg/gif/webp 图片' })
  const ext = m[1] === 'jpeg' ? 'jpg' : m[1]
  const buf = Buffer.from(m[2], 'base64')
  if (buf.length > 3 * 1024 * 1024) return res.status(413).json({ error: '图片太大（请 ≤ 3MB）' })
  const name = Date.now().toString(36) + Math.random().toString(36).slice(2, 8) + '.' + ext
  fs.writeFileSync(path.join(UPLOAD_DIR, name), buf)
  res.json({ url: '/uploads/' + name })
})

// ---------- 帖子 ----------
app.get('/api/posts', (req, res) => {
  const u = readUser(req)
  res.json(Q.allPosts.all().map((r) => shapePost(r, u && u.id)))
})

app.post('/api/posts', auth, (req, res) => {
  const { title, body, cat, cover, image, tags, festival } = req.body || {}
  if (!(title || '').trim() || !(body || '').trim()) return res.status(400).json({ error: '标题和正文不能为空' })
  const me = Q.userById.get(req.user.id)
  const id = 'u' + Date.now() + Math.floor(Math.random() * 1000)
  Q.insertPost.run(
    id, cat || 'rec', me.username, me.avatar, title.trim(), body.trim(), cover || null, image || null,
    JSON.stringify((tags || []).slice(0, 5)), festival ? 1 : 0, (body || '').length > 60 ? 1 : 0, Date.now(),
  )
  res.json(shapePost(Q.postById.get(id), req.user.id))
})

app.post('/api/posts/:id/comments', auth, (req, res) => {
  const post = Q.postById.get(req.params.id)
  if (!post) return res.status(404).json({ error: '帖子不存在' })
  const text = (req.body?.text || '').trim()
  if (!text) return res.status(400).json({ error: '评论不能为空' })
  const me = Q.userById.get(req.user.id)
  Q.insertComment.run(post.id, me.username, me.avatar, text, Date.now())
  res.json(shapePost(Q.postById.get(post.id), req.user.id))
})

function makeToggle(addStmt, delStmt, isSetStmt) {
  return (req, res) => {
    const post = Q.postById.get(req.params.id)
    if (!post) return res.status(404).json({ error: '帖子不存在' })
    if (isSetStmt.get(req.user.id, post.id)) delStmt.run(req.user.id, post.id)
    else addStmt.run(req.user.id, post.id)
    res.json(shapePost(Q.postById.get(post.id), req.user.id))
  }
}
app.post('/api/posts/:id/like', auth, makeToggle(Q.addLike, Q.delLike, Q.liked))
app.post('/api/posts/:id/collect', auth, makeToggle(Q.addCollect, Q.delCollect, Q.collected))

// ---------- 关注 ----------
app.post('/api/follow/:name', auth, (req, res) => {
  const target = req.params.name
  const me = Q.userById.get(req.user.id)
  if (target === me.username) return res.status(400).json({ error: '不能关注自己' })
  if (Q.isFollowing.get(req.user.id, target)) Q.unfollow.run(req.user.id, target)
  else Q.follow.run(req.user.id, target)
  res.json(meObject(me.username))
})

// ---------- 私信 ----------
app.get('/api/messages', auth, (req, res) => {
  const me = Q.userById.get(req.user.id).username
  const convos = {}
  for (const m of Q.msgsInvolving.all(me, me)) {
    const other = m.from_name === me ? m.to_name : m.from_name
    if (!convos[other]) convos[other] = { name: other, avatar: avatarOf(other), last: m.text, ts: m.created_at, unread: 0 }
    if (m.to_name === me && !m.read) convos[other].unread++
  }
  res.json(Object.values(convos).sort((a, b) => b.ts - a.ts))
})

app.get('/api/messages/:name', auth, (req, res) => {
  const me = Q.userById.get(req.user.id).username
  const other = req.params.name
  Q.markRead.run(other, me)
  res.json({ name: other, avatar: avatarOf(other), messages: Q.thread.all(me, other, other, me) })
})

app.post('/api/messages', auth, (req, res) => {
  const { to, text } = req.body || {}
  const me = Q.userById.get(req.user.id).username
  const t = (text || '').trim()
  if (!t) return res.status(400).json({ error: '消息不能为空' })
  if (to === me) return res.status(400).json({ error: '不能给自己发私信' })
  if (!Q.userByName.get(to)) return res.status(404).json({ error: '对方还不是注册用户，暂时无法私信' })
  Q.insertMsg.run(me, to, t, Date.now())
  res.json({ ok: true })
})

// ---------- 静态资源 ----------
app.use('/uploads', express.static(UPLOAD_DIR))
app.use(express.static(PUBLIC_DIR))
app.get('*', (req, res) => res.sendFile(path.join(PUBLIC_DIR, 'index.html')))

app.listen(PORT, () => console.log(`[dalanshu] 服务已启动 http://0.0.0.0:${PORT}`))
