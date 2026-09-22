const BASE_PATH = '/api/v1'
const TIMEOUT_MS = 15000

let onPersonalUnauthorized: (() => void) | null = null

// Called when a personal-scoped request comes back 401, so the session
// layer can drop its "connected" flag without waiting for the next poll.
export function setPersonalUnauthorizedHandler(handler: (() => void) | null) {
  onPersonalUnauthorized = handler
}

export class ApiError extends Error {
  status: number
  retryAfter?: number

  constructor(message: string, status: number, retryAfter?: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.retryAfter = retryAfter
  }
}

interface ErrorPayload {
  error?: string
}

function isErrorPayload(value: unknown): value is ErrorPayload {
  return typeof value === 'object' && value !== null && 'error' in value
}

async function request<T>(path: string, personal: boolean, body?: unknown): Promise<T> {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), TIMEOUT_MS)

  try {
    const response = await fetch(`${BASE_PATH}${path}`, {
      signal: controller.signal,
      ...(body === undefined ? {} : { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) }),
    })
    const data: unknown = await response.json().catch(() => null)

    if (!response.ok) {
      if (response.status === 401 && personal) {
        onPersonalUnauthorized?.()
      }
      const retryAfterHeader = response.headers.get('Retry-After')
      const retryAfter = retryAfterHeader ? Number(retryAfterHeader) : undefined
      const message = isErrorPayload(data) && typeof data.error === 'string' ? data.error : 'Не удалось выполнить запрос.'
      throw new ApiError(message, response.status, retryAfter)
    }

    return data as T
  } catch (error) {
    if (error instanceof ApiError) throw error
    if (error instanceof DOMException && error.name === 'AbortError') {
      throw new ApiError('Сервер не ответил вовремя. Повторите попытку.', 0)
    }
    if (error instanceof TypeError) {
      throw new ApiError('Нет соединения с приложением. Проверьте, запущен ли сервер.', 0)
    }
    throw error
  } finally {
    clearTimeout(timeout)
  }
}

// `personal` explicitly picks /api/v1/personal/... over /api/v1/... — passed
// by the caller (derived from render state) rather than a hidden module-level
// flag, so a scope switch can't race a page's own fetch effect.
export function apiGet<T>(path: string, personal = false): Promise<T> {
  return request<T>(`${personal ? '/personal' : ''}${path}`, personal)
}

export function apiPost<T>(path: string, body: unknown, personal = false): Promise<T> {
  return request<T>(`${personal ? '/personal' : ''}${path}`, personal, body)
}
