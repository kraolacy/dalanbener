import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { viteSingleFile } from 'vite-plugin-singlefile'

// 纯前端单页应用。
// viteSingleFile：build 时把 JS/CSS 全部内联进一个 index.html，
// 这样 dist/index.html 可以直接双击(file://)打开，无需服务器、可离线。
export default defineConfig({
  plugins: [react(), viteSingleFile()],
  base: './',
  // 本地联调：前端 :5173 的 /api 代理到 Go 后端 :8080
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
