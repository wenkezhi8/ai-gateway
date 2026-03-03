import { beforeEach, describe, expect, it, vi, afterEach } from 'vitest'
import { ApiError } from './envelope'
import {
  getUiSettings,
  getSettingsDefaults,
  updateUiSettings,
  updateRoutingUiSettings,
  updateModelManagementUiSettings,
  updateGeneralUiSettings,
  updateGeneralUiSettingsThrottled,
  flushThrottledSettings
} from './settings-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('settings-domain', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.put.mockReset()
  })

  describe('getUiSettings', () => {
    it('unwraps success envelope', async () => {
      requestMock.get.mockResolvedValue({
        success: true,
        data: { routing: { auto_save_enabled: true } }
      })
      const data = await getUiSettings()
      expect(requestMock.get).toHaveBeenCalledWith('/api/admin/settings/ui')
      expect(data.routing?.auto_save_enabled).toBe(true)
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.get.mockResolvedValue({
        success: false,
        error: { code: 'not_found', message: 'Settings not found' }
      })
      await expect(getUiSettings()).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.get.mockResolvedValue({ routing: { auto_save_enabled: false } })
      const data = await getUiSettings()
      expect(data.routing?.auto_save_enabled).toBe(false)
    })
  })

  describe('getSettingsDefaults', () => {
    it('calls /admin/settings/defaults and unwraps payload', async () => {
      requestMock.get.mockResolvedValue({
        success: true,
        data: {
          gateway: { host: '0.0.0.0' },
          cache: { enabled: true },
          logging: { level: 'info' },
          security: { enabled: true }
        }
      })

      const data = await getSettingsDefaults()
      expect(requestMock.get).toHaveBeenCalledWith('/api/admin/settings/defaults')
      expect(data.gateway.host).toBe('0.0.0.0')
    })

    it('throws on failed defaults response', async () => {
      requestMock.get.mockResolvedValue({
        success: false,
        error: { code: 'defaults_failed', message: 'defaults failed' }
      })

      await expect(getSettingsDefaults()).rejects.toThrow(ApiError)
    })
  })

  describe('updateUiSettings', () => {
    it('unwraps success envelope', async () => {
      requestMock.put.mockResolvedValue({
        success: true,
        data: { routing: { auto_save_enabled: true } }
      })
      const data = await updateUiSettings({ routing: { auto_save_enabled: true } })
      expect(requestMock.put).toHaveBeenCalledWith('/api/admin/settings/ui', { routing: { auto_save_enabled: true } })
      expect(data.routing?.auto_save_enabled).toBe(true)
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.put.mockResolvedValue({
        success: false,
        error: { code: 'invalid_request', message: 'Invalid payload' }
      })
      await expect(updateUiSettings({})).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.put.mockResolvedValue({ routing: { auto_save_enabled: true } })
      const data = await updateUiSettings({})
      expect(data.routing?.auto_save_enabled).toBe(true)
    })
  })

  describe('updateRoutingUiSettings', () => {
    it('calls updateUiSettings with routing payload', async () => {
      requestMock.put.mockResolvedValue({ success: true, data: { routing: { auto_save_enabled: true } } })
      await updateRoutingUiSettings({ auto_save_enabled: true })
      expect(requestMock.put).toHaveBeenCalledWith('/api/admin/settings/ui', { routing: { auto_save_enabled: true } })
    })
  })

  describe('updateModelManagementUiSettings', () => {
    it('calls updateUiSettings with model_management payload', async () => {
      requestMock.put.mockResolvedValue({ success: true, data: { model_management: { last_saved_at: '2026-02-28' } } })
      await updateModelManagementUiSettings({ last_saved_at: '2026-02-28' })
      expect(requestMock.put).toHaveBeenCalledWith('/api/admin/settings/ui', { model_management: { last_saved_at: '2026-02-28' } })
    })
  })

  describe('updateGeneralUiSettings', () => {
    it('calls updateUiSettings with settings payload', async () => {
      requestMock.put.mockResolvedValue({ success: true, data: { settings: { theme: 'dark' } } })
      await updateGeneralUiSettings({ theme: 'dark' })
      expect(requestMock.put).toHaveBeenCalledWith('/api/admin/settings/ui', { settings: { theme: 'dark' } })
    })
  })
})

describe('updateGeneralUiSettingsThrottled', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.put.mockReset()
    flushThrottledSettings()
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('merges multiple calls within delay period', async () => {
    requestMock.put.mockResolvedValue({ success: true, data: { settings: { a: 1, b: 2 } } })

    const p1 = updateGeneralUiSettingsThrottled({ a: 1 }, 500)
    const p2 = updateGeneralUiSettingsThrottled({ b: 2 }, 500)

    vi.advanceTimersByTime(500)
    await Promise.all([p1, p2])

    expect(requestMock.put).toHaveBeenCalledTimes(1)
    expect(requestMock.put).toHaveBeenCalledWith('/api/admin/settings/ui', { settings: { a: 1, b: 2 } })
  })

  it('resolves all promises with same result', async () => {
    requestMock.put.mockResolvedValue({ success: true, data: { settings: { x: 1 } } })

    const p1 = updateGeneralUiSettingsThrottled({ x: 1 }, 500)
    const p2 = updateGeneralUiSettingsThrottled({ x: 2 }, 500)

    vi.advanceTimersByTime(500)
    const [r1, r2] = await Promise.all([p1, p2])

    expect(r1).toEqual(r2)
  })

  it('rejects all promises on error', async () => {
    requestMock.put.mockResolvedValue({
      success: false,
      error: { code: 'error', message: 'fail' }
    })

    const p1 = updateGeneralUiSettingsThrottled({ a: 1 }, 500)
    vi.advanceTimersByTime(500)

    await expect(p1).rejects.toThrow(ApiError)
  })
})
