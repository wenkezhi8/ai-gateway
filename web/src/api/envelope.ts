export interface EnvelopeError {
  code?: string
  message?: string
  detail?: string
}

export interface EnvelopeResponse<T> {
  success: boolean
  data?: T
  error?: EnvelopeError
}

export interface UnwrapOptions {
  allowPlain?: boolean
  status?: number
}

export class ApiError extends Error {
  code: string
  status?: number
  detail?: string

  constructor(code: string, message: string, status?: number, detail?: string) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
    this.detail = detail
  }
}

function isEnvelope(raw: unknown): raw is EnvelopeResponse<unknown> {
  return Boolean(raw) && typeof raw === 'object' && Object.prototype.hasOwnProperty.call(raw, 'success')
}

export function unwrapEnvelope<T>(raw: unknown, options: UnwrapOptions = {}): T {
  if (isEnvelope(raw)) {
    if (raw.success) {
      return raw.data as T
    }

    const err = raw.error ?? {}
    throw new ApiError(
      err.code || 'api_error',
      err.message || 'API request failed',
      options.status,
      err.detail
    )
  }

  if (options.allowPlain) {
    return raw as T
  }

  throw new ApiError('invalid_envelope', 'Response is not a standard envelope', options.status)
}
