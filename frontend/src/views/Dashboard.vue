<template>
  <div class="dashboard-container">
    <!-- 顶部导航栏 -->
    <div class="dashboard-header">
      <div class="logo">
        <h2>系统监控平台</h2>
      </div>
      <div class="user-menu">
        <el-dropdown trigger="click" @command="handleCommand">
          <span class="el-dropdown-link">
            <el-avatar :size="32" icon="User" />
            {{ currentUser.username || '未登录' }}
            <el-icon><ArrowDown /></el-icon>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item v-if="isAdmin" command="users">用户管理</el-dropdown-item>
              <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </div>
    
    <!-- 错误提示 -->
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      :closable="true"
      show-icon
      style="margin: 0 20px 20px 20px;"
      @close="clearError"
    />
    
    <!-- 主要内容 -->
    <el-row :gutter="20">
      <!-- 概览卡片 -->
      <el-col :span="8">
        <el-card class="overview-card">
          <template #header>
            <div class="card-header">
              <span>总服务器</span>
            </div>
          </template>
          <div class="card-content">
            <div class="stat-value">{{ totalAgents }}</div>
            <div class="stat-label">已连接服务器</div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="8">
        <el-card class="overview-card">
          <template #header>
            <div class="card-header">
              <span>在线服务器</span>
            </div>
          </template>
          <div class="card-content">
            <div class="stat-value success">{{ onlineAgents }}</div>
            <div class="stat-label">正常运行</div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="8">
        <el-card class="overview-card">
          <template #header>
            <div class="card-header">
              <span>离线服务器</span>
            </div>
          </template>
          <div class="card-content">
            <div class="stat-value danger">{{ offlineAgents }}</div>
            <div class="stat-label">未响应</div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 服务器列表 -->
    <el-row :gutter="20" class="server-list">
      <el-col :span="24">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>服务器列表</span>
              <el-button type="primary" @click="refreshData" :loading="refreshing">
                <el-icon><RefreshRight /></el-icon>
                刷新
              </el-button>
            </div>
          </template>
          
          <!-- 空状态 -->
          <el-empty 
            v-if="!loading && agents.length === 0" 
            description="暂无服务器数据"
          >
            <template #default>
              <el-button type="primary" @click="refreshData">刷新数据</el-button>
            </template>
          </el-empty>
          
          <el-table
            v-else
            :data="agents"
            style="width: 100%"
            v-loading="loading"
          >
            <el-table-column prop="name" label="服务器名称" min-width="180">
              <template #default="scope">
                <div class="server-name">
                  <el-icon v-if="scope.row.is_online" style="color: #67C23A;"><CircleCheckFilled /></el-icon>
                  <el-icon v-else style="color: #F56C6C;"><CircleCloseFilled /></el-icon>
                  <span>{{ scope.row.name || scope.row.hostname }}</span>
                </div>
              </template>
            </el-table-column>
            
            <el-table-column prop="hostname" label="主机名" min-width="150" />
            
            <el-table-column prop="platform" label="平台" min-width="120" />
            
            <el-table-column prop="ip_address" label="IP地址" min-width="130" />
            
            <el-table-column prop="last_seen" label="最后在线" min-width="160">
              <template #default="scope">
                {{ timeSince(scope.row.last_seen) }}
              </template>
            </el-table-column>
            
            <el-table-column prop="created_at" label="创建时间" min-width="160">
              <template #default="scope">
                {{ formatDate(scope.row.created_at) }}
              </template>
            </el-table-column>
            
            <el-table-column label="CPU使用率" min-width="120">
              <template #default="{ row }">
                <el-progress
                  :percentage="getMetricValue(row.id, 'cpu_usage')"
                  :status="getProgressStatus(getMetricValue(row.id, 'cpu_usage'))"
                  :stroke-width="10"
                />
              </template>
            </el-table-column>
            
            <el-table-column label="内存使用率" min-width="120">
              <template #default="{ row }">
                <el-progress
                  :percentage="getMetricValue(row.id, 'memory_percent')"
                  :status="getProgressStatus(getMetricValue(row.id, 'memory_percent'))"
                  :stroke-width="10"
                />
              </template>
            </el-table-column>
            
            <el-table-column label="磁盘使用率" min-width="120">
              <template #default="{ row }">
                <el-progress
                  :percentage="getMetricValue(row.id, 'disk_percent')"
                  :status="getProgressStatus(getMetricValue(row.id, 'disk_percent'))"
                  :stroke-width="10"
                />
              </template>
            </el-table-column>
            
            <el-table-column label="操作" width="120" fixed="right">
              <template #default="{ row }">
                <el-button
                  type="primary"
                  size="small"
                  @click="viewAgentDetails(row.id)"
                >
                  详情
                </el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useStore } from 'vuex'
