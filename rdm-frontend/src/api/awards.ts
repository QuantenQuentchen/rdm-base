import api from './client'
import config from '../config'
import type { Category, EntityId, ListPayload, Suggestion, SuggestionDraft } from '../types'
import type { RequestOptions } from './client'

export const awardsApi = {
  listCategories(options?: RequestOptions) {
    return api.get<ListPayload<Category>>(config.endpoints.categories, options)
  },

  /** Every suggestion the signed-in user has made, across all categories. */
  listMySuggestions(options?: RequestOptions) {
    return api.get<ListPayload<Suggestion>>(config.endpoints.mySuggestions, options)
  },

  /**
   * Replace the user's full set of suggestions for one category. Existing
   * rows carry their id so the backend can keep them stable; new rows omit
   * it. `id !== null`, not truthiness — a numeric id of 0 is still a real id.
   */
  saveCategorySuggestions(categoryId: EntityId, entries: SuggestionDraft[], options?: RequestOptions) {
    return api.put<ListPayload<Suggestion>>(
      config.endpoints.categorySuggestions(categoryId),
      { suggestions: entries.map(({ id, text }) => (id !== null ? { id, text } : { text })) },
      options,
    )
  },
}

export default awardsApi
