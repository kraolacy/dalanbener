import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 纯前端单页应用；base 相对路径，方便部署到 GitHub Pages 子路径
export default defineConfig({
  plugins: [react()],
  base: './',
})
