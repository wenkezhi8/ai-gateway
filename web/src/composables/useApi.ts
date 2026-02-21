import { ref, computed, type Ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { eventBus } from '@/utils/eventBus'

interface UseApiOptions<T> {
  immediate?: boolean
  silent?: boolean
  retryCount?: number
  retryDelay?: number
  dedupe?: boolean
  dedupeKey?: string
  onSuccess?: (data: T) => void
  onError?: (error: Error) => void
  successMessage?: string
  confirmMessage?: string
}

interface ApiState<T> {
  data: Ref<T | null>
  loading: Ref<boolean>
  error: Ref<Error | null>
  executed: Ref<boolean>
}

const pendingRequests = new Map<string, Promise<any>>()

function generateDedupeKey(key: string): string {
  return `api:${key}:${Date.now().toString(36)}`
}

export function useApiRequest<T = any>(
  apiFn: () => Promise<T>,
  options: UseApiOptions<T> = {}
): ApiState<T> & { execute: () => Promise<T | null>; reset: () => void } {
  const {
    immediate = false,
    silent = false,
    retryCount = 0,
    retryDelay = 1000,
    dedupe = false,
    dedupeKey,
    onSuccess,
    onError,
    successMessage,
    confirmMessage
  } = options

  const data = ref<T | null>(null) as Ref<T | null>
  const loading = ref(false)
  const error = ref<Error | null>(null)
  const executed = ref(false)

  const executeWithRetry = async (attempt: number = 0): Promise<T> => {
    try {
      const result = await apiFn()
      return result
    } catch (e) {
      if (attempt < retryCount) {
        await new Promise(resolve => setTimeout(resolve, retryDelay * (attempt + 1)))
        return executeWithRetry(attempt + 1)
      }
      throw e
    }
  }

  const execute = async (): Promise<T | null> => {
    if (confirmMessage) {
      try {
        await ElMessageBox.confirm(confirmMessage, '确认操作', {
          type: 'warning'
        })
      } catch {
        return null
      }
    }

    const key = dedupeKey || generateDedupeKey(apiFn.toString().slice(0, 50))
    
    if (dedupe && pendingRequests.has(key)) {
      return pendingRequests.get(key)!
    }

    loading.value = true
    error.value = null

    const promise = (async () => {
      try {
        const result = await executeWithRetry()
        data.value = result
        executed.value = true
        
        if (!silent && successMessage) {
          ElMessage.success(successMessage)
        }
        
        onSuccess?.(result)
        return result
      } catch (e: any) {
        const err = e instanceof Error ? e : new Error(e?.message || '请求失败')
        error.value = err
        
        if (!silent) {
          const message = err.message || e?.response?.data?.error?.message || '操作失败'
          ElMessage.error(message)
        }
        
        onError?.(err)
        return null
      } finally {
        loading.value = false
        if (dedupe) {
          pendingRequests.delete(key)
        }
      }
    })()

    if (dedupe) {
      pendingRequests.set(key, promise)
    }

    return promise
  }

  const reset = () => {
    data.value = null
    loading.value = false
    error.value = null
    executed.value = false
  }

  if (immediate) {
    execute()
  }

  return {
    data,
    loading,
    error,
    executed,
    execute,
    reset
  }
}

export function useListApi<T>(
  apiFn: () => Promise<{ success: boolean; data: T[] }>,
  options: Omit<UseApiOptions<{ success: boolean; data: T[] }>, 'successMessage'> = {}
) {
  const result = useApiRequest(apiFn, { ...options, silent: options.silent ?? true })
  
  const items = computed(() => (result.data.value as any)?.data || [])
  const isEmpty = computed(() => items.value.length === 0)
  
  const refresh = () => result.execute()
  
  return {
    ...result,
    items,
    isEmpty,
    refresh
  }
}

export function useCrudApi<T extends { id: string | number }>(
  api: {
    list: () => Promise<{ success: boolean; data: T[] }>
    create: (data: Partial<T>) => Promise<T>
    update: (id: string | number, data: Partial<T>) => Promise<T>
    delete: (id: string | number) => Promise<void>
  },
  dataEvent: string,
  options: { listSilent?: boolean } = {}
) {
  const items = ref<T[]>([]) as Ref<T[]>
  const loading = ref(false)
  const submitting = ref(false)
  const error = ref<Error | null>(null)

  const fetchList = async () => {
    loading.value = true
    error.value = null
    try {
      const res = await api.list()
      items.value = res.data || []
    } catch (e: any) {
      error.value = e
      if (!options.listSilent) {
        ElMessage.error(e?.message || '获取数据失败')
      }
    } finally {
      loading.value = false
    }
  }

  const create = async (data: Partial<T>): Promise<boolean> => {
    submitting.value = true
    try {
      await api.create(data)
      ElMessage.success('创建成功')
      await fetchList()
      eventBus.emit(dataEvent)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '创建失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const update = async (id: string | number, data: Partial<T>): Promise<boolean> => {
    submitting.value = true
    try {
      await api.update(id, data)
      ElMessage.success('更新成功')
      await fetchList()
      eventBus.emit(dataEvent)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '更新失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const remove = async (id: string | number): Promise<boolean> => {
    try {
      await ElMessageBox.confirm('确定要删除此项吗？', '确认删除', {
        type: 'warning'
      })
    } catch {
      return false
    }

    submitting.value = true
    try {
      await api.delete(id)
      ElMessage.success('删除成功')
      await fetchList()
      eventBus.emit(dataEvent)
      return true
    } catch (e: any) {
      ElMessage.error(e?.message || '删除失败')
      return false
    } finally {
      submitting.value = false
    }
  }

  const findById = (id: string | number): T | undefined => {
    return items.value.find(item => item.id === id)
  }

  return {
    items,
    loading,
    submitting,
    error,
    fetchList,
    create,
    update,
    remove,
    findById
  }
}

export function useSubmitWithLoading<T extends (...args: any[]) => Promise<any>>(
  fn: T,
  options: { successMessage?: string; errorMessage?: string } = {}
): { submit: T; loading: Ref<boolean> } {
  const loading = ref(false)

  const submit = (async (...args: Parameters<T>) => {
    if (loading.value) return null
    
    loading.value = true
    try {
      const result = await fn(...args)
      if (options.successMessage) {
        ElMessage.success(options.successMessage)
      }
      return result
    } catch (e: any) {
      if (options.errorMessage) {
        ElMessage.error(options.errorMessage)
      } else {
        ElMessage.error(e?.message || '操作失败')
      }
      return null
    } finally {
      loading.value = false
    }
  }) as T

  return { submit, loading }
}

const debounceTimers = new Map<string, ReturnType<typeof setTimeout>>()

export function useDebouncedRef<T>(value: T, delay: number = 300): Ref<T> {
  const debouncedValue = ref(value) as Ref<T>
  
  return computed({
    get: () => debouncedValue.value,
    set: (newValue: T) => {
      const key = `debounce_${debouncedValue.toString()}`
      
      if (debounceTimers.has(key)) {
        clearTimeout(debounceTimers.get(key)!)
      }
      
      debounceTimers.set(key, setTimeout(() => {
        debouncedValue.value = newValue
        debounceTimers.delete(key)
      }, delay))
    }
  })
}

export function useAutoRefresh(
  fetchFn: () => Promise<void>,
  interval: number = 30000,
  options: { immediate?: boolean; pauseOnHidden?: boolean } = {}
) {
  const { immediate = true, pauseOnHidden = true } = options
  const isRefreshing = ref(false)
  const isPaused = ref(false)
  let timer: number | null = null

  const refresh = async () => {
    if (isRefreshing.value || isPaused.value) return
    isRefreshing.value = true
    try {
      await fetchFn()
    } finally {
      isRefreshing.value = false
    }
  }

  const start = () => {
    if (timer) return
    if (immediate) refresh()
    timer = window.setInterval(refresh, interval)
  }

  const stop = () => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  const pause = () => {
    isPaused.value = true
  }

  const resume = () => {
    isPaused.value = false
  }

  if (pauseOnHidden && typeof document !== 'undefined') {
    document.addEventListener('visibilitychange', () => {
      if (document.hidden) {
        pause()
      } else {
        resume()
        refresh()
      }
    })
  }

  return {
    refresh,
    start,
    stop,
    pause,
    resume,
    isRefreshing,
    isPaused
  }
}
