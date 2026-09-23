import { reactive, computed, readonly, type DeepReadonly } from 'vue'
import awardsApi from '../api/awards'
import type { Category, EntityId, ListPayload, Suggestion, SuggestionDraft } from '../types'

export type AwardsStatus = 'idle' | 'loading' | 'ready' | 'error'

interface AwardsState {
  status: AwardsStatus
  error: string | null
  categories: Category[]
  suggestionsByCategory: Record<string, Suggestion[]>
  saving: Record<string, boolean>
  saveErrors: Record<string, string>
}

const state = reactive<AwardsState>({
  status: 'idle',
  error: null,
  categories: [],
  suggestionsByCategory: {},
  saving: {},
  saveErrors: {},
})

/**
 * Category/suggestion ids may be numbers or strings depending on the
 * backend table. Always key internal maps by the string form so either
 * representation works without the store caring which one it got.
 */
function keyOf(id: EntityId): string {
  return String(id)
}

/** Endpoints may return a bare array or a wrapped object; accept both. */
function unwrap<T>(payload: ListPayload<T>, key: string): T[] {
  if (Array.isArray(payload)) return payload
  const record = payload as Record<string, unknown>
  if (Array.isArray(record[key])) return record[key] as T[]
  if (Array.isArray(record.data)) return record.data as T[]
  return []
}

function indexByCategory(list: Suggestion[]): Record<string, Suggestion[]> {
  const map: Record<string, Suggestion[]> = {}
  for (const suggestion of list) {
    if (suggestion.categoryId === undefined || suggestion.categoryId === null) continue
    const key = keyOf(suggestion.categoryId)
    ;(map[key] ??= []).push({ ...suggestion })
  }
  return map
}

/** Copy of `record` without `key` — avoids unused-rest-sibling lint noise. */
function omit<T>(record: Record<string, T>, key: string): Record<string, T> {
  const next = { ...record }
  delete next[key]
  return next
}

export const awards: DeepReadonly<AwardsState> = readonly(state)
export const categories = computed(() => state.categories)
export const isLoading = computed(() => state.status === 'loading')

export function suggestionsFor(categoryId: EntityId): Suggestion[] {
  return state.suggestionsByCategory[keyOf(categoryId)] ?? []
}

export function isSaving(categoryId: EntityId): boolean {
  return state.saving[keyOf(categoryId)] === true
}

export function saveErrorFor(categoryId: EntityId): string | null {
  return state.saveErrors[keyOf(categoryId)] ?? null
}

export const totalSuggestions = computed(() =>
  Object.values(state.suggestionsByCategory).reduce((sum, list) => sum + list.length, 0),
)

export const categoriesWithEntries = computed(
  () => state.categories.filter((category) => suggestionsFor(category.id).length > 0).length,
)

/** Load categories and the user's existing suggestions together. */
export async function loadAwards(options: { force?: boolean } = {}): Promise<void> {
  if (state.status === 'loading') return
  if (state.status === 'ready' && !options.force) return

  state.status = 'loading'
  state.error = null

  try {
    const [categoryPayload, suggestionPayload] = await Promise.all([
      awardsApi.listCategories(),
      awardsApi.listMySuggestions(),
    ])

    state.categories = unwrap<Category>(categoryPayload, 'categories')
    state.suggestionsByCategory = indexByCategory(unwrap<Suggestion>(suggestionPayload, 'suggestions'))
    state.status = 'ready'
  } catch (error) {
    state.error = (error as Error).message
    state.status = 'error'
  }
}

/** Persist the full set of entries for one category. Returns true when saved. */
export async function saveSuggestions(categoryId: EntityId, entries: SuggestionDraft[]): Promise<boolean> {
  const key = keyOf(categoryId)
  state.saving = { ...state.saving, [key]: true }
  state.saveErrors = omit(state.saveErrors, key)

  const cleaned = entries
    .map((entry) => ({ ...entry, text: entry.text.trim() }))
    .filter((entry) => entry.text.length > 0)

  try {
    const payload = await awardsApi.saveCategorySuggestions(categoryId, cleaned)
    const saved = unwrap<Suggestion>(payload, 'suggestions').map((item) => ({
      ...item,
      categoryId: item.categoryId ?? categoryId,
    }))

    // If the backend echoes nothing useful, keep what we sent so a
    // successful save doesn't blank the panel out.
    const fallback: Suggestion[] = cleaned.map((entry, index) => ({
      id: entry.id ?? `local-${key}-${index}`,
      categoryId,
      text: entry.text,
    }))

    state.suggestionsByCategory = {
      ...state.suggestionsByCategory,
      [key]: saved.length > 0 || cleaned.length === 0 ? saved : fallback,
    }
    return true
  } catch (error) {
    state.saveErrors = { ...state.saveErrors, [key]: (error as Error).message }
    return false
  } finally {
    state.saving = omit(state.saving, key)
  }
}

/** Drop everything on sign-out so the next user starts clean. */
export function resetAwards(): void {
  state.status = 'idle'
  state.error = null
  state.categories = []
  state.suggestionsByCategory = {}
  state.saving = {}
  state.saveErrors = {}
}
