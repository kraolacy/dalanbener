// 后端 API 客户端。与后端同源（Node 同时托管前端），故用相对路径 /api。
// 若后端不可达（GitHub Pages / 双击 file://），store 会自动退回 localStorage 演示模式。
let _token = null
export function setToken(t) { _token = t }

async function req(path, { method = 'GET', body } = {}) {
  const headers = {}
  if (body) headers['Content-Type'] = 'application/json'
  if (_token) headers.Authorization = 'Bearer ' + _token
  const res = await fetch(path, { method, headers, body: body ? JSON.stringify(body) : undefined })
  let data = null
  try { data = await res.json() } catch { /* 非 JSON */ }
  if (!res.ok) throw new Error((data && data.error) || `HTTP ${res.status}`)
  return data
}

export const api = {
  async health() {
    try {
      const res = await fetch('/api/health', { cache: 'no-store' })
      if (!res.ok) return false
      const j = await res.json()
      return !!(j && j.ok)
    } catch { return false }
  },
  register: (b) => req('/api/register', { method: 'POST', body: b }),
  login: (b) => req('/api/login', { method: 'POST', body: b }),
  me: () => req('/api/me'),
  posts: () => req('/api/posts'),
  helps: () => req('/api/helps'),
  createPost: (b) => req('/api/posts', { method: 'POST', body: b }),
  createHelp: (b) => req('/api/helps', { method: 'POST', body: b }),
  addComment: (id, text) => req(`/api/posts/${id}/comments`, { method: 'POST', body: { text } }),
  toggleLike: (id) => req(`/api/posts/${id}/like`, { method: 'POST' }),
  toggleCollect: (id) => req(`/api/posts/${id}/collect`, { method: 'POST' }),
  upload: (dataUrl) => req('/api/upload', { method: 'POST', body: { dataUrl } }),
  follow: (name) => req(`/api/follow/${encodeURIComponent(name)}`, { method: 'POST' }),
  conversations: () => req('/api/messages'),
  thread: (name) => req(`/api/messages/${encodeURIComponent(name)}`),
  sendMessage: (to, text) => req('/api/messages', { method: 'POST', body: { to, text } }),
}
