export const USE_AUTO_MODE_AUTO = 'auto'
export const USE_AUTO_MODE_DEFAULT = 'default'
export const USE_AUTO_MODE_FIXED = 'fixed'
export const USE_AUTO_MODE_LATEST = 'latest'

export const USE_AUTO_MODE_MIGRATION_HINT = '检测到 use_auto_mode=latest 已废弃，系统已自动迁移为 auto。'

export const USE_AUTO_MODE_ALLOWED_VALUES = [
  USE_AUTO_MODE_AUTO,
  USE_AUTO_MODE_DEFAULT,
  USE_AUTO_MODE_FIXED
] as const

export type UseAutoMode = typeof USE_AUTO_MODE_ALLOWED_VALUES[number]

export type UseAutoModeContract = {
  allowed_modes?: string[]
  deprecated_mappings?: Record<string, string>
  migration_hint?: string
}

export const USE_AUTO_MODE_LABELS: Record<UseAutoMode, string> = {
  [USE_AUTO_MODE_AUTO]: 'Auto 智能选择',
  [USE_AUTO_MODE_DEFAULT]: 'Default 服务商默认',
  [USE_AUTO_MODE_FIXED]: '固定模型'
}

function buildAllowedModeSet(contract?: UseAutoModeContract): Set<string> {
  if (Array.isArray(contract?.allowed_modes) && contract?.allowed_modes.length > 0) {
    return new Set(contract.allowed_modes)
  }
  return new Set(USE_AUTO_MODE_ALLOWED_VALUES)
}

function buildDeprecatedMappings(contract?: UseAutoModeContract): Record<string, string> {
  const defaultMappings: Record<string, string> = {
    [USE_AUTO_MODE_LATEST]: USE_AUTO_MODE_AUTO
  }
  if (!contract?.deprecated_mappings) {
    return defaultMappings
  }
  return {
    ...defaultMappings,
    ...contract.deprecated_mappings
  }
}

function isUseAutoMode(value: string): value is UseAutoMode {
  return USE_AUTO_MODE_ALLOWED_VALUES.includes(value as UseAutoMode)
}

export function normalizeUseAutoMode(value: unknown, contract?: UseAutoModeContract): UseAutoMode {
  const raw = typeof value === 'string' ? value.trim() : ''
  const deprecatedMappings = buildDeprecatedMappings(contract)
  const candidate = deprecatedMappings[raw] || raw

  const allowedSet = buildAllowedModeSet(contract)
  if (!allowedSet.has(candidate)) {
    return USE_AUTO_MODE_AUTO
  }

  return isUseAutoMode(candidate) ? candidate : USE_AUTO_MODE_AUTO
}

export function resolveUseAutoModeMigrationNotice(value: unknown, contract?: UseAutoModeContract): string {
  const raw = typeof value === 'string' ? value.trim() : ''
  const deprecatedMappings = buildDeprecatedMappings(contract)
  if (!raw || !deprecatedMappings[raw]) {
    return ''
  }
  return contract?.migration_hint || USE_AUTO_MODE_MIGRATION_HINT
}
