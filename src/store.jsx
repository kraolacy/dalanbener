import { createContext, useContext, useEffect, useMemo, useState } from 'react'
import { SEED_POSTS, SEED_HELP } from './data.js'
import { api, setToken as apiSetToken } from './api.js'

// ===== 本地存储封装 =====
const LS = {
  get(key, fallback) {
    try { const v = localStorage.getItem(key); return v ? JSON.parse(v) : fallback } catch { return fallback }
  },
  set(key, val) { try { localStorage.setItem(key, JSON.stringify(val)) } catch { /* 隐私模式忽略 */ } },
  del(key) { try { localStorage.removeItem(key) } catch { /* ignore */ } },
}
const K = {
  userPosts: 'dls_userPosts', likes: 'dls_likes', collects: 'dls_collects',
  comments: 'dls_comments', userHelp: 'dls_userHelp', accounts: 'dls_accounts',
  currentUser: 'dls_currentUser', token: 'dls_token', following: 'dls_following',
}
const GUEST = { name: '游客', avatar: '🙂', bio: '登录后开启你的散帅之旅', guest: true, following: [], followers: 0, unread: 0 }

const StoreCtx = createContext(null)

export function StoreProvider({ children }) {
  // —— 运行模式 ——
  const [backend, setBackend] = useState(false)   // 后端是否可用
  const [ready, setReady] = useState(false)       // 首屏数据是否就绪
  // —— 后端模式状态 ——
  const [apiPosts, setApiPosts] = useState([])
  const [apiHelps, setApiHelps] = useState([])
  const [apiMe, setApiMe] = useState(null)
  const [token, setTok] = useState(() => LS.get(K.token, null))
  // —— localStorage 模式状态 ——
  const [userPosts, setUserPosts] = useState(() => LS.get(K.userPosts, []))
  const [lLikes, setLLikes] = useState(() => LS.get(K.likes, {}))
  const [lCollects, setLCollects] = useState(() => LS.get(K.collects, {}))
  const [lComments, setLComments] = useState(() => LS.get(K.comments, {}))
  const [userHelp, setUserHelp] = useState(() => LS.get(K.userHelp, []))
  const [accounts, setAccounts] = useState(() => LS.get(K.accounts, {}))
  const [currentUser, setCurrentUser] = useState(() => LS.get(K.currentUser, null))
  const [lFollowing, setLFollowing] = useState(() => LS.get(K.following, {}))
  const [authOpen, setAuthOpen] = useState(false)

  // localStorage 持久化
  useEffect(() => LS.set(K.userPosts, userPosts), [userPosts])
  useEffect(() => LS.set(K.likes, lLikes), [lLikes])
  useEffect(() => LS.set(K.collects, lCollects), [lCollects])
  useEffect(() => LS.set(K.comments, lComments), [lComments])
  useEffect(() => LS.set(K.userHelp, userHelp), [userHelp])
  useEffect(() => LS.set(K.accounts, accounts), [accounts])
  useEffect(() => LS.set(K.currentUser, currentUser), [currentUser])
  useEffect(() => LS.set(K.following, lFollowing), [lFollowing])

  // —— 启动：探测后端 ——
  useEffect(() => {
    let alive = true
    ;(async () => {
      const online = await api.health()
      if (!alive) return
      if (online) {
        setBackend(true)
        if (token) {
          apiSetToken(token)
          try { setApiMe(await api.me()) } catch { apiSetToken(null); setTok(null); LS.del(K.token) }
        }
        try {
          const [p, h] = await Promise.all([api.posts(), api.helps()])
          if (alive) { setApiPosts(p); setApiHelps(h) }
        } catch { /* 忽略，留空 */ }
      }
      if (alive) setReady(true)
    })()
    return () => { alive = false }
  }, []) // eslint-disable-line

  const refresh = async () => {
    const [p, h] = await Promise.all([api.posts(), api.helps()])
    setApiPosts(p); setApiHelps(h)
  }
  const refreshMe = async () => { if (backend && token) { try { setApiMe(await api.me()) } catch { /* ignore */ } } }
  const persistToken = (t) => { setTok(t); apiSetToken(t); if (t) LS.set(K.token, t); else LS.del(K.token) }

  // —— 当前用户 ——
  const me = useMemo(() => {
    if (backend) return apiMe
      ? { name: apiMe.name, avatar: apiMe.avatar || '😎', bio: apiMe.bio || '', guest: false,
          following: apiMe.following || [], followers: apiMe.followers || 0, unread: apiMe.unread || 0 }
      : GUEST
    if (currentUser && accounts[currentUser]) {
      const a = accounts[currentUser]
      return { name: currentUser, avatar: a.avatar || '😎', bio: a.bio || '', guest: false,
        following: Object.keys(lFollowing), followers: 0, unread: 0 }
    }
    return GUEST
  }, [backend, apiMe, currentUser, accounts, lFollowing])
  const isLoggedIn = !me.guest

  // —— 帖子/互助（按模式取源）——
  const lsPosts = useMemo(() => {
    return [...userPosts, ...SEED_POSTS].map((p) => {
      const extra = lComments[p.id] || []
      const liked = !!lLikes[p.id]
      const collected = !!lCollects[p.id]
      return {
        ...p, liked, collected,
        likeCount: (p.likes || 0) + (liked ? 1 : 0),
        collectCount: (p.collects || 0) + (collected ? 1 : 0),
        comments: [...(p.comments || []), ...extra],
      }
    })
  }, [userPosts, lLikes, lCollects, lComments])

  const posts = backend ? apiPosts : lsPosts
  const helps = backend ? apiHelps : [...userHelp, ...SEED_HELP]

  // Me 页统计用的 likes/collects 对象（两模式都提供兼容结构）
  const likes = backend
    ? Object.fromEntries(apiPosts.filter((p) => p.liked).map((p) => [p.id, true]))
    : lLikes
  const collects = backend
    ? Object.fromEntries(apiPosts.filter((p) => p.collected).map((p) => [p.id, true]))
    : lCollects

  const openAuth = () => setAuthOpen(true)
  const closeAuth = () => setAuthOpen(false)
  const gate = () => { if (isLoggedIn) return true; openAuth(); return false }

  // localStorage 模式的点赞/收藏切换
  const lsToggle = (setter) => (id) => setter((m) => {
    const next = { ...m }; if (next[id]) delete next[id]; else next[id] = true; return next
  })

  const actions = {
    // ---- 账号 ----
    async register({ username, password, avatar }) {
      if (backend) {
        try {
          const { token: t, user } = await api.register({ username, password, avatar })
          persistToken(t); setApiMe(user); setAuthOpen(false); await refresh(); return { ok: true }
        } catch (e) { return { ok: false, error: e.message } }
      }
      const name = (username || '').trim()
      if (name.length < 2) return { ok: false, error: '用户名至少 2 个字符' }
      if ((password || '').length < 4) return { ok: false, error: '密码至少 4 位' }
      if (accounts[name]) return { ok: false, error: '这个用户名已被注册' }
      setAccounts((m) => ({ ...m, [name]: { password, avatar: avatar || '😎', bio: '新来的散帅，请多关照 🌞' } }))
      setCurrentUser(name); setAuthOpen(false); return { ok: true }
    },
    async login({ username, password }) {
      if (backend) {
        try {
          const { token: t, user } = await api.login({ username, password })
          persistToken(t); setApiMe(user); setAuthOpen(false); await refresh(); return { ok: true }
        } catch (e) { return { ok: false, error: e.message } }
      }
      const name = (username || '').trim()
      const a = accounts[name]
      if (!a) return { ok: false, error: '用户不存在，去注册一个吧' }
      if (a.password !== password) return { ok: false, error: '密码不对' }
      setCurrentUser(name); setAuthOpen(false); return { ok: true }
    },
    logout() {
      if (backend) { persistToken(null); setApiMe(null); refresh() }
      else setCurrentUser(null)
    },
    openAuth, closeAuth,

    // ---- 互动（均需登录）----
    async toggleLike(id) {
      if (!gate()) return
      if (backend) { try { const p = await api.toggleLike(id); setApiPosts((list) => list.map((x) => x.id === id ? p : x)) } catch { /* ignore */ } }
      else lsToggle(setLLikes)(id)
    },
    async toggleCollect(id) {
      if (!gate()) return
      if (backend) { try { const p = await api.toggleCollect(id); setApiPosts((list) => list.map((x) => x.id === id ? p : x)) } catch { /* ignore */ } }
      else lsToggle(setLCollects)(id)
    },
    async addComment(postId, text) {
      if (!gate()) return
      const t = (text || '').trim(); if (!t) return
      if (backend) { try { const p = await api.addComment(postId, t); setApiPosts((list) => list.map((x) => x.id === postId ? p : x)) } catch { /* ignore */ } }
      else setLComments((m) => ({ ...m, [postId]: [...(m[postId] || []), { author: me.name, avatar: me.avatar, text: t }] }))
    },
    async addPost({ title, body, cat, cover, image, tags, festival }) {
      if (!gate()) return null
      if (backend) {
        try {
          let img = image || null
          if (img && img.startsWith('data:')) { const u = await api.upload(img); img = u.url } // 先上传图片拿到 URL
          const p = await api.createPost({ title, body, cat, cover, image: img, tags, festival })
          setApiPosts((list) => [p, ...list]); return p
        } catch { return null }
      }
      const post = {
        id: 'u' + Date.now(), cat, author: me.name, avatar: me.avatar,
        title: title.trim(), body: body.trim(), cover, image: image || null, tags: tags || [],
        likes: 0, collects: 0, comments: [], festival: !!festival, tall: (body || '').length > 60,
      }
      setUserPosts((list) => [post, ...list]); return post
    },
    // ---- 关注 / 私信 ----
    async toggleFollow(name) {
      if (!gate()) return
      if (name === me.name) return
      if (backend) { try { setApiMe(await api.follow(name)) } catch { /* ignore */ } }
      else setLFollowing((m) => { const n = { ...m }; if (n[name]) delete n[name]; else n[name] = true; return n })
    },
    refreshMe,
    async getConversations() { if (!backend) return []; try { return await api.conversations() } catch { return [] } },
    async getThread(name) { if (!backend) return null; try { const t = await api.thread(name); refreshMe(); return t } catch { return null } },
    async sendMessage(to, text) {
      if (!isLoggedIn) { openAuth(); return { ok: false } }
      if (!backend) return { ok: false, error: '私信是联网版（NAS/服务器）功能，本地演示版暂不支持' }
      try { await api.sendMessage(to, text); return { ok: true } } catch (e) { return { ok: false, error: e.message } }
    },
    async addHelp({ type, title, body, city }) {
      if (!gate()) return null
      if (backend) {
        try { const h = await api.createHelp({ type, title, body, city }); setApiHelps((list) => [h, ...list]); return h }
        catch { return null }
      }
      const item = {
        id: 'uh' + Date.now(), type, author: me.name, avatar: me.avatar,
        title: title.trim(), body: body.trim(), city: (city || '').trim() || '同城',
        reward: type === 'need' ? '当面感谢' : '交个朋友', ts: '刚刚',
      }
      setUserHelp((list) => [item, ...list]); return item
    },
  }

  const value = { posts, helps, likes, collects, me, isLoggedIn, authOpen, ready, backend, ...actions }
  return <StoreCtx.Provider value={value}>{children}</StoreCtx.Provider>
}

export const useStore = () => {
  const ctx = useContext(StoreCtx)
  if (!ctx) throw new Error('useStore 必须在 StoreProvider 内使用')
  return ctx
}
