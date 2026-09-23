<script setup lang="ts">
import TopBar from './TopBar.vue'
import SideNav from './SideNav.vue'
import { matched } from '../router'
import { isDebugActive } from '../config'

const debugActive = isDebugActive()
</script>

<template>
  <div class="shell">
    <p v-if="debugActive" class="banner">
      Debug mode — signed in as a fixture user, data served from src/mock. Nothing here reaches the backend.
    </p>

    <TopBar />

    <div class="shell__body">
      <SideNav />
      <main class="stage" id="main">
        <component :is="matched.component" />
      </main>
    </div>
  </div>
</template>

<style scoped>
.shell {
  min-height: 100vh;
  min-width: var(--shell-min);
  background:
    radial-gradient(1100px 460px at 50% -180px, rgba(110, 134, 255, 0.16), transparent 70%),
    radial-gradient(700px 320px at 50% -60px, rgba(201, 162, 75, 0.1), transparent 70%),
    var(--stage);
}

.banner {
  margin: 0;
  padding: 0.45rem 1.5rem;
  background: rgba(110, 134, 255, 0.12);
  border-bottom: 1px solid rgba(110, 134, 255, 0.28);
  color: #b9c5ff;
  font-size: 0.8rem;
  letter-spacing: 0.02em;
}

.shell__body {
  display: flex;
  align-items: flex-start;
}

.stage {
  flex: 1;
  min-width: 0;
  padding: 3.5rem 2.5rem 6rem;
}
</style>