import { useRouter } from 'vue-router'
import { 
  CircleCheckFilled, 
  CircleCloseFilled, 
  RefreshRight,
  ArrowDown,
  User
} from '@element-plus/icons-vue'
import agentApi from '@/api/agent'
import { ElMessage } from 'element-plus'

const router = useRouter()
const store = useStore()
const loading = ref(true)
const refreshing = ref(false)
const refreshInterval = ref(null)
const agentMetrics = ref({}) // 存储每个agent的最新指标

// 从store中获取agents和用户信息
const agents = computed(() => store.getters.agents)
const totalAgents = computed(() => agents.value.length)
const onlineAgents = computed(() => agents.value.filter(agent => agent.is_online).length)
const offlineAgents = computed(() => agents.value.filter(agent => !agent.is_online).length)
const currentUser = computed(() => store.state.user || {})
const isAdmin = computed(() => currentUser.value && currentUser.value.role === 'admin')

// 获取错误信息
const error = computed(() => store.state.error)

// 清除错误
const clearError = () => {
  store.commit('CLEAR_ERROR')
}

// 下拉菜单命令处理
const handleCommand = (command) => {
  switch (command) {
    case 'users':
      router.push('/users')
      break
    case 'logout':
      store.dispatch('logout')
      router.push('/login')
      break
  }
}

// 获取代理数据
const fetchAgents = async () => {
  loading.value = true
  try {
    console.log('开始获取代理列表，Dashboard组件...')
    const agentsData = await store.dispatch('fetchAgents')
    console.log('Dashboard获取到的代理列表:', agentsData)
    
    // 检查数据是否为空数组，为空时显示提示
    if (Array.isArray(agentsData) && agentsData.length === 0) {
      console.log('Dashboard: 代理列表为空')
      ElMessage.info('当前没有已注册的代理')
    } else if (Array.isArray(agentsData)) {
      console.log(`Dashboard: 获取到${agentsData.length}个代理`)
      
      // 打印每个代理的基本信息，方便调试
      agentsData.forEach((agent, index) => {
        console.log(`Dashboard: 代理 ${index+1}: ID=${agent.id}, 主机名=${agent.hostname}`)
      })
    }
    
    // 获取每个代理的最新指标
    if (agentsData.length > 0) {
      await fetchAllAgentMetrics()
    }
  } catch (error) {
    console.error('Dashboard: 获取代理列表失败:', error)
    ElMessage.error('获取代理列表失败')
  } finally {
    loading.value = false
  }
}

// 获取所有在线代理的最新指标
const fetchAllAgentMetrics = async () => {
  const onlineAgentsList = agents.value.filter(agent => agent.is_online)
  console.log('获取在线代理的指标数据, 在线代理数量:', onlineAgentsList.length)
  
  for (const agent of onlineAgentsList) {
    try {
      console.log(`开始获取代理 ${agent.id} 的指标数据...`)
      // 获取过去5分钟的指标
      const now = Math.floor(Date.now() / 1000)
      const fiveMinutesAgo = now - 300
      
      // 只获取最近一条记录
      const params = {
        from: fiveMinutesAgo,
        to: now,
        limit: 1
      }
      
      const metrics = await agentApi.getAgentMetrics(agent.id, params)
      console.log(`代理 ${agent.id} 的指标数据:`, metrics)
      
      if (metrics && metrics.length > 0) {
        agentMetrics.value[agent.id] = metrics[0]
      } else {
        console.log(`代理 ${agent.id} 没有有效的指标数据`)
      }
    } catch (error) {
      console.error(`获取代理 ${agent.id} 的指标失败:`, error)
    }
  }
}

