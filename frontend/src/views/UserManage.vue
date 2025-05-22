<!--
  用户管理页面组件
  
  这个页面用于管理系统用户，仅对管理员开放。
  主要功能:
  - 显示所有用户的列表
  - 创建新用户（指定用户名、密码和角色）
  - 删除现有用户（包括其他管理员账户）
  - 显示用户的基本信息（用户名、角色、创建时间）
  - 权限检查（只有管理员可访问）
-->

<template>
  <div class="user-manage-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <div class="header-left">
            <el-button icon="ArrowLeft" @click="goBack">返回</el-button>
            <span class="title">用户管理</span>
          </div>
          <el-button type="primary" @click="dialogVisible = true">
            <el-icon><Plus /></el-icon>
            添加用户
          </el-button>
        </div>
      </template>
      
      <el-table
        :data="users"
        style="width: 100%"
        v-loading="loading"
      >
        <el-table-column prop="username" label="用户名" min-width="120" />
        <el-table-column prop="role" label="角色" min-width="100">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'info'">
              {{ row.role === 'admin' ? '管理员' : '普通用户' }}
            </el-tag>
          </template>
        </el-table-column>
        
        <el-table-column label="最后登录时间" min-width="160">
          <template #default="{ row }">
            {{ formatTimestamp(row.lastLogin) }}
          </template>
        </el-table-column>
        
        <el-table-column label="创建时间" min-width="160">
          <template #default="{ row }">
            {{ formatTimestamp(row.createdAt) }}
          </template>
        </el-table-column>
        
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button
              type="danger"
              size="small"
              @click="confirmDelete(row)"
              :disabled="row.username === 'admin' || row.username === currentUser.username"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    
    <!-- 添加用户对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="添加用户"
      width="500px"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" placeholder="请输入用户名" />
        </el-form-item>
        
        <el-form-item label="密码" prop="password">
          <el-input 
            v-model="form.password" 
            type="password"
            placeholder="请输入密码"
            show-password
          />
        </el-form-item>
        
        <el-form-item label="确认密码" prop="confirmPassword">
          <el-input 
            v-model="form.confirmPassword" 
            type="password"
            placeholder="请再次输入密码"
            show-password
          />
        </el-form-item>
        
        <el-form-item label="角色">
          <el-radio-group v-model="form.role">
            <el-radio label="user">普通用户</el-radio>
            <el-radio label="admin">管理员</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="createUser" :loading="creating">
            确认
          </el-button>
        </div>
      </template>
    </el-dialog>

    <div v-if="isAdmin" class="webhook-settings">
      <el-card>
        <template #header>
          <span>Webhook设置</span>
          <el-button type="primary" size="small" @click="addWebhook">添加Webhook</el-button>
        </template>
        <el-table :data="webhooks" style="width: 100%">
          <el-table-column prop="name" label="名称" />
          <el-table-column prop="type" label="类型" />
          <el-table-column prop="sendkey" label="Server酱API Key" v-if="colType==='serverchan'" />
          <el-table-column prop="url" label="Webhook URL" v-if="colType==='custom'" />
          <el-table-column label="操作">
            <template #default="{ row, $index }">
              <el-button size="small" @click="editWebhook($index)">编辑</el-button>
              <el-button size="small" type="danger" @click="removeWebhook($index)">删除</el-button>
              <el-button size="small" type="success" @click="testWebhook(row)">测试</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-dialog v-model="webhookDialogVisible" title="Webhook配置">
          <el-form :model="webhookForm" label-width="100px">
            <el-form-item label="名称">
              <el-input v-model="webhookForm.name" />
            </el-form-item>
            <el-form-item label="类型">
              <el-select v-model="webhookForm.type">
                <el-option label="Server酱" value="serverchan" />
                <el-option label="自定义Webhook" value="custom" />
              </el-select>
            </el-form-item>
            <el-form-item label="Server酱API Key" v-if="webhookForm.type==='serverchan'">
              <el-input v-model="webhookForm.sendkey" />
            </el-form-item>
            <el-form-item label="Webhook URL" v-if="webhookForm.type==='custom'">
              <el-input v-model="webhookForm.url" />
            </el-form-item>
            <el-form-item label="启用">
              <el-switch v-model="webhookForm.enabled" />
            </el-form-item>
          </el-form>
          <template #footer>
            <el-button @click="webhookDialogVisible=false">取消</el-button>
            <el-button type="primary" @click="saveWebhook">保存</el-button>
          </template>
        </el-dialog>
        <el-button type="primary" @click="saveAllWebhooks" style="margin-top:16px">保存所有Webhook</el-button>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useStore } from 'vuex'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useRouter } from 'vue-router'
import axios from 'axios'
import userApi from '@/api/user'
import { Plus, ArrowLeft } from '@element-plus/icons-vue'
import agentApi from '@/api/agent'

const store = useStore()
const router = useRouter()
const loading = ref(false)
const creating = ref(false)
const dialogVisible = ref(false)
const users = ref([])
const formRef = ref(null)
const isAdmin = computed(() => store.state.user && store.state.user.role === 'admin')
const webhooks = ref([])
const webhookDialogVisible = ref(false)
const webhookForm = ref({ name: '', type: 'serverchan', sendkey: '', url: '', enabled: true })
let editIndex = -1
const colType = ref('')

const form = ref({
  username: '',
  password: '',
  confirmPassword: '',
  role: 'user'
})

