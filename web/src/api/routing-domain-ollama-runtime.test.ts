import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getOllamaRuntimeConfig, preloadOllamaModels, updateOllamaRuntimeConfig } from './routing-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('routing domain ollama runtime apis', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.post.mockReset()
    requestMock.put.mockReset()
  })

  it('should request ollama runtime config endpoint', async () => {
    requestMock.get.mockResolvedValue({
      success: true,
      data: {
        config: {
          startup_mode: 'auto',
          preload: {
            auto_on_startup: false,
            targets: ['intent', 'embedding'],
            timeout_seconds: 180
          }
        },
        monitoring_stats: {
          health_status: 'healthy'
        }
      }
    })

    const data = await getOllamaRuntimeConfig()
    expect(requestMock.get).toHaveBeenCalledWith('/admin/router/ollama/runtime-config')
    expect(data.config.startup_mode).toBe('auto')
  })

  it('should update ollama runtime config endpoint', async () => {
    const payload = {
      startup_mode: 'cli' as const,
      monitoring: {
        enabled: true,
        check_interval_seconds: 15,
        auto_restart: true,
        max_restart_attempts: 3,
        restart_cooldown_seconds: 10
      }
    }
    requestMock.put.mockResolvedValue({ success: true, data: payload })

    const data = await updateOllamaRuntimeConfig(payload)
    expect(requestMock.put).toHaveBeenCalledWith('/admin/router/ollama/runtime-config', payload)
    expect(data.startup_mode).toBe('cli')
  })

  it('should trigger ollama preload endpoint', async () => {
    requestMock.post.mockResolvedValue({
      success: true,
      data: {
        results: [
          { kind: 'intent', model: 'qwen2.5:0.5b-instruct', status: 'success', duration_ms: 120 }
        ]
      }
    })

    const data = await preloadOllamaModels({ targets: ['intent', 'embedding'] })
    expect(requestMock.post).toHaveBeenCalledWith('/admin/router/ollama/preload', { targets: ['intent', 'embedding'] })
    expect(Array.isArray(data.results)).toBe(true)
  })
})
