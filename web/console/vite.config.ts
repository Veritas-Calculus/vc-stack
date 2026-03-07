import { defineConfig, type Plugin } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { fileURLToPath, URL } from 'node:url'
import { readFileSync } from 'node:fs'

const pkg = JSON.parse(readFileSync(new URL('./package.json', import.meta.url), 'utf-8'))

// noVNC 1.6 has a broken Babel transpilation: line 179 of browser.js
// uses top-level `await` inside a CJS module, which Rollup cannot parse.
// This plugin rewrites it to a .then() pattern at build time.
function fixNoVNCTopLevelAwait(): Plugin {
  return {
    name: 'fix-novnc-top-level-await',
    transform(code, id) {
      if (id.includes('@novnc/novnc') && id.includes('browser.js')) {
        return code.replace(
          /exports\.supportsWebCodecsH264Decode\s*=\s*supportsWebCodecsH264Decode\s*=\s*await\s+_checkWebCodecsH264DecodeSupport\(\);/,
          '_checkWebCodecsH264DecodeSupport().then(function(r) { exports.supportsWebCodecsH264Decode = supportsWebCodecsH264Decode = r; });'
        )
      }
      return null
    }
  }
}

export default defineConfig({
  plugins: [react(), fixNoVNCTopLevelAwait()],
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
    assetsInlineLimit: 0,
    rollupOptions: {
      input: {
        main: fileURLToPath(new URL('./index.html', import.meta.url)),
        novnc: fileURLToPath(new URL('./src/novnc-entry.ts', import.meta.url))
      },
      output: {
        manualChunks: {
          'react-vendor': ['react', 'react-dom', 'react-router-dom'],
          'ui-vendor': ['xterm'],
          'utils-vendor': ['axios', 'zustand']
        },
        entryFileNames: (chunkInfo) => {
          if (chunkInfo.name === 'novnc') {
            return 'assets/novnc-[hash].js'
          }
          return 'assets/[name]-[hash].js'
        }
      }
    },
    sourcemap: false,
    chunkSizeWarningLimit: 1000,
    cssCodeSplit: true,
    minify: 'esbuild',
    target: 'es2022'
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./vitest.setup.ts'],
    exclude: ['e2e/**', 'node_modules/**'],
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
  optimizeDeps: {
    exclude: ['@novnc/novnc']
  },
  preview: {
    port: 5173
  }
})
