import { describe, expect, it } from 'vitest'

import { applyCacheConfigPayload, buildCacheConfigPayload } from './cache-tier-config'

describe('cache tier config helpers', () => {
  it('should build payload with cold tier fields', () => {
    const payload = buildCacheConfigPayload({
      enabled: true,
      strategy: 'semantic',
      similarityThreshold: 92,
      defaultTTLSeconds: 1800,
      maxEntries: 10000,
      evictionPolicy: 'lru',
      vectorEnabled: true,
      vectorDimension: 768,
      vectorQueryTimeoutMs: 1000,
      vectorThresholds: { qa: 0.93 },
      coldVectorEnabled: true,
      coldVectorQueryEnabled: true,
      coldVectorBackend: 'qdrant',
      coldVectorDualWriteEnabled: true,
      coldVectorSimilarityThreshold: 0.9,
      coldVectorTopK: 3,
      hotMemoryHighWatermarkPercent: 80,
      hotMemoryReliefPercent: 70,
      hotToColdBatchSize: 256,
      hotToColdIntervalSeconds: 25,
      coldVectorQdrantURL: 'http://127.0.0.1:6333',
      coldVectorQdrantAPIKey: 'k',
      coldVectorQdrantCollection: 'c',
      coldVectorQdrantTimeoutMs: 1800
    })

    expect(payload.cold_vector_enabled).toBe(true)
    expect(payload.cold_vector_backend).toBe('qdrant')
    expect(payload.hot_to_cold_batch_size).toBe(256)
    expect(payload.cold_vector_qdrant_url).toBe('http://127.0.0.1:6333')
  })

  it('should normalize server payload into local model', () => {
    const model = {
      enabled: false,
      strategy: 'exact',
      similarityThreshold: 0,
      defaultTTLSeconds: 0,
      maxEntries: 0,
      evictionPolicy: '',
      vectorEnabled: false,
      vectorDimension: 0,
      vectorQueryTimeoutMs: 0,
      vectorThresholds: {},
      coldVectorEnabled: false,
      coldVectorQueryEnabled: true,
      coldVectorBackend: 'sqlite',
      coldVectorDualWriteEnabled: false,
      coldVectorSimilarityThreshold: 0.92,
      coldVectorTopK: 1,
      hotMemoryHighWatermarkPercent: 75,
      hotMemoryReliefPercent: 65,
      hotToColdBatchSize: 500,
      hotToColdIntervalSeconds: 30,
      coldVectorQdrantURL: '',
      coldVectorQdrantAPIKey: '',
      coldVectorQdrantCollection: '',
      coldVectorQdrantTimeoutMs: 0
    }

    applyCacheConfigPayload(model, {
      enabled: true,
      strategy: 'semantic',
      similarity_threshold: 0.91,
      default_ttl_seconds: 3600,
      max_entries: 5000,
      eviction_policy: 'lfu',
      vector_enabled: true,
      vector_dimension: 1024,
      vector_query_timeout_ms: 1200,
      vector_thresholds: { chat: 0.9 },
      cold_vector_enabled: true,
      cold_vector_query_enabled: false,
      cold_vector_backend: 'qdrant',
      cold_vector_dual_write_enabled: true,
      cold_vector_similarity_threshold: 0.88,
      cold_vector_top_k: 4,
      hot_memory_high_watermark_percent: 78,
      hot_memory_relief_percent: 66,
      hot_to_cold_batch_size: 128,
      hot_to_cold_interval_seconds: 20,
      cold_vector_qdrant_url: 'http://127.0.0.1:6333',
      cold_vector_qdrant_api_key: 'abc',
      cold_vector_qdrant_collection: 'tier',
      cold_vector_qdrant_timeout_ms: 1900
    })

    expect(model.similarityThreshold).toBe(91)
    expect(model.coldVectorEnabled).toBe(true)
    expect(model.coldVectorQueryEnabled).toBe(false)
    expect(model.coldVectorBackend).toBe('qdrant')
    expect(model.hotToColdBatchSize).toBe(128)
    expect(model.coldVectorQdrantCollection).toBe('tier')
  })
})
