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
}

// 当前用户（演示固定账号）
export const ME = { name: '蓝书散帅', avatar: '😎', bio: '一个爱散步、爱撸铁、爱交朋友的阳光散帅' }

const StoreCtx = createContext(null)

export function StoreProvider({ children }) {
  const [userPosts, setUserPosts] = useState(() => LS.get(K.userPosts, []))
  const [likes, setLikes] = useState(() => LS.get(K.likes, {}))
  const [collects, setCollects] = useState(() => LS.get(K.collects, {}))
  const [comments, setComments] = useState(() => LS.get(K.comments, {})) // { postId: [ {author,avatar,text} ] }
  const [userHelp, setUserHelp] = useState(() => LS.get(K.userHelp, []))

  useEffect(() => LS.set(K.userPosts, userPosts), [userPosts])
  useEffect(() => LS.set(K.likes, likes), [likes])
  useEffect(() => LS.set(K.collects, collects), [collects])
  useEffect(() => LS.set(K.comments, comments), [comments])
  useEffect(() => LS.set(K.userHelp, userHelp), [userHelp])

  // 合并：用户帖在前，附带运行时状态（点赞数、评论）
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

  const toggle = (setter) => (id) =>
    setter((m) => {
      const next = { ...m }
      if (next[id]) delete next[id]
      else next[id] = true
      return next
    })

  const actions = {
    toggleLike: toggle(setLikes),
    toggleCollect: toggle(setCollects),
    addComment(postId, text) {
      const t = text.trim()
      if (!t) return
      setComments((m) => ({
        ...m,
        [postId]: [...(m[postId] || []), { author: ME.name, avatar: ME.avatar, text: t, mine: true }],
      }))
    },
    addPost({ title, body, cat, cover, tags, festival }) {
      const post = {
        id: 'u' + Date.now(),
        cat,
        author: ME.name,
        avatar: ME.avatar,
        title: title.trim(),
        body: body.trim(),
        cover, // emoji
        tags: tags || [],
        likes: 0,
        collects: 0,
        comments: [],
        festival: !!festival,
        mine: true,
        tall: (body || '').length > 60,
      }
      setUserPosts((list) => [post, ...list])
      return post
    },
    addHelp({ type, title, body, city }) {
      const item = {
        id: 'uh' + Date.now(),
        type, // 'need' | 'offer'
        author: ME.name,
        avatar: ME.avatar,
        title: title.trim(),
        body: body.trim(),
        city: city.trim() || '同城',
        reward: type === 'need' ? '当面感谢' : '交个朋友',
        ts: '刚刚',
        mine: true,
      }
      setUserHelp((list) => [item, ...list])
      return item
    },
  }

  const value = { posts, helps, likes, collects, me: ME, ...actions }
  return <StoreCtx.Provider value={value}>{children}</StoreCtx.Provider>
}

export const useStore = () => {
  const ctx = useContext(StoreCtx)
  if (!ctx) throw new Error('useStore 必须在 StoreProvider 内使用')
  return ctx
}
