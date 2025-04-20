<!--
  根组件App.vue
  
  这是Vue应用的根组件，作为整个应用的容器和入口组件。
  主要职责:
  - 提供应用的基本布局框架
  - 在应用启动时检查用户认证状态
  - 恢复用户会话（如果存在）
  - 处理全局路由守卫和认证逻辑
  - 定义全局CSS样式
-->

<template>
  <el-config-provider>
    <router-view></router-view>
  </el-config-provider>
</template>

<script setup>
import { ElConfigProvider } from 'element-plus'
import { onMounted } from 'vue'
import { useStore } from 'vuex'
import { useRouter } from 'vue-router'

const store = useStore()
const router = useRouter()

// 页面加载时检查登录状态
onMounted(() => {
  console.log('App mounted, checking authentication state')
  
  // 从 localStorage 读取 token
  const token = localStorage.getItem('token')
  const userStr = localStorage.getItem('user')
  
  console.log('从localStorage读取的token:', token ? token.substring(0, 10) + '...' : 'null')
  console.log('从localStorage读取的user:', userStr ? '有用户数据' : 'null')
  
  // 如果有 token 但 vuex 中没有用户信息，尝试恢复会话
  if (token && !store.state.user && userStr) {
    try {
      const user = JSON.parse(userStr)
      console.log('恢复会话状态，用户名:', user.username)
      store.commit('SET_TOKEN', token)
      store.commit('SET_USER', user)
    } catch (e) {
      console.error('会话恢复失败:', e)
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      router.push('/')
    }
  } else if (!token && router.currentRoute.value.meta.requiresAuth) {
    // 如果没有token但当前路由需要认证，重定向到登录页
    console.log('没有有效的认证令牌，重定向到登录页')
    router.push('/')
  }
})
</script>

<style>
html, body {
  margin: 0;
  padding: 0;
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

#app {
  height: 100%;
  color: #2c3e50;
  background-color: #f5f7fa;
}

.el-card {
  border-radius: 4px;
  border: none;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.el-card__header {
  border-bottom: 1px solid #ebeef5;
  padding: 15px 20px;
}

.el-table th.el-table__cell {
  background-color: #f5f7fa;
}
</style> 