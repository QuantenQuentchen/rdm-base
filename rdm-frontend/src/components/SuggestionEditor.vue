<script setup lang="ts">
import { ref, computed, watch, nextTick, type ComponentPublicInstance } from 'vue'
import config from '../config'
import UiButton from './UiButton.vue'
import type { EntityId, Suggestion, SuggestionDraft } from '../types'

interface EditorRow extends SuggestionDraft {
  key: string
}

const props = withDefaults(
  defineProps<{
    categoryId: EntityId
    suggestions?: Suggestion[]
    max?: number
    locked?: boolean
    saving?: boolean
    saveError?: string | null
  }>(),
  {
    suggestions: () => [],
    max: config.suggestions.maxPerCategory,
    locked: false,
    saving: false,
    saveError: null,
  },
)

const emit = defineEmits<{ save: [entries: SuggestionDraft[]] }>()

let keySeed = 0
const nextKey = () => `row-${props.categoryId}-${keySeed++}`

const rows = ref<EditorRow[]>([])
const baseline = ref('')
const inputs = ref<(HTMLInputElement | null)[]>([])
const justSaved = ref(false)

function toRows(source: Suggestion[]): EditorRow[] {
  const list = source.map((item) => ({ key: nextKey(), id: item.id ?? null, text: item.text ?? '' }))
  return list.length > 0 ? list : [{ key: nextKey(), id: null, text: '' }]
}

/** Comparable snapshot of the meaningful content, for dirty checking. */
function fingerprint(list: EditorRow[]): string {
  return JSON.stringify(
    list
      .map((row): [string, string] => [row.id === null ? '' : String(row.id), row.text.trim()])
      .filter(([, text]) => text.length > 0),
  )
}

function reset(source: Suggestion[] = props.suggestions): void {
  inputs.value = [] // drop element refs from the previous row set
  rows.value = toRows(source)
  baseline.value = fingerprint(rows.value)
  justSaved.value = false
}

reset()
watch(() => props.suggestions, (value) => reset(value), { deep: true })

function setInput(el: Element | ComponentPublicInstance | null, index: number): void {
  inputs.value[index] = (el as HTMLInputElement | null) ?? null
}

const filledCount = computed(() => rows.value.filter((row) => row.text.trim().length > 0).length)
const canAdd = computed(() => !props.locked && rows.value.length < props.max)
const isDirty = computed(() => fingerprint(rows.value) !== baseline.value)
const tooLong = computed(() =>
  rows.value.some((row) => row.text.trim().length > config.suggestions.maxLength),
)
const canSave = computed(() => !props.locked && !props.saving && isDirty.value && !tooLong.value)

async function addRow(afterIndex: number = rows.value.length - 1): Promise<void> {
  if (!canAdd.value) return
  rows.value.splice(afterIndex + 1, 0, { key: nextKey(), id: null, text: '' })
  await nextTick()
  inputs.value[afterIndex + 1]?.focus()
}

function removeRow(index: number): void {
  rows.value.splice(index, 1)
  if (rows.value.length === 0) rows.value.push({ key: nextKey(), id: null, text: '' })
}

function onEnter(index: number): void {
  const row = rows.value[index]
  if (index === rows.value.length - 1 && row && row.text.trim().length > 0) void addRow(index)
  else inputs.value[index + 1]?.focus()
}

function save(): void {
  if (!canSave.value) return
  justSaved.value = false
  emit(
    'save',
    rows.value.map(({ id, text }) => ({ id, text })),
  )
}

// The parent swaps in server-confirmed rows after a successful save, which
// resets the baseline — that's our cue to confirm it in the UI.
watch(
  () => props.saving,
  (isSaving, wasSaving) => {
    if (wasSaving && !isSaving && !props.saveError) {
      justSaved.value = true
      window.setTimeout(() => (justSaved.value = false), 3200)
    }
  },
)

defineExpose({ focusFirst: () => inputs.value[0]?.focus() })
</script>

