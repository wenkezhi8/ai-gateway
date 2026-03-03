/**
 * Chat API Tests
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { streamCompletion } from './chat'
import type { StreamChunk } from '@/types/chat'

// Mock fetch
const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

// Mock localStorage
const mockLocalStorage = {
  getItem: vi.fn(() => 'test-token'),
  setItem: vi.fn(),
  removeItem: vi.fn()
}
vi.stubGlobal('localStorage', mockLocalStorage)

describe('streamCompletion', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('stream parameter handling', () => {
    it('should send stream request when stream=true', async () => {
      const mockResponse = new Response('', {
        headers: { 'x-local-cache-hit': '0' }
      })
      mockFetch.mockResolvedValueOnce(mockResponse)
      
      const onChunk = vi.fn()
      const onError = vi.fn()
      const onComplete = vi.fn()
      
      streamCompletion(
        { model: 'test', messages: [{ role: 'user', content: 'hi' }], stream: true },
        onChunk,
        onError,
        onComplete
      )
      
      // Wait for async execution
      await new Promise(resolve => setTimeout(resolve, 10))
      
      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          body: expect.stringContaining('"stream":true')
        })
      )
    })

    it('should send non-stream request when stream=false', async () => {
      const mockChunk: StreamChunk = {
        choices: [{ index: 0, delta: { content: 'Hello' }, finish_reason: null }],
        usage: { total_tokens: 10, prompt_tokens: 5, completion_tokens: 5 }
      }
      const mockResponse = new Response(JSON.stringify(mockChunk), {
        headers: { 
          'x-local-cache-hit': '1',
          'x-cache-layer': 'exact'
        }
      })
      mockFetch.mockResolvedValueOnce(mockResponse)
      
      const onChunk = vi.fn()
      const onError = vi.fn()
      const onComplete = vi.fn()
      
      streamCompletion(
        { model: 'test', messages: [{ role: 'user', content: 'hi' }], stream: false },
        onChunk,
        onError,
        onComplete
      )
      
      // Wait for async execution
      await new Promise(resolve => setTimeout(resolve, 10))
      
      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          body: expect.stringContaining('"stream":false')
        })
      )
    })

    it('should default to stream when stream is undefined', async () => {
      const mockResponse = new Response('', {
        headers: { 'x-local-cache-hit': '0' }
      })
      mockFetch.mockResolvedValueOnce(mockResponse)
      
      const onChunk = vi.fn()
      const onError = vi.fn()
      const onComplete = vi.fn()
      
      streamCompletion(
        { model: 'test', messages: [{ role: 'user', content: 'hi' }] },
        onChunk,
        onError,
        onComplete
      )
      
      // Wait for async execution
      await new Promise(resolve => setTimeout(resolve, 10))
      
      expect(mockFetch).toHaveBeenCalledWith(
        expect.any(String),
        expect.objectContaining({
          body: expect.stringContaining('"stream":true')
        })
      )
    })
  })

  describe('cache header parsing', () => {
    it('should parse cache headers into completion meta', async () => {
      const mockChunk: StreamChunk = {
        choices: [{ index: 0, delta: { content: 'Hello' }, finish_reason: null }],
        usage: { total_tokens: 10, prompt_tokens: 5, completion_tokens: 5 }
      }
      const mockResponse = new Response(JSON.stringify(mockChunk), {
        headers: { 
          'x-local-cache-hit': '1',
          'x-cache-layer': 'semantic'
        }
      })
      mockFetch.mockResolvedValueOnce(mockResponse)
      
      const onChunk = vi.fn()
      const onError = vi.fn()
      const onComplete = vi.fn()
      
      streamCompletion(
        { model: 'test', messages: [{ role: 'user', content: 'hi' }], stream: false },
        onChunk,
        onError,
        onComplete
      )
      
      // Wait for async execution
      await new Promise(resolve => setTimeout(resolve, 10))
      
      expect(onComplete).toHaveBeenCalledWith(
        expect.objectContaining({
          cacheHit: true,
          cacheLayer: 'semantic',
          requestMode: 'non_stream',
          totalTokens: 10,
          promptTokens: 5,
          completionTokens: 5
        })
      )
    })

    it('should mark cache status as unknown when header missing', async () => {
      const mockChunk: StreamChunk = {
        choices: [{ index: 0, delta: { content: 'Hello' }, finish_reason: null }],
        usage: { total_tokens: 10 }
      }
      const mockResponse = new Response(JSON.stringify(mockChunk), {
        headers: {} // no cache headers
      })
      mockFetch.mockResolvedValueOnce(mockResponse)
      
      const onChunk = vi.fn()
      const onError = vi.fn()
      const onComplete = vi.fn()
      
      streamCompletion(
        { model: 'test', messages: [{ role: 'user', content: 'hi' }], stream: false },
        onChunk,
        onError,
        onComplete
      )
      
      // Wait for async execution
      await new Promise(resolve => setTimeout(resolve, 10))
      
      expect(onComplete).toHaveBeenCalledWith(
        expect.objectContaining({
          cacheHit: undefined,
          cacheLayer: undefined
        })
      )
    })

    it('should set cacheHit to false when header is 0', async () => {
      const mockChunk: StreamChunk = {
        choices: [{ index: 0, delta: { content: 'Hello' }, finish_reason: null }],
        usage: { total_tokens: 10 }
      }
      const mockResponse = new Response(JSON.stringify(mockChunk), {
        headers: { 
          'x-local-cache-hit': '0',
          'x-cache-layer': 'none'
        }
      })
      mockFetch.mockResolvedValueOnce(mockResponse)
      
      const onChunk = vi.fn()
      const onError = vi.fn()
      const onComplete = vi.fn()
      
      streamCompletion(
        { model: 'test', messages: [{ role: 'user', content: 'hi' }], stream: false },
        onChunk,
        onError,
        onComplete
      )
      
      // Wait for async execution
      await new Promise(resolve => setTimeout(resolve, 10))
      
      expect(onComplete).toHaveBeenCalledWith(
        expect.objectContaining({
          cacheHit: false,
          cacheLayer: 'none'
        })
      )
    })
  })
})
