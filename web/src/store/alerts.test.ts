import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

import { useAlertsStore } from './alerts'

const alertApiMock = vi.hoisted(() => ({
  getHistory: vi.fn(),
  resolveAlert: vi.fn()
}))

const emitMock = vi.hoisted(() => vi.fn())

vi.mock('@/api/alert', () => ({
  alertApi: alertApiMock
}))

vi.mock('@/utils/eventBus', () => ({
  eventBus: {
    emit: emitMock
  },
  DATA_EVENTS: {
    ALERTS_CHANGED: 'data:alerts:changed'
  }
}))

vi.mock('element-plus', () => ({
  ElMessage: {
    success: vi.fn(),
    error: vi.fn()
  }
}))

describe('alerts store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    alertApiMock.getHistory.mockReset()
    alertApiMock.resolveAlert.mockReset()
    emitMock.mockReset()
  })

  it('should fetch pending/resolved alerts from alert facade history api', async () => {
    alertApiMock.getHistory.mockResolvedValue({
      data: {
        list: [
          {
            id: 'a-1',
            time: '2026-03-01T00:00:00Z',
            level: 'critical',
            source: 'gateway',
            message: 'CPU high',
            status: 'pending'
          },
          {
            id: 'a-2',
            time: '2026-03-01T00:01:00Z',
            level: 'warning',
            source: 'gateway',
            message: 'Recovered',
            status: 'resolved'
          }
        ],
        total: 2
      }
    })

    const store = useAlertsStore()
    await store.fetchAlerts()

    expect(alertApiMock.getHistory).toHaveBeenCalledTimes(1)
    expect(store.pendingAlerts).toHaveLength(1)
    expect(store.resolvedAlerts).toHaveLength(1)
    expect(store.alertCount.pending).toBe(1)
    expect(store.alertCount.resolved).toBe(1)
  })

  it('should resolve an alert and emit changed event', async () => {
    alertApiMock.getHistory.mockResolvedValue({
      data: {
        list: [
          {
            id: 'a-1',
            time: '2026-03-01T00:00:00Z',
            level: 'critical',
            source: 'gateway',
            message: 'CPU high',
            status: 'pending'
          }
        ],
        total: 1
      }
    })
    alertApiMock.resolveAlert.mockResolvedValue({ success: true })

    const store = useAlertsStore()
    await store.fetchAlerts()
    const ok = await store.resolveAlert('a-1')

    expect(ok).toBe(true)
    expect(store.findById('a-1')?.status).toBe('resolved')
    expect(emitMock).toHaveBeenCalledWith('data:alerts:changed')
  })

  it('should drop acknowledged workflow and expose pending/resolved only', () => {
    const store = useAlertsStore() as any

    expect(store.acknowledgeAlert).toBeUndefined()
    expect(store.acknowledgeAll).toBeUndefined()
    expect(store.clearResolved).toBeUndefined()
    expect(store.acknowledgedAlerts).toBeUndefined()
  })
})
