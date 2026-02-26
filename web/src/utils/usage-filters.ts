type UsageFilterInput = {
  model?: string
  experimentTag?: string
  domainTag?: string
}

type UsageRowLike = {
  model: string
  experimentTag: string
  domainTag: string
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
    if (filter.experimentTag && row.experimentTag !== filter.experimentTag) {
      return false
    }
    if (filter.domainTag && row.domainTag !== filter.domainTag) {
      return false
    }
    return true
  })
}
