function parseJson(raw: string): any {
  try {
    return JSON.parse(raw)
  } catch {
    return null
  }
}

function decodeBase64Json(raw: string): any {
  try {
    const atobFn = typeof globalThis.atob === 'function'
      ? globalThis.atob.bind(globalThis)
      : (input: string) => Buffer.from(input, 'base64').toString('binary')

    const binary = atobFn(raw)
    const bytes = Uint8Array.from(binary, ch => ch.charCodeAt(0))
    const text = new TextDecoder().decode(bytes)
    return JSON.parse(text)
  } catch {
    return null
  }
}

function extractUserMessageFromPayload(payload: any): string {
  if (!payload || typeof payload !== 'object' || !Array.isArray(payload.messages)) {
    return ''
  }

  const userMsg = payload.messages.find((m: any) => m?.role === 'user')
  if (!userMsg?.content) return ''

  if (typeof userMsg.content === 'string') {
    return userMsg.content.trim()
  }

  try {
    return JSON.stringify(userMsg.content)
  } catch {
    return ''
  }
}

function extractAIResponseFromPayload(payload: any): string {
  if (!payload || typeof payload !== 'object') return ''
  if (!Array.isArray(payload.choices) || !payload.choices[0]) return ''

  const content = payload.choices[0]?.message?.content
  return typeof content === 'string' && content.trim() ? content : ''
}

function toDisplayString(value: any): string {
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

export function extractUserMessageFull(row: any): string {
  if (typeof row?.user_message === 'string' && row.user_message.trim()) {
    return row.user_message
  }

  const value = row?.value
  if (!value) return '-'

  const parsedValue = typeof value === 'string' ? parseJson(value) : value
  if (!parsedValue) return '-'

  const direct = extractUserMessageFromPayload(parsedValue)
  if (direct) return direct

  const nested = parsedValue.body ?? parsedValue.Body ?? parsedValue.request ?? parsedValue.Request
  if (!nested) return '-'

  const nestedValue = typeof nested === 'string' ? parseJson(nested) : nested
  const nestedMessage = extractUserMessageFromPayload(nestedValue)
  return nestedMessage || '-'
}

export function extractAIResponseFull(row: any): string {
  const value = row?.value
  if (!value) return '-'

  if (typeof value === 'object') {
    const direct = extractAIResponseFromPayload(value)
    if (direct) return direct

    const body = value.body ?? value.Body
    if (body) {
      let payload = typeof body === 'string' ? parseJson(body) : body
      if (!payload && typeof body === 'string') {
        payload = decodeBase64Json(body)
      }
      const fromBody = extractAIResponseFromPayload(payload)
      if (fromBody) return fromBody
    }

    const response = value.response ?? value.Response
    if (response) {
      let payload = typeof response === 'string' ? parseJson(response) : response
      if (!payload && typeof response === 'string') {
        payload = decodeBase64Json(response)
      }
      const fromResponse = extractAIResponseFromPayload(payload)
      if (fromResponse) return fromResponse
    }
  }

  if (typeof value === 'string') {
    const parsed = parseJson(value)
    const fromParsed = extractAIResponseFromPayload(parsed)
    if (fromParsed) return fromParsed
    return value
  }

  return toDisplayString(value)
}
