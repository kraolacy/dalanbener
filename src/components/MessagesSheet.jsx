import { useEffect, useRef, useState } from 'react'
import { useStore } from '../store.jsx'

export default function MessagesSheet({ onClose, initialTo }) {
  const { getConversations, getThread, sendMessage, me, backend } = useStore()
  const [convos, setConvos] = useState([])
  const [active, setActive] = useState(initialTo || null)
  const [thread, setThread] = useState(null)
  const [text, setText] = useState('')
  const [err, setErr] = useState('')
  const bottomRef = useRef(null)

  const loadConvos = async () => setConvos(await getConversations())
  const openThread = async (name) => { setActive(name); setThread(await getThread(name)) }

  useEffect(() => {
    if (!backend) return
    if (initialTo) openThread(initialTo)
    else loadConvos()
  }, []) // eslint-disable-line
  useEffect(() => { bottomRef.current?.scrollIntoView() }, [thread])

  const send = async () => {
    const t = text.trim()
    if (!t || !active) return
    const r = await sendMessage(active, t)
    if (!r.ok) { setErr(r.error || '发送失败'); return }
    setText(''); setErr('')
    setThread(await getThread(active))
  }

  // 本地演示版：没有后端就没有私信
  if (!backend) {
    return (
      <div className="modal-mask" onClick={onClose}>
        <div className="sheet" onClick={(e) => e.stopPropagation()} style={{ maxWidth: 420 }}>
          <div className="sheet-head"><span style={{ width: 32 }} /><h3>私信</h3><button className="icon-btn" onClick={onClose}>✕</button></div>
          <div className="empty"><div className="big">📭</div>私信是<b style={{ color: 'var(--cyan)' }}>联网版</b>（部署到 NAS / 服务器）的功能，<br />当前是本地演示版，暂不支持。</div>
        </div>
      </div>
    )
  }

  return (
    <div className="modal-mask" onClick={onClose}>
      <div className="sheet" onClick={(e) => e.stopPropagation()} style={{ maxWidth: 460, display: 'flex', flexDirection: 'column' }}>
        <div className="sheet-head">
          {active
            ? <button className="icon-btn" onClick={() => { setActive(null); setThread(null); loadConvos() }}>‹</button>
            : <span style={{ width: 32 }} />}
          <h3>{active ? `与 ${active}` : '私信'}</h3>
          <button className="icon-btn" onClick={onClose}>✕</button>
        </div>

        {!active ? (
          <div className="convo-list">
            {convos.length === 0 && (
              <div className="empty"><div className="big">📭</div>还没有私信。<br />去帖子里点作者的「私信」开聊吧～</div>
            )}
            {convos.map((c) => (
              <button className="convo" key={c.name} onClick={() => openThread(c.name)}>
                <span className="avatar" style={{ width: 42, height: 42, fontSize: 22 }}>{c.avatar}</span>
                <div className="convo-mid"><b>{c.name}</b><p>{c.last}</p></div>
                {c.unread > 0 && <span className="unread-dot">{c.unread}</span>}
              </button>
            ))}
          </div>
        ) : (
          <>
            <div className="thread">
              {thread?.messages?.length
                ? thread.messages.map((m, i) => (
                  <div key={i} className={`bubble ${m.from_name === me.name ? 'mine' : ''}`}>{m.text}</div>
                ))
                : <div className="empty" style={{ padding: '50px 20px' }}>还没有消息，发第一条打个招呼 👋</div>}
              <div ref={bottomRef} />
            </div>
            {err && <p style={{ color: 'var(--pink)', fontSize: 12, padding: '0 14px 6px' }}>⚠️ {err}</p>}
            <div className="action-bar">
              <input value={text} onChange={(e) => setText(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && send()} placeholder={`私信 ${active}…`} />
              <button className="act" onClick={send} style={{ color: 'var(--cyan)' }}><span className="ic">📨</span>发送</button>
            </div>
          </>
        )}
      </div>
    </div>
  )
}
