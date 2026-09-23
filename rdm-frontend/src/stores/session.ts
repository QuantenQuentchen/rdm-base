import { reactive, computed, readonly, type DeepReadonly } from 'vue'
import authApi from '../api/auth'
import { checkIsAdmin } from '../api/admin'
import { onUnauthorized, ApiError } from '../api/client'
import { debug } from '../config'
import type { User } from '../types'

export type SessionStatus = 'unknown' | 'restoring' | 'authenticated' | 'anonymous'

interface SessionState {
  status: SessionStatus
  user: User | null
  isAdmin: boolean
  error: string | null
}

/**
 * Plain `reactive()` state — no Pinia dependency, no client-held token. The
 * backend owns the session entirely via an HttpOnly cookie; this store just
 * mirrors what the backend last told us about it.
 */
const state = reactive<SessionState>({
  status: 'unknown',
  user: null,
  isAdmin: false,
  error: null,
})

// A rejected session (expired/missing cookie) means the session is over,
// wherever in the app it happened.
onUnauthorized(() => {
  if (debug.bypassAuth) return
  state.user = null
  state.isAdmin = false
  state.status = 'anonymous'
})

export const session: DeepReadonly<SessionState> = readonly(state)
export const currentUser = computed(() => state.user)
export const isSignedIn = computed(() => state.status === 'authenticated' && state.user !== null)
export const isAdmin = computed(() => state.isAdmin)
export const isResolvingSession = computed(() => state.status === 'unknown' || state.status === 'restoring')

/**
 * Ask the backend whether this user is an admin. Safe to call repeatedly —
 * views call it on mount so a revoked role takes effect on navigation.
 */
export async function refreshAdminStatus(): Promise<boolean> {
  state.isAdmin = state.user ? await checkIsAdmin() : false
  console.log("Admin status refreshed: ", state.isAdmin)
  return state.isAdmin
}

/** Called once at boot: ask the backend if the session cookie (if any) is valid. */
export async function restoreSession(): Promise<User | null> {
  if (debug.bypassAuth) {
    const { mockUser } = await import('../mock/fixtures')
    // If the mock API is also in play, it tracks its own "signed in" flag —
    // bypassing auth in the store doesn't automatically bypass it there too.
    if (debug.useMockApi) {
      const { mockSignIn } = await import('../mock/server')
      mockSignIn()
    }
    state.user = mockUser
    state.status = 'authenticated'
    await refreshAdminStatus()
    return state.user
  }

  state.status = 'restoring'
  try {
    const result = await authApi.getSession()
    state.user = result?.user ?? null
    state.status = state.user ? 'authenticated' : 'anonymous'
    console.log("Session restored: ", state.user)
    console.log("Session status: ", state.status)
    if (state.user) await refreshAdminStatus()
    return state.user
  } catch (error) {
    state.user = null
    state.isAdmin = false
    state.status = 'anonymous'
    // A missing/expired cookie (401) is the normal "not logged in yet" case
    // and isn't worth surfacing; a genuinely dead server is.
    state.error = error instanceof ApiError && error.isAuthError ? null : (error as Error).message
    return null
  }
}

/**
 * Real mode: a full-page navigation to the backend's login route — Discord
 * has to redirect the top-level browser window, this can't be a fetch.
 * Mock/bypass mode: nothing to navigate to, so sign in locally instead.
 */
export async function signInWithDiscord(): Promise<void> {
  state.error = null

  if (debug.useMockApi || debug.bypassAuth) {
    const { mockUser } = await import('../mock/fixtures')
    const { mockSignIn } = await import('../mock/server')
    mockSignIn()
    state.user = mockUser
    state.status = 'authenticated'
    await refreshAdminStatus()
    return
  }

  window.location.assign(authApi.discordLoginUrl())
}

export async function signOut(): Promise<void> {
  const wasSignedIn = state.status === 'authenticated'
  state.user = null
  state.isAdmin = false
  state.status = 'anonymous'

  if (!wasSignedIn) return

  if (debug.useMockApi) {
    const { mockSignOut } = await import('../mock/server')
    mockSignOut()
  } else if (!debug.bypassAuth) {
    // Best effort: local state is already cleared either way.
    await authApi.signOut().catch(() => undefined)
  }
}

export function clearSessionError(): void {
  state.error = null
}
