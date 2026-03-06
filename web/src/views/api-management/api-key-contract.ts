export interface ApiKeyApiRecord {
  id: string
  name: string
  key: string
  description?: string
  created_at: string
  last_used?: string
  last_used_at?: string
  enabled: boolean
}

export interface ApiKeyViewRecord {
  id: string
  name: string
  key: string
  description?: string
  created_at: string
  last_used?: string
  enabled: boolean
}

export function normalizeApiKeyRecord(record: ApiKeyApiRecord): ApiKeyViewRecord {
  return {
    id: record.id,
    name: record.name,
    key: record.key,
    description: record.description,
    created_at: record.created_at,
    last_used: record.last_used ?? record.last_used_at,
    enabled: record.enabled
  }
}
