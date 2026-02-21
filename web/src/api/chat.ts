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
    const errorData = await response.json().catch(() => ({}))
    const errorMsg = errorData?.error?.message || errorData?.message || `HTTP error ${response.status}`
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
        const errorData = await response.json().catch(() => ({}))
        const errorMsg = errorData?.error?.message || errorData?.message || `HTTP error ${response.status}`
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
      onError?.(error instanceof Error ? error : new Error(String(error)))
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

export default chatApi
