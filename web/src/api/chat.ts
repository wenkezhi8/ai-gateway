/**
 * Chat API - Handles chat completion requests
 */
import type { ChatCompletionParams, CompletionMeta, StreamChunk } from '@/types/chat'
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
 * @param onComplete Callback when stream completes (new signature with CompletionMeta object)
 * @returns AbortController to cancel the request
 */
export function streamCompletion(
  params: ChatCompletionParams,
  onChunk: (chunk: StreamChunk) => void,
  onError?: (error: Error) => void,
  onComplete?: (meta: CompletionMeta) => void
): AbortController {
  const controller = new AbortController()
  const token = localStorage.getItem('token')
  const requestMode: 'stream' | 'non_stream' = params.stream !== false ? 'stream' : 'non_stream'

  // Non-streaming mode: use fetch and bridge to callbacks
  if (params.stream === false) {
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
            stream: false
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

        const cacheHitHeader = response.headers.get('x-local-cache-hit')
        const cacheHit = cacheHitHeader === null ? undefined : cacheHitHeader === '1'
        const cacheLayer = response.headers.get('x-cache-layer') || undefined

        const data: StreamChunk = await response.json()

        if (data.error) {
          throw new Error(data.error.message)
        }

        // Bridge non-streaming response to onChunk callback
        onChunk(data)

        onComplete?.({
          totalTokens: data.usage?.total_tokens,
          promptTokens: data.usage?.prompt_tokens,
          completionTokens: data.usage?.completion_tokens,
          cacheHit,
          cacheLayer,
          requestMode,
          reasoningEffortDowngraded: data.gateway_meta?.reasoning_effort_downgraded === true
        })
      } catch (error) {
        if (error instanceof Error && error.name === 'AbortError') {
          console.log('Non-stream request aborted')
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

  // Streaming mode: use SSE
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

      const cacheHitHeader = response.headers.get('x-local-cache-hit')
      const cacheHit = cacheHitHeader === null ? undefined : cacheHitHeader === '1'
      const cacheLayer = response.headers.get('x-cache-layer') || undefined

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('Response body is not readable')
      }

      const decoder = new TextDecoder()
      let buffer = ''
      let currentEvent = 'message'
      let lastUsage: { prompt_tokens?: number; completion_tokens?: number; total_tokens?: number } | undefined
      let reasoningEffortDowngraded = false

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

          if (!trimmedLine || trimmedLine === '') {
            continue
          }

          if (trimmedLine.startsWith('event:')) {
            currentEvent = trimmedLine.replace(/^event:\s*/, '') || 'message'
            continue
          }

          if (trimmedLine === 'data: [DONE]' || trimmedLine === 'data:[DONE]') {
            onComplete?.({
              totalTokens: lastUsage?.total_tokens,
              promptTokens: lastUsage?.prompt_tokens,
              completionTokens: lastUsage?.completion_tokens,
              cacheHit,
              cacheLayer,
              requestMode,
              reasoningEffortDowngraded
            })
            return
          }

          if (trimmedLine.startsWith('data:')) {
            try {
              const jsonStr = trimmedLine.replace(/^data:\s*/, '')
              if (currentEvent === 'error') {
                const errPayload = JSON.parse(jsonStr)
                const errMessage =
                  (typeof errPayload?.error === 'string' && errPayload.error) ||
                  errPayload?.error?.message ||
                  errPayload?.message ||
                  'Stream error'
                onError?.(new Error(errMessage))
                return
              }

              const chunk: StreamChunk = JSON.parse(jsonStr)

              if (chunk.error) {
                throw new Error(chunk.error.message)
              }

              if (chunk.usage) {
                lastUsage = chunk.usage
              }
              if (chunk.gateway_meta?.reasoning_effort_downgraded === true) {
                reasoningEffortDowngraded = true
              }

              onChunk(chunk)
            } catch (parseError) {
              console.warn('Failed to parse SSE chunk:', trimmedLine, parseError)
            }
            currentEvent = 'message'
          }
        }
      }

      onComplete?.({
        totalTokens: lastUsage?.total_tokens,
        promptTokens: lastUsage?.prompt_tokens,
        completionTokens: lastUsage?.completion_tokens,
        cacheHit,
        cacheLayer,
        requestMode,
        reasoningEffortDowngraded
      })
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
