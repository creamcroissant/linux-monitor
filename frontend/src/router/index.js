/*
 * 路由配置文件
 * 
 * 这个文件定义了前端应用的路由配置，管理页面间的导航和访问控制。
 * 主要功能:
 * - 定义应用的所有路由及其对应的组件
 * - 配置路由元信息（如权限要求）
 * - 实现全局路由守卫，处理用户认证和授权
 * - 处理登录状态的路由重定向
 * - 使用hash模式路由确保兼容性
 */

import { createRouter, createWebHistory, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: () => import('@/views/Dashboard.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/agent/:id',
    name: 'AgentDetail',
    component: () => import('@/views/AgentDetail.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/manage',
    name: 'Manage',
    component: () => import('@/views/Manage.vue'),
    meta: { requiresAuth: true }
  },
  {
    path: '/users',
    name: 'UserManage',
    component: () => import('@/views/UserManage.vue'),
    meta: { requiresAuth: true, requiresAdmin: true }
  },
  {
    path: '/login',
    redirect: '/'
  }
]

const router = createRouter({
  // 使用createWebHashHistory而不是createWebHistory
  // 这样可以确保在任何路径下都能正确加载资源
  history: createWebHashHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, from, next) => {
  const isAuthenticated = localStorage.getItem('token')
  
  // 访问需要认证的页面但未登录时重定向到登录页
  if (to.meta.requiresAuth && !isAuthenticated) {
    next({ path: '/', query: { redirect: to.fullPath } })
    return
  }
  
  // 如果用户已登录且访问的是登录页，重定向到仪表盘
  if ((to.path === '/' || to.path === '/login') && isAuthenticated) {
    next({ name: 'Dashboard' })
    return
  }
  
  // 访问需要管理员权限的页面
  if (to.meta.requiresAdmin) {
    const userStr = localStorage.getItem('user')
    if (userStr) {
      try {
        const user = JSON.parse(userStr)
        if (user.role !== 'admin') {
          next({ name: 'Dashboard' })
          return
        }
      } catch (e) {
        next({ path: '/' })
        return
      }
    }
  }
  
  next()
})

export default router 