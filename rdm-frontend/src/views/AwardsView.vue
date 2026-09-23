<script setup lang="ts">
import { onMounted } from 'vue'
import config from '../config'
import {
  awards,
  categories,
  isLoading,
  loadAwards,
  saveSuggestions,
  suggestionsFor,
  isSaving,
  saveErrorFor,
  totalSuggestions,
  categoriesWithEntries,
} from '../stores/awards'
import CategoryPanel from '../components/CategoryPanel.vue'
import StatusNote from '../components/StatusNote.vue'
import UiButton from '../components/UiButton.vue'
import type { EntityId, SuggestionDraft } from '../types'

onMounted(() => loadAwards())

function onSave(categoryId: EntityId, entries: SuggestionDraft[]): void {
  void saveSuggestions(categoryId, entries)
}
</script>

<template>
  <div class="view">
    <header class="masthead">
      <p class="masthead__edition">{{ config.brand.edition }}</p>
      <h1 class="display masthead__title">{{ config.brand.name }}</h1>
      <div class="masthead__rule" aria-hidden="true" />
      <p class="masthead__lede">
        Nominations are open to the floor. Put forward the games, studios and performances that
        defined the year — the jury takes it from there.
      </p>
    </header>

    <section class="board" aria-labelledby="categories-heading">
      <div class="board__head">
        <h2 id="categories-heading" class="plate board__title">Categories</h2>
        <p v-if="awards.status === 'ready'" class="board__count">
          {{ categoriesWithEntries }} of {{ categories.length }} entered ·
          {{ totalSuggestions }} {{ totalSuggestions === 1 ? 'suggestion' : 'suggestions' }} in total
        </p>
      </div>

      <div v-if="isLoading" class="skeletons" aria-hidden="true">
        <div v-for="n in 5" :key="n" class="skeleton" />
      </div>

      <StatusNote
        v-else-if="awards.status === 'error'"
        tone="error"
        title="Categories didn't load"
        :message="awards.error"
      >
        <div class="note__actions">
          <UiButton variant="ghost" size="sm" @click="loadAwards({ force: true })">Try again</UiButton>
        </div>
      </StatusNote>

      <StatusNote
        v-else-if="categories.length === 0"
        title="No categories yet"
        message="The ballot hasn't been published. Check back once the season opens."
      />

      <div v-else class="panels">
        <CategoryPanel
          v-for="category in categories"
          :key="category.id"
          :category="category"
          :suggestions="suggestionsFor(category.id)"
          :saving="isSaving(category.id)"
          :save-error="saveErrorFor(category.id)"
          @save="(entries: SuggestionDraft[]) => onSave(category.id, entries)"
        />
      </div>
    </section>
  </div>
</template>

<style scoped>
.view {
  max-width: var(--shell-max);
  margin: 0 auto;
}

/* --- masthead: the one loud moment on the page --- */
.masthead {
  text-align: center;
  padding-bottom: 3.5rem;
}

.masthead__edition {
  margin: 0 0 1.1rem;
  font-size: 0.78rem;
  letter-spacing: 0.44em;
  text-indent: 0.44em;
  color: var(--champagne);
}

.masthead__title {
  font-size: 5.25rem;
  background: linear-gradient(175deg, #fdf6e3 8%, var(--champagne-bright) 45%, #8a6c28 100%);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
  text-shadow: 0 0 80px rgba(201, 162, 75, 0.25);
}

.masthead__rule {
  width: 360px;
  height: 1px;
  margin: 1.75rem auto;
  background: linear-gradient(90deg, transparent, var(--champagne), transparent);
}

.masthead__lede {
  margin: 0 auto;
  max-width: 54ch;
  color: var(--muted);
  font-size: 1.02rem;
}

/* --- category board --- */
.board__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 1rem;
  padding-bottom: 0.9rem;
  border-bottom: 1px solid var(--hairline);
  margin-bottom: 1.5rem;
}

.board__title {
  font-size: 0.95rem;
  letter-spacing: 0.24em;
  color: var(--platinum);
}

.board__count {
  margin: 0;
  font-size: 0.82rem;
  color: var(--faint);
}

.panels,
.skeletons {
  display: grid;
  gap: 0.75rem;
}

.skeleton {
  height: 88px;
  border-radius: 0 var(--radius-lg) var(--radius-lg) 0;
  border-left: 2px solid var(--hairline);
  background: linear-gradient(90deg, rgba(20, 25, 38, 0.5), rgba(27, 33, 48, 0.75), rgba(20, 25, 38, 0.5));
  background-size: 200% 100%;
  animation: shimmer 1.6s var(--ease) infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}
</style>
