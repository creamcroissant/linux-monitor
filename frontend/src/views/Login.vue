<template>
  <div class="login-container">
    <div class="login-logo">
      <h1>系统监控平台</h1>
    </div>
    
    <el-card class="login-card" v-loading="loading">
      <template #header>
        <div class="card-header">
          <h2>用户登录</h2>
        </div>
      </template>
      
      <el-form
        ref="loginForm"
        :model="loginForm"
        :rules="rules"
        label-width="80px"
        @submit.prevent="handleLogin"
      >
        <el-form-item label="用户名" prop="username">
          <el-input
            v-model="loginForm.username"
            placeholder="请输入用户名"
            prefix-icon="User"
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        
        <el-form-item label="密码" prop="password">
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="请输入密码"
            prefix-icon="Lock"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>
        
        <el-form-item>
          <el-checkbox v-model="loginForm.remember">记住登录状态</el-checkbox>
        </el-form-item>
        
        <el-form-item>
          <el-button
            type="primary"
            :loading="loading"
            class="login-button"
            @click="handleLogin"
          >
            登录
          </el-button>
        </el-form-item>
        
        <div class="login-tips" v-if="loginError">
          <el-alert
            :title="loginError"
            type="error"
            show-icon
            :closable="false"
          />
        </div>
        
        <div class="default-account">
          <p>默认账号：admin</p>
          <p>默认密码：admin</p>
        </div>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useStore } from 'vuex'
import { ElMessage } from 'element-plus'
import { User, Lock } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const store = useStore()
const loginForm = reactive({
  username: '',
  password: '',
  remember: false
})
const loading = ref(false)
const loginError = ref('')
const loginFormRef = ref(null)

const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' }
  ]
}

const handleLogin = async () => {
  if (loading.value) return
  
  loading.value = true
  loginError.value = ''
  
  try {
    console.log('尝试登录, 用户名:', loginForm.username)
    const response = await store.dispatch('login', {
      username: loginForm.username,
      password: loginForm.password
    })
    
    console.log('登录成功, 响应:', response ? '有响应数据' : '无响应数据')
    
    // 确保在本地存储用户名，如果启用了"记住登录状态"
    if (loginForm.remember) {
      localStorage.setItem('username', loginForm.username)
    } else {
      localStorage.removeItem('username')
    }
    
    // 验证token是否已经存储到localStorage
    const token = localStorage.getItem('token')
    console.log('登录后token是否存在:', !!token)
    
    ElMessage.success('登录成功')
    
    // 等待一小段时间确保状态更新完毕
    setTimeout(() => {
      // 获取登录前的路径，如果有则跳转回去，否则跳转到首页
      const redirectPath = route.query.redirect || '/dashboard'
      console.log('登录后重定向到:', redirectPath)
      router.push(redirectPath)
    }, 100)
  } catch (error) {
    console.error('登录失败:', error)
    loginError.value = error.response?.data?.error || '用户名或密码错误'
  } finally {
    loading.value = false
  }
}

// 如果已经登录，跳转到首页
onMounted(() => {
  const token = localStorage.getItem('token')
  console.log('Login.vue mounted, token存在:', !!token)
  
  if (token) {
    console.log('已经登录，跳转到dashboard')
    router.push('/dashboard')
  } else {
    console.log('未登录，显示登录界面')
  }
  
  // 自动填充用户名
  const savedUsername = localStorage.getItem('username')
  if (savedUsername) {
    loginForm.username = savedUsername
    loginForm.remember = true
    console.log('自动填充用户名:', savedUsername)
  }
})
</script>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  background-color: #f5f7fa;
}

.login-logo {
  margin-bottom: 30px;
}

.login-logo h1 {
  color: #409EFF;
  font-size: 28px;
  text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.1);
}

.login-card {
  width: 400px;
  border-radius: 8px;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1);
}

.card-header {
  text-align: center;
}

.card-header h2 {
  margin: 0;
  color: #303133;
}

.login-button {
  width: 100%;
}

.login-tips {
  margin-top: 15px;
}

.default-account {
  margin-top: 20px;
  text-align: center;
  font-size: 12px;
  color: #909399;
}

.default-account p {
  margin: 5px 0;
}
</style> 