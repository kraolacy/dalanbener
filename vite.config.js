import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { viteSingleFile } from 'vite-plugin-singlefile'

// 纯前端单页应用。
// viteSingleFile：build 时把 JS/CSS 全部内联进一个 index.html，
// 这样 dist/index.html 可以直接双击(file://)打开，无需服务器、可离线。
export default defineConfig({
  plugins: [react(), viteSingleFile()],
  base: './',
})
