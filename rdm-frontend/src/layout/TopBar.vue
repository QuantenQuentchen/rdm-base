<script setup lang="ts">
import { ref, computed } from 'vue'
import config from '../config'
import { currentUser, isAdmin, signOut } from '../stores/session'
import { resetAwards } from '../stores/awards'
import { navigate, isActive, DEFAULT_PATH } from '../router'
import UiButton from '../components/UiButton.vue'
import DebugMenu from '../components/DebugMenu.vue'

const menuOpen = ref(false)
const isDev = import.meta.env.DEV

const user = currentUser
const initials = computed(() => {
  const name = user.value?.displayName ?? user.value?.username ?? '?'
  return name.trim().slice(0, 2).toUpperCase()
})

const onAdmin = computed(() => isActive('/admin'))

function toggleAdmin(): void {
  navigate(onAdmin.value ? DEFAULT_PATH : '/admin')
}

async function handleSignOut(): Promise<void> {
  menuOpen.value = false
  resetAwards()
  await signOut()
}
</script>

<template>
  <header class="topbar">
    <a class="wordmark" href="#/awards">
      <span class="wordmark__name">{{ config.brand.name }}</span>
      <span class="wordmark__edition">{{ config.brand.edition }}</span>
    </a>

    <div class="tools">
      <DebugMenu v-if="isDev" />

      <!-- Only rendered once the admin check comes back true. -->
      <UiButton v-if="isAdmin" size="sm" :variant="onAdmin ? 'gold' : 'ghost'" @click="toggleAdmin">
        {{ onAdmin ? 'Back to awards' : 'Admin panel' }}
      </UiButton>

      <div v-if="user" class="account">
        <button
          class="chip"
          type="button"
          :aria-expanded="menuOpen"
          aria-haspopup="menu"
          @click="menuOpen = !menuOpen"
        >
          <img v-if="user.avatarUrl" class="chip__avatar" :src="user.avatarUrl" alt="" />
          <span v-else class="chip__avatar chip__avatar--letters">{{ initials }}</span>
          <span class="chip__name">{{ user.displayName ?? user.username }}</span>
        </button>

        <div v-if="menuOpen" class="menu" role="menu">
          <p class="menu__meta">
            Signed in with Discord<span v-if="user.username"> as {{ user.username }}</span>
          </p>
          <UiButton variant="quiet" size="sm" @click="handleSignOut">Sign out</UiButton>
        </div>
      </div>
    </div>
  </header>
</template>

<style scoped>
.topbar {
  position: sticky;
  top: 0;
  z-index: 30;
  height: var(--topbar);
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 0 1.5rem;
  background: rgba(4, 5, 10, 0.86);
  backdrop-filter: blur(14px);
  border-bottom: 1px solid var(--hairline);
}

.wordmark {
  display: flex;
  align-items: baseline;
  gap: 0.7rem;
  text-decoration: none;
  color: inherit;
  margin-right: auto;
}

.wordmark__name {
  font-family: var(--display);
  font-variation-settings: 'wdth' 112, 'wght' 750;
  text-transform: uppercase;
  letter-spacing: 0.1em;
  font-size: 0.95rem;
}

.wordmark__edition {
  font-size: 0.72rem;
  letter-spacing: 0.22em;
  color: var(--champagne);
}

.tools {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.account {
  position: relative;
}

.chip {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  padding: 0.3rem 0.75rem 0.3rem 0.3rem;
  background: transparent;
  border: 1px solid var(--hairline);
  border-radius: 999px;
  color: var(--platinum);
  font-family: var(--body);
  font-size: 0.875rem;
  cursor: pointer;
  transition: border-color 0.18s var(--ease);
}

.chip:hover {
  border-color: var(--hairline-strong);
}

.chip__avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  background: var(--raised);
}

.chip__avatar--letters {
  display: grid;
  place-items: center;
  font-size: 0.7rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  color: var(--champagne);
  border: 1px solid var(--champagne-dim);
}

.menu {
  position: absolute;
  top: calc(100% + 0.6rem);
  right: 0;
  min-width: 232px;
  padding: 0.9rem;
  background: var(--elevated);
  border: 1px solid var(--hairline-strong);
  border-radius: var(--radius-lg);
  box-shadow: 0 24px 48px rgba(0, 0, 0, 0.55);
}

.menu__meta {
  margin: 0 0 0.6rem;
  font-size: 0.8rem;
  color: var(--faint);
}
</style>
