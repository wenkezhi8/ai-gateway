import type { EditionConfig } from '../../api/edition-domain'
import { DASHBOARD_ROUTE } from '../../constants/navigation'

export interface MenuItem {
  path: string
  title: string
  icon: string
  minEdition?: 'standard' | 'enterprise'
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
  { path: '/ollama', title: 'Ollama 管理', icon: 'Cpu', minEdition: 'standard' },
  { path: '/settings', title: '系统设置', icon: 'Setting' }
]

const EDITION_LEVEL = {
  basic: 1,
  standard: 2,
  enterprise: 3
} as const

export function getMenuItems(edition: EditionConfig | null): MenuItem[] {
  if (!edition) {
    return BASE_MENUS.filter((item) => !item.minEdition)
  }

  const currentLevel = EDITION_LEVEL[edition.type]
  return BASE_MENUS.filter((item) => {
    if (!item.minEdition) return true
    return currentLevel >= EDITION_LEVEL[item.minEdition]
  })
}
