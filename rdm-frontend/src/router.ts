import { shallowRef, computed, markRaw, type Component } from 'vue'
import AwardsView from './views/AwardsView.vue'
import AdminView from './views/AdminView.vue'

/**
 * A small hash router — the project needs nothing beyond Vue itself. The
 * shape (routes array, `route`, `navigate`) mirrors vue-router closely
 * enough that dropping the real one in later is this file plus the
 * `<component :is>` in AppShell.vue.
 *
 * There's no `/auth/callback` route here: the backend handles the entire
 * Discord round trip itself (redirect to Discord, receive the callback, set
 * the cookie, redirect back to the frontend's root). The frontend never
 * sees an OAuth code or state param — it just calls `GET /auth/session` on
 * boot and finds out it's logged in.
 */
export interface AppRoute {
  path: string
  name: string
  component: Component
  label?: string
  hint?: string
  requiresAuth?: boolean
  requiresAdmin?: boolean
  hidden?: boolean
}

export interface RouteLocation {
  path: string
  query: Record<string, string>
}

export const DEFAULT_PATH = '/awards'

export const routes: AppRoute[] = [
  {
    path: '/awards',
    name: 'awards',
    label: 'Awards',
    hint: 'Categories and your suggestions',
    component: markRaw(AwardsView),
    requiresAuth: true,
  },
  {
    path: '/admin',
    name: 'admin',
    label: 'Admin',
    component: markRaw(AdminView),
    requiresAuth: true,
    requiresAdmin: true,
    hidden: true, // reached from the top bar's admin switch, not the rail
  },
]

export const navRoutes = routes.filter((entry) => !entry.hidden)

function parseLocation(): RouteLocation {
  const raw = window.location.hash.replace(/^#/, '') || DEFAULT_PATH
  const [path, queryString = ''] = raw.split('?')
  return {
    path: path || DEFAULT_PATH,
    query: Object.fromEntries(new URLSearchParams(queryString)),
  }
}

const location = shallowRef<RouteLocation>(parseLocation())
window.addEventListener('hashchange', () => {
  location.value = parseLocation()
})

export const route = computed(() => location.value)

// @ts-ignore
export const matched = computed<AppRoute>(
  () => routes.find((entry) => entry.path === location.value.path) ?? routes[0],
)

export function navigate(path: string, query: Record<string, string> = {}): void {
  const queryString = new URLSearchParams(query).toString()
  const next = `#${path}${queryString ? `?${queryString}` : ''}`
  if (window.location.hash === next) {
    location.value = parseLocation()
    return
  }
  window.location.hash = next
}

export function isActive(path: string): boolean {
  return location.value.path === path
}
