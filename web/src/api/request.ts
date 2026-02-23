import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'

// 创建axios实例
const service: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// 标记是否正在处理401
let isHandling401 = false

// 请求拦截器
service.interceptors.request.use(
  (config) => {
    // 在这里可以添加token等认证信息
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    console.error('Request error:', error)
    return Promise.reject(error)
  }
)

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse) => {
    const { data } = response
    return data
  },
  (error) => {
    const { response, config } = error

    // 检查是否为静默请求（不显示错误提示）
    const silent = config?.silent === true

    if (response) {
      switch (response.status) {
        case 401:
          if (!silent && !isHandling401) {
            isHandling401 = true
            ElMessage.error('登录已过期，请重新登录')
            // 清除token并跳转到登录页
            localStorage.removeItem('token')
            localStorage.removeItem('userInfo')
            // 延迟跳转，让用户看到提示
            setTimeout(() => {
              isHandling401 = false
              window.location.href = '/login'
            }, 1000)
          }
          break
        case 403:
          if (!silent) {
            ElMessage.error('没有权限访问')
          }
          break
        case 404:
          // 404 错误静默处理，不显示错误提示，让调用方优雅降级
          console.warn(`Resource not found: ${config?.url}`)
          break
        case 500:
          if (!silent) {
            ElMessage.error('服务器错误')
          }
          break
        default:
          if (!silent) {
            ElMessage.error(response.data?.message || '请求失败')
          }
      }
    } else {
      // 网络错误也静默处理，避免频繁提示
      console.error('Network error:', error.message)
    }
    return Promise.reject(error)
  }
)

// 扩展 AxiosRequestConfig 以支持 silent 选项
interface CustomAxiosRequestConfig extends AxiosRequestConfig {
  silent?: boolean
}

// 封装请求方法
export const request = {
  get<T>(url: string, config?: CustomAxiosRequestConfig): Promise<T> {
    return service.get(url, config)
  },
  post<T>(url: string, data?: any, config?: CustomAxiosRequestConfig): Promise<T> {
    return service.post(url, data, config)
  },
  put<T>(url: string, data?: any, config?: CustomAxiosRequestConfig): Promise<T> {
    return service.put(url, data, config)
  },
  delete<T>(url: string, config?: CustomAxiosRequestConfig): Promise<T> {
    return service.delete(url, config)
  }
}

export default service
