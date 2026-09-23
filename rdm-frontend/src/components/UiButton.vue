<script setup lang="ts">
withDefaults(
  defineProps<{
    variant?: 'gold' | 'ghost' | 'quiet' | 'danger'
    size?: 'sm' | 'md' | 'lg'
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    loading?: boolean
  }>(),
  {
    variant: 'ghost',
    size: 'md',
    type: 'button',
    disabled: false,
    loading: false,
  },
)
</script>

<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    class="btn"
    :class="[`btn--${variant}`, `btn--${size}`]"
  >
    <span v-if="loading" class="spinner" aria-hidden="true" />
    <slot />
  </button>
</template>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border: 1px solid transparent;
  border-radius: var(--radius);
  font-family: var(--body);
  font-weight: 600;
  letter-spacing: 0.02em;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.18s var(--ease), border-color 0.18s var(--ease), color 0.18s var(--ease);
}

.btn:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.btn--sm {
  padding: 0.35rem 0.7rem;
  font-size: 0.85rem;
}
.btn--md {
  padding: 0.55rem 1.05rem;
  font-size: 0.925rem;
}
.btn--lg {
  padding: 0.8rem 1.6rem;
  font-size: 1rem;
}

.btn--gold {
  background: linear-gradient(180deg, var(--champagne-bright), var(--champagne));
  color: #120d02;
  border-color: var(--champagne-bright);
}
.btn--gold:hover:not(:disabled) {
  background: var(--champagne-bright);
}

.btn--ghost {
  background: transparent;
  color: var(--platinum);
  border-color: var(--hairline-strong);
}
.btn--ghost:hover:not(:disabled) {
  border-color: var(--champagne);
  color: var(--champagne-bright);
}

.btn--quiet {
  background: transparent;
  color: var(--muted);
  border-color: transparent;
}
.btn--quiet:hover:not(:disabled) {
  color: var(--platinum);
  background: rgba(231, 236, 247, 0.06);
}

.btn--danger {
  background: transparent;
  color: var(--danger);
  border-color: transparent;
}
.btn--danger:hover:not(:disabled) {
  background: rgba(255, 107, 107, 0.1);
}

.spinner {
  width: 0.85em;
  height: 0.85em;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
