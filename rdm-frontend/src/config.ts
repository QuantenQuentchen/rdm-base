/**
 * Brand strings, backend endpoint paths, and the debug switches.
 *
 * Debug flags resolve in this order:
 *   1. a localStorage override (set by the Debug pill in the top bar)
 *   2. a VITE_ env var
 *   3. the default — on in `npm run dev`, off in a production build
 */

import type { EntityId } from './types'

const DEBUG_STORAGE_KEY = 'rdm.debug'

export interface DebugFlags {
  /** Skip auth entirely and sign in as a fixture user. */
  bypassAuth: boolean
  /** Answer every API call from src/mock instead of the network. */
  useMockApi: boolean
  /** Make the fixture user an admin, so the admin switch shows up. */
  forceAdmin: boolean
}

function readOverrides(): Partial<DebugFlags> {
  try {
    const raw = window.localStorage.getItem(DEBUG_STORAGE_KEY)
    return raw ? (JSON.parse(raw) as Partial<DebugFlags>) : {}
  } catch {
    return {}
  }
}

function envFlag(value: unknown, fallback: boolean): boolean {
  if (value === undefined || value === '') return fallback
  return value === true || value === 'true' || value === '1'
}

const isDev = import.meta.env.DEV
const overrides = readOverrides()

export const debug: DebugFlags = {
  bypassAuth: overrides.bypassAuth ?? envFlag(import.meta.env.VITE_DEBUG_BYPASS_AUTH, isDev),
  useMockApi: overrides.useMockApi ?? envFlag(import.meta.env.VITE_DEBUG_MOCK_API, isDev),
  forceAdmin: overrides.forceAdmin ?? envFlag(import.meta.env.VITE_DEBUG_FORCE_ADMIN, isDev),
}

/** Persist a flag and reload, so stores re-bootstrap cleanly. */
export function setDebugFlag<K extends keyof DebugFlags>(key: K, value: DebugFlags[K]): void {
  const next = { ...readOverrides(), [key]: value }
  try {
    window.localStorage.setItem(DEBUG_STORAGE_KEY, JSON.stringify(next))
  } catch {
    /* ignore: storage unavailable */
  }
  window.location.reload()
}

export function clearDebugOverrides(): void {
  try {
    window.localStorage.removeItem(DEBUG_STORAGE_KEY)
  } catch {
    /* ignore */
  }
  window.location.reload()
}

/** True when any flag is making the app lie to you. */
export const isDebugActive = (): boolean => debug.bypassAuth || debug.useMockApi

/**
 * Every path the backend exposes, gathered in one place so a wrong guess is
 * a one-line fix instead of a grep. Paths marked ASSUMED are my best read of
 * the Go handlers you've shown me, not confirmed against your actual
 * mux/HandleFunc registration — check these against your router and adjust.
 */
export const endpoints = {
  login: '/api/auth/discord/login', // confirmed — full browser navigation, never fetched
  logout: '/api/auth/logout', // ASSUMED — mirrors /login; mount path wasn't shown
  session: '/api/auth/session', // NEW — doesn't exist server-side yet, see chat
  adminCheck: '/api/auth/admin', // ASSUMED — isAdmin's mount path wasn't shown
  categories: '/api/awards/categories', // ASSUMED — from earlier in the chat
  mySuggestions: '/api/awards/suggestions/mine', // ASSUMED — same
  categorySuggestions: (categoryId: EntityId) =>
    `/api/awards/categories/${encodeURIComponent(String(categoryId))}/suggestions`, // ASSUMED
  adminViewSuggestions: '/api/admin/viewSuggestions', // confirmed
  adminUpsertCategory: '/api/admin/upsert/Categories', // confirmed
  adminDeleteCategory: '/api/admin/delete/Categories', // confirmed
}

export const config = {
  api: {
    /**
     * Full origin, not a same-origin dev-proxy path — the backend runs on
     * its own port, so every call is a cross-origin `fetch` with
     * `credentials: 'include'` carrying the session cookie.
     */
    baseUrl: import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080',
  },

  brand: {
    name: 'Working Title Awards',
    shortName: 'Working',
    edition: 'MMXXVI',
    tagline: 'Honouring the year in play.',
  },

  suggestions: {
    maxPerCategory: 5,
    maxLength: 120,
  },

  debug,
  endpoints,
}

export default config
