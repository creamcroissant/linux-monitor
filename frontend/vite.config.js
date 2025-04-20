/*
 * Vite前端构建工具配置文件
 *
 * 这个文件定义了Vite构建工具的配置选项，用于前端项目的开发和生产构建。
 * 主要功能:
 * - 配置Vue插件
 * - 设置路径别名（@指向src目录）
 * - 配置开发服务器选项（端口、代理设置等）
 * - 配置生产构建选项（代码分割、压缩等）
 * - 设置API代理，用于开发环境中请求后端服务
 */

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