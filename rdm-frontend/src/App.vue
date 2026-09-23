<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { matched } from './router'
import { isSignedIn, isAdmin, isResolvingSession, restoreSession } from './stores/session'
import AppShell from './layout/AppShell.vue'
import SignInView from './views/SignInView.vue'
import StatusNote from './components/StatusNote.vue'

onMounted(() => {
  void restoreSession()
})

const needsAuth = computed(() => matched.value.requiresAuth !== false)
const blockedByRole = computed(() => matched.value.requiresAdmin === true && !isAdmin.value)
</script>

<template>
  <!-- Boot: hold the stage dark rather than flashing the sign-in screen. -->
  <div v-if="isResolvingSession" class="boot">
    <div class="boot__ring" aria-hidden="true" />
    <span class="sr-only">Loading</span>
  </div>

  <!-- Landing on "/" with no session lands here — this is the whole gate. -->
  <SignInView v-else-if="needsAuth && !isSignedIn" />

  <!-- Route-level role gate. AdminView re-checks on mount as well. -->
  <div v-else-if="blockedByRole" class="blocked">
    <StatusNote tone="error" title="Not available" message="This area is limited to administrators." />
  </div>

  <AppShell v-else />
</template>

<style scoped>
.boot {
  min-height: 100vh;
  display: grid;
  place-items: center;
  background: var(--stage);
}

.boot__ring {
  width: 30px;
  height: 30px;
  border: 2px solid var(--champagne-dim);
  border-top-color: var(--champagne);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.blocked {
  min-height: 100vh;
  display: grid;
  place-items: center;
  background: var(--stage);
  padding: 2rem;
}
</style>
