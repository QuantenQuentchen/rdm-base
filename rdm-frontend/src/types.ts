/** Everything the backend hands us, in one place. */

/**
 * Category/suggestion ids may be numeric (Go int64) or string, depending on
 * the entity. Kept intentionally loose — always compare/key by `String(id)`
 * so the frontend doesn't care which one a given backend table actually uses.
 */
export type EntityId = string | number

export interface User {
  id: string
  displayName: string
  username?: string | null
  avatarUrl?: string | null
  roles?: string[]
}

/** The admin endpoints return the same shape under a different Go type name
 *  (`UserReply`). Kept as a separate alias so a future divergence doesn't
 *  silently break both call sites at once. */
export type UserReply = User

export interface Category {
  id: EntityId
  name: string
  description?: string
  /** One line of "what counts" guidance, shown when the panel is open. */
  criteria?: string
  /** Submissions closed — entries become read-only. */
  locked?: boolean
  /** Per-user cap for this category; overrides the app default. */
  maxSuggestions?: number
}

/** What the admin category form sends — no `id`, that's tracked separately
 *  by the upsert wrapper (`null` means "create"). */
export type CategoryInput = Omit<Category, 'id'>

export interface Suggestion {
  id: EntityId
  categoryId: EntityId
  text: string
  updatedAt?: string
}

/** A row in the editor. `id` is null until the backend assigns one. */
export interface SuggestionDraft {
  id: EntityId | null
  text: string
}

/* --- response envelopes ------------------------------------------------- */

export interface SessionResponse {
  user: User
}

export interface AdminStatusResponse {
  isAdmin: boolean
}

/** List endpoints may return a bare array or a wrapped object. */
export type ListPayload<T> = T[] | { data?: T[]; [key: string]: unknown }

/* --- admin --------------------------------------------------------------- */

export interface NominationSuggestionRich {
  suggestion: Suggestion
  user: UserReply
}

export interface ViewAllSuggestionsResponse {
  suggestions: NominationSuggestionRich[]
}

export interface CategoryUpsertRequest {
  /** null creates a new category; an id updates that one. */
  id: EntityId | null
  category: CategoryInput
}

export interface CategoryDeletionRequest {
  id: EntityId
}
