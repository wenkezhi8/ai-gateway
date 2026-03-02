export type CacheTaskKey =
  | 'fact'
  | 'code'
  | 'math'
  | 'chat'
  | 'creative'
  | 'reasoning'
  | 'translate'
  | 'long_text'
  | 'unknown'

export type CacheTaskTTLItem = {
  key: CacheTaskKey
  name: string
  description: string
  ttl: number
}

export type CacheTaskTypeOption = {
  label: string
  value: CacheTaskKey
}
