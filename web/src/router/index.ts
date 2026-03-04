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
import { canAccessEditionRoute } from './guards/edition-guard'

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
        path: '/ollama',
        name: 'Ollama',
        component: () => import('@/views/ollama/index.vue'),
        meta: { title: 'Ollama 管理', icon: 'Cpu' }
      },
      {
        path: '/cache',
        name: 'Cache',
        component: () => import('@/views/cache/index.vue'),
        meta: { title: '缓存管理', icon: 'Box' }
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
        meta: { title: 'AI服务商', icon: 'Key' }
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
    path: '/vector-db',
    component: () => import('@/components/Layout/VectorDBLayout.vue'),
    redirect: '/vector-db/collections',
    children: [
      {
        path: 'collections',
        name: 'VectorDBCollections',
        component: () => import('@/views/vector-db/collections/index.vue'),
        meta: { title: '向量集合', icon: 'Collection' }
      },
      {
        path: 'search',
        name: 'VectorDBSearch',
        component: () => import('@/views/vector-db/search/index.vue'),
        meta: { title: '向量检索', icon: 'Search' }
      },
      {
        path: 'import',
        name: 'VectorDBImport',
        component: () => import('@/views/vector-db/import/index.vue'),
        meta: { title: '向量导入', icon: 'Upload' }
      },
      {
        path: 'monitoring',
        name: 'VectorDBMonitoring',
        component: () => import('@/views/vector-db/monitoring/index.vue'),
        meta: { title: '向量监控', icon: 'DataLine' }
      },
      {
        path: 'monitoring/alerts',
        name: 'VectorDBMonitoringAlerts',
        component: () => import('@/views/vector-db/monitoring/alerts.vue'),
        meta: { title: '向量告警', icon: 'Bell' }
      },
      {
        path: 'permissions',
        name: 'VectorDBPermissions',
        component: () => import('@/views/vector-db/permissions/index.vue'),
        meta: { title: '向量权限', icon: 'Key' }
      },
      {
        path: 'backup',
        name: 'VectorDBBackup',
        component: () => import('@/views/vector-db/backup/index.vue'),
        meta: { title: '备份恢复', icon: 'Folder' }
      },
      {
        path: 'audit',
        name: 'VectorDBAudit',
        component: () => import('@/views/vector-db/audit/index.vue'),
        meta: { title: '向量审计', icon: 'Tickets' }
      },
      {
        path: 'visualization',
        name: 'VectorDBVisualization',
        component: () => import('@/views/vector-db/visualization/index.vue'),
        meta: { title: '向量可视化', icon: 'DataAnalysis' }
      }
    ]
  },
  {
    path: '/knowledge',
    component: () => import('@/components/Layout/KnowledgeLayout.vue'),
    redirect: '/knowledge/documents',
    children: [
      {
        path: 'documents',
        name: 'KnowledgeDocuments',
        component: () => import('@/views/knowledge/documents/index.vue'),
        meta: { title: '知识库文档', icon: 'Document' }
      },
      {
        path: 'chat',
        name: 'KnowledgeChat',
        component: () => import('@/views/knowledge/chat/index.vue'),
        meta: { title: '知识库问答', icon: 'ChatDotRound' }
      },
      {
        path: 'config',
        name: 'KnowledgeConfig',
        component: () => import('@/views/knowledge/config/index.vue'),
        meta: { title: '知识库配置', icon: 'Setting' }
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
router.beforeEach(async (to, _from, next) => {
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

  const allowed = await canAccessEditionRoute(to.path)
  if (!allowed) {
    next(DASHBOARD_ROUTE)
    return
  }

  next()
})

export default router
