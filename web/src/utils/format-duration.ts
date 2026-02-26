/**
 * 格式化毫秒为人类可读的时间格式
 * 9080ms -> "9.08s"
 * 1500ms -> "1.50s"
 * 500ms -> "0.50s"
 * 100ms -> "0.10s"
 * 10ms -> "0.01s"
 * null/undefined -> "-"
 */
export function formatDuration(ms: number | null | undefined): string {
  if (ms === null || ms === undefined || isNaN(ms)) {
    return '-'
  }
  
  const seconds = ms / 1000
  return `${seconds.toFixed(2)}s`
}
