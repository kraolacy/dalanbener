const path = require('path')
const express = require('express')
const bcrypt = require('bcryptjs')
const jwt = require('jsonwebtoken')
const db = require('./db')

const PORT = process.env.PORT || 3000
const JWT_SECRET = process.env.JWT_SECRET || 'dev-secret-change-me'
const PUBLIC_DIR = process.env.PUBLIC_DIR || path.join(__dirname, 'public')

const app = express()
app.use(express.json({ limit: '1mb' }))

// ---------- 预编译语句 ----------
const Q = {
  userByName: db.prepare('SELECT * FROM users WHERE username = ?'),
  userById: db.prepare('SELECT id, username, avatar, bio FROM users WHERE id = ?'),
  createUser: db.prepare('INSERT INTO users (username, password_hash, avatar, bio, created_at) VALUES (?,?,?,?,?)'),
  allPosts: db.prepare('SELECT * FROM posts ORDER BY created_at DESC'),
  postById: db.prepare('SELECT * FROM posts WHERE id = ?'),
  insertPost: db.prepare(`INSERT INTO posts (id,cat,author,avatar,title,body,cover,tags,festival,tall,base_likes,base_collects,created_at)
    VALUES (?,?,?,?,?,?,?,?,?,?,0,0,?)`),
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
  insertHelp: db.prepare(`INSERT INTO helps (id,type,author,avatar,title,body,city,reward,ts,created_at)
    VALUES (?,?,?,?,?,?,?,?,?,?)`),
}

// ---------- 辅助 ----------
function sign(user) { return jwt.sign({ id: user.id, name: user.username }, JWT_SECRET, { expiresIn: '30d' }) }
function publicUser(u) { return { name: u.username, avatar: u.avatar, bio: u.bio } }

// 可选鉴权：有 token 就解析出 userId，没有则为 null（不拦截）
function readUser(req) {
  const h = req.headers.authorization || ''
  const t = h.startsWith('Bearer ') ? h.slice(7) : null
  if (!t) return null
  try { return jwt.verify(t, JWT_SECRET) } catch { return null }
}
// 强制鉴权中间件
function auth(req, res, next) {
  const u = readUser(req)
  if (!u) return res.status(401).json({ error: '请先登录' })
  req.user = u
  next()
}

function shapePost(row, userId) {
  return {
    id: row.id, cat: row.cat, author: row.author, avatar: row.avatar,
    title: row.title, body: row.body, cover: row.cover || null,
    tags: JSON.parse(row.tags || '[]'), festival: !!row.festival, tall: !!row.tall,
    likeCount: row.base_likes + Q.likeCount.get(row.id).n,
    collectCount: row.base_collects + Q.collectCount.get(row.id).n,
    liked: userId ? !!Q.liked.get(userId, row.id) : false,
    collected: userId ? !!Q.collected.get(userId, row.id) : false,
    comments: Q.commentsOf.all(row.id),
  }
}

// ---------- API ----------
app.get('/api/health', (req, res) => res.json({ ok: true, name: 'dalanshu', version: 1 }))

app.post('/api/register', (req, res) => {
  const { username, password, avatar } = req.body || {}
  const name = (username || '').trim()
  if (name.length < 2) return res.status(400).json({ error: '用户名至少 2 个字符' })
  if ((password || '').length < 4) return res.status(400).json({ error: '密码至少 4 位' })
  if (Q.userByName.get(name)) return res.status(409).json({ error: '这个用户名已被注册' })
  const hash = bcrypt.hashSync(password, 10)
  const info = Q.createUser.run(name, hash, avatar || '😎', '新来的散帅，请多关照 🌞', Date.now())
  const user = Q.userById.get(Number(info.lastInsertRowid))
  res.json({ token: sign({ id: user.id, username: name }), user: publicUser(user) })
})

app.post('/api/login', (req, res) => {
  const { username, password } = req.body || {}
  const row = Q.userByName.get((username || '').trim())
  if (!row) return res.status(404).json({ error: '用户不存在，去注册一个吧' })
  if (!bcrypt.compareSync(password || '', row.password_hash)) return res.status(401).json({ error: '密码不对' })
  res.json({ token: sign(row), user: publicUser(row) })
})

app.get('/api/me', auth, (req, res) => {
  const u = Q.userById.get(req.user.id)
  if (!u) return res.status(401).json({ error: '账号不存在' })
  res.json(publicUser(u))
})

app.get('/api/posts', (req, res) => {
  const u = readUser(req)
  res.json(Q.allPosts.all().map((r) => shapePost(r, u && u.id)))
})

app.post('/api/posts', auth, (req, res) => {
  const { title, body, cat, cover, tags, festival } = req.body || {}
  if (!(title || '').trim() || !(body || '').trim()) return res.status(400).json({ error: '标题和正文不能为空' })
  const me = Q.userById.get(req.user.id)
  const id = 'u' + Date.now() + Math.floor(Math.random() * 1000)
  Q.insertPost.run(
    id, cat || 'rec', me.username, me.avatar, title.trim(), body.trim(), cover || null,
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

app.get('/api/helps', (req, res) => res.json(Q.allHelps.all()))

app.post('/api/helps', auth, (req, res) => {
  const { type, title, body, city } = req.body || {}
  if (!(title || '').trim() || !(body || '').trim()) return res.status(400).json({ error: '标题和说明不能为空' })
  const me = Q.userById.get(req.user.id)
  const id = 'uh' + Date.now() + Math.floor(Math.random() * 1000)
  const item = {
    id, type: type === 'offer' ? 'offer' : 'need', author: me.username, avatar: me.avatar,
    title: title.trim(), body: body.trim(), city: (city || '').trim() || '同城',
    reward: type === 'need' ? '当面感谢' : '交个朋友', ts: '刚刚', created_at: Date.now(),
  }
  Q.insertHelp.run(item.id, item.type, item.author, item.avatar, item.title, item.body, item.city, item.reward, item.ts, item.created_at)
  res.json(item)
})

// ---------- 托管前端（SPA） ----------
app.use(express.static(PUBLIC_DIR))
app.get('*', (req, res) => res.sendFile(path.join(PUBLIC_DIR, 'index.html')))

app.listen(PORT, () => console.log(`[dalanshu] 服务已启动 http://0.0.0.0:${PORT}`))
