import { describe, expect, it } from 'vitest'

import {
  CACHE_REQUEST_SOURCES,
  TRACE_ANSWER_SOURCE_FALLBACK,
  TRACE_ANSWER_SOURCE_LABELS,
  TRACE_ANSWER_SOURCES
} from './trace-answer-source'

describe('trace-answer-source constants', () => {
  it('should expose stable protocol values and fallback', () => {
    expect(TRACE_ANSWER_SOURCES).toEqual(['exact_raw', 'exact_prompt', 'semantic', 'v2', 'provider_chat'])
    expect(TRACE_ANSWER_SOURCE_FALLBACK).toBe('provider_chat')
  })

  it('should keep labels in sync with protocol values', () => {
    for (const source of TRACE_ANSWER_SOURCES) {
      expect(TRACE_ANSWER_SOURCE_LABELS[source]).toBeTruthy()
    }
    expect(Object.keys(TRACE_ANSWER_SOURCE_LABELS).sort()).toEqual([...TRACE_ANSWER_SOURCES].sort())
    expect(CACHE_REQUEST_SOURCES).toEqual(['all', ...TRACE_ANSWER_SOURCES])
  })
})
