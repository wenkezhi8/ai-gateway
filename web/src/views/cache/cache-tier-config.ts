export interface CacheConfigModel {
  enabled: boolean
  strategy: string
  similarityThreshold: number
  defaultTTLSeconds: number
  maxEntries: number
  evictionPolicy: string
  vectorEnabled: boolean
  vectorDimension: number
  vectorQueryTimeoutMs: number
  vectorThresholds: Record<string, number>
  coldVectorEnabled: boolean
  coldVectorQueryEnabled: boolean
  coldVectorBackend: string
  coldVectorDualWriteEnabled: boolean
  coldVectorSimilarityThreshold: number
  coldVectorTopK: number
  hotMemoryHighWatermarkPercent: number
  hotMemoryReliefPercent: number
  hotToColdBatchSize: number
  hotToColdIntervalSeconds: number
  coldVectorQdrantURL: string
  coldVectorQdrantAPIKey: string
  coldVectorQdrantCollection: string
  coldVectorQdrantTimeoutMs: number
}

export type CacheConfigPayload = Record<string, unknown>

function readNumber(value: unknown, fallback: number): number {
  const num = Number(value)
  if (Number.isFinite(num)) {
    return num
  }
  return fallback
}

function readBoolean(value: unknown, fallback: boolean): boolean {
  if (typeof value === 'boolean') {
    return value
  }
  return fallback
}

function clamp(value: number, min: number, max: number): number {
  if (value < min) {
    return min
  }
  if (value > max) {
    return max
  }
  return value
}

function readField<T = unknown>(cfg: Record<string, any>, snakeKey: string, camelKey: string): T | undefined {
  if (cfg[snakeKey] !== undefined) {
    return cfg[snakeKey] as T
  }
  if (cfg[camelKey] !== undefined) {
    return cfg[camelKey] as T
  }
  return undefined
}

export function buildCacheConfigPayload(model: CacheConfigModel): CacheConfigPayload {
  const similarityRatio = model.similarityThreshold > 1
    ? model.similarityThreshold / 100
    : model.similarityThreshold

  return {
    enabled: model.enabled,
    strategy: model.strategy,
    similarity_threshold: clamp(similarityRatio, 0, 1),
    default_ttl_seconds: model.defaultTTLSeconds,
    max_entries: model.maxEntries,
    eviction_policy: model.evictionPolicy,
    vector_enabled: model.vectorEnabled,
    vector_dimension: model.vectorDimension,
    vector_query_timeout_ms: model.vectorQueryTimeoutMs,
    vector_thresholds: model.vectorThresholds,
    cold_vector_enabled: model.coldVectorEnabled,
    cold_vector_query_enabled: model.coldVectorQueryEnabled,
    cold_vector_backend: model.coldVectorBackend,
    cold_vector_dual_write_enabled: model.coldVectorDualWriteEnabled,
    cold_vector_similarity_threshold: model.coldVectorSimilarityThreshold,
    cold_vector_top_k: model.coldVectorTopK,
    hot_memory_high_watermark_percent: model.hotMemoryHighWatermarkPercent,
    hot_memory_relief_percent: model.hotMemoryReliefPercent,
    hot_to_cold_batch_size: model.hotToColdBatchSize,
    hot_to_cold_interval_seconds: model.hotToColdIntervalSeconds,
    cold_vector_qdrant_url: model.coldVectorQdrantURL,
    cold_vector_qdrant_api_key: model.coldVectorQdrantAPIKey,
    cold_vector_qdrant_collection: model.coldVectorQdrantCollection,
    cold_vector_qdrant_timeout_ms: model.coldVectorQdrantTimeoutMs
  }
}

