/**
 * 日期格式化
 */
export const formatDate = (date: Date | string | number, format = 'YYYY-MM-DD HH:mm:ss'): string => {
  const d = new Date(date)

  const tokens: Record<string, () => string> = {
    YYYY: () => d.getFullYear().toString(),
    MM: () => (d.getMonth() + 1).toString().padStart(2, '0'),
    DD: () => d.getDate().toString().padStart(2, '0'),
    HH: () => d.getHours().toString().padStart(2, '0'),
    mm: () => d.getMinutes().toString().padStart(2, '0'),
    ss: () => d.getSeconds().toString().padStart(2, '0')
  }

  let result = format
  for (const [token, fn] of Object.entries(tokens)) {
    result = result.replace(token, fn())
  }

  return result
}

/**
 * 相对时间
 */
export const timeAgo = (date: Date | string | number): string => {
  const now = new Date()
  const d = new Date(date)
  const diff = now.getTime() - d.getTime()

  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) return `${days}天前`
  if (hours > 0) return `${hours}小时前`
  if (minutes > 0) return `${minutes}分钟前`
  return '刚刚'
}
