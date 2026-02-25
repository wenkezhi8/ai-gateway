import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import Layout from '@/components/Layout/index.vue'
import { useUserStore } from '@/store/user'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      {
        path: 'docs',
        name: 'Docs',
        component: () => import('@/views/docs/index.vue'),
        meta: { title: '文档中心', icon: 'Document' }
      },
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '监控仪表盘', icon: 'Monitor' }
      },
      {
        path: 'ops',
        name: 'Ops',
        component: () => import('@/views/ops/index.vue'),
        meta: { title: '运维监控', icon: 'Operation' }
      },
      {
        path: 'routing',
        name: 'Routing',
        component: () => import('@/views/routing/index.vue'),
        meta: { title: '路由策略', icon: 'Guide' }
      },
      {
        path: 'cache',
        name: 'Cache',
        component: () => import('@/views/cache/index.vue'),
        meta: { title: '缓存管理', icon: 'Box' }
      },
      {
        path: 'alerts',
        name: 'Alerts',
        component: () => import('@/views/alerts/index.vue'),
        meta: { title: '告警管理', icon: 'Bell' }
      },
      {
        path: 'api-management',
        name: 'ApiManagement',
        component: () => import('@/views/api-management/index.vue'),
        meta: { title: 'API 管理', icon: 'Connection' }
      },
      {
        path: 'model-management',
        name: 'ModelManagement',
        component: () => import('@/views/model-management/index.vue'),
        meta: { title: '模型管理', icon: 'Collection' }
      },
      {
        path: 'providers-accounts',
        name: 'ProvidersAccounts',
        component: () => import('@/views/accounts-limit/index.vue'),
        meta: { title: '账号与限额', icon: 'Key' }
      },
      {
        path: 'limit-management',
        name: 'LimitManagement',
        component: () => import('@/views/accounts-limit/index.vue'),
        meta: { title: '账号与限额', icon: 'Key' }
      },
      {
        path: 'chat',
        name: 'Chat',
        component: () => import('@/views/chat/index.vue'),
        meta: { title: 'AI 对话', icon: 'ChatDotRound' }
      },
      {
        path: 'settings',
        name: 'Settings',
        component: () => import('@/views/settings/index.vue'),
        meta: { title: '系统设置', icon: 'Setting' }
      }
    ]
  },
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: { title: '登录' }
  },
  {
    path: '/p/chat',
    name: 'PublicChat',
    component: () => import('@/views/chat/index.vue'),
    meta: { title: 'AI 智能助手', public: true }
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/error/404.vue'),
    meta: { title: '404' }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫
router.beforeEach((to, _from, next) => {
  // 设置页面标题
  document.title = `${to.meta.title || 'AI Gateway'} - AI Gateway`

  // 公开页面不需要认证
  if (to.meta.public || to.path === '/login') {
    next()
    return
  }

  // 检查是否已登录
  const userStore = useUserStore()
  const token = userStore.token || localStorage.getItem('token')

  if (!token && to.path !== '/login') {
    next('/login')
    return
  }

  next()
})

export default router
