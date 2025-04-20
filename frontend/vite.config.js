import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src')
    }
  },
  server: {
    port: 5173,
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
        rewrite: (path) => path
      },
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true
      }
    }
  },
  build: {
    chunkSizeWarningLimit: 1600,
    outDir: 'dist',
    assetsDir: 'assets',
    base: '/',
    rollupOptions: {
      output: {
        manualChunks: {
          'vendor': ['vue', 'vue-router', 'vuex'],
          'element-plus': ['element-plus', '@element-plus/icons-vue'],
          'echarts': ['echarts']
        }
      }
    },
    sourcemap: false,
    minify: 'terser',
    terserOptions: {
      compress: {
        drop_console: true,
        drop_debugger: true
      }
    }
  }
}) 