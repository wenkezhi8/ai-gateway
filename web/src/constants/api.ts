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
    BASE: '/api/v1',
    CHAT_COMPLETIONS: '/api/v1/chat/completions',
    COMPLETIONS: '/api/v1/completions',
    EMBEDDINGS: '/api/v1/embeddings',
    PROVIDERS: '/api/v1/providers',
    MODELS: '/api/v1/models',
    CONFIG_PROVIDERS: '/api/v1/config/providers',
    SEARCH: '/api/v1/search',
  },

  // Anthropic 兼容接口
  ANTHROPIC: {
    BASE: '/api/anthropic',
    MESSAGES: '/api/anthropic/v1/messages',
  },
  
  // 认证接口
  AUTH: {
    LOGIN: '/api/auth/login',
    LOGOUT: '/api/auth/logout',
    ME: '/api/auth/me',
    REFRESH: '/api/auth/refresh',
    CHANGE_PASSWORD: '/api/auth/change-password',
    PROFILE: '/api/auth/profile',
    VALIDATE: '/api/auth/validate',
  },
  
  // 管理接口
  ADMIN: {
    ACCOUNTS: '/api/admin/accounts',
    PROVIDERS: '/api/admin/providers',
    ROUTING: '/api/admin/routing',
    ROUTER: '/api/admin/router',
    DASHBOARD: '/api/admin/dashboard',
    CACHE: '/api/admin/cache',
    API_KEYS: '/api/admin/api-keys',
    FEEDBACK: '/api/admin/feedback',
    ALERTS: '/api/admin/alerts',
    SETTINGS: {
      DEFAULTS: '/admin/settings/defaults',
      UI: '/admin/settings/ui',
    },
  },
  
  // 子路径
  ROUTER: {
    CONFIG: '/api/admin/router/config',
    MODEL_REGISTRY: '/api/admin/router/model-registry',
    AVAILABLE_MODELS: '/api/admin/router/available-models',
    TOP_MODELS: '/api/admin/router/top-models',
    PROVIDER_DEFAULTS: '/api/admin/router/provider-defaults',
    TTL_CONFIG: '/api/admin/router/ttl-config',
    CASCADE_RULES: '/api/admin/router/cascade-rules',
    CLASSIFIER_HEALTH: '/api/admin/router/classifier/health',
    CLASSIFIER_STATS: '/api/admin/router/classifier/stats',
    CLASSIFIER_SWITCH: '/api/admin/router/classifier/switch',
    CLASSIFIER_SWITCH_ASYNC: '/api/admin/router/classifier/switch-async',
    CLASSIFIER_SWITCH_TASKS: '/api/admin/router/classifier/switch-tasks',
    OLLAMA_STOP: '/api/admin/router/ollama/stop',
  },
  
  FEEDBACK: {
    STATS: '/api/admin/feedback/stats',
    PERFORMANCE: '/api/admin/feedback/performance',
    TOP_MODELS: '/api/admin/feedback/top-models',
    RECENT: '/api/admin/feedback/recent',
    TASK_TYPE_DISTRIBUTION: '/api/admin/feedback/task-type-distribution',
    OPTIMIZE: '/api/admin/feedback/optimize',
  },
  
  CACHE: {
    STATS: '/api/admin/cache/stats',
    CONFIG: '/api/admin/cache/config',
    HEALTH: '/api/admin/cache/health',
    SUMMARY: '/api/admin/cache/summary',
    SEMANTIC_SIGNATURES: '/api/admin/cache/semantic-signatures',
    QUALITY_CONFIG: '/api/admin/cache/quality-config',
    INVALIDATE_LOW_QUALITY: '/api/admin/cache/invalidate-low-quality',
    RULES: '/api/admin/cache/rules',
    EXPORT: '/api/admin/cache/export',
  },
  
  USAGE: {
    LOGS: '/admin/usage/logs',
    STATS: '/admin/usage/stats',
    CLEAR: '/admin/usage/logs',
  },
  
  TRACES: {
    LIST: '/admin/traces',
    DETAIL: '/admin/traces/:request_id',
    CLEAR: '/admin/traces',
  },
} as const
