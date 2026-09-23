import { ApiError, type HttpMethod } from '../api/client'
import config from '../config'
import { mockCategories, mockSuggestions, mockUser } from './fixtures'
import type { Category, CategoryInput, EntityId, NominationSuggestionRich, Suggestion } from '../types'

/**
 * A tiny in-memory stand-in for the backend, used when `debug.useMockApi` is
 * on. Saves persist for the life of the tab. There's no token here — the
 * real backend authenticates via an HttpOnly cookie the client never sees,
 * so this just tracks a `signedIn` boolean instead, flipped by whichever
 * flow is standing in for login (see `stores/session.ts`).
 *
 * Routes are matched against `config.endpoints`, not hardcoded strings, so
 * this stays in sync automatically if a path there gets corrected.
 */

const LATENCY_MS = 320
const wait = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

let signedIn = false
let suggestions: Suggestion[] = mockSuggestions.map((item) => ({ ...item }))
let categories: Category[] = mockCategories.map((item) => ({ ...item }))
let suggestionIdSeed = 1000
let categoryIdSeed = 1000

export function mockSignIn(): void {
  signedIn = true
}

export function mockSignOut(): void {
  signedIn = false
}

export interface MockRequest {
  method: HttpMethod
  body?: unknown
  query?: Record<string, string | number | boolean | null | undefined>
  auth?: boolean
}

function requireAuth(req: MockRequest): void {
  if (req.auth !== false && !signedIn) {
    throw new ApiError('Your session has expired. Sign in again.', { status: 401 })
  }
}

function escapeRegex(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function saveSuggestions(categoryId: EntityId, body: unknown): Suggestion[] {
  const incoming = (body as { suggestions?: { id?: EntityId; text?: string }[] } | undefined)?.suggestions ?? []
  const now = new Date().toISOString()

  const saved: Suggestion[] = incoming
    .map((entry) => ({
      id: entry.id ?? `sg_${suggestionIdSeed++}`,
      categoryId,
      text: (entry.text ?? '').trim(),
      updatedAt: now,
    }))
    .filter((entry) => entry.text.length > 0)

  suggestions = [...suggestions.filter((item) => String(item.categoryId) !== String(categoryId)), ...saved]
  return saved
}

/** Resolves the same shapes the real endpoints return. */
export async function mockRequest<T>(path: string, req: MockRequest): Promise<T> {
  await wait(LATENCY_MS)
  const route = `${req.method} ${path}`

  if (route === `GET ${config.endpoints.session}`) {
    requireAuth(req)
    return { user: mockUser } as T
  }

  if (route === `POST ${config.endpoints.logout}`) {
    mockSignOut()
    return null as T
  }

  if (route === `GET ${config.endpoints.adminCheck}`) {
    requireAuth(req)
    return { isAdmin: config.debug.forceAdmin } as T
  }

  if (route === `GET ${config.endpoints.categories}`) {
    requireAuth(req)
    return categories.map((item) => ({ ...item })) as T
  }

  if (route === `GET ${config.endpoints.mySuggestions}`) {
    requireAuth(req)
    return suggestions.map((item) => ({ ...item })) as T
  }

  if (route === `GET ${config.endpoints.adminViewSuggestions}`) {
    requireAuth(req)
    const rich: NominationSuggestionRich[] = suggestions.map((suggestion) => ({
      suggestion: { ...suggestion },
      user: mockUser,
    }))
    return { suggestions: rich } as T
  }

  if (route === `POST ${config.endpoints.adminUpsertCategory}`) {
    requireAuth(req)
    const payload = req.body as { id: EntityId | null; category: CategoryInput } | undefined
    if (!payload) throw new ApiError('Missing request body.', { status: 400 })

    if (payload.id !== null && payload.id !== undefined) {
      const idx = categories.findIndex((c) => String(c.id) === String(payload.id))
      if (idx === -1) throw new ApiError('Category not found.', { status: 404 })
      categories[idx] = { ...payload.category, id: payload.id }
    } else {
      categories = [...categories, { ...payload.category, id: `cat_${categoryIdSeed++}` }]
    }
    return null as T
  }

  if (route === `POST ${config.endpoints.adminDeleteCategory}`) {
    requireAuth(req)
    const payload = req.body as { id: EntityId } | undefined
    if (!payload) throw new ApiError('Missing request body.', { status: 400 })
    categories = categories.filter((c) => String(c.id) !== String(payload.id))
    return null as T
  }

  const saveMatch = new RegExp(`^PUT ${escapeRegex(config.endpoints.categories)}/([^/]+)/suggestions$`).exec(route)
  if (saveMatch) {
    requireAuth(req)
    // @ts-ignore
    const categoryId = decodeURIComponent(saveMatch[1])
    const category = categories.find((item) => String(item.id) === categoryId)
    if (!category) throw new ApiError('That category no longer exists.', { status: 404 })
    if (category.locked) throw new ApiError('Submissions for this category are closed.', { status: 409 })
    return saveSuggestions(category.id, req.body) as T
  }

  throw new ApiError(`No mock handler for ${route}.`, { status: 404 })
}

/** Handy from the console: `__resetMockData()` */
export function resetMockData(): void {
  suggestions = mockSuggestions.map((item) => ({ ...item }))
  categories = mockCategories.map((item) => ({ ...item }))
}
