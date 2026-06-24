import {defineConfig} from 'vite'
import react from '@vitejs/plugin-react'

// Aland 前端构建配置。
// M0 只用 React 插件；后续可加 path alias、代码分割等。
export default defineConfig({
  plugins: [react()],
  server: {
    port: 34115, // 与 Wails dev server 配合，避免冲突
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
