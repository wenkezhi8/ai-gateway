export const ALERT_LEVEL_OPTIONS = [
  { label: '全部', value: '' },
  { label: '严重', value: 'critical' },
  { label: '警告', value: 'warning' },
  { label: '信息', value: 'info' }
] as const

export const ALERT_METRIC_OPTIONS = [
  { label: 'API延迟', value: 'latency' },
  { label: '错误率', value: 'error_rate' },
  { label: '额度使用率', value: 'quota' },
  { label: '服务可用性', value: 'availability' },
  { label: '缓存命中率', value: 'cache_hit_rate' }
] as const

export const ALERT_OPERATOR_OPTIONS = [
  { label: '大于', value: '>' },
  { label: '小于', value: '<' },
  { label: '等于', value: '=' },
  { label: '大于等于', value: '>=' },
  { label: '小于等于', value: '<=' }
] as const

export const ALERT_UNIT_OPTIONS = [
  { label: 'ms', value: 'ms' },
  { label: '%', value: '%' },
  { label: '次', value: 'count' }
] as const

export const ALERT_NOTIFY_CHANNEL_OPTIONS = [
  { label: '邮件', value: 'email' },
  { label: '钉钉', value: 'dingtalk' },
  { label: '企业微信', value: 'wechat' },
  { label: 'Webhook', value: 'webhook' }
] as const

export const ALERT_RULE_LEVEL_RADIOS = [
  { label: '严重', value: 'critical' },
  { label: '警告', value: 'warning' },
  { label: '信息', value: 'info' }
] as const
