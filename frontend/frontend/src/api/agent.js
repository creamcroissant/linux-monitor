import axios from 'axios'
import { ElMessage } from 'element-plus'

const api = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
api.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
      console.log(`API请求: ${config.method.toUpperCase()} ${config.url}`)
      console.log(`使用的token (前10字符): ${token.substring(0, 10)}...`)
    } else {
      console.warn(`API请求没有token: ${config.method.toUpperCase()} ${config.url}`)
    }
    return config
  },
  error => {
    console.error('Request error:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  response => {
    // 检查响应数据格式
    if (!response.data) {
      throw new Error('Invalid response format')
    }
    return response
  },
  error => {
    if (error.response) {
      // 服务器返回错误
      const status = error.response.status
      const errorData = error.response.data
      
      let errorMessage = errorData.error || '未知错误'
      let errorDetail = errorData.detail || ''
      
      console.error(`API error: ${status} - ${errorMessage}`, errorDetail)
      
      // 显示错误信息
      ElMessage.error(errorDetail ? `${errorMessage}: ${errorDetail}` : errorMessage)
      
      if (status === 401) {
        // 未授权，清除token并重定向到登录页
        localStorage.removeItem('token')
        window.location.href = '/login'
      }
    } else if (error.request) {
      // 请求已发出但没有收到响应
      console.error('Network error:', error.request)
      ElMessage.error('网络错误：服务器未响应')
    } else {
      // 请求配置出错
      console.error('Request configuration error:', error.message)
      ElMessage.error(`请求错误：${error.message}`)
    }
    return Promise.reject(error)
  }
)

export default {
  // 获取所有代理
  async getAgents() {
    try {
      const response = await api.get('/agents')
      if (!Array.isArray(response.data)) {
        throw new Error('Invalid agents data format')
      }
      return response.data
    } catch (error) {
      console.error('Failed to fetch agents:', error)
      throw error
    }
  },
  
  // 获取单个代理详情
  async getAgent(id) {
    try {
      // 移除可能存在的花括号
      if (id.startsWith('{') && id.endsWith('}')) {
        id = id.substring(1, id.length - 1);
      }
      
      const response = await api.get(`/agents/${id}`)
      if (!response.data || typeof response.data !== 'object') {
        throw new Error('Invalid agent data format')
      }
      return response.data
    } catch (error) {
      console.error(`Failed to fetch agent ${id}:`, error)
      throw error
    }
  },
  
  // 获取代理指标数据
  async getAgentMetrics(id, params) {
    try {
      // 移除可能存在的花括号
      if (id.startsWith('{') && id.endsWith('}')) {
        id = id.substring(1, id.length - 1);
      }
      
      const response = await api.get(`/agents/${id}/metrics`, { params })
      
      // 处理不同格式的响应数据
      if (!response.data) {
        throw new Error('Invalid metrics data format')
      }
      
      // 如果响应直接是数组格式
      if (Array.isArray(response.data)) {
        return response.data;
      }
      
      // 如果响应包含 data 字段，且为数组
      if (response.data.data && Array.isArray(response.data.data)) {
        return response.data.data;
      }
      
      // 其他情况，返回整个响应数据
      return response.data;
    } catch (error) {
      console.error(`Failed to fetch metrics for agent ${id}:`, error)
      throw error
    }
  },
  
  // 更新代理信息
  async updateAgent(id, data) {
    try {
      const response = await api.put(`/agents/${id}`, data)
      if (!response.data || typeof response.data !== 'object') {
        throw new Error('Invalid update response format')
      }
      return response.data
    } catch (error) {
      console.error(`Failed to update agent ${id}:`, error)
      throw error
    }
  },
  
  // 删除代理
  async deleteAgent(id) {
    try {
      await api.delete(`/agents/${id}`)
    } catch (error) {
      console.error(`Failed to delete agent ${id}:`, error)
      throw error
    }
  }
} 