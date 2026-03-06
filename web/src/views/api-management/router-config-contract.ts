import {
  USE_AUTO_MODE_AUTO,
  normalizeUseAutoMode,
  resolveUseAutoModeMigrationNotice,
  type UseAutoMode,
  type UseAutoModeContract
} from '@/constants/router-mode'

type RouterConfigApiPayload = {
  use_auto_mode?: unknown
  default_strategy?: unknown
  default_model?: unknown
  migration_notice?: unknown
  use_auto_mode_contract?: UseAutoModeContract
}

export type RouterConfigViewModel = {
  useAutoMode: UseAutoMode
  defaultStrategy: string
  defaultModel: string
  migrationNotice: string
}

export function mapRouterConfigForView(payload: RouterConfigApiPayload | null | undefined): RouterConfigViewModel {
  const contract = payload?.use_auto_mode_contract
  const useAutoMode = normalizeUseAutoMode(payload?.use_auto_mode, contract)
  const migrationNotice = typeof payload?.migration_notice === 'string' && payload.migration_notice.trim()
    ? payload.migration_notice
    : resolveUseAutoModeMigrationNotice(payload?.use_auto_mode, contract)

  return {
    useAutoMode,
    defaultStrategy: typeof payload?.default_strategy === 'string' && payload.default_strategy.trim()
      ? payload.default_strategy
      : USE_AUTO_MODE_AUTO,
    defaultModel: typeof payload?.default_model === 'string' ? payload.default_model : '',
    migrationNotice
  }
}
