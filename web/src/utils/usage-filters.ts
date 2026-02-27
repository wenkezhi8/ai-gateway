type UsageFilterInput = {
  model?: string
  taskType?: string
}

type UsageRowLike = {
  model: string
  taskType: string
}

export function getTagOptions(values: Array<string | null | undefined>): string[] {
  const set = new Set<string>()
  values.forEach(value => {
    if (value && value !== '-') {
      set.add(value)
    }
  })
  return Array.from(set)
}

export function filterUsageRows<T extends UsageRowLike>(rows: T[], filter: UsageFilterInput): T[] {
  return rows.filter(row => {
    if (filter.model && row.model !== filter.model) {
      return false
    }
    if (filter.taskType && row.taskType !== filter.taskType) {
      return false
    }
    return true
  })
}
