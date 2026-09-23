<script setup lang="ts">
withDefaults(
  defineProps<{
    tone?: 'neutral' | 'error' | 'success'
    title?: string
    message?: string | null
  }>(),
  { tone: 'neutral', title: '', message: '' },
)
</script>

<template>
  <div class="note" :class="`note--${tone}`" :role="tone === 'error' ? 'alert' : 'status'">
    <p v-if="title" class="note__title">{{ title }}</p>
    <p v-if="message" class="note__message">{{ message }}</p>
    <slot />
  </div>
</template>

<style scoped>
.note {
  border: 1px solid var(--hairline);
  border-left-width: 2px;
  border-radius: var(--radius);
  padding: 1rem 1.15rem;
  background: rgba(231, 236, 247, 0.02);
}

.note--error {
  border-left-color: var(--danger);
}
.note--success {
  border-left-color: var(--success);
}
.note--neutral {
  border-left-color: var(--champagne);
}

.note__title {
  margin: 0 0 0.25rem;
  font-weight: 600;
  color: var(--platinum);
}

.note__message {
  margin: 0;
  color: var(--muted);
  font-size: 0.95rem;
  max-width: 62ch;
}

.note :deep(.note__actions) {
  margin-top: 0.9rem;
  display: flex;
  gap: 0.6rem;
}
</style>
