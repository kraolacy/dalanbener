import { useState } from 'react'
import { useStore } from '../store.jsx'

const AVATARS = ['😎', '🧔', '🤴', '🦸', '🧑‍💻', '🏋️', '🎮', '🎣', '🏀', '☕', '🚗', '🐺']

export default function AuthModal() {
  const { authOpen, closeAuth, login, register } = useStore()
  const [mode, setMode] = useState('login') // 'login' | 'register'
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [avatar, setAvatar] = useState('😎')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  if (!authOpen) return null

  const thirdParty = (name) =>
    setNotice(`${name}登录需接入后端与「${name}」开放平台开发者认证，演示版暂未开通，请先用账号密码 🙏`)

  const submit = () => {
    setError('')
    const res = mode === 'login'
      ? login({ username, password })
      : register({ username, password, avatar })
    if (!res.ok) setError(res.error)
    else { setUsername(''); setPassword('') }
  }

  return (
    <div className="modal-mask" onClick={closeAuth}>
      <div className="sheet" onClick={(e) => e.stopPropagation()} style={{ maxWidth: 420 }}>
        <div className="sheet-head">
          <button className="icon-btn" onClick={closeAuth}>✕</button>
          <h3>{mode === 'login' ? '登录大蓝书' : '注册大蓝书'}</h3>
          <span style={{ width: 32 }} />
        </div>

        <div style={{ padding: '4px 18px 0', textAlign: 'center' }}>
          <div style={{ fontSize: 40, margin: '10px 0 2px' }}>💙</div>
          <p style={{ color: 'var(--ink-3)', fontSize: 13 }}>散散帅 · 交交友 —— 加入散帅大家庭</p>
        </div>

        {mode === 'register' && (
          <div className="form-field">
            <label>选个头像</label>
            <div className="emoji-picker">
              {AVATARS.map((e) => (
                <button key={e} className={`emoji-opt ${avatar === e ? 'sel' : ''}`} onClick={() => setAvatar(e)}>{e}</button>
              ))}
            </div>
          </div>
        )}

        <div className="form-field">
          <label>用户名</label>
          <input type="text" value={username} maxLength={16} placeholder="给自己起个散帅昵称"
            onChange={(e) => setUsername(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && submit()} />
        </div>
        <div className="form-field">
          <label>密码</label>
          <input type="password" value={password} maxLength={32} placeholder="至少 4 位"
            onChange={(e) => setPassword(e.target.value)} onKeyDown={(e) => e.key === 'Enter' && submit()} />
        </div>

        {error && (
          <div style={{ padding: '4px 18px', color: 'var(--like)', fontSize: 13, fontWeight: 600 }}>⚠️ {error}</div>
        )}

        <div className="submit-bar">
          <button className="submit-btn" onClick={submit} disabled={!username || !password}>
            {mode === 'login' ? '登录' : '注册并登录'}
          </button>
          <p style={{ textAlign: 'center', marginTop: 14, fontSize: 13.5, color: 'var(--ink-2)' }}>
            {mode === 'login' ? '还没有账号？' : '已经有账号了？'}
            <button
              style={{ color: 'var(--blue)', fontWeight: 700, marginLeft: 4 }}
              onClick={() => { setMode(mode === 'login' ? 'register' : 'login'); setError('') }}
            >
              {mode === 'login' ? '去注册' : '去登录'}
            </button>
          </p>
          <div className="third-party">
            <div className="tp-divider"><span>或使用第三方账号登录</span></div>
            <div className="tp-icons">
              <button className="tp tp-wechat" onClick={() => thirdParty('微信')} title="微信登录">💬</button>
              <button className="tp tp-qq" onClick={() => thirdParty('QQ')} title="QQ登录">🐧</button>
              <button className="tp tp-douyin" onClick={() => thirdParty('抖音')} title="抖音登录">🎵</button>
              <button className="tp tp-phone" onClick={() => thirdParty('手机号')} title="手机号登录">📱</button>
            </div>
            {notice && <p className="tp-notice">🚧 {notice}</p>}
          </div>

          <p style={{ textAlign: 'center', marginTop: 12, fontSize: 11.5, color: 'var(--ink-3)' }}>
            演示用途，账号仅保存在本机浏览器，请勿使用真实密码
          </p>
        </div>
      </div>
    </div>
  )
}
