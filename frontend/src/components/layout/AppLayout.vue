<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import { ThemeSwitcher } from '@/components/shared'
import { 
  LayoutDashboard, 
  Users, 
  ShieldCheck, 
  Cable, 
  LogOut,
  Building2,
  AppWindow,
  KeyRound,
  ScrollText
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const navLinks = [
  { name: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { name: 'organizations', label: 'Orgs', icon: Building2 },
  { name: 'applications', label: 'Apps', icon: AppWindow },
  { name: 'api-keys', label: 'API Keys', icon: KeyRound },
  { name: 'accounts', label: 'Accounts', icon: Users },
  { name: 'whitelist', label: 'Whitelist', icon: ShieldCheck },
  { name: 'tunnels', label: 'Tunnels', icon: Cable },
  { name: 'audit', label: 'Audit', icon: ScrollText },
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
          <div class="w-7 h-7 border-2 border-[var(--accent-primary)] rounded-md flex items-center justify-center">
            <div class="w-3 h-3 bg-[var(--accent-primary)] rounded-sm rotate-45" />
          </div>
          <span class="font-[var(--font-display)] text-xl font-semibold hidden lg:inline">digit-link</span>
        </RouterLink>

        <!-- Nav Links -->
        <div class="flex gap-0.5 ml-6 lg:ml-12 overflow-x-auto">
          <RouterLink
            v-for="link in navLinks"
            :key="link.name"
            :to="{ name: link.name }"
            class="nav-link"
            :class="{ 'nav-link--active': currentRoute === link.name }"
          >
            <component :is="link.icon" class="w-4 h-4" />
            <span class="hidden md:inline">{{ link.label }}</span>
          </RouterLink>
        </div>

        <!-- Spacer -->
        <div class="flex-1" />

        <!-- Theme Switcher -->
        <ThemeSwitcher />

        <!-- Logout -->
        <button 
          class="flex items-center gap-2 px-3 py-2 ml-2 rounded-md text-sm text-[var(--text-muted)] border border-[var(--border-subtle)] hover:text-[var(--accent-red)] hover:border-[var(--accent-red)] transition-all"
          @click="logout"
        >
          <LogOut class="w-4 h-4" />
          <span class="hidden sm:inline">Logout</span>
        </button>
      </div>
    </nav>

    <!-- Main Content -->
    <main class="max-w-[1400px] mx-auto px-4 lg:px-8 py-8">
      <slot />
    </main>
  </div>
</template>

<style scoped>
.nav-link {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.5rem 0.75rem;
  border-radius: 0.375rem;
  font-size: 0.875rem;
  transition: all 0.2s ease;
  white-space: nowrap;
  color: var(--text-secondary);
}

.nav-link:hover {
  color: var(--text-primary);
  background: var(--bg-elevated);
}

.nav-link--active {
  color: var(--accent-primary);
  background: rgba(var(--accent-primary-rgb), 0.1);
}

.nav-link--active:hover {
  color: var(--accent-primary);
  background: rgba(var(--accent-primary-rgb), 0.15);
}
</style>
