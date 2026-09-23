<script setup lang="ts">
import { navRoutes, isActive, navigate } from '../router'
</script>

<template>
  <nav class="rail" aria-label="Sections">
    <ul class="rail__list">
      <li v-for="entry in navRoutes" :key="entry.path">
        <button
          type="button"
          class="rail__item"
          :class="{ 'rail__item--active': isActive(entry.path) }"
          :aria-current="isActive(entry.path) ? 'page' : undefined"
          @click="navigate(entry.path)"
        >
          <svg class="rail__icon" viewBox="0 0 24 24" aria-hidden="true">
            <path
              d="M7 4h10v3h3v2a4 4 0 0 1-4 4h-.6A5 5 0 0 1 13 15.9V18h3v2H8v-2h3v-2.1a5 5 0 0 1-2.4-2.9H8a4 4 0 0 1-4-4V7h3V4Zm0 5H6a2 2 0 0 0 1 1.7V9Zm10 1.7A2 2 0 0 0 18 9h-1v1.7Z"
              fill="currentColor"
            />
          </svg>
          <span class="rail__text">
            <span class="rail__label">{{ entry.label }}</span>
            <span v-if="entry.hint" class="rail__hint">{{ entry.hint }}</span>
          </span>
        </button>
      </li>
    </ul>

    <p class="rail__footnote">More sections open as the season progresses.</p>
  </nav>
</template>

<style scoped>
.rail {
  position: sticky;
  top: var(--topbar);
  align-self: flex-start;
  flex: none;
  height: calc(100vh - var(--topbar));
  width: var(--rail);
  padding: 1.5rem 0.9rem;
  border-right: 1px solid var(--hairline);
  display: flex;
  flex-direction: column;
}

.rail__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  gap: 0.25rem;
}

.rail__item {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.65rem 0.7rem;
  background: transparent;
  border: none;
  border-left: 2px solid transparent;
  border-radius: 0 var(--radius) var(--radius) 0;
  color: var(--muted);
  font-family: var(--body);
  text-align: left;
  cursor: pointer;
  transition: color 0.18s var(--ease), background 0.18s var(--ease);
}

.rail__item:hover {
  color: var(--platinum);
  background: rgba(231, 236, 247, 0.04);
}

.rail__item--active {
  color: var(--platinum);
  background: rgba(201, 162, 75, 0.07);
  border-left-color: var(--champagne);
}

.rail__item--active .rail__icon {
  color: var(--champagne);
}

.rail__icon {
  width: 20px;
  height: 20px;
  flex: none;
  color: var(--faint);
}

.rail__text {
  display: grid;
  gap: 0.1rem;
  min-width: 0;
}

.rail__label {
  font-size: 0.95rem;
  font-weight: 500;
}

.rail__hint {
  font-size: 0.74rem;
  color: var(--faint);
  line-height: 1.3;
}

.rail__footnote {
  margin: auto 0 0;
  padding: 0 0.7rem;
  font-size: 0.75rem;
  color: var(--faint);
  max-width: 24ch;
}
</style>
