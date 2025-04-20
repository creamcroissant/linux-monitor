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
        const agents = await agentApi.getAgents()
        commit('SET_AGENTS', agents)
        return agents
      } catch (error) {
        const message = error.response?.data?.error || error.message || '获取代理列表失败'
        commit('SET_ERROR', message)
        console.error('Error fetching agents:', error)
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
        commit('SET_TOKEN', response.data.token)
        commit('SET_USER', response.data.user)
        
        // 验证设置是否生效
        console.log('登录后，token已设置:', {
          storeToken: store.state.token,
          localStorageToken: localStorage.getItem('token')
        })
        
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