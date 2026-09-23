<script setup lang="ts">
import { ref, computed } from 'vue'
import SuggestionEditor from './SuggestionEditor.vue'
import config from '../config'
import type { Category, Suggestion, SuggestionDraft } from '../types'

const props = withDefaults(
  defineProps<{
    category: Category
    suggestions?: Suggestion[]
    saving?: boolean
    saveError?: string | null
  }>(),
  { suggestions: () => [], saving: false, saveError: null },
)

const emit = defineEmits<{ save: [entries: SuggestionDraft[]] }>()

const expanded = ref(false)
const panelId = computed(() => `category-${props.category.id}`)
const entryCount = computed(() => props.suggestions.length)
const max = computed(() => props.category.maxSuggestions ?? config.suggestions.maxPerCategory)
</script>

<template>
  <article class="panel" :class="{ 'panel--open': expanded, 'panel--locked': category.locked }">
    <button
      type="button"
      class="head"
      :aria-expanded="expanded"
      :aria-controls="panelId"
      @click="expanded = !expanded"
    >
      <div class="head__text">
        <h3 class="plate head__name">{{ category.name }}</h3>
        <p v-if="category.description" class="head__desc">{{ category.description }}</p>
      </div>

      <div class="head__meta">
        <span v-if="category.locked" class="tag tag--locked">Closed</span>
        <span v-else-if="entryCount > 0" class="tag tag--filled">
          {{ entryCount }} {{ entryCount === 1 ? 'suggestion' : 'suggestions' }}
        </span>
        <span v-else class="tag">No suggestions yet</span>

        <svg class="chev" viewBox="0 0 16 16" aria-hidden="true">
          <path d="M4 6l4 4 4-4" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
        </svg>
      </div>
    </button>

    <div v-show="expanded" :id="panelId" class="body">
      <p v-if="category.criteria" class="criteria">{{ category.criteria }}</p>

      <p v-if="category.locked" class="closed">
        Submissions for this category are closed. Your entries are locked in.
      </p>

      <SuggestionEditor
        v-else
        :category-id="category.id"
        :suggestions="suggestions"
        :max="max"
        :locked="category.locked ?? false"
        :saving="saving"
        :save-error="saveError"
        @save="(entries: SuggestionDraft[]) => emit('save', entries)"
      />
    </div>
  </article>
</template>

<style scoped>
.panel {
  border: 1px solid var(--hairline);
  border-left: 2px solid var(--hairline-strong);
  border-radius: 0 var(--radius-lg) var(--radius-lg) 0;
  background: linear-gradient(180deg, rgba(20, 25, 38, 0.72), rgba(10, 13, 22, 0.5));
  transition: border-color 0.2s var(--ease);
}

.panel--open {
  border-left-color: var(--champagne);
  border-color: var(--hairline-strong);
}

.panel--locked {
  opacity: 0.78;
}

.head {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1.5rem;
  padding: 1.35rem 1.5rem;
  background: none;
  border: none;
  color: inherit;
  text-align: left;
  cursor: pointer;
}

.head__name {
  font-size: 1.25rem;
  color: var(--platinum);
}

.panel--open .head__name {
  color: var(--champagne-bright);
}

.head__desc {
  margin: 0.4rem 0 0;
  font-size: 0.9rem;
  color: var(--muted);
  max-width: 58ch;
}

.head__meta {
  display: flex;
  align-items: center;
  gap: 0.9rem;
  flex: none;
}

.tag {
  font-size: 0.74rem;
  letter-spacing: 0.04em;
  color: var(--faint);
  white-space: nowrap;
}

.tag--filled {
  color: var(--champagne);
}

.tag--locked {
  color: var(--muted);
  border: 1px solid var(--hairline);
  border-radius: 999px;
  padding: 0.1rem 0.55rem;
}

.chev {
  width: 16px;
  height: 16px;
  color: var(--faint);
  transition: transform 0.2s var(--ease);
}

.panel--open .chev {
  transform: rotate(180deg);
  color: var(--champagne);
}

.body {
  padding: 1.25rem 1.5rem 1.5rem;
  border-top: 1px solid var(--hairline);
}

.criteria {
  margin: 0 0 1.1rem;
  font-size: 0.875rem;
  color: var(--muted);
  max-width: 62ch;
}

.closed {
  margin: 0;
  color: var(--muted);
  font-size: 0.9rem;
}
</style>
