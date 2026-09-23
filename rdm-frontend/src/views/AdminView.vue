<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { refreshAdminStatus, currentUser } from '../stores/session'
import { navigate, DEFAULT_PATH } from '../router'
import { adminApi } from '../api/admin'
import { awardsApi } from '../api/awards'
import StatusNote from '../components/StatusNote.vue'
import UiButton from '../components/UiButton.vue'
import type { Category, CategoryInput, EntityId, NominationSuggestionRich } from '../types'

type FetchStatus = 'idle' | 'loading' | 'ready' | 'error'

const checking = ref(true)
const allowed = ref(false)
const tab = ref<'suggestions' | 'categories'>('suggestions')

/* --- suggestions (read-only) --------------------------------------------- */

const suggestionsStatus = ref<FetchStatus>('idle')
const suggestionsError = ref<string | null>(null)
const suggestions = ref<NominationSuggestionRich[]>([])

async function loadSuggestions(): Promise<void> {
  suggestionsStatus.value = 'loading'
  suggestionsError.value = null
  try {
    suggestions.value = await adminApi.listAllSuggestions()
    suggestionsStatus.value = 'ready'
  } catch (error) {
    suggestionsError.value = (error as Error).message
    suggestionsStatus.value = 'error'
  }
}

/* --- categories (CRUD) ---------------------------------------------------- */

const categoriesStatus = ref<FetchStatus>('idle')
const categoriesError = ref<string | null>(null)
const categories = ref<Category[]>([])

async function loadCategories(): Promise<void> {
  categoriesStatus.value = 'loading'
  categoriesError.value = null
  try {
    const payload = await awardsApi.listCategories()
    categories.value = Array.isArray(payload)
      ? payload
      : ((payload as { categories?: Category[] })?.categories ?? [])
    categoriesStatus.value = 'ready'
  } catch (error) {
    categoriesError.value = (error as Error).message
    categoriesStatus.value = 'error'
  }
}

function emptyDraft(): CategoryInput {
  return { name: '', description: '', criteria: '', locked: false, maxSuggestions: undefined }
}

const formOpen = ref(false)
const editingId = ref<EntityId | null>(null)
const draft = reactive<CategoryInput>(emptyDraft())
const formBusy = ref(false)
const formError = ref<string | null>(null)

function openCreate(): void {
  editingId.value = null
  Object.assign(draft, emptyDraft())
  formError.value = null
  formOpen.value = true
}

function openEdit(category: Category): void {
  editingId.value = category.id
  const { id: _id, ...rest } = category
  Object.assign(draft, emptyDraft(), rest)
  formError.value = null
  formOpen.value = true
}

function closeForm(): void {
  formOpen.value = false
}

async function submitForm(): Promise<void> {
  if (!draft.name.trim()) {
    formError.value = 'Name is required.'
    return
  }
  formBusy.value = true
  formError.value = null
  try {
    await adminApi.upsertCategory(editingId.value, { ...draft, name: draft.name.trim() })
    formOpen.value = false
    await loadCategories()
  } catch (error) {
    formError.value = (error as Error).message
  } finally {
    formBusy.value = false
  }
}

const deletingId = ref<EntityId | null>(null)

async function removeCategory(category: Category): Promise<void> {
  if (!window.confirm(`Delete "${category.name}"? This cannot be undone.`)) return
  deletingId.value = category.id
  try {
    await adminApi.deleteCategory(category.id)
    await loadCategories()
  } catch (error) {
    categoriesError.value = (error as Error).message
  } finally {
    deletingId.value = null
  }
}

