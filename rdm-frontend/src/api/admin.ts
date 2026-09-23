import api, { ApiError } from './client'
import config from '../config'
import type {
  AdminStatusResponse,
  Category,
  CategoryInput,
  CategoryUpsertRequest,
  CategoryDeletionRequest,
  EntityId,
  NominationSuggestionRich,
  ViewAllSuggestionsResponse,
} from '../types'

/**
 * Whether the current user is an admin. Never throws — a failed or
 * unreachable check just means "not an admin", which is the safe answer.
 * The backend is the actual gate; this only decides what the UI shows.
 */
export async function checkIsAdmin(): Promise<boolean> {
  if (config.debug.forceAdmin) return true
  try {
    const result = await api.get<AdminStatusResponse>(config.endpoints.adminCheck)
    console.log("Admin check result: ", result)
    console.log("typeof:", typeof result)
    console.log("stringified:", JSON.stringify(result))
    console.log("Admin status: ", result.isAdmin)
    return result?.isAdmin
  } catch (error) {
    console.error("Error during admin check: ", error)
    if (error instanceof ApiError) return false
    throw error
  }
}

export const adminApi = {
  /** Every suggestion across every user, with the submitter attached. */
  async listAllSuggestions(): Promise<NominationSuggestionRich[]> {
    const result = await api.get<ViewAllSuggestionsResponse | NominationSuggestionRich[]>(
      config.endpoints.adminViewSuggestions,
    )
    return Array.isArray(result) ? result : (result?.suggestions ?? [])
  },

  /** `id: null` creates a new category; a given id updates that one. */
  upsertCategory(id: EntityId | null, category: CategoryInput) {
    const body: CategoryUpsertRequest = { id, category }
    return api.post<void>(config.endpoints.adminUpsertCategory, body)
  },

  deleteCategory(id: EntityId) {
    const body: CategoryDeletionRequest = { id }
    return api.post<void>(config.endpoints.adminDeleteCategory, body)
  },
}

export default adminApi

export type { Category }
