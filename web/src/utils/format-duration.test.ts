import { describe, it, expect } from 'vitest'
import { formatDuration } from './format-duration'

describe('formatDuration', () => {
  // 边界情况：null/undefined/NaN
  it('should format null/undefined as "-"', () => {
    expect(formatDuration(null as any)).toBe('-')
    expect(formatDuration(undefined as any)).toBe('-')
    expect(formatDuration(NaN)).toBe('-')
  })

  // 0ms → 0.00s
  it('should format 0ms as "0.00s"', () => {
    expect(formatDuration(0)).toBe('0.00s')
  })

  // 小于1秒的情况
  it('should format 1ms as "0.00s"', () => {
    expect(formatDuration(1)).toBe('0.00s')
  })

  it('should format 100ms as "0.10s"', () => {
    expect(formatDuration(100)).toBe('0.10s')
  })

  it('should format 500ms as "0.50s"', () => {
    expect(formatDuration(500)).toBe('0.50s')
  })

  it('should format 999ms as "1.00s"', () => {
    expect(formatDuration(999)).toBe('1.00s')
  })

  // 秒级
  it('should format 1000ms as "1.00s"', () => {
    expect(formatDuration(1000)).toBe('1.00s')
  })

  it('should format 1500ms as "1.50s"', () => {
    expect(formatDuration(1500)).toBe('1.50s')
  })

  // 用户需求示例：9080ms → 9.08s
  it('should format 9080ms as "9.08s"', () => {
    expect(formatDuration(9080)).toBe('9.08s')
  })

  it('should format 10000ms as "10.00s"', () => {
    expect(formatDuration(10000)).toBe('10.00s')
  })

  // 大数值（仍然是秒，不做 min/h 转换）
  it('should format 60000ms as "60.00s"', () => {
    expect(formatDuration(60000)).toBe('60.00s')
  })

  it('should format 90000ms as "90.00s"', () => {
    expect(formatDuration(90000)).toBe('90.00s')
  })

  it('should format 3600000ms as "3600.00s"', () => {
    expect(formatDuration(3600000)).toBe('3600.00s')
  })
})