function formatDate(value?: string): string {
  if (!value) return '—'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

onMounted(async () => {
  allowed.value = await refreshAdminStatus()
  checking.value = false
  if (allowed.value) {
    await Promise.all([loadSuggestions(), loadCategories()])
  }
})
</script>

<template>
  <div class="admin">
    <p v-if="checking" class="checking">Checking permissions…</p>

    <StatusNote
      v-else-if="!allowed"
      tone="error"
      title="Not available"
      message="This area is limited to administrators."
    >
      <div class="note__actions">
        <UiButton variant="ghost" size="sm" @click="navigate(DEFAULT_PATH)">Back to awards</UiButton>
      </div>
    </StatusNote>

    <template v-else>
      <header class="head">
        <div>
          <h1 class="title">Admin</h1>
          <p class="meta">Signed in as {{ currentUser?.displayName }}</p>
        </div>
      </header>

      <nav class="tabs" aria-label="Admin sections">
        <button
          type="button"
          class="tab"
          :class="{ 'tab--active': tab === 'suggestions' }"
          @click="tab = 'suggestions'"
        >
          Suggestions
        </button>
        <button
          type="button"
          class="tab"
          :class="{ 'tab--active': tab === 'categories' }"
          @click="tab = 'categories'"
        >
          Categories
        </button>
      </nav>

      <!-- ---------------------------------------------------------------- -->
      <section v-if="tab === 'suggestions'" class="panel">
        <div class="panel__head">
          <h2 class="panel__title">All suggestions</h2>
          <UiButton variant="ghost" size="sm" :disabled="suggestionsStatus === 'loading'" @click="loadSuggestions">
            Refresh
          </UiButton>
        </div>

        <StatusNote v-if="suggestionsStatus === 'error'" tone="error" title="Couldn't load suggestions" :message="suggestionsError" />
        <p v-else-if="suggestionsStatus === 'loading'" class="loading">Loading…</p>
        <p v-else-if="suggestions.length === 0" class="empty">No suggestions submitted yet.</p>

        <table v-else class="table">
          <thead>
            <tr>
              <th>Category</th>
              <th>Suggestion</th>
              <th>Submitted by</th>
              <th>Updated</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, index) in suggestions" :key="`${row.suggestion.id}-${index}`">
              <td class="mono">{{ row.suggestion.categoryId }}</td>
              <td>{{ row.suggestion.text }}</td>
              <td>
                <span class="user">
                  <img v-if="row.user.avatarUrl" class="user__avatar" :src="row.user.avatarUrl" alt="" />
                  <span class="user__name">{{ row.user.displayName || row.user.username }}</span>
                </span>
              </td>
              <td class="mono">{{ formatDate(row.suggestion.updatedAt) }}</td>
            </tr>
          </tbody>
        </table>
      </section>

      <!-- ---------------------------------------------------------------- -->
      <section v-else class="panel">
        <div class="panel__head">
          <h2 class="panel__title">Categories</h2>
          <div class="panel__actions">
            <UiButton variant="ghost" size="sm" :disabled="categoriesStatus === 'loading'" @click="loadCategories">
              Refresh
            </UiButton>
            <UiButton variant="gold" size="sm" @click="openCreate">New category</UiButton>
          </div>
        </div>

        <StatusNote v-if="categoriesStatus === 'error'" tone="error" title="Couldn't load categories" :message="categoriesError" />
        <p v-else-if="categoriesStatus === 'loading'" class="loading">Loading…</p>
        <p v-else-if="categories.length === 0" class="empty">No categories yet.</p>

        <table v-else class="table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Id</th>
              <th>Locked</th>
              <th>Max</th>
              <th class="table__actions-head">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="category in categories" :key="String(category.id)">
              <td>
                <div class="category-name">{{ category.name }}</div>
                <div v-if="category.description" class="category-desc">{{ category.description }}</div>
              </td>
              <td class="mono">{{ category.id }}</td>
              <td>
                <span class="badge" :class="{ 'badge--on': category.locked }">
                  {{ category.locked ? 'Locked' : 'Open' }}
                </span>
              </td>
              <td class="mono">{{ category.maxSuggestions ?? '—' }}</td>
              <td class="table__actions">
                <UiButton variant="quiet" size="sm" @click="openEdit(category)">Edit</UiButton>
                <UiButton
                  variant="danger"
                  size="sm"
                  :loading="deletingId === category.id"
                  @click="removeCategory(category)"
                >
                  Delete
                </UiButton>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-if="formOpen" class="form-panel">
          <h3 class="form-panel__title">{{ editingId !== null ? 'Edit category' : 'New category' }}</h3>

          <div class="form-grid">
            <label class="form-field">
              <span class="form-label">Name</span>
              <input v-model="draft.name" class="field" type="text" placeholder="Game of the Year" />
            </label>

            <label class="form-field">
              <span class="form-label">Max suggestions</span>
              <input v-model.number="draft.maxSuggestions" class="field" type="number" min="1" placeholder="5" />
            </label>

            <label class="form-field form-field--wide">
              <span class="form-label">Description</span>
              <input v-model="draft.description" class="field" type="text" placeholder="Shown on the category plaque" />
            </label>

            <label class="form-field form-field--wide">
              <span class="form-label">Criteria</span>
              <input v-model="draft.criteria" class="field" type="text" placeholder="What counts as eligible" />
            </label>

            <label class="form-field form-field--checkbox">
              <input v-model="draft.locked" type="checkbox" />
              <span class="form-label">Locked (submissions closed)</span>
            </label>
          </div>

          <p v-if="formError" class="form-error" role="alert">{{ formError }}</p>

          <div class="form-actions">
            <UiButton variant="quiet" size="sm" :disabled="formBusy" @click="closeForm">Cancel</UiButton>
            <UiButton variant="gold" size="sm" :loading="formBusy" @click="submitForm">
              {{ editingId !== null ? 'Save changes' : 'Create category' }}
            </UiButton>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
