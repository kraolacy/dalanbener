import { useState } from 'react'
import { useStore } from '../store.jsx'
import Feed from './Feed.jsx'

export default function Me({ onOpen }) {
  const { posts, me, likes, collects, openAuth, logout } = useStore()
  const [tab, setTab] = useState('mine')

  // 未登录：显示登录引导
  if (me.guest) {
    return (
      <div className="empty" style={{ paddingTop: 80 }}>
        <div className="big">🙂</div>
        <p style={{ fontSize: 16, fontWeight: 700, color: 'var(--ink)' }}>还没登录</p>
        <p style={{ margin: '8px 0 20px' }}>登录后才能发帖、点赞、评论、找搭子</p>
        <button className="submit-btn" style={{ maxWidth: 220, margin: '0 auto' }} onClick={openAuth}>
          登录 / 注册
        </button>
      </div>
    )
  }

  const minePosts = posts.filter((p) => p.author === me.name)
  const collectedPosts = posts.filter((p) => p.collected)
  const likeCount = Object.keys(likes).length
  const collectCount = Object.keys(collects).length

  return (
    <div>
      <section className="profile">
        <div className="row">
          <div className="big-avatar">{me.avatar}</div>
          <div style={{ flex: 1 }}>
            <h2>{me.name}</h2>
            <div className="bio">{me.bio}</div>
          </div>
          <button className="logout-btn" onClick={logout}>退出</button>
        </div>
        <div className="stats">
          <div><b>{minePosts.length}</b><span>发布</span></div>
          <div><b>{likeCount}</b><span>点赞</span></div>
          <div><b>{collectCount}</b><span>收藏</span></div>
        </div>
      </section>

      <div className="badge-row">
        <span className="badge">🌞 散帅节首批用户</span>
        <span className="badge">💙 大蓝本儿 · 蓝V</span>
        <span className="badge">🤝 热心搭子</span>
      </div>

      <div className="page" style={{ paddingTop: 4, paddingBottom: 0 }}>
        <div className="tabs" style={{ position: 'static', padding: '6px 0' }}>
          <button className={`chip ${tab === 'mine' ? 'active' : ''}`} onClick={() => setTab('mine')}>📝 我的发布</button>
          <button className={`chip ${tab === 'fav' ? 'active' : ''}`} onClick={() => setTab('fav')}>⭐ 我的收藏</button>
        </div>
      </div>

      <Feed posts={tab === 'mine' ? minePosts : collectedPosts} onOpen={onOpen} />
    </div>
  )
}
