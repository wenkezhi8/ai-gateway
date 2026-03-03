import { beforeEach, describe, expect, it, vi } from 'vitest'

import { getTaskTypeDistribution } from './routing-domain'

const requestMock = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  delete: vi.fn(),
  post: vi.fn()
}))

vi.mock('./request', () => ({
  request: requestMock
}))

describe('routing domain task-type distribution api', () => {
  beforeEach(() => {
    requestMock.get.mockReset()
  })

  it('calls task-type distribution endpoint without refresh flag by default', async () => {
    requestMock.get.mockResolvedValue({ distribution: [] })

    await getTaskTypeDistribution()

    expect(requestMock.get).toHaveBeenCalledWith('/admin/feedback/task-type-distribution')
  })

  it('passes refresh query when force refresh is requested', async () => {
    requestMock.get.mockResolvedValue({ distribution: [] })

    await getTaskTypeDistribution({ refresh: true })

    expect(requestMock.get).toHaveBeenCalledWith('/admin/feedback/task-type-distribution', {
      params: { refresh: 'true' }
    })
  })
})
