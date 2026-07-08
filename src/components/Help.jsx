import { useState } from 'react'
import { useStore } from '../store.jsx'

function HelpForm({ onClose }) {
  const { addHelp } = useStore()
  const [type, setType] = useState('need')
  const [title, setTitle] = useState('')
  const [body, setBody] = useState('')
  const [city, setCity] = useState('')
  const canPost = title.trim() && body.trim()

  return (
    <div className="modal-mask" onClick={onClose}>
      <div className="sheet" onClick={(e) => e.stopPropagation()}>
        <div className="sheet-head">
          <button className="icon-btn" onClick={onClose}>✕</button>
          <h3>发起互助</h3>
          <span style={{ width: 32 }} />
        </div>

        <div className="form-field">
          <label>类型</label>
          <div className="cat-picker">
            <button className={`cat-opt ${type === 'need' ? 'sel' : ''}`} onClick={() => setType('need')}>🙋 我需要帮助</button>
            <button className={`cat-opt ${type === 'offer' ? 'sel' : ''}`} onClick={() => setType('offer')}>🤝 我能帮忙</button>
          </div>
        </div>
        <div className="form-field">
          <label>标题</label>
          <input type="text" maxLength={40} value={title} placeholder={type === 'need' ? '例如：周末求搬家帮手' : '例如：免费帮装机清灰'} onChange={(e) => setTitle(e.target.value)} />
        </div>
        <div className="form-field">
          <label>说明</label>
          <textarea maxLength={300} value={body} placeholder="说清楚时间、地点、需要什么…" onChange={(e) => setBody(e.target.value)} />
        </div>
        <div className="form-field">
          <label>城市</label>
          <input type="text" maxLength={12} value={city} placeholder="例如：杭州" onChange={(e) => setCity(e.target.value)} />
        </div>
        <div className="submit-bar">
          <button className="submit-btn" disabled={!canPost} onClick={() => { addHelp({ type, title, body, city }); onClose() }}>
            发布互助
          </button>
        </div>
      </div>
    </div>
  )
}

export default function Help() {
  const { helps } = useStore()
  const [open, setOpen] = useState(false)

  return (
    <div className="page">
      <div className="section-title" style={{ justifyContent: 'space-between' }}>
        <span>🤝 散帅互助 · 搭子广场</span>
        <button className="pill-btn" onClick={() => setOpen(true)}>+ 发起</button>
      </div>
      <p style={{ color: 'var(--ink-2)', fontSize: 13.5, margin: '0 2px 14px' }}>
        搬家、修电脑、找搭子、约球局……兄弟之间搭把手，散帅节更要互相照应。
      </p>

      {helps.map((h) => (
        <div className="help-card" key={h.id}>
          <div className="top">
            <span className="avatar" style={{ width: 26, height: 26 }}>{h.avatar}</span>
            <b style={{ fontSize: 13.5 }}>{h.author}</b>
            <span className={`help-badge ${h.type}`}>{h.type === 'need' ? '求助' : '帮忙'}</span>
            <span style={{ marginLeft: 'auto', color: 'var(--ink-3)', fontSize: 12 }}>{h.ts}</span>
          </div>
          <h3>{h.title}</h3>
          <p>{h.body}</p>
          <div className="foot">
            <span style={{ color: 'var(--ink-3)', fontSize: 12.5 }}>📍 {h.city} · 🎁 {h.reward}</span>
            <button className="pill-btn ghost">{h.type === 'need' ? '我来帮' : '联系他'}</button>
          </div>
        </div>
      ))}

      {open && <HelpForm onClose={() => setOpen(false)} />}
    </div>
  )
}
