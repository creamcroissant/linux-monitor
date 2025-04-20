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
  // 从 localStorage 读取 token
  const token = localStorage.getItem('token')
  const userStr = localStorage.getItem('user')
  
  // 如果有 token 但 vuex 中没有用户信息，尝试恢复会话
  if (token && !store.state.user && userStr) {
    try {
      const user = JSON.parse(userStr)
      store.commit('SET_TOKEN', token)
      store.commit('SET_USER', user)
    } catch (e) {
      console.error('Failed to restore session:', e)
      localStorage.removeItem('token')
      localStorage.removeItem('user')
    }
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