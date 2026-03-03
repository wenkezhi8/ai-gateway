import type { ProviderOption } from '@/views/providers-accounts/provider-options'

/**
 * Provider selection state for the add/edit provider form
 */
export interface ProviderSelectState {
  id: string
  label: string
  color: string
  logoFile: File | null
  logoPreview: string
  idAutoFilled: boolean
}

/**
 * Auto-fill context to track whether ID was auto-filled
 */
export interface AutoFillContext {
  idAutoFilled: boolean
}

/**
 * Create a new auto-fill context
 */
export function createAutoFillContext(): AutoFillContext {
  return {
    idAutoFilled: false
  }
}

/**
 * Handle provider label change in the form
 *
 * Rules:
 * 1. If ID is empty, auto-fill with selected provider's ID
 * 2. Auto-fill color and logoPreview if available
 * 3. Mark ID as auto-filled for future reference
 * 4. If ID was manually edited (idAutoFilled=false), don't override when switching options
 * 5. If ID was auto-filled (idAutoFilled=true), allow re-filling when switching options
 */
export function handleProviderLabelChange(
  state: ProviderSelectState,
  options: ProviderOption[],
  selectedLabel: string,
  context: AutoFillContext
): void {
  const selectedOption = options.find(option => option.label === selectedLabel)
  if (!selectedOption) return

  // Auto-fill ID only if:
  // - ID is currently empty, OR
  // - ID was previously auto-filled (not manually edited)
  const shouldAutoFillId = !state.id || context.idAutoFilled

  if (shouldAutoFillId) {
    state.id = selectedOption.value
    context.idAutoFilled = true
  }

  // Always update color and logo preview when a standard provider is selected
  if (selectedOption.color) {
    state.color = selectedOption.color
  }

  // Update logo preview only if no custom logo file is uploaded
  if (selectedOption.logo && !state.logoFile) {
    state.logoPreview = selectedOption.logo
  }
}

/**
 * Mark ID as manually edited (user input)
 */
export function markIdAsManuallyEdited(context: AutoFillContext): void {
  context.idAutoFilled = false
}