<template>
  <div class="editor">
    <ol class="rows">
      <li v-for="(row, index) in rows" :key="row.key" class="row">
        <span class="row__ordinal" aria-hidden="true">{{ index + 1 }}</span>

        <label class="sr-only" :for="`${row.key}-input`">Suggestion {{ index + 1 }}</label>
        <input
          :id="`${row.key}-input`"
          :ref="(el) => setInput(el, index)"
          v-model="row.text"
          class="field"
          type="text"
          :maxlength="config.suggestions.maxLength"
          :disabled="locked || saving"
          placeholder="Name a game, studio, or performance"
          @keydown.enter.prevent="onEnter(index)"
        />

        <button
          type="button"
          class="row__remove"
          :disabled="locked || saving || (rows.length === 1 && row.text.length === 0)"
          :aria-label="`Remove suggestion ${index + 1}`"
          @click="removeRow(index)"
        >
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path d="M3 8h10" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" />
          </svg>
        </button>
      </li>
    </ol>

    <button v-if="canAdd" type="button" class="add" @click="addRow()">
      <span class="add__plus" aria-hidden="true">+</span>
      Add another suggestion
    </button>
    <p v-else-if="!locked" class="limit">You've used all {{ max }} slots in this category.</p>

    <footer class="bar">
      <p class="bar__count">
        {{ filledCount }} of {{ max }} filled
        <span v-if="isDirty" class="bar__flag">Unsaved</span>
        <span v-else-if="justSaved" class="bar__flag bar__flag--ok">Saved</span>
      </p>

      <div class="bar__actions">
        <UiButton v-if="isDirty" variant="quiet" size="sm" :disabled="saving" @click="reset()">
          Discard changes
        </UiButton>
        <UiButton variant="gold" size="sm" :loading="saving" :disabled="!canSave" @click="save">
          Save suggestions
        </UiButton>
      </div>
    </footer>

    <p v-if="tooLong" class="error">
      Keep each suggestion under {{ config.suggestions.maxLength }} characters.
    </p>
    <p v-if="saveError" class="error" role="alert">{{ saveError }}</p>
  </div>
</template>

<style scoped>
.editor {
  display: grid;
  gap: 0.9rem;
}

.rows {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 0.5rem;
}

.row {
  display: grid;
  grid-template-columns: 1.4rem minmax(0, 1fr) 2rem;
  align-items: center;
  gap: 0.6rem;
}

.row__ordinal {
  font-family: var(--display);
  font-variation-settings: 'wdth' 100, 'wght' 600;
  font-size: 0.8rem;
  color: var(--faint);
  text-align: right;
}

.row__remove {
  width: 2rem;
  height: 2rem;
  display: grid;
  place-items: center;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius);
  color: var(--faint);
  cursor: pointer;
  transition: color 0.18s var(--ease), border-color 0.18s var(--ease);
}

.row__remove svg {
  width: 16px;
  height: 16px;
}

.row__remove:hover:not(:disabled) {
  color: var(--danger);
  border-color: rgba(255, 107, 107, 0.35);
}

.row__remove:disabled {
  opacity: 0.25;
  cursor: not-allowed;
}

.add {
  justify-self: start;
  margin-left: 2rem;
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  padding: 0.4rem 0.75rem 0.4rem 0.4rem;
  background: transparent;
  border: 1px dashed var(--hairline-strong);
  border-radius: var(--radius);
  color: var(--muted);
  font-family: var(--body);
  font-size: 0.875rem;
  cursor: pointer;
  transition: color 0.18s var(--ease), border-color 0.18s var(--ease);
}

.add:hover {
  color: var(--champagne-bright);
  border-color: var(--champagne);
}

.add__plus {
  display: grid;
  place-items: center;
  width: 1.4rem;
  height: 1.4rem;
  border-radius: 50%;
  border: 1px solid currentColor;
  font-size: 0.9rem;
  line-height: 1;
}

.limit {
  margin: 0 0 0 2rem;
  font-size: 0.8rem;
  color: var(--faint);
}

.bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--hairline);
}

.bar__count {
  margin: 0;
  font-size: 0.82rem;
  color: var(--faint);
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.bar__flag {
  color: var(--champagne);
  border: 1px solid var(--champagne-dim);
  border-radius: 999px;
  padding: 0.05rem 0.5rem;
  font-size: 0.72rem;
}

.bar__flag--ok {
  color: var(--success);
  border-color: rgba(87, 217, 163, 0.3);
}

.bar__actions {
  display: flex;
  gap: 0.5rem;
}

.error {
  margin: 0;
  color: var(--danger);
  font-size: 0.85rem;
}
</style>
