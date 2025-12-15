<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import { 
  LayoutDashboard, 
  Users, 
  ShieldCheck, 
  Cable, 
  LogOut 
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const navLinks = [
  { name: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { name: 'accounts', label: 'Accounts', icon: Users },
  { name: 'whitelist', label: 'Whitelist', icon: ShieldCheck },
  { name: 'tunnels', label: 'Tunnels', icon: Cable },
]

const currentRoute = computed(() => route.name)

function logout() {
  authStore.clearToken()
  router.push({ name: 'login' })
}
</script>

<template>
  <div class="min-h-screen bg-[var(--bg-deep)]">
    <!-- Navigation -->
    <nav class="sticky top-0 z-50 bg-[var(--bg-surface)] border-b border-[var(--border-subtle)]">
      <div class="h-[60px] px-8 flex items-center">
        <!-- Brand -->
        <RouterLink 
          to="/dashboard" 
          class="flex items-center gap-3 text-[var(--text-primary)] no-underline"
        >
          <div class="w-7 h-7 border-2 border-[var(--accent-copper)] rounded-md flex items-center justify-center">
            <div class="w-3 h-3 bg-[var(--accent-copper)] rounded-sm rotate-45" />
          </div>
          <span class="font-[var(--font-display)] text-xl font-semibold">digit-link</span>
        </RouterLink>

        <!-- Nav Links -->
        <div class="flex gap-1 ml-12">
          <RouterLink
            v-for="link in navLinks"
            :key="link.name"
            :to="{ name: link.name }"
            class="flex items-center gap-2 px-4 py-2 rounded-md text-sm transition-all"
            :class="[
              currentRoute === link.name 
                ? 'text-[var(--accent-copper)] bg-[rgba(201,149,108,0.1)]' 
                : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-elevated)]'
            ]"
          >
            <component :is="link.icon" class="w-4 h-4" />
            <span class="hidden sm:inline">{{ link.label }}</span>
          </RouterLink>
        </div>

        <!-- Spacer -->
        <div class="flex-1" />

        <!-- Logout -->
        <button 
          class="flex items-center gap-2 px-4 py-2 rounded-md text-sm text-[var(--text-muted)] border border-[var(--border-subtle)] hover:text-[var(--accent-red)] hover:border-[var(--accent-red)] transition-all"
          @click="logout"
        >
          <LogOut class="w-4 h-4" />
          <span class="hidden sm:inline">Logout</span>
        </button>
      </div>
    </nav>

    <!-- Main Content -->
    <main class="max-w-[1200px] mx-auto px-8 py-8">
      <slot />
    </main>
  </div>
</template>
