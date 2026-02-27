export const SETTINGS_MENU_ITEMS = [
  { key: 'appearance', label: '外观设置', icon: 'Brush' },
  { key: 'gateway', label: '网关配置', icon: 'Connection' },
  { key: 'cache', label: '缓存配置', icon: 'Box' },
  { key: 'logging', label: '日志配置', icon: 'Document' },
  { key: 'security', label: '安全配置', icon: 'Lock' },
  { key: 'about', label: '关于', icon: 'InfoFilled' }
] as const

export const THEME_COLOR_OPTIONS = [
  '#007AFF',
  '#34C759',
  '#FF9500',
  '#FF3B30',
  '#AF52DE',
  '#FF2D55',
  '#5856D6',
  '#00C7BE'
] as const

export const SETTINGS_DEFAULT_VALUES = {
  theme: 'auto',
  themeVariant: 'apple',
  primaryColor: '#007AFF',
  borderRadius: 16,
  enableAnimation: true,
  gateway: {
    host: '0.0.0.0',
    port: 8080,
    timeout: 30,
    maxConnections: 1000,
    enableCors: true,
    corsOrigins: '*'
  },
  cache: {
    enabled: true,
    type: 'memory',
    defaultTTL: 3600,
    maxSize: 1024,
    redis: {
      host: 'localhost:6379',
      password: '',
      db: 0
    }
  },
  logging: {
    level: 'info',
    format: 'json',
    outputs: ['console'],
    filePath: '/var/log/ai-gateway',
    maxFileSize: 100,
    maxBackups: 7
  },
  security: {
    enabled: true,
    type: 'apikey',
    rateLimit: true,
    rateLimitRPM: 100,
    ipWhitelist: ''
  }
} as const

export function createSettingsDefaults() {
  return JSON.parse(JSON.stringify(SETTINGS_DEFAULT_VALUES))
}
