import { useState } from 'react'
import { catByKey } from '../data.js'
import { useStore } from '../store.jsx'

export default function PostDetail({ post, onClose, onMessage }) {
  const { toggleLike, toggleCollect, addComment, toggleFollow, me } = useStore()
  const [text, setText] = useState('')
  const cat = catByKey(post.cat)
  const emoji = post.cover || cat.emoji
  const grad = post.festival ? ['#ffb200', '#ff7a3d'] : cat.grad
  const isMine = !me.guest && post.author === me.name
  const following = (me.following || []).includes(post.author)

  const send = () => {
    if (!text.trim()) return
    addComment(post.id, text)
    setText('')
  }

  return (
    <div className="modal-mask" onClick={onClose}>
      <div className="sheet" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-head">
          <h3>{cat.emoji} {cat.name}</h3>
          <button className="icon-btn" onClick={onClose}>✕</button>
        </div>

        {post.image ? (
          <div className="detail-cover" style={{ padding: 0 }}>
            <img className="detail-img" src={post.image} alt="" />
          </div>
        ) : (
          <div className="detail-cover" style={{ background: `linear-gradient(135deg, ${grad[0]}, ${grad[1]})` }}>
            <span className="emoji">{emoji}</span>
          </div>
        )}

        <div className="detail-body">
          <h2>{post.title}</h2>
          <div className="text">{post.body}</div>

          {post.tags?.length > 0 && (
            <div className="tag-row">
              {post.tags.map((t) => (
                <span className="tag" key={t}>#{t}</span>
              ))}
            </div>
          )}

          <div className="detail-author">
            <span className="avatar">{post.avatar}</span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <b>{post.author}</b>
              <br />
              <small>{post.festival ? '散帅节参与者 · 阳光男孩' : '大蓝本儿用户'}</small>
            </div>
            {!isMine && (
              <div style={{ display: 'flex', gap: 8, flex: '0 0 auto' }}>
                <button className={`follow-btn ${following ? 'on' : ''}`} onClick={() => toggleFollow(post.author)}>
                  {following ? '已关注' : '+ 关注'}
                </button>
                <button className="msg-btn" onClick={() => onMessage && onMessage(post.author)}>私信</button>
              </div>
            )}
          </div>

          <div className="comments">
            <h4>共 {post.comments.length} 条评论</h4>
            {post.comments.length === 0 && (
              <p style={{ color: 'var(--ink-3)', fontSize: 14 }}>还没有评论，来抢沙发～</p>
            )}
            {post.comments.map((c, i) => (
              <div className="comment" key={i}>
                <span className="avatar">{c.avatar}</span>
                <div className="c-body">
                  <b>{c.author}</b>{!me.guest && c.author === me.name && <small> · 我</small>}
                  <p>{c.text}</p>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="action-bar">
          <input
            value={text}
            onChange={(e) => setText(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && send()}
            placeholder="说点什么，散帅之间要互相鼓励 💬"
          />
          <button className={`act ${post.liked ? 'on' : ''}`} onClick={() => toggleLike(post.id)}>
            <span className="ic">{post.liked ? '❤️' : '🤍'}</span>
            {post.likeCount}
          </button>
          <button className="act" onClick={() => toggleCollect(post.id)} style={post.collected ? { color: 'var(--sun)' } : undefined}>
            <span className="ic">{post.collected ? '⭐' : '☆'}</span>
            {post.collectCount}
          </button>
          <button className="act" onClick={send} style={{ color: 'var(--blue)' }}>
            <span className="ic">📨</span>发送
          </button>
        </div>
      </div>
    </div>
  )
}