// 获取指标值
const getMetricValue = (agentId, metricName) => {
  const metrics = agentMetrics.value[agentId]
  if (!metrics) return 0
  
  switch (metricName) {
    case 'cpu_usage':
      return parseFloat(metrics.cpu_usage || 0).toFixed(1)
    case 'memory_percent':
      return parseFloat(metrics.memory_info?.percent || 0).toFixed(1)
    case 'disk_percent':
      return parseFloat(metrics.disk_info?.percent || 0).toFixed(1)
    default:
      return 0
  }
}

// 根据使用率返回不同的进度条状态
const getProgressStatus = (value) => {
  const numValue = parseFloat(value)
  if (numValue >= 90) return 'exception'
  if (numValue >= 70) return 'warning'
  return 'success'
}

// 刷新数据
const refreshData = async () => {
  refreshing.value = true
  try {
    await store.dispatch('fetchAgents')
    await fetchAllAgentMetrics()
  } catch (error) {
    console.error('Failed to refresh agents:', error)
  } finally {
    refreshing.value = false
  }
}

// 查看代理详情
const viewAgentDetails = (agentId) => {
  router.push(`/agent/${agentId}`)
}

// 格式化时间
const formatDate = (date) => {
  if (!date) return '未知';
  if (typeof date === 'string') {
    date = new Date(date);
  }
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(date);
};

// 时间差计算
const timeSince = (date) => {
  if (!date) return '未知';
  
  if (typeof date === 'string') {
    date = new Date(date);
  }
  
  const seconds = Math.floor((new Date() - date) / 1000);
  
  let interval = seconds / 31536000;
  if (interval > 1) return Math.floor(interval) + ' 年前';
  
  interval = seconds / 2592000;
  if (interval > 1) return Math.floor(interval) + ' 个月前';
  
  interval = seconds / 86400;
  if (interval > 1) return Math.floor(interval) + ' 天前';
  
  interval = seconds / 3600;
  if (interval > 1) return Math.floor(interval) + ' 小时前';
  
  interval = seconds / 60;
  if (interval > 1) return Math.floor(interval) + ' 分钟前';
  
  return Math.floor(seconds) + ' 秒前';
};

// 生命周期钩子
onMounted(() => {
  console.log('Dashboard组件已挂载，开始获取数据')
  fetchAgents()
  // 每60秒自动刷新一次
  refreshInterval.value = setInterval(refreshData, 60000)
})

onUnmounted(() => {
  console.log('Dashboard组件已卸载，清除定时器')
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
})
</script>

<style scoped>
.dashboard-container {
  padding: 0 0 20px 0;
}

.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
  height: 60px;
  background-color: #fff;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  margin-bottom: 20px;
}

.logo h2 {
  margin: 0;
  font-size: 20px;
  color: #409EFF;
}

.user-menu {
  display: flex;
  align-items: center;
}

.el-dropdown-link {
  display: flex;
  align-items: center;
  cursor: pointer;
  padding: 5px 10px;
  border-radius: 4px;
}

.el-dropdown-link:hover {
  background-color: #f5f7fa;
}

.el-avatar {
  margin-right: 8px;
}

.overview-card {
  margin: 0 20px 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-content {
  text-align: center;
  padding: 20px 0;
}

.stat-value {
  font-size: 36px;
  font-weight: bold;
  margin-bottom: 10px;
}

.stat-value.success {
  color: #67c23a;
}

.stat-value.danger {
  color: #f56c6c;
}

.stat-label {
  color: #909399;
  font-size: 14px;
}

.server-list {
  margin: 0 20px;
}

.server-name {
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-tag {
  margin-right: 8px;
}
</style> 