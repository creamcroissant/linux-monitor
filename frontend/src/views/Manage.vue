<!--
  代理管理页面组件
  
  这个页面用于管理被监控的服务器（代理）。
  主要功能:
  - 显示所有代理服务器的列表
  - 提供代理服务器编辑功能（修改名称）
  - 提供代理服务器删除功能
  - 显示代理服务器的基本信息（主机名、IP地址、状态等）
  - 提供状态标识（在线/离线）
  - 显示最后连接时间
-->

<template>
  <div class="manage-container">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>代理管理</span>
        </div>
      </template>
      
      <el-table
        :data="agents"
        style="width: 100%"
        v-loading="loading"
      >
        <el-table-column prop="name" label="名称" min-width="120">
          <template #default="{ row }">
            <el-tag :type="row.isOnline ? 'success' : 'danger'" class="status-tag">
              {{ row.isOnline ? '在线' : '离线' }}
            </el-tag>
            <span>{{ row.name || row.hostname }}</span>
          </template>
        </el-table-column>
        
        <el-table-column prop="hostname" label="主机名" min-width="150" />
        
        <el-table-column prop="ip_address" label="IP地址" min-width="120" />
        
        <el-table-column prop="platform" label="平台" min-width="100" />
        
        <el-table-column label="最后连接时间" min-width="160">
          <template #default="{ row }">
            {{ formatTimestamp(row.last_seen) }}
          </template>
        </el-table-column>
        
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button
              type="primary"
              size="small"
              @click="editAgent(row)"
            >
              编辑
            </el-button>
            <el-button
              type="danger"
              size="small"
              @click="confirmDelete(row)"
              :disabled="row.isOnline"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    
    <!-- 编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="编辑代理"
      width="500px"
    >
      <el-form :model="editForm" label-width="100px">
        <el-form-item label="名称">
          <el-input v-model="editForm.name" placeholder="代理名称" />
        </el-form-item>
        
        <el-form-item label="主机名">
          <el-input v-model="editForm.hostname" disabled />
        </el-form-item>
        
        <el-form-item label="IP地址">
          <el-input v-model="editForm.ip_address" disabled />
        </el-form-item>
        
        <el-form-item label="平台">
          <el-input v-model="editForm.platform" disabled />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="saveAgent" :loading="saving">
            保存
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useStore } from 'vuex'
import { ElMessage, ElMessageBox } from 'element-plus'
import agentApi from '@/api/agent'

const store = useStore()
const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const editForm = ref({
  id: null,
  name: '',
  hostname: '',
  ip_address: '',
  platform: ''
})

// 计算属性
const agents = computed(() => store.state.agents)

// 格式化时间戳
const formatTimestamp = (timestamp) => {
  if (!timestamp) return '从未连接'
  return new Date(timestamp * 1000).toLocaleString()
}

// 刷新数据
const refreshData = async () => {
  loading.value = true
  try {
    await store.dispatch('fetchAgents')
  } catch (error) {
    ElMessage.error('获取代理列表失败')
  } finally {
    loading.value = false
  }
}

// 编辑代理
const editAgent = (agent) => {
  editForm.value = {
    id: agent.id,
    name: agent.name || '',
    hostname: agent.hostname,
    ip_address: agent.ip_address,
    platform: agent.platform
  }
  dialogVisible.value = true
}

// 保存代理
const saveAgent = async () => {
  if (!editForm.value.id) return
  
  saving.value = true
  try {
    await agentApi.updateAgent(editForm.value.id, {
      name: editForm.value.name
    })
    
    ElMessage.success('保存成功')
    dialogVisible.value = false
    refreshData()
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

// 确认删除
const confirmDelete = (agent) => {
  ElMessageBox.confirm(
    `确定要删除代理 ${agent.name || agent.hostname} 吗？此操作不可恢复。`,
    '删除确认',
    {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning'
    }
  )
    .then(() => {
      deleteAgent(agent.id)
    })
    .catch(() => {
      // 用户取消，不做操作
    })
}

// 删除代理
const deleteAgent = async (id) => {
  try {
    await agentApi.deleteAgent(id)
    ElMessage.success('删除成功')
    refreshData()
  } catch (error) {
    ElMessage.error('删除失败')
  }
}

// 生命周期钩子
onMounted(async () => {
  await refreshData()
})
</script>

<style scoped>
.manage-container {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-tag {
  margin-right: 8px;
}
</style> 