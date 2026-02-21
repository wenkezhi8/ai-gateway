import { ref, reactive, computed, type Ref, type UnwrapNestedRefs } from 'vue'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'

interface UseFormOptions<T> {
  initialValues: T
  rules?: FormRules
  onSubmit: (values: T) => Promise<boolean | void>
  onSuccess?: () => void
  onError?: (error: Error) => void
  successMessage?: string
  errorMessage?: string
  confirmMessage?: string
  resetOnSuccess?: boolean
}

export function useForm<T extends Record<string, any>>(options: UseFormOptions<T>) {
  const {
    initialValues,
    rules,
    onSubmit,
    onSuccess,
    onError,
    successMessage = '保存成功',
    errorMessage = '保存失败',
    confirmMessage,
    resetOnSuccess = true
  } = options

  const formRef = ref<FormInstance>()
  const form = reactive({ ...initialValues }) as UnwrapNestedRefs<T>
  const originalValues = { ...initialValues }
  const submitting = ref(false)
  const hasChanges = computed(() => {
    return JSON.stringify(form) !== JSON.stringify(originalValues)
  })

  const validate = async (): Promise<boolean> => {
    if (!formRef.value) return true
    try {
      await formRef.value.validate()
      return true
    } catch {
      return false
    }
  }

  const handleSubmit = async (): Promise<boolean> => {
    const isValid = await validate()
    if (!isValid) return false

    if (submitting.value) return false
    
    if (confirmMessage) {
      try {
        const { ElMessageBox } = await import('element-plus')
        await ElMessageBox.confirm(confirmMessage, '确认', { type: 'warning' })
      } catch {
        return false
      }
    }

    submitting.value = true
    try {
      const result = await onSubmit({ ...form } as T)
      if (result !== false) {
        if (successMessage) ElMessage.success(successMessage)
        if (resetOnSuccess) reset()
        onSuccess?.()
        return true
      }
      return false
    } catch (e: any) {
      if (errorMessage) ElMessage.error(e?.message || errorMessage)
      onError?.(e)
      return false
    } finally {
      submitting.value = false
    }
  }

  const reset = () => {
    Object.assign(form, originalValues)
    formRef.value?.clearValidate()
  }

  const setValues = (values: Partial<T>) => {
    Object.assign(form, values)
  }

  const setInitialValues = (values: T) => {
    Object.assign(originalValues, values)
    Object.assign(form, values)
  }

  return {
    formRef,
    form,
    rules,
    submitting,
    hasChanges,
    validate,
    handleSubmit,
    reset,
    setValues,
    setInitialValues
  }
}

interface UseDialogFormOptions<T> extends UseFormOptions<T> {
  width?: string
  destroyOnClose?: boolean
}

export function useDialogForm<T extends Record<string, any>>(options: UseDialogFormOptions<T>) {
  const form = useForm(options)
  const visible = ref(false)
  const isEdit = ref(false)
  const editingId = ref<string | number | null>(null)

  const open = (data?: Partial<T> & { id?: string | number }) => {
    if (data) {
      isEdit.value = true
      editingId.value = data.id ?? null
      form.setInitialValues({ ...options.initialValues, ...data } as T)
    } else {
      isEdit.value = false
      editingId.value = null
      form.reset()
    }
    visible.value = true
  }

  const close = () => {
    visible.value = false
    form.reset()
  }

  const title = computed(() => isEdit.value ? '编辑' : '新增')

  return {
    ...form,
    visible,
    isEdit,
    editingId,
    title,
    open,
    close
  }
}

export function useTableSelection<T extends { id: string | number }>() {
  const selectedItems = ref<T[]>([]) as Ref<T[]>
  const selectedIds = computed(() => selectedItems.value.map(item => item.id))
  const isSelected = computed(() => selectedItems.value.length > 0)
  const isAllSelected = computed(() => selectedItems.value.length > 0)
  const selectionCount = computed(() => selectedItems.value.length)

  const handleSelectionChange = (items: T[]) => {
    selectedItems.value = items
  }

  const clearSelection = () => {
    selectedItems.value = []
  }

  const isSelectedById = (id: string | number): boolean => {
    return selectedIds.value.includes(id)
  }

  return {
    selectedItems,
    selectedIds,
    isSelected,
    isAllSelected,
    selectionCount,
    handleSelectionChange,
    clearSelection,
    isSelectedById
  }
}

export function usePagination(fetchFn: (page: number, pageSize: number) => Promise<void>) {
  const currentPage = ref(1)
  const pageSize = ref(10)
  const total = ref(0)
  const loading = ref(false)

  const totalPages = computed(() => Math.ceil(total.value / pageSize.value))

  const goToPage = async (page: number) => {
    currentPage.value = page
    loading.value = true
    try {
      await fetchFn(page, pageSize.value)
    } finally {
      loading.value = false
    }
  }

  const changePageSize = async (size: number) => {
    pageSize.value = size
    currentPage.value = 1
    loading.value = true
    try {
      await fetchFn(1, size)
    } finally {
      loading.value = false
    }
  }

  const refresh = async () => {
    await goToPage(currentPage.value)
  }

  return {
    currentPage,
    pageSize,
    total,
    loading,
    totalPages,
    goToPage,
    changePageSize,
    refresh
  }
}
