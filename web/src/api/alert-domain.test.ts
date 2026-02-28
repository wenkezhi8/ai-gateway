import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from './envelope'
import {
  getAlerts,
  acknowledgeAlert,
  resolveAlert,
  acknowledgeAllAlerts,
  clearResolvedAlerts
} from './alert-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn(),
  delete: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('alert-domain', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
    requestMock.put.mockReset()
    requestMock.post.mockReset()
    requestMock.delete.mockReset()
  })

  describe('getAlerts', () => {
    it('unwraps success envelope', async () => {
      requestMock.get.mockResolvedValue({
        success: true,
        data: { alerts: [{ id: '1', status: 'firing' }] }
      })
      const data = await getAlerts()
      expect(requestMock.get).toHaveBeenCalledWith('/admin/alerts', { silent: true })
      expect((data as any).alerts[0].id).toBe('1')
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.get.mockResolvedValue({
        success: false,
        error: { code: 'internal_error', message: 'Failed to fetch' }
      })
      await expect(getAlerts()).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.get.mockResolvedValue({ alerts: [{ id: '2', status: 'resolved' }] })
      const data = await getAlerts()
      expect((data as any).alerts[0].id).toBe('2')
    })
  })

  describe('acknowledgeAlert', () => {
    it('unwraps success envelope', async () => {
      requestMock.put.mockResolvedValue({ success: true, data: { id: '1', status: 'acknowledged' } })
      const data = await acknowledgeAlert('1')
      expect(requestMock.put).toHaveBeenCalledWith('/admin/alerts/1/acknowledge', {})
      expect((data as any).id).toBe('1')
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.put.mockResolvedValue({
        success: false,
        error: { code: 'not_found', message: 'Alert not found' }
      })
      await expect(acknowledgeAlert('999')).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.put.mockResolvedValue({ id: '1', status: 'acknowledged' })
      const data = await acknowledgeAlert('1')
      expect((data as any).id).toBe('1')
    })
  })

  describe('resolveAlert', () => {
    it('unwraps success envelope', async () => {
      requestMock.put.mockResolvedValue({ success: true, data: { id: '1', status: 'resolved' } })
      const data = await resolveAlert('1')
      expect(requestMock.put).toHaveBeenCalledWith('/admin/alerts/1/resolve', {})
      expect((data as any).id).toBe('1')
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.put.mockResolvedValue({
        success: false,
        error: { code: 'not_found', message: 'Alert not found' }
      })
      await expect(resolveAlert('999')).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.put.mockResolvedValue({ id: '1', status: 'resolved' })
      const data = await resolveAlert('1')
      expect((data as any).id).toBe('1')
    })
  })

  describe('acknowledgeAllAlerts', () => {
    it('unwraps success envelope', async () => {
      requestMock.post.mockResolvedValue({ success: true, data: { count: 5 } })
      const data = await acknowledgeAllAlerts()
      expect(requestMock.post).toHaveBeenCalledWith('/admin/alerts/acknowledge-all', {})
      expect((data as any).count).toBe(5)
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.post.mockResolvedValue({
        success: false,
        error: { code: 'internal_error', message: 'Failed' }
      })
      await expect(acknowledgeAllAlerts()).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.post.mockResolvedValue({ count: 3 })
      const data = await acknowledgeAllAlerts()
      expect((data as any).count).toBe(3)
    })
  })

  describe('clearResolvedAlerts', () => {
    it('unwraps success envelope', async () => {
      requestMock.delete.mockResolvedValue({ success: true, data: { count: 2 } })
      const data = await clearResolvedAlerts()
      expect(requestMock.delete).toHaveBeenCalledWith('/admin/alerts/clear-resolved')
      expect((data as any).count).toBe(2)
    })

    it('throws ApiError on failure envelope', async () => {
      requestMock.delete.mockResolvedValue({
        success: false,
        error: { code: 'internal_error', message: 'Failed' }
      })
      await expect(clearResolvedAlerts()).rejects.toThrow(ApiError)
    })

    it('accepts plain payload', async () => {
      requestMock.delete.mockResolvedValue({ count: 1 })
      const data = await clearResolvedAlerts()
      expect((data as any).count).toBe(1)
    })
  })
})
