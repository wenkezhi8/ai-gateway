import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import Layout from '@/components/Layout/index.vue'
import { useUserStore } from '@/store/user'
import {
  DASHBOARD_ROUTE,
  DOCS_ROUTE,
  HOME_ROUTE,
  LOGIN_ROUTE,
  PUBLIC_CHAT_ROUTE,
  UNAUTHORIZED_REDIRECT
} from '@/constants/navigation'

const routes: RouteRecordRaw[] = [
  {
    path: HOME_ROUTE,
    name: 'Home',
    component: () => import('@/views/home/index.vue'),
    meta: { title: 'AI Gateway 首页', public: true }
  },
  {
    path: DOCS_ROUTE,
    component: () => import('@/views/docs/index.vue'),
    meta: { title: '文档中心', public: true },
    children: [
      {
        path: '',
        redirect: '/docs/getting-started'
      },
      {
        path: 'getting-started',
        name: 'DocsGettingStarted',
        component: () => import('@/views/docs/pages/getting-started.vue'),
        meta: { title: '入门指南', public: true }
      },
      {
        path: 'wizard',
        name: 'DocsWizard',
        component: () => import('@/views/docs/pages/wizard.vue'),
        meta: { title: '安装向导', public: true }
      },
      {
        path: 'api',
        name: 'DocsApi',
        component: () => import('@/views/docs/pages/api.vue'),
        meta: { title: 'API 参考', public: true }
      },
      {
        path: 'sdk',
        name: 'DocsSdk',
        component: () => import('@/views/docs/pages/sdk.vue'),
        meta: { title: 'SDK 示例', public: true }
      },
      {
        path: 'providers',
        name: 'DocsProviders',
        component: () => import('@/views/docs/pages/providers.vue'),
        meta: { title: '服务商文档', public: true }
      },
      {
        path: 'admin',
        name: 'DocsAdmin',
        component: () => import('@/views/docs/pages/admin.vue'),
        meta: { title: '管理 API 文档', public: true }
      },
      {
        path: 'errors',
        name: 'DocsErrors',
        component: () => import('@/views/docs/pages/errors.vue'),
        meta: { title: '错误码文档', public: true }
      }
    ]
  },
  {
    path: '/console',
    component: Layout,
    redirect: DASHBOARD_ROUTE,
    children: [
      {
        path: DASHBOARD_ROUTE,
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '监控仪表盘', icon: 'Monitor' }
      },
      {
        path: '/ops',
        name: 'Ops',
        component: () => import('@/views/ops/index.vue'),
        meta: { title: '运维监控', icon: 'Operation' }
      },
      {
        path: '/routing',
        name: 'Routing',
        component: () => import('@/views/routing/index.vue'),
        meta: { title: '路由策略', icon: 'Guide' }
      },
      {
        path: '/cache',
        name: 'Cache',
        component: () => import('@/views/cache/index.vue'),
        meta: { title: '缓存管理', icon: 'Box' }
      },
      {
        path: '/vector-db/collections',
        name: 'VectorDBCollections',
        component: () => import('@/views/vector-db/collections/index.vue'),
        meta: { title: '向量集合', icon: 'Collection' }
      },
      {
        path: '/knowledge/documents',
        name: 'KnowledgeDocuments',
        component: () => import('@/views/knowledge/documents/index.vue'),
        meta: { title: '知识库文档', icon: 'Document' }
      },
      {
        path: '/knowledge/chat',
        name: 'KnowledgeChat',
        component: () => import('@/views/knowledge/chat/index.vue'),
        meta: { title: '知识库问答', icon: 'ChatDotRound' }
      },
      {
        path: '/knowledge/config',
        name: 'KnowledgeConfig',
        component: () => import('@/views/knowledge/config/index.vue'),
        meta: { title: '知识库配置', icon: 'Setting' }
      },
      {
        path: '/alerts',
        name: 'Alerts',
        component: () => import('@/views/alerts/index.vue'),
        meta: { title: '告警管理', icon: 'Bell' }
      },
      {
        path: '/api-management',
        name: 'ApiManagement',
        component: () => import('@/views/api-management/index.vue'),
        meta: { title: 'API 管理', icon: 'Connection' }
      },
      {
        path: '/model-management',
        name: 'ModelManagement',
        component: () => import('@/views/model-management/index.vue'),
        meta: { title: '模型管理', icon: 'Collection' }
      },
      {
        path: '/providers-accounts',
        name: 'ProvidersAccounts',
        component: () => import('@/views/accounts-limit/index.vue'),
        meta: { title: '账号与限额', icon: 'Key' }
      },
      {
        path: '/providers',
        name: 'Providers',
        component: () => import('@/views/providers/index.vue'),
        meta: { title: '服务商管理', icon: 'Collection' }
      },
      {
        path: '/accounts',
        name: 'Accounts',
        component: () => import('@/views/accounts/index.vue'),
        meta: { title: '账号管理', icon: 'Key' }
      },
      {
        path: '/limit-management',
        name: 'LimitManagement',
        component: () => import('@/views/limit-management/index.vue'),
        meta: { title: '限额管理', icon: 'DataLine' }
      },
      {
        path: '/usage',
        name: 'Usage',
        component: () => import('@/views/usage/index.vue'),
        meta: { title: 'API 使用统计', icon: 'DataLine' }
      },
      {
        path: '/trace',
        name: 'Trace',
        component: () => import('@/views/trace/index.vue'),
        meta: { title: '请求链路追踪', icon: 'Connection' }
      },
      {
        path: '/chat',
        name: 'Chat',
        component: () => import('@/views/chat/index.vue'),
        meta: { title: 'AI 对话', icon: 'ChatDotRound' }
      },
      {
        path: '/settings',
        name: 'Settings',
        component: () => import('@/views/settings/index.vue'),
        meta: { title: '系统设置', icon: 'Setting' }
      }
    ]
  },
  {
    path: LOGIN_ROUTE,
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: { title: '登录' }
  },
  {
    path: PUBLIC_CHAT_ROUTE,
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
  if (to.meta.public || to.path === LOGIN_ROUTE) {
    next()
    return
  }

  // 检查是否已登录
  const userStore = useUserStore()
  const token = userStore.token || localStorage.getItem('token')

  if (!token && to.path !== LOGIN_ROUTE) {
    next(UNAUTHORIZED_REDIRECT)
    return
  }

  next()
})

export default router
