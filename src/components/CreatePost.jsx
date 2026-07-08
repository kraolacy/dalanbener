import { useState } from 'react'
import { CATEGORIES } from '../data.js'

const EMOJIS = ['💪', '⌨️', '🎣', '🏀', '📱', '🚗', '🍳', '⛺', '☕', '💻', '👟', '📷', '🌞', '😎', '🍺', '🎮']
const POST_CATS = CATEGORIES.filter((c) => c.key !== 'rec') // “推荐”是聚合，不作为发帖分类

export default function CreatePost({ onClose, onSubmit }) {
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [cat, setCat] = useState('fitness')
  const [cover, setCover] = useState('💪')
  const [tagStr, setTagStr] = useState('')
  const [festival, setFestival] = useState(false)

  const canPost = title.trim().length > 0 && body.trim().length > 0

  const submit = () => {
    if (!canPost) return
    const tags = tagStr
      .split(/[,，#\s]+/)
      .map((s) => s.trim())
      .filter(Boolean)
      .slice(0, 5)
    onSubmit({ title, body, cat: festival ? 'festival' : cat, cover, tags, festival: festival || cat === 'festival' })
  }

  return (
    <div className="modal-mask" onClick={onClose}>
      <div className="sheet" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-head">
          <button className="icon-btn" onClick={onClose}>✕</button>
          <h3>发布散帅动态</h3>
          <button
            className="icon-btn"
            style={{ background: canPost ? 'var(--blue)' : undefined, color: canPost ? '#fff' : undefined, width: 'auto', padding: '0 14px', borderRadius: 999 }}
            onClick={submit}
            disabled={!canPost}
          >
            发布
          </button>
        </div>

        <div className="form-field">
          <label>封面</label>
          <div className="emoji-picker">
            {EMOJIS.map((e) => (
              <button key={e} className={`emoji-opt ${cover === e ? 'sel' : ''}`} onClick={() => setCover(e)}>
                {e}
              </button>
            ))}
          </div>
        </div>

        <div className="form-field">
          <label>标题</label>
          <input
            type="text"
            value={title}
            maxLength={40}
            placeholder="起个吸引兄弟们的标题…"
            onChange={(e) => setTitle(e.target.value)}
          />
        </div>

        <div className="form-field">
          <label>正文</label>
          <textarea
            value={body}
            maxLength={800}
            placeholder="分享你的兴趣、生活、经验，或者发起一个散帅节的约局…"
            onChange={(e) => setBody(e.target.value)}
          />
        </div>

        <div className="form-field">
          <label>圈子</label>
          <div className="cat-picker">
            {POST_CATS.map((c) => (
              <button
                key={c.key}
                className={`cat-opt ${cat === c.key ? 'sel' : ''}`}
                onClick={() => { setCat(c.key); if (c.key === 'festival') setFestival(true) }}
              >
                {c.emoji} {c.name}
              </button>
            ))}
          </div>
        </div>

        <div className="form-field">
          <label>话题标签（用空格或逗号分隔，最多 5 个）</label>
          <input
            type="text"
            value={tagStr}
            placeholder="例如：健身 增肌 我的散帅时刻"
            onChange={(e) => setTagStr(e.target.value)}
          />
        </div>

        <div className="form-field" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <label style={{ margin: 0 }}>🌞 参加 8·3 散帅节话题</label>
          <button
            onClick={() => setFestival((v) => !v)}
            style={{
              width: 48, height: 28, borderRadius: 999,
              background: festival ? 'var(--sun)' : 'var(--line)',
              position: 'relative', transition: '.15s',
            }}
            aria-label="切换散帅节话题"
          >
            <span style={{
              position: 'absolute', top: 3, left: festival ? 23 : 3,
              width: 22, height: 22, borderRadius: '50%', background: '#fff', transition: '.15s',
              boxShadow: '0 1px 3px rgba(0,0,0,.2)',
            }} />
          </button>
        </div>

        <div className="submit-bar">
          <button className="submit-btn" onClick={submit} disabled={!canPost}>
            发布到大蓝本儿
          </button>
        </div>
      </div>
    </div>
  )
}
