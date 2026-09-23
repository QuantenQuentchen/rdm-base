<script setup lang="ts">
import { ref } from 'vue'
import { debug, setDebugFlag, clearDebugOverrides, type DebugFlags } from '../config'

/**
 * Dev-only switchboard. Rendered by TopBar behind `import.meta.env.DEV`, so it
 * disappears from a production build entirely. Flipping a flag persists it and
 * reloads, which is the simplest way to re-bootstrap the stores cleanly.
 */
const open = ref(false)

const flags: { key: keyof DebugFlags; label: string; hint: string }[] = [
  { key: 'bypassAuth', label: 'Skip sign-in', hint: 'Boot straight in as the fixture user' },
  { key: 'useMockApi', label: 'Mock API', hint: 'Answer requests from src/mock instead of the network' },
  { key: 'forceAdmin', label: 'Force admin', hint: 'Treat the current user as an admin' },
]

function toggle(key: keyof DebugFlags): void {
  setDebugFlag(key, !debug[key])
}
</script>

<template>
  <div class="debug">
    <button class="pill" type="button" :aria-expanded="open" @click="open = !open">
      <span class="dot" aria-hidden="true" />
      Debug
    </button>

    <div v-if="open" class="sheet">
      <p class="sheet__title">Local overrides</p>

      <label v-for="flag in flags" :key="flag.key" class="row">
        <input type="checkbox" :checked="debug[flag.key]" @change="toggle(flag.key)" />
        <span class="row__text">
          <span class="row__label">{{ flag.label }}</span>
          <span class="row__hint">{{ flag.hint }}</span>
        </span>
      </label>

      <button class="reset" type="button" @click="clearDebugOverrides()">
        Reset to env defaults
      </button>
    </div>
  </div>
</template>

<style scoped>
.debug {
  position: relative;
}

.pill {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  padding: 0.28rem 0.7rem;
  background: rgba(110, 134, 255, 0.1);
  border: 1px solid rgba(110, 134, 255, 0.35);
  border-radius: 999px;
  color: #a9b8ff;
  font-family: var(--body);
  font-size: 0.78rem;
  letter-spacing: 0.06em;
  cursor: pointer;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--spotlight);
}

.sheet {
  position: absolute;
  top: calc(100% + 0.6rem);
  right: 0;
  z-index: 40;
  width: 288px;
  padding: 0.9rem;
  background: var(--elevated);
  border: 1px solid var(--hairline-strong);
  border-radius: var(--radius-lg);
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.55);
}

.sheet__title {
  margin: 0 0 0.7rem;
  font-size: 0.75rem;
  letter-spacing: 0.14em;
  color: var(--faint);
}

.row {
  display: flex;
  gap: 0.6rem;
  padding: 0.4rem 0;
  cursor: pointer;
}

.row input {
  margin-top: 0.3rem;
  accent-color: var(--spotlight);
}

.row__text {
  display: grid;
}

.row__label {
  font-size: 0.9rem;
  color: var(--platinum);
}

.row__hint {
  font-size: 0.75rem;
  color: var(--faint);
  line-height: 1.35;
}

.reset {
  margin-top: 0.7rem;
  width: 100%;
  padding: 0.4rem;
  background: transparent;
  border: 1px solid var(--hairline);
  border-radius: var(--radius);
  color: var(--muted);
  font-family: var(--body);
  font-size: 0.8rem;
  cursor: pointer;
}

.reset:hover {
  color: var(--platinum);
  border-color: var(--hairline-strong);
}
</style>
