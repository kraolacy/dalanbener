import { useState } from 'react'
import { CATEGORIES, catByKey } from './data.js'
import { useStore } from './store.jsx'
import Feed from './components/Feed.jsx'
import Festival from './components/Festival.jsx'
import Help from './components/Help.jsx'
import Me from './components/Me.jsx'
import PostDetail from './components/PostDetail.jsx'
import CreatePost from './components/CreatePost.jsx'
import AuthModal from './components/AuthModal.jsx'
import ColumnIntro from './components/ColumnIntro.jsx'

export default function App() {
  const { posts, addPost, me, openAuth } = useStore()
  const [tab, setTab] = useState('home')
  const [cat, setCat] = useState('rec')
  const [q, setQ] = useState('')
  const [openPost, setOpenPost] = useState(null)
  const [creating, setCreating] = useState(false)
  const [toast, setToast] = useState('')

  const flash = (msg) => {
    setToast(msg)
    setTimeout(() => setToast(''), 1800)
  }

  const query = q.trim().toLowerCase()
  const homePosts = posts.filter((p) => {
    const catOk = cat === 'rec' || p.cat === cat || (cat === 'festival' && p.festival)
    if (!catOk) return false
    if (!query) return true
    const hay = (p.title + p.body + p.author + (p.tags || []).join(' ')).toLowerCase()
    return hay.includes(query)
  })

  // 详情用最新数据（点赞/评论会变），根据 id 从 store 重新取
  const livePost = openPost ? posts.find((p) => p.id === openPost.id) || openPost : null

  return (
    <div className="app-shell">
      <header className="topbar">
        <div className="topbar-inner">
          <div className="brand">
            <span className="logo">蓝</span>
            <span className="brand-text">
              <b>大蓝本儿</b>
              <small>Sunshine-note</small>
            </span>
          </div>
          <div className="search">
            🔍
            <input
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="搜兴趣、搜搭子、搜散帅…"
            />
          </div>
          {me.guest ? (
            <button className="login-btn" onClick={openAuth}>登录</button>
          ) : (
            <button className="topbar-avatar" onClick={() => setTab('me')} title={me.name}>{me.avatar}</button>
          )}
        </div>
      </header>

      {tab === 'home' && (
        <>
          <nav className="tabs">
            {CATEGORIES.map((c) => (
              <button
                key={c.key}
                className={`chip ${c.sun ? 'sun' : ''} ${cat === c.key ? 'active' : ''}`}
                onClick={() => setCat(c.key)}
              >
                {c.emoji} {c.name}
              </button>
            ))}
          </nav>
          <ColumnIntro cat={catByKey(cat)} />
          <Feed posts={homePosts} onOpen={setOpenPost} />
        </>
      )}

      {tab === 'festival' && <Festival posts={posts} onOpen={setOpenPost} />}
      {tab === 'help' && <Help />}
      {tab === 'me' && <Me onOpen={setOpenPost} />}

      {/* 底部导航 */}
      <nav className="bottom-nav">
        <div className="bottom-nav-inner">
          <button className={`nav-item ${tab === 'home' ? 'active' : ''}`} onClick={() => setTab('home')}>
            <span className="ic">🏠</span>首页
          </button>
          <button className={`nav-item ${tab === 'festival' ? 'active' : ''}`} onClick={() => setTab('festival')}>
            <span className="ic">🌞</span>散帅节
          </button>
          <button className="nav-fab" onClick={() => (me.guest ? openAuth() : setCreating(true))} aria-label="发布">＋</button>
          <button className={`nav-item ${tab === 'help' ? 'active' : ''}`} onClick={() => setTab('help')}>
            <span className="ic">🤝</span>互助
          </button>
          <button className={`nav-item ${tab === 'me' ? 'active' : ''}`} onClick={() => setTab('me')}>
            <span className="ic">😎</span>我
          </button>
        </div>
      </nav>

      {livePost && <PostDetail post={livePost} onClose={() => setOpenPost(null)} />}
      {creating && (
        <CreatePost
          onClose={() => setCreating(false)}
          onSubmit={(data) => {
            const created = addPost(data)
            setCreating(false)
            if (created) {
              setTab('home')
              setCat('rec')
              flash('发布成功，散帅节快乐 🌞')
            }
          }}
        />
      )}

      <AuthModal />

      {toast && <div className="toast">{toast}</div>}
    </div>
  )
}
