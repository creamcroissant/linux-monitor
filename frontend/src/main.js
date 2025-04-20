/*
 * 前端应用入口文件
 * 
 * 这个文件是Vue.js前端应用的主入口点，负责创建Vue实例并挂载到DOM中。
 * 主要功能:
 * - 导入并初始化Vue应用
 * - 配置Element Plus UI库
 * - 注册全局图标组件
 * - 加载路由和状态管理
 * - 设置全局错误处理
 */

import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'
import store from './store'
import './assets/main.css'

const app = createApp(App)

// 注册所有图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

// 处理错误
app.config.errorHandler = (err, vm, info) => {
  console.error('Vue错误:', err, info)
}

app.use(ElementPlus)
app.use(store)
app.use(router)

app.mount('#app') 