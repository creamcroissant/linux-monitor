/*
 * 用户API模块
 *
 * 这个模块封装了与用户认证和管理相关的所有API请求。
 * 主要功能:
 * - 用户登录和注销
 * - 用户注册
 * - 获取当前登录用户信息
 * - 更新用户密码
 * - 管理员功能（获取用户列表、创建和删除用户）
 */

import axios from 'axios'
import { ElMessage } from 'element-plus'

// 创建用户API实例
const userApi = axios.create({
  baseURL: '/api',
  headers: {
    'Content-Type': 'application/json'
  }
})

// 请求拦截器
userApi.interceptors.request.use(
  config => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
      console.log(`用户API请求: ${config.method.toUpperCase()} ${config.url}`)
      console.log(`使用的token (前10字符): ${token.substring(0, 10)}...`)
    } else {
      console.warn(`用户API请求没有token: ${config.method.toUpperCase()} ${config.url}`)
    }
    return config
  },
  error => {
    console.error('Request error:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
userApi.interceptors.response.use(
  response => {
    return response
  },
  error => {
    if (error.response) {
      // 服务器返回错误
      const status = error.response.status
      const errorData = error.response.data
      
      let errorMessage = errorData.error || '未知错误'
      let errorDetail = errorData.detail || ''
      
      console.error(`用户API错误: ${status} - ${errorMessage}`, errorDetail)
      
      // 显示错误信息
      ElMessage.error(errorDetail ? `${errorMessage}: ${errorDetail}` : errorMessage)
      
      if (status === 401) {
        // 未授权，清除token并重定向到登录页
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        window.location.href = '/#/'
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
  // 获取用户列表
  getUsers() {
    return userApi.get('/admin/users')
      .then(response => response.data)
  },
  
  // 创建用户
  createUser(userData) {
    return userApi.post('/admin/users', userData)
      .then(response => response.data)
  },
  
  // 删除用户
  deleteUser(username) {
    return userApi.delete(`/admin/users/${username}`)
      .then(response => response.data)
  },
  
  // 获取当前用户信息
  getCurrentUser() {
    return userApi.get('/users/me')
      .then(response => response.data)
  },
  
  // 更新密码
  updatePassword(passwordData) {
    return userApi.put('/users/password', passwordData)
      .then(response => response.data)
  }
} 