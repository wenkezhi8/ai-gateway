import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

describe('model management settings loading', () => {
  it('keeps provider defaults visible when models API is unavailable', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/model-management/index.vue'), 'utf-8')

    expect(viewFile).toContain('await fetchModelLabels().catch(() => undefined)')
    expect(viewFile).toContain('getModelRegistry().catch(() => [])')
    expect(viewFile).toContain('...Object.keys(providerDefaults)')
    expect(viewFile).toContain('providerDefaults[providerId] || meta?.default_model || models[0] ||')
  })

  it('falls back to provider icon when logo image fails', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/model-management/index.vue'), 'utf-8')

    expect(viewFile).toContain('brokenLogoProviders')
    expect(viewFile).toContain('@error="handleLogoError(row.id)"')
    expect(viewFile).toContain('row.logo && !brokenLogoProviders.has(row.id)')
  })

  it('supports manual input and dropdown selection for provider label', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/model-management/index.vue'), 'utf-8')

    expect(viewFile).toContain('<el-form-item label="服务商名称" prop="label">')
    expect(viewFile).toContain('<el-select')
    expect(viewFile).toContain('allow-create')
    expect(viewFile).toContain('filterable')
    expect(viewFile).toContain('v-for="option in publicProviderOptions"')
    expect(viewFile).toContain('@change="handleProviderLabelChange"')
    expect(viewFile).toContain('publicProviderOptions.value = buildProviderOptions(')
  })

  it('allows deleting all providers and supports catalog re-add options', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/model-management/index.vue'), 'utf-8')

    expect(viewFile).toContain('@click="handleDeleteProvider(row)"')
    expect(viewFile).not.toContain('v-if="row.custom"')
    expect(viewFile).toContain('providerApi.delete(row.id)')
    expect(viewFile).toContain('buildProviderOptions')
    expect(viewFile).toContain('PROVIDER_CATALOG')
  })

  it('uses model-registry endpoints instead of deprecated router models endpoints', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/model-management/index.vue'), 'utf-8')

    expect(viewFile).toContain('getModelRegistry')
    expect(viewFile).toContain('upsertModelRegistry')
    expect(viewFile).toContain('deleteModelRegistry')
    expect(viewFile).not.toContain('/admin/router/models')
  })

  it('subscribes to MODELS_CHANGED and unsubscribes on unmount', () => {
    const viewFile = readFileSync(join(process.cwd(), 'src/views/model-management/index.vue'), 'utf-8')

    expect(viewFile).toContain('eventBus.on(DATA_EVENTS.MODELS_CHANGED')
    expect(viewFile).toContain('onUnmounted(() =>')
    expect(viewFile).toContain('offModelsChanged()')
  })
})
