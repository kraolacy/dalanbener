import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import { SEED_POSTS, SEED_HELP } from './data.js'

// ===== 本地存储封装 =====
const LS = {
  get(key, fallback) {
    try {
      const v = localStorage.getItem(key)
      return v ? JSON.parse(v) : fallback
    } catch {
      return fallback
    }
  },
  set(key, val) {
    try { localStorage.setItem(key, JSON.stringify(val)) } catch { /* 隐私模式等，忽略 */ }
  },
}

const K = {
  userPosts: 'dls_userPosts',
  likes: 'dls_likes',
  collects: 'dls_collects',
  comments: 'dls_comments',
  userHelp: 'dls_userHelp',
  accounts: 'dls_accounts',       // { [用户名]: { password, avatar, bio } }
  currentUser: 'dls_currentUser', // 当前登录用户名，或 null
}

const GUEST = { name: '游客', avatar: '🙂', bio: '登录后开启你的散帅之旅', guest: true }

const StoreCtx = createContext(null)

export function StoreProvider({ children }) {
  const [userPosts, setUserPosts] = useState(() => LS.get(K.userPosts, []))
  const [likes, setLikes] = useState(() => LS.get(K.likes, {}))
  const [collects, setCollects] = useState(() => LS.get(K.collects, {}))
  const [comments, setComments] = useState(() => LS.get(K.comments, {}))
  const [userHelp, setUserHelp] = useState(() => LS.get(K.userHelp, []))
  const [accounts, setAccounts] = useState(() => LS.get(K.accounts, {}))
  const [currentUser, setCurrentUser] = useState(() => LS.get(K.currentUser, null))
  const [authOpen, setAuthOpen] = useState(false) // 登录/注册弹窗

  useEffect(() => LS.set(K.userPosts, userPosts), [userPosts])
  useEffect(() => LS.set(K.likes, likes), [likes])
  useEffect(() => LS.set(K.collects, collects), [collects])
  useEffect(() => LS.set(K.comments, comments), [comments])
  useEffect(() => LS.set(K.userHelp, userHelp), [userHelp])
  useEffect(() => LS.set(K.accounts, accounts), [accounts])
  useEffect(() => LS.set(K.currentUser, currentUser), [currentUser])

  // 当前用户：已登录取账号信息，否则游客
  const me = useMemo(() => {
    if (currentUser && accounts[currentUser]) {
      const a = accounts[currentUser]
      return { name: currentUser, avatar: a.avatar || '😎', bio: a.bio || '', guest: false }
    }
    return GUEST
  }, [currentUser, accounts])

  const isLoggedIn = !me.guest
  const openAuth = () => setAuthOpen(true)
  const closeAuth = () => setAuthOpen(false)
  // 需要登录的操作：未登录则弹出登录窗并返回 false
  const gate = () => {
    if (isLoggedIn) return true
    openAuth()
    return false
  }

  // 合并帖子并附带运行时状态
  const posts = useMemo(() => {
    const all = [...userPosts, ...SEED_POSTS]
    return all.map((p) => {
      const extra = comments[p.id] || []
      const liked = !!likes[p.id]
      const collected = !!collects[p.id]
      return {
        ...p,
        liked,
        collected,
        likeCount: (p.likes || 0) + (liked ? 1 : 0),
        collectCount: (p.collects || 0) + (collected ? 1 : 0),
        comments: [...(p.comments || []), ...extra],
      }
    })
  }, [userPosts, likes, collects, comments])

  const helps = useMemo(() => [...userHelp, ...SEED_HELP], [userHelp])

  const toggle = (setter) => (id) => {
    if (!gate()) return
    setter((m) => {
      const next = { ...m }
      if (next[id]) delete next[id]
      else next[id] = true
      return next
    })
  }

  const actions = {
    // ---- 账号 ----
    register({ username, password, avatar }) {
      const name = (username || '').trim()
      if (name.length < 2) return { ok: false, error: '用户名至少 2 个字符' }
      if ((password || '').length < 4) return { ok: false, error: '密码至少 4 位' }
      if (accounts[name]) return { ok: false, error: '这个用户名已被注册' }
      setAccounts((m) => ({ ...m, [name]: { password, avatar: avatar || '😎', bio: '新来的散帅，请多关照 🌞' } }))
      setCurrentUser(name)
      setAuthOpen(false)
      return { ok: true }
    },
    login({ username, password }) {
      const name = (username || '').trim()
      const a = accounts[name]
      if (!a) return { ok: false, error: '用户不存在，去注册一个吧' }
      if (a.password !== password) return { ok: false, error: '密码不对' }
      setCurrentUser(name)
      setAuthOpen(false)
      return { ok: true }
    },
    logout() { setCurrentUser(null) },
    openAuth,
    closeAuth,

    // ---- 内容（均需登录）----
    toggleLike: toggle(setLikes),
    toggleCollect: toggle(setCollects),
    addComment(postId, text) {
      if (!gate()) return
      const t = text.trim()
      if (!t) return
      setComments((m) => ({
        ...m,
        [postId]: [...(m[postId] || []), { author: me.name, avatar: me.avatar, text: t }],
      }))
    },
    addPost({ title, body, cat, cover, tags, festival }) {
      if (!gate()) return null
      const post = {
        id: 'u' + Date.now(),
        cat,
        author: me.name,
        avatar: me.avatar,
        title: title.trim(),
        body: body.trim(),
        cover,
        tags: tags || [],
        likes: 0,
        collects: 0,
        comments: [],
        festival: !!festival,
        tall: (body || '').length > 60,
      }
      setUserPosts((list) => [post, ...list])
      return post
    },
    addHelp({ type, title, body, city }) {
      if (!gate()) return null
      const item = {
        id: 'uh' + Date.now(),
        type,
        author: me.name,
        avatar: me.avatar,
        title: title.trim(),
        body: body.trim(),
        city: city.trim() || '同城',
        reward: type === 'need' ? '当面感谢' : '交个朋友',
        ts: '刚刚',
      }
      setUserHelp((list) => [item, ...list])
      return item
    },
  }

  const value = { posts, helps, likes, collects, me, isLoggedIn, authOpen, ...actions }
  return <StoreCtx.Provider value={value}>{children}</StoreCtx.Provider>
}

export const useStore = () => {
  const ctx = useContext(StoreCtx)
  if (!ctx) throw new Error('useStore 必须在 StoreProvider 内使用')
  return ctx
}
