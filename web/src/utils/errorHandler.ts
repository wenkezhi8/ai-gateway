import { ElMessage } from 'element-plus'
import { useRouter } from 'vue-router'

const router = useRouter()

/**
 * 统一API错误处理
 * @param error 错误对象
 * @param defaultMessage 默认错误消息
 * @param options 处理选项
 */
export function handleApiError(
  error: any,
  defaultMessage = '操作失败',
  options: {
    showMessage?: boolean
    rethrow?: boolean
    onError?: (error: any) => void
  } = {}
) {
  const { showMessage = true, rethrow = false, onError } = options
  
  let message = defaultMessage
  
  if (error.response) {
    // HTTP错误
    switch (error.response.status) {
      case 400:
        message = error.response.data?.message || '请求参数错误'
        break
      case 401:
        message = '登录已过期，请重新登录'
        // 跳转到登录页
        router.push('/login')
        break
      case 403:
        message = '没有权限执行此操作'
        break
      case 404:
        message = '请求的资源不存在'
        break
      case 500:
        message = '服务器内部错误'
        break
      case 502:
      case 503:
      case 504:
        message = '服务暂时不可用，请稍后重试'
        break
      default:
        message = error.response.data?.message || `请求失败 (${error.response.status})`
    }
  } else if (error.request) {
    // 网络错误
    message = '网络连接失败，请检查网络设置'
  } else {
    // 其他错误
    message = error.message || defaultMessage
  }
  
  // 显示错误消息
  if (showMessage) {
    ElMessage.error(message)
  }
  
  // 执行自定义错误处理
  if (onError) {
    onError(error)
  }
  
  // 控制台记录错误
  console.error('API Error:', error)
  
  // 是否重新抛出错误
  if (rethrow) {
    throw error
  }
  
  return message
}

/**
 * 处理表单验证错误
 * @param error 验证错误
 */
export function handleValidationError(error: any) {
  if (error?.errors?.length > 0) {
    const messages = error.errors.map((err: any) => err.message).join(', ')
    ElMessage.warning(messages)
    return messages
  }
  return '表单验证失败'
}

/**
 * 处理操作成功
 * @param message 成功消息
 */
export function handleSuccess(message = '操作成功') {
  ElMessage.success(message)
}

/**
 * 处理操作警告
 * @param message 警告消息
 */
export function handleWarning(message: string) {
  ElMessage.warning(message)
}

/**
 * 处理操作信息
 * @param message 信息消息
 */
export function handleInfo(message: string) {
  ElMessage.info(message)
}
