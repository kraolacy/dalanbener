const path = require('path')
const fs = require('fs')
const { DatabaseSync } = require('node:sqlite') // Node 内置 SQLite（Node 22.5+/24），免原生编译
const { SEED_POSTS, SEED_HELP } = require('./seed')

const DB_PATH = process.env.DB_PATH || path.join(__dirname, 'data', 'dalanshu.db')
fs.mkdirSync(path.dirname(DB_PATH), { recursive: true })

const db = new DatabaseSync(DB_PATH)
db.exec('PRAGMA journal_mode = WAL;')

db.exec(`
  CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    avatar TEXT DEFAULT '😎',
    bio TEXT DEFAULT '',
    created_at INTEGER NOT NULL
  );
  CREATE TABLE IF NOT EXISTS posts (
    id TEXT PRIMARY KEY,
    cat TEXT NOT NULL,
    author TEXT NOT NULL,
    avatar TEXT DEFAULT '😎',
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    cover TEXT,
    image TEXT,
    tags TEXT DEFAULT '[]',
    festival INTEGER DEFAULT 0,
    tall INTEGER DEFAULT 0,
    base_likes INTEGER DEFAULT 0,
    base_collects INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL
  );
  CREATE TABLE IF NOT EXISTS comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_id TEXT NOT NULL,
    author TEXT NOT NULL,
    avatar TEXT DEFAULT '😎',
    text TEXT NOT NULL,
    created_at INTEGER NOT NULL
  );
  CREATE TABLE IF NOT EXISTS likes (
    user_id INTEGER NOT NULL, post_id TEXT NOT NULL, PRIMARY KEY (user_id, post_id)
  );
  CREATE TABLE IF NOT EXISTS collects (
    user_id INTEGER NOT NULL, post_id TEXT NOT NULL, PRIMARY KEY (user_id, post_id)
  );
  CREATE TABLE IF NOT EXISTS helps (
    id TEXT PRIMARY KEY, type TEXT NOT NULL, author TEXT NOT NULL, avatar TEXT DEFAULT '😎',
    title TEXT NOT NULL, body TEXT NOT NULL, city TEXT DEFAULT '同城', reward TEXT DEFAULT '交个朋友',
    ts TEXT DEFAULT '刚刚', created_at INTEGER NOT NULL
  );
  CREATE TABLE IF NOT EXISTS follows (
    follower_id INTEGER NOT NULL, target TEXT NOT NULL, PRIMARY KEY (follower_id, target)
  );
  CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_name TEXT NOT NULL, to_name TEXT NOT NULL, text TEXT NOT NULL,
    read INTEGER DEFAULT 0, created_at INTEGER NOT NULL
  );
`)

// 老库平滑升级：补 posts.image 列（新库已含，忽略"重复列"错误）
try { db.exec('ALTER TABLE posts ADD COLUMN image TEXT') } catch { /* 列已存在 */ }

// 首次启动写入种子数据
function seedIfEmpty() {
  const n = db.prepare('SELECT COUNT(*) AS n FROM posts').get().n
  if (n > 0) return
  const now = Date.now()
  const insPost = db.prepare(`INSERT INTO posts
    (id,cat,author,avatar,title,body,cover,tags,festival,tall,base_likes,base_collects,created_at)
    VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`)
  const insComment = db.prepare('INSERT INTO comments (post_id,author,avatar,text,created_at) VALUES (?,?,?,?,?)')
  const insHelp = db.prepare(`INSERT INTO helps (id,type,author,avatar,title,body,city,reward,ts,created_at)
    VALUES (?,?,?,?,?,?,?,?,?,?)`)

  db.exec('BEGIN')
  try {
    SEED_POSTS.forEach((p, i) => {
      const t = now - i * 60000
      insPost.run(p.id, p.cat, p.author, p.avatar, p.title, p.body, p.cover || null,
        JSON.stringify(p.tags || []), p.festival ? 1 : 0, p.tall ? 1 : 0, p.likes || 0, p.collects || 0, t)
      ;(p.comments || []).forEach((c, j) => insComment.run(p.id, c.author, c.avatar, c.text, t + j * 1000))
    })
    SEED_HELP.forEach((h, i) => insHelp.run(h.id, h.type, h.author, h.avatar, h.title, h.body, h.city, h.reward, h.ts, now - i * 60000))
    db.exec('COMMIT')
  } catch (e) { db.exec('ROLLBACK'); throw e }
  console.log(`[db] 已写入种子：${SEED_POSTS.length} 帖 / ${SEED_HELP.length} 互助`)
}
seedIfEmpty()

module.exports = db
