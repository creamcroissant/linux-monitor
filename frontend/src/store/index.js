import { createStore } from 'vuex'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import agentApi from '@/api/agent'

export default createStore({
  state: {
    agents: [],
    token: localStorage.getItem('token') || null,
    user: null,
    error: null,
    loading: false
  },
  mutations: {
    SET_AGENTS(state, agents) {
      state.agents = agents.map(agent => ({
        ...agent,
        last_seen: agent.last_seen ? new Date(agent.last_seen) : null,
        created_at: agent.created_at ? new Date(agent.created_at) : null,
        updated_at: agent.updated_at ? new Date(agent.updated_at) : null,
      }))
    },
    SET_TOKEN(state, token) {
      state.token = token
      localStorage.setItem('token', token)
    },
    SET_USER(state, user) {
      state.user = user
      localStorage.setItem('user', JSON.stringify(user))
    },
    SET_ERROR(state, error) {
      state.error = error
    },
    CLEAR_ERROR(state) {
      state.error = null
    },
    SET_LOADING(state, status) {
      state.loading = status
    },
    LOGOUT(state) {
      state.token = null
      state.user = null
      localStorage.removeItem('token')
      localStorage.removeItem('user')
    }
  },
  actions: {
    async fetchAgents({ commit }) {
      commit('SET_LOADING', true)
      try {
        commit('CLEAR_ERROR')
        console.log('Vuex 开始获取代理列表...')
        
        // 检查 token 是否存在
        const token = localStorage.getItem('token')
        if (!token) {
          console.error('没有发现认证token，无法获取代理列表')
          throw new Error('未授权')
        }
        
        const agents = await agentApi.getAgents()
        
        // 确保代理列表是有效的数组
        if (Array.isArray(agents)) {
          console.log(`Vuex 成功获取代理列表，数量: ${agents.length}`)
          
          // 直接打印每个代理的详细信息，便于调试
          agents.forEach((agent, index) => {
            console.log(`代理 ${index+1}: ID=${agent.id}, 名称=${agent.name}, 在线=${agent.is_online}, 系统=${agent.platform}`)
          })
          
          commit('SET_AGENTS', agents)
          return agents
        } else {
          console.error('代理列表格式错误，期望数组但收到:', agents)
          commit('SET_AGENTS', [])
          return []
        }
      } catch (error) {
        const message = error.response?.data?.error || error.message || '获取代理列表失败'
        console.error('Vuex 获取代理列表错误:', message)
        console.error('错误详情:', error)
        
        if (error.response?.status === 401) {
          console.log('检测到401未授权错误，清除认证信息')
          commit('LOGOUT')
        }
        
        commit('SET_ERROR', message)
        commit('SET_AGENTS', []) // 确保置空代理列表
        return []
      } finally {
        commit('SET_LOADING', false)
      }
    },
    async login({ commit }, credentials) {
      commit('SET_LOADING', true)
      try {
        commit('CLEAR_ERROR')
        console.log('尝试登录，凭据:', credentials.username)
        const response = await axios.post('/api/login', credentials)
        console.log('登录成功，服务器响应:', response.data)
        
        if (!response.data || !response.data.token) {
          throw new Error('无效的登录响应')
        }
        
        // 确保token设置正确
        const token = response.data.token
        commit('SET_TOKEN', token)
        
        // 确保用户数据存储正确
        if (response.data.user) {
          commit('SET_USER', response.data.user)
        } else {
          // 尝试获取用户信息
          try {
            const userResponse = await axios.get('/api/users/me', {
              headers: {
                'Authorization': `Bearer ${token}`
              }
            })
            commit('SET_USER', userResponse.data)
          } catch (userError) {
            console.error('获取用户信息失败:', userError)
          }
        }
        
        ElMessage.success('登录成功')
        return response.data
      } catch (error) {
        const message = error.response?.data?.error || error.message || '登录失败'
        const detail = error.response?.data?.detail || ''
        commit('SET_ERROR', message)
        ElMessage.error(detail ? `${message}: ${detail}` : message)
        console.error('Login error:', error)
        throw error
      } finally {
        commit('SET_LOADING', false)
      }
    },
    logout({ commit }) {
      commit('LOGOUT')
      commit('CLEAR_ERROR')
      ElMessage.success('已退出登录')
    }
  },
  getters: {
    isAuthenticated: state => !!state.token,
    agents: state => state.agents,
    onlineAgents: state => state.agents.filter(agent => agent.is_online),
    offlineAgents: state => state.agents.filter(agent => !agent.is_online),
    error: state => state.error,
    isLoading: state => state.loading
  }
}) 