export function applyCacheConfigPayload(model: CacheConfigModel, rawCfg: Record<string, any>) {
  const cfg = rawCfg || {}
  model.enabled = readBoolean(readField(cfg, 'enabled', 'enabled'), model.enabled)
  model.strategy = String(readField(cfg, 'strategy', 'strategy') ?? model.strategy)
  const similarity = readNumber(readField(cfg, 'similarity_threshold', 'similarityThreshold'), 0.92)
  model.similarityThreshold = Math.round(clamp(similarity, 0, 1) * 100)
  model.defaultTTLSeconds = readNumber(readField(cfg, 'default_ttl_seconds', 'defaultTTLSeconds'), model.defaultTTLSeconds)
  model.maxEntries = readNumber(readField(cfg, 'max_entries', 'maxEntries'), model.maxEntries)
  model.evictionPolicy = String(readField(cfg, 'eviction_policy', 'evictionPolicy') ?? model.evictionPolicy)
  model.vectorEnabled = readBoolean(readField(cfg, 'vector_enabled', 'vectorEnabled'), model.vectorEnabled)
  model.vectorDimension = readNumber(readField(cfg, 'vector_dimension', 'vectorDimension'), model.vectorDimension || 1024)
  model.vectorQueryTimeoutMs = readNumber(readField(cfg, 'vector_query_timeout_ms', 'vectorQueryTimeoutMs'), model.vectorQueryTimeoutMs || 1200)
  model.vectorThresholds = (readField(cfg, 'vector_thresholds', 'vectorThresholds') as Record<string, number>) || model.vectorThresholds
  model.coldVectorEnabled = readBoolean(readField(cfg, 'cold_vector_enabled', 'coldVectorEnabled'), model.coldVectorEnabled)
  model.coldVectorQueryEnabled = readBoolean(readField(cfg, 'cold_vector_query_enabled', 'coldVectorQueryEnabled'), model.coldVectorQueryEnabled)
  model.coldVectorBackend = String(readField(cfg, 'cold_vector_backend', 'coldVectorBackend') ?? model.coldVectorBackend)
  model.coldVectorDualWriteEnabled = readBoolean(readField(cfg, 'cold_vector_dual_write_enabled', 'coldVectorDualWriteEnabled'), model.coldVectorDualWriteEnabled)
  model.coldVectorSimilarityThreshold = readNumber(readField(cfg, 'cold_vector_similarity_threshold', 'coldVectorSimilarityThreshold'), model.coldVectorSimilarityThreshold)
  model.coldVectorTopK = readNumber(readField(cfg, 'cold_vector_top_k', 'coldVectorTopK'), model.coldVectorTopK || 1)
  model.hotMemoryHighWatermarkPercent = readNumber(readField(cfg, 'hot_memory_high_watermark_percent', 'hotMemoryHighWatermarkPercent'), model.hotMemoryHighWatermarkPercent || 75)
  model.hotMemoryReliefPercent = readNumber(readField(cfg, 'hot_memory_relief_percent', 'hotMemoryReliefPercent'), model.hotMemoryReliefPercent || 65)
  model.hotToColdBatchSize = readNumber(readField(cfg, 'hot_to_cold_batch_size', 'hotToColdBatchSize'), model.hotToColdBatchSize || 500)
  model.hotToColdIntervalSeconds = readNumber(readField(cfg, 'hot_to_cold_interval_seconds', 'hotToColdIntervalSeconds'), model.hotToColdIntervalSeconds || 30)
  model.coldVectorQdrantURL = String(readField(cfg, 'cold_vector_qdrant_url', 'coldVectorQdrantURL') ?? model.coldVectorQdrantURL)
  model.coldVectorQdrantAPIKey = String(readField(cfg, 'cold_vector_qdrant_api_key', 'coldVectorQdrantAPIKey') ?? model.coldVectorQdrantAPIKey)
  model.coldVectorQdrantCollection = String(readField(cfg, 'cold_vector_qdrant_collection', 'coldVectorQdrantCollection') ?? model.coldVectorQdrantCollection)
  model.coldVectorQdrantTimeoutMs = readNumber(readField(cfg, 'cold_vector_qdrant_timeout_ms', 'coldVectorQdrantTimeoutMs'), model.coldVectorQdrantTimeoutMs || 1500)
}
