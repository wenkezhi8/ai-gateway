import type { EditionConfig } from '../../api/edition-domain'
import { DASHBOARD_ROUTE } from '../../constants/navigation'

export interface MenuItem {
  path: string
  title: string
  icon: string
}

const BASE_MENUS: MenuItem[] = [
  { path: DASHBOARD_ROUTE, title: '监控仪表盘', icon: 'Monitor' },
  { path: '/ops', title: '运维监控', icon: 'Operation' },
  { path: '/chat', title: 'AI 对话', icon: 'ChatDotRound' },
  { path: '/api-management', title: 'API 管理', icon: 'Connection' },
  { path: '/model-management', title: '模型管理', icon: 'Collection' },
  { path: '/providers-accounts', title: '账号与限额', icon: 'Key' },
  { path: '/usage', title: 'API 使用统计', icon: 'DataLine' },
  { path: '/trace', title: '请求链路追踪', icon: 'Share' },
  { path: '/routing', title: '路由策略', icon: 'Guide' },
  { path: '/cache', title: '缓存管理', icon: 'Box' },
  { path: '/alerts', title: '告警管理', icon: 'Bell' },
  { path: '/settings', title: '系统设置', icon: 'Setting' }
]

export function getMenuItems(_edition: EditionConfig | null): MenuItem[] {
  return BASE_MENUS
}
