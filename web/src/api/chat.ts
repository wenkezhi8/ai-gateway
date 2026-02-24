/**
 * Chat API - Handles chat completion requests
 */
import type { ChatCompletionParams, StreamChunk } from '@/types/chat'
import { API } from '@/constants/api'

/**
 * Non-streaming chat completion
 */
export async function completion(params: ChatCompletionParams): Promise<StreamChunk> {
  const token = localStorage.getItem('token')
  const response = await fetch(API.V1.CHAT_COMPLETIONS, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    body: JSON.stringify({
      ...params,
      stream: false
    })
  })

  if (!response.ok) {
    let errorMsg = `HTTP ${response.status}`
    try {
      const errorData = await response.json()
      if (typeof errorData?.error === 'string') {
        errorMsg = errorData.error
      } else if (errorData?.error?.message) {
        errorMsg = errorData.error.message
      } else if (errorData?.message) {
        errorMsg = errorData.message
      } else if (errorData?.error?.code) {
        errorMsg = `${errorData.error.code}: ${errorData.error.message || 'Unknown error'}`
      }
    } catch {
      errorMsg = `HTTP ${response.status}: ${response.statusText}`
    }
    throw new Error(errorMsg)
  }

  return response.json()
}

/**
 * Streaming chat completion using SSE
 * @param params Chat completion parameters
 * @param onChunk Callback for each chunk received
 * @param onError Callback for errors
 * @param onComplete Callback when stream completes
 * @returns AbortController to cancel the request
 */
export function streamCompletion(
  params: ChatCompletionParams,
  onChunk: (chunk: StreamChunk) => void,
  onError?: (error: Error) => void,
  onComplete?: (totalTokens?: number, promptTokens?: number, completionTokens?: number) => void
): AbortController {
  const controller = new AbortController()
  const token = localStorage.getItem('token')

	;(async () => {
    try {
      const response = await fetch(API.V1.CHAT_COMPLETIONS, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {})
        },
        body: JSON.stringify({
          ...params,
          stream: true
        }),
        signal: controller.signal
      })

      if (!response.ok) {
        let errorMsg = `HTTP ${response.status}`
        try {
          const errorData = await response.json()
          if (typeof errorData?.error === 'string') {
            errorMsg = errorData.error
          } else if (errorData?.error?.message) {
            errorMsg = errorData.error.message
          } else if (errorData?.message) {
            errorMsg = errorData.message
          } else if (errorData?.error?.code) {
            errorMsg = `${errorData.error.code}: ${errorData.error.message || 'Unknown error'}`
          }
        } catch {
          errorMsg = `HTTP ${response.status}: ${response.statusText}`
        }
        throw new Error(errorMsg)
      }

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('Response body is not readable')
      }

      const decoder = new TextDecoder()
      let buffer = ''
      let lastUsage: { prompt_tokens?: number; completion_tokens?: number; total_tokens?: number } | undefined

      while (true) {
        const { done, value } = await reader.read()

        if (done) {
          break
        }

        buffer += decoder.decode(value, { stream: true })

        // Process complete SSE messages
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          const trimmedLine = line.trim()

          if (!trimmedLine || trimmedLine === '' || trimmedLine.startsWith('event:')) {
            continue
          }

          if (trimmedLine === 'data: [DONE]' || trimmedLine === 'data:[DONE]') {
            onComplete?.(lastUsage?.total_tokens, lastUsage?.prompt_tokens, lastUsage?.completion_tokens)
            return
          }

          if (trimmedLine.startsWith('data:')) {
            try {
              const jsonStr = trimmedLine.replace(/^data:\s*/, '')
              const chunk: StreamChunk = JSON.parse(jsonStr)

              if (chunk.error) {
                throw new Error(chunk.error.message)
              }

              if (chunk.usage) {
                lastUsage = chunk.usage
              }

              onChunk(chunk)
            } catch (parseError) {
              console.warn('Failed to parse SSE chunk:', trimmedLine, parseError)
            }
          }
        }
      }

      onComplete?.(lastUsage?.total_tokens, lastUsage?.prompt_tokens, lastUsage?.completion_tokens)
    } catch (error) {
      if (error instanceof Error && error.name === 'AbortError') {
        console.log('Stream request aborted')
        return
      }
      let errorMessage = 'Unknown error'
      if (error instanceof Error) {
        errorMessage = error.message
      } else if (typeof error === 'string') {
        errorMessage = error
      } else if (error && typeof error === 'object' && 'message' in error) {
        errorMessage = String((error as any).message)
      }
      onError?.(new Error(errorMessage))
    }
  })()

  return controller
}

/**
 * Chat API object
 */
export const chatApi = {
  completion,
  streamCompletion
}

export interface SearchResult {
  title: string
  link: string
  snippet: string
}

export interface SearchResponse {
  success: boolean
  data?: SearchResult[]
  error?: string
}

export async function search(query: string, limit: number = 5): Promise<SearchResponse> {
  const token = localStorage.getItem('token')
  const response = await fetch(API.V1.SEARCH, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {})
    },
    body: JSON.stringify({ query, limit })
  })

  if (!response.ok) {
    let errorMsg = `HTTP ${response.status}`
    try {
      const errorData = await response.json()
      if (typeof errorData?.error === 'string') {
        errorMsg = errorData.error
      } else if (errorData?.error?.message) {
        errorMsg = errorData.error.message
      } else if (errorData?.message) {
        errorMsg = errorData.message
      }
    } catch {
      errorMsg = `HTTP ${response.status}: ${response.statusText}`
    }
    return { success: false, error: errorMsg }
  }

  return response.json()
}

export default chatApi
