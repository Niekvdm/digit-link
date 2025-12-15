<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import { 
  LayoutDashboard, 
  ShieldCheck, 
  LogOut,
  AppWindow,
  KeyRound,
  Building2,
  Settings
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const navLinks = [
  { name: 'org-dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { name: 'org-applications', label: 'Applications', icon: AppWindow },
  { name: 'org-api-keys', label: 'API Keys', icon: KeyRound },
  { name: 'org-whitelist', label: 'Whitelist', icon: ShieldCheck },
  { name: 'org-settings', label: 'Settings', icon: Settings },
]

const currentRoute = computed(() => route.name)

function logout() {
  authStore.clearToken()
  router.push({ name: 'org-login' })
}
</script>

<template>
  <div class="min-h-screen bg-[var(--bg-deep)]">
    <!-- Navigation -->
    <nav class="sticky top-0 z-50 bg-[var(--bg-surface)] border-b border-[var(--border-subtle)]">
      <div class="h-[60px] px-8 flex items-center">
        <!-- Brand -->
        <RouterLink 
          to="/portal" 
          class="flex items-center gap-3 text-[var(--text-primary)] no-underline"
        >
          <div class="w-7 h-7 border-2 border-[var(--accent-emerald)] rounded-md flex items-center justify-center">
            <Building2 class="w-4 h-4 text-[var(--accent-emerald)]" />
          </div>
          <span class="font-[var(--font-display)] text-xl font-semibold hidden lg:inline">Organization Portal</span>
        </RouterLink>

        <!-- Nav Links -->
        <div class="flex gap-0.5 ml-6 lg:ml-12 overflow-x-auto">
          <RouterLink
            v-for="link in navLinks"
            :key="link.name"
            :to="{ name: link.name }"
            class="flex items-center gap-1.5 px-3 py-2 rounded-md text-sm transition-all whitespace-nowrap"
            :class="[
              currentRoute === link.name || (currentRoute === 'org-application-detail' && link.name === 'org-applications')
                ? 'text-[var(--accent-emerald)] bg-[rgba(74,159,126,0.1)]' 
                : 'text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-elevated)]'
            ]"
          >
            <component :is="link.icon" class="w-4 h-4" />
            <span class="hidden md:inline">{{ link.label }}</span>
          </RouterLink>
        </div>

        <!-- Spacer -->
        <div class="flex-1" />

        <!-- Logout -->
        <button 
          class="flex items-center gap-2 px-3 py-2 rounded-md text-sm text-[var(--text-muted)] border border-[var(--border-subtle)] hover:text-[var(--accent-red)] hover:border-[var(--accent-red)] transition-all"
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