/* Deliberately plain and utilitarian — a tool, not the show. No champagne. */

.admin {
  max-width: 920px;
  margin: 0 auto;
}

.checking {
  color: var(--muted);
}

.head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 1.25rem;
}

.title {
  margin: 0 0 0.25rem;
  font-family: var(--body);
  font-size: 1.3rem;
  font-weight: 600;
  letter-spacing: -0.01em;
}

.meta {
  margin: 0;
  color: var(--faint);
  font-size: 0.85rem;
}

.tabs {
  display: flex;
  gap: 0.25rem;
  margin-bottom: 1.5rem;
  border-bottom: 1px solid var(--hairline);
}

.tab {
  padding: 0.6rem 0.9rem;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  color: var(--muted);
  font-family: var(--body);
  font-size: 0.9rem;
  cursor: pointer;
}

.tab:hover {
  color: var(--platinum);
}

.tab--active {
  color: var(--platinum);
  border-bottom-color: var(--spotlight);
}

.panel {
  display: grid;
  gap: 1rem;
}

.panel__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.panel__title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
}

.panel__actions {
  display: flex;
  gap: 0.5rem;
}

.loading,
.empty {
  margin: 0;
  padding: 1.5rem 0;
  color: var(--faint);
  font-size: 0.9rem;
  text-align: center;
}

.table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.table th {
  text-align: left;
  padding: 0.55rem 0.7rem;
  font-size: 0.72rem;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--faint);
  border-bottom: 1px solid var(--hairline-strong);
}

.table td {
  padding: 0.65rem 0.7rem;
  border-bottom: 1px solid var(--hairline);
  vertical-align: top;
  color: var(--platinum);
}

.table tbody tr:hover td {
  background: rgba(231, 236, 247, 0.02);
}

.mono {
  font-variant-numeric: tabular-nums;
  color: var(--muted);
  font-size: 0.82rem;
}

.category-name {
  font-weight: 500;
}

.category-desc {
  margin-top: 0.15rem;
  color: var(--faint);
  font-size: 0.8rem;
  max-width: 42ch;
}

.badge {
  display: inline-block;
  padding: 0.1rem 0.55rem;
  border-radius: 999px;
  border: 1px solid var(--hairline-strong);
  color: var(--muted);
  font-size: 0.72rem;
}

.badge--on {
  color: var(--danger);
  border-color: rgba(255, 107, 107, 0.35);
}

.table__actions-head {
  text-align: right;
}

.table__actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.4rem;
  white-space: nowrap;
}

.user {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.user__avatar {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--raised);
}

.user__name {
  font-size: 0.85rem;
}

.form-panel {
  margin-top: 0.5rem;
  padding: 1.1rem;
  border: 1px solid var(--hairline-strong);
  border-radius: var(--radius);
  background: rgba(110, 134, 255, 0.03);
}

.form-panel__title {
  margin: 0 0 0.9rem;
  font-size: 0.9rem;
  font-weight: 600;
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0.8rem;
}

.form-field {
  display: grid;
  gap: 0.3rem;
}

.form-field--wide {
  grid-column: 1 / -1;
}

.form-field--checkbox {
  grid-column: 1 / -1;
  display: flex;
  flex-direction: row;
  align-items: center;
  gap: 0.5rem;
}

.form-field--checkbox input {
  accent-color: var(--spotlight);
}

.form-label {
  font-size: 0.78rem;
  color: var(--muted);
}

.form-error {
  margin: 0.8rem 0 0;
  color: var(--danger);
  font-size: 0.85rem;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1rem;
}
</style>