// 当前用户
const currentUser = computed(() => store.state.user || {})

// 表单验证规则
const rules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 3, max: 20, message: '用户名长度应为3-20个字符', trigger: 'blur' }
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 5, max: 30, message: '密码长度应为5-30个字符', trigger: 'blur' }
  ],
  confirmPassword: [
    { 
      required: true, 
      message: '请再次输入密码', 
      trigger: 'blur' 
    },
    { 
      validator: (rule, value, callback) => {
        if (value !== form.value.password) {
          callback(new Error('两次输入的密码不一致'));
        } else {
          callback();
        }
      }, 
      trigger: 'blur' 
    }
  ]
}

// 格式化时间戳
const formatTimestamp = (timestamp) => {
  if (!timestamp) return '未知'
  return new Date(timestamp * 1000).toLocaleString()
}

// 返回主界面
const goBack = () => {
  router.push('/')
}

// 获取用户列表
const fetchUsers = async () => {
  loading.value = true
  try {
    // 打印当前token信息进行调试
    console.log('当前使用的token:', store.state.token)
    console.log('localStorage中的token:', localStorage.getItem('token'))
    console.log('当前用户信息:', store.state.user)
    
    // 使用userApi服务
    const users_data = await userApi.getUsers()
    
    console.log('获取用户列表成功:', users_data)
    users.value = users_data || []
  } catch (error) {
    console.error('获取用户列表失败:', error.response || error)
    ElMessage.error('获取用户列表失败: ' + (error.response?.data?.error || error.message))
  } finally {
    loading.value = false
  }
}

// 创建用户
const createUser = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (!valid) return
    
    creating.value = true
    try {
      const userData = {
        username: form.value.username,
        password: form.value.password,
        role: form.value.role
      }
      
      // 使用userApi服务创建用户
      const response = await userApi.createUser(userData)
      
      console.log('创建用户成功，服务器响应:', response)
      ElMessage.success('添加用户成功')
      dialogVisible.value = false
      
      // 重置表单
      form.value = {
        username: '',
        password: '',
        confirmPassword: '',
        role: 'user'
      }
      
      // 采用更可靠的方式刷新用户列表
      // 1. 先延迟一点时间确保数据库操作已完成
      await new Promise(resolve => setTimeout(resolve, 500))
      
      // 2. 手动构建新用户数据，先添加到当前列表中 
      const newUser = {
        username: response.username,
        role: response.role,
        lastLogin: null,
        createdAt: response.createdAt
      }
      
      // 将新用户添加到列表最前面
      users.value = [newUser, ...users.value]
      
      // 3. 然后再异步获取完整列表
      setTimeout(() => {
        fetchUsers()
      }, 1000)
    } catch (error) {
      console.error('添加用户失败:', error.response || error)
      ElMessage.error(error.response?.data?.error || '添加用户失败')
    } finally {
      creating.value = false
    }
  })
}

// 确认删除
const confirmDelete = (user) => {
  ElMessageBox.confirm(
    `确定要删除用户 ${user.username} 吗？此操作不可恢复。`,
    '删除确认',
    {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    }
  )
    .then(() => {
      deleteUser(user.username)
    })
    .catch(() => {
      // 用户取消，不做操作
    })
}

// 删除用户
const deleteUser = async (username) => {
  try {
    await userApi.deleteUser(username)
    
    ElMessage.success('删除用户成功')
    // 刷新用户列表
    fetchUsers()
  } catch (error) {
    console.error('删除用户失败:', error.response || error)
    ElMessage.error(error.response?.data?.error || '删除用户失败')
  }
}

const loadWebhooks = async () => {
  webhooks.value = await agentApi.getWebhook()
}
const addWebhook = () => {
  webhookForm.value = { name: '', type: 'serverchan', sendkey: '', url: '', enabled: true }
  editIndex = -1
  webhookDialogVisible.value = true
}
const editWebhook = (idx) => {
  webhookForm.value = { ...webhooks.value[idx] }
  editIndex = idx
  webhookDialogVisible.value = true
}
const removeWebhook = (idx) => {
  webhooks.value.splice(idx, 1)
}
const testWebhook = async (webhook) => {
  try {
    const res = await agentApi.testWebhook(webhook)
    if (res.message === 'SUCCESS') {
      ElMessage.success('Webhook测试成功')
    } else {
      ElMessage.error('Webhook测试失败: ' + (res.detail || res.raw || '未知错误'))
    }
  } catch (e) {
    ElMessage.error('Webhook测试请求异常')
  }
}
const saveWebhook = () => {
  if (editIndex === -1) {
    webhooks.value.push({ ...webhookForm.value })
  } else {
    webhooks.value[editIndex] = { ...webhookForm.value }
  }
  webhookDialogVisible.value = false
}
const saveAllWebhooks = async () => {
  await agentApi.setWebhook(webhooks.value)
  ElMessage.success('Webhook已保存')
  // 保存后自动测试所有已启用webhook
  for (const wh of webhooks.value) {
    if (wh.enabled) await testWebhook(wh)
  }
}

// 生命周期钩子
onMounted(() => {
  fetchUsers()
  loadWebhooks()
})
</script>

<style scoped>
.user-manage-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 15px;
}

.title {
  font-size: 18px;
  font-weight: bold;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
}

.webhook-settings { margin-top: 32px; }
</style> 