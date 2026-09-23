import config from '../config'

export type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface RequestOptions {
  method?: HttpMethod
  body?: unknown
  query?: Record<string, string | number | boolean | null | undefined>
  /** Whether this call requires a valid session. Default: true. */
  auth?: boolean
  signal?: AbortSignal
}

export class ApiError extends Error {
  readonly status: number
  readonly code: string | null
  readonly details: unknown

  constructor(message: string, init: { status?: number; code?: string | null; details?: unknown } = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = init.status ?? 0
    this.code = init.code ?? null
    this.details = init.details ?? null
  }

  get isAuthError(): boolean {
    return this.status === 401 || this.status === 403
  }
}

/** Called whenever the backend rejects the session (expired/missing cookie). */
let unauthorizedHandler: (error: ApiError) => void = () => {}

export function onUnauthorized(fn: (error: ApiError) => void): void {
  unauthorizedHandler = fn
}

function buildUrl(path: string, query?: RequestOptions['query']): string {
  const base = config.api.baseUrl.replace(/\/$/, '')
  const url = new URL(`${base}${path}`, window.location.origin)
  for (const [key, value] of Object.entries(query ?? {})) {
    if (value !== undefined && value !== null && value !== '') {
      url.searchParams.set(key, String(value))
    }
  }
  return url.toString()
}

async function parseBody(response: Response): Promise<unknown> {
  if (response.status === 204) return null
  const type = response.headers.get('content-type') ?? ''
  if (type.includes('application/json')) return response.json().catch(() => null)
  return response.text().catch(() => null)
}

/**
 * Go's `http.Error(w, "message", status)` writes a plain-text body, not
 * JSON — which is what this backend uses everywhere. Handle both shapes so
 * the actual backend message ("not eligible for login", "invalid state", …)
 * surfaces instead of a generic "Request failed (403)".
 */
function messageFrom(payload: unknown, status: number): string {
  if (typeof payload === 'string' && payload.trim().length > 0) return payload.trim()
  if (payload && typeof payload === 'object') {
    const record = payload as Record<string, unknown>
    const message = record.message ?? record.error
    if (typeof message === 'string') return message
  }
  return `Request failed (${status}).`
}

export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, query, auth = true, signal } = options

  // Debug mode: never touch the network. Dynamically imported so the
  // fixtures are code-split out of a production build.
  if (config.debug.useMockApi) {
    const { mockRequest } = await import('../mock/server')
    return mockRequest<T>(path, { method, body, query, auth })
  }

  const headers: Record<string, string> = { Accept: 'application/json' }
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  let response: Response
  try {
    response = await fetch(buildUrl(path, query), {
      method,
      headers,
      signal,
      // The backend authenticates via an HttpOnly session cookie, not a
      // header we attach ourselves — this is what makes the browser send
      // and accept it on a cross-origin (different port) request.
      credentials: 'include',
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === 'AbortError') throw cause
    throw new ApiError("Can't reach the server. Check your connection and try again.", { status: 0 })
  }

  const payload = await parseBody(response)

  if (!response.ok) {
    const record = (payload ?? null) as Record<string, unknown> | null
    const error = new ApiError(messageFrom(payload, response.status), {
      status: response.status,
      code: typeof record?.code === 'string' ? record.code : null,
      details: payload,
    })
    if (error.isAuthError && auth) unauthorizedHandler(error)
    throw error
  }

  return payload as T
}

export const api = {
  get: <T>(path: string, options?: RequestOptions) => request<T>(path, { ...options, method: 'GET' }),
  post: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: 'POST', body }),
  put: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: 'PUT', body }),
  patch: <T>(path: string, body?: unknown, options?: RequestOptions) =>
    request<T>(path, { ...options, method: 'PATCH', body }),
  del: <T>(path: string, options?: RequestOptions) => request<T>(path, { ...options, method: 'DELETE' }),
}

export default api
