import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { fileURLToPath, URL } from 'node:url'
import { readFileSync } from 'node:fs'

const pkg = JSON.parse(readFileSync(new URL('./package.json', import.meta.url), 'utf-8'))

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
    __APP_COMMIT__: JSON.stringify(process.env.GIT_COMMIT || ''),
    __APP_BUILD_TIME__: JSON.stringify(new Date().toISOString())
  },
  build: {
    // 确保所有资源都打包到本地
    assetsInlineLimit: 0, // 禁用小文件内联，确保资源可追踪
    rollupOptions: {
      input: {
        main: fileURLToPath(new URL('./index.html', import.meta.url)),
        novnc: fileURLToPath(new URL('./src/novnc-entry.ts', import.meta.url))
      },
      output: {
        // 确保chunk文件名稳定
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          'ui-vendor': ['xterm'],
          'utils-vendor': ['axios', 'zustand']
        },
        // 为novnc-entry创建独立的输出
        entryFileNames: (chunkInfo) => {
          if (chunkInfo.name === 'novnc') {
            return 'assets/novnc-[hash].js'
          }
          return 'assets/[name]-[hash].js'
        }
      }
    },
    // 生成source map便于调试
    sourcemap: false,
    // 优化chunk大小
    chunkSizeWarningLimit: 1000,
    // 确保CSS也被正确处理
    cssCodeSplit: true,
    // 使用esbuild压缩，速度更快且无需额外依赖
    minify: 'esbuild',
    // 使用ES2022以支持top-level await等特性（noVNC需要）
    target: 'es2022'
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./vitest.setup.ts']
  },
  server: {
    port: 5173,
    open: true,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false
      }
    }
  },
  preview: {
    port: 5173
  }
})
