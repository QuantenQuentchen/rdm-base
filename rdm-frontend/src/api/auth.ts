import api from './client'
import config from '../config'
import type { SessionResponse } from '../types'

export const authApi = {
  /**
   * Not a fetch — a full browser navigation. Discord needs to redirect the
   * top-level window, and the backend sets the session cookie on its own
   * origin as part of that redirect chain, then sends the browser back to
   * the frontend once it's done. `window.location.assign(discordLoginUrl())`
   * is the entire client side of this.
   */
  discordLoginUrl(): string {
    return `${config.api.baseUrl}${config.endpoints.login}`
  },

  /** Ask the backend whether the session cookie (if any) is still valid. */
  getSession() {
    return api.get<SessionResponse>(config.endpoints.session)
  },

  /** Clears the cookie server-side — JS can't touch an HttpOnly cookie itself. */
  signOut() {
    return api.post<null>(config.endpoints.logout)
  },
}

export default authApi
