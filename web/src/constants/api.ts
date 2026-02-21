/**
 * 端口号常量 - 前后端统一
 */
export const PORTS = {
  /** 服务端口 (前端和后端统一) */
  SERVER: 8566,
  /** Metrics 端口 */
  METRICS: 9090,
} as const

/**
 * API 基础 URL
 */
export const API_BASE_URL = `http://localhost:${PORTS.SERVER}`

/**
 * API 路径常量 - 统一管理所有 API 路径
 * 修改接口时只需改这里
 */
export const API = {
  // v1 接口 (统一入口)
  V1: {
    CHAT_COMPLETIONS: '/api/v1/chat/completions',
    COMPLETIONS: '/api/v1/completions',
    EMBEDDINGS: '/api/v1/embeddings',
    PROVIDERS: '/api/v1/providers',
    MODELS: '/api/v1/models',
    CONFIG_PROVIDERS: '/api/v1/config/providers',
  },
  
  // 认证接口
  AUTH: {
    LOGIN: '/api/auth/login',
    LOGOUT: '/api/auth/logout',
    ME: '/api/auth/me',
    REFRESH: '/api/auth/refresh',
  },
  
  // 管理接口
  ADMIN: {
    ACCOUNTS: '/api/admin/accounts',
    PROVIDERS: '/api/admin/providers',
    ROUTER: '/api/admin/router',
    DASHBOARD: '/api/admin/dashboard',
    CACHE: '/api/admin/cache',
    API_KEYS: '/api/admin/api-keys',
  },
} as const
