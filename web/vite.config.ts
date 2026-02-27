import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { PUBLIC_PRECHECK_PATHS, UNAUTHORIZED_REDIRECT } from './src/constants/navigation'

// 端口常量 - 与 src/constants/api.ts 保持一致
const SERVER_PORT = 8566
const INJECTED_PUBLIC_PRECHECK_PATHS = JSON.stringify(PUBLIC_PRECHECK_PATHS)
const INJECTED_UNAUTHORIZED_REDIRECT = JSON.stringify(UNAUTHORIZED_REDIRECT)

const injectPublicPathsPlugin = {
  name: 'inject-public-precheck-paths',
  transformIndexHtml(html: string) {
    return html
      .replace('__PUBLIC_PRECHECK_PATHS__', INJECTED_PUBLIC_PRECHECK_PATHS)
      .replace('__UNAUTHORIZED_REDIRECT__', INJECTED_UNAUTHORIZED_REDIRECT)
  }
}

export default defineConfig({
  base: '/',
  plugins: [vue(), injectPublicPathsPlugin],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src')
    }
  },
  server: {
    port: SERVER_PORT,
    proxy: {
      '/api/': {
        target: process.env.VITE_API_TARGET || `http://localhost:${SERVER_PORT}`,
        changeOrigin: true
      },
      '/health': {
        target: process.env.VITE_API_TARGET || `http://localhost:${SERVER_PORT}`,
        changeOrigin: true
      },
      '/swagger': {
        target: process.env.VITE_API_TARGET || `http://localhost:${SERVER_PORT}`,
        changeOrigin: true
      },
      '/metrics': {
        target: process.env.VITE_API_TARGET || `http://localhost:${SERVER_PORT}`,
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    sourcemap: false,
    chunkSizeWarningLimit: 1500,
    rollupOptions: {
      output: {
        manualChunks: {
          'element-plus': ['element-plus'],
          'vue-vendor': ['vue', 'vue-router', 'pinia']
        }
      }
    }
  }
})
