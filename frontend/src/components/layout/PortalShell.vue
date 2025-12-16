<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter, RouterView } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import { usePortalContext } from '@/composables/usePortalContext'
import { useToast } from '@/composables/useToast'
import { ThemeSwitcher } from '@/components/shared'
import { Toast } from '@/components/ui'
import { 
  LayoutDashboard, 
  Users, 
  ShieldCheck, 
  Cable, 
  LogOut,
  Building2,
  AppWindow,
  KeyRound,
  ScrollText,
  Settings,
  ChevronLeft,
  ChevronRight,
  Menu,
  X
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { isAdmin, username, currentOrgName } = usePortalContext()
const { toasts, dismiss } = useToast()

// Sidebar state
const sidebarCollapsed = ref(false)
const mobileMenuOpen = ref(false)

// Navigation links based on user type
const adminLinks = [
  { name: 'admin-dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { name: 'admin-organizations', label: 'Organizations', icon: Building2 },
  { name: 'admin-applications', label: 'Applications', icon: AppWindow },
  { name: 'admin-accounts', label: 'Accounts', icon: Users },
  { name: 'admin-api-keys', label: 'API Keys', icon: KeyRound },
  { name: 'admin-whitelist', label: 'Whitelist', icon: ShieldCheck },
  { name: 'admin-tunnels', label: 'Tunnels', icon: Cable },
  { name: 'admin-audit', label: 'Audit Logs', icon: ScrollText },
]

const orgLinks = [
  { name: 'org-dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { name: 'org-applications', label: 'Applications', icon: AppWindow },
  { name: 'org-api-keys', label: 'API Keys', icon: KeyRound },
  { name: 'org-whitelist', label: 'Whitelist', icon: ShieldCheck },
  { name: 'org-settings', label: 'Settings', icon: Settings },
]

const navLinks = computed(() => isAdmin.value ? adminLinks : orgLinks)

const currentRoute = computed(() => {
  const name = route.name as string
  // Handle detail pages showing parent as active
  if (name?.includes('application-detail')) {
    return isAdmin.value ? 'admin-applications' : 'org-applications'
  }
  return name
})

const portalTitle = computed(() => isAdmin.value ? 'Admin Portal' : currentOrgName.value || 'Organization Portal')
const accentColor = computed(() => isAdmin.value ? 'primary' : 'secondary')

function logout() {
  authStore.clearToken()
  router.push({ name: 'login' })
}

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

function toggleMobileMenu() {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

function navigateTo(name: string) {
  router.push({ name })
  mobileMenuOpen.value = false
}
</script>

<template>
  <div class="min-h-screen flex bg-bg-deep relative overflow-hidden">
    <!-- Animated background -->
    <div class="bg-pattern" />
    <div 
      class="fixed inset-0 pointer-events-none z-0 transition-[background] duration-600"
      :class="accentColor === 'primary' ? 'bg-gradient-primary' : 'bg-gradient-secondary'"
    />
    
    <!-- Decorative corners -->
    <div class="corner corner-tl" />
    <div class="corner corner-br" />

    <!-- Mobile header -->
    <header class="mobile-header">
      <button 
        class="w-10 h-10 flex items-center justify-center border border-border-subtle rounded-xs bg-transparent text-text-secondary cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary"
        @click="toggleMobileMenu"
      >
        <Menu v-if="!mobileMenuOpen" class="w-5 h-5" />
        <X v-else class="w-5 h-5" />
      </button>
      <div class="flex-1 flex items-center gap-3">
        <div 
          class="w-9 h-9 border-2 rounded-[10px] flex items-center justify-center shrink-0 transition-colors duration-300"
          :class="accentColor === 'primary' ? 'border-accent-primary' : 'border-accent-secondary'"
        >
          <div 
            class="w-3.5 h-3.5 rounded transition-colors duration-300 rotate-45"
            :class="accentColor === 'primary' ? 'bg-accent-primary' : 'bg-accent-secondary'"
          />
        </div>
        <span class="font-display text-xl font-semibold whitespace-nowrap">digit-link</span>
      </div>
      <ThemeSwitcher />
    </header>

    <!-- Mobile navigation overlay -->
    <Transition name="fade">
      <div 
        v-if="mobileMenuOpen" 
        class="mobile-overlay" 
        @click="mobileMenuOpen = false"
      />
    </Transition>

    <!-- Sidebar -->
    <aside 
      class="sidebar" 
      :class="{ 
        'sidebar--collapsed': sidebarCollapsed,
        'sidebar--mobile-open': mobileMenuOpen 
      }"
    >
      <!-- Brand -->
      <div class="flex items-center justify-between mb-6">
        <RouterLink to="/" class="flex items-center gap-3 no-underline text-text-primary">
          <div 
            class="w-9 h-9 border-2 rounded-[10px] flex items-center justify-center shrink-0 transition-colors duration-300"
            :class="accentColor === 'primary' ? 'border-accent-primary' : 'border-accent-secondary'"
          >
            <div 
              class="w-3.5 h-3.5 rounded transition-colors duration-300 rotate-45"
              :class="accentColor === 'primary' ? 'bg-accent-primary' : 'bg-accent-secondary'"
            />
          </div>
          <Transition name="fade">
            <span v-if="!sidebarCollapsed" class="font-display text-xl font-semibold whitespace-nowrap">digit-link</span>
          </Transition>
        </RouterLink>
        <button 
          class="collapse-btn w-7 h-7 flex items-center justify-center border border-border-subtle rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary"
          @click="toggleSidebar"
        >
          <ChevronLeft v-if="!sidebarCollapsed" class="w-4 h-4" />
          <ChevronRight v-else class="w-4 h-4" />
        </button>
      </div>

      <!-- Portal type badge -->
      <div 
        class="flex items-center gap-2 py-2.5 px-3.5 rounded-xs text-xs font-medium mb-6 whitespace-nowrap overflow-hidden"
        :class="[
          accentColor === 'primary' ? 'bg-[rgba(var(--accent-primary-rgb),0.1)] text-accent-primary' : 'bg-[rgba(var(--accent-secondary-rgb),0.1)] text-accent-secondary',
          sidebarCollapsed ? 'justify-center px-2.5' : ''
        ]"
      >
        <Building2 v-if="!isAdmin" class="w-3.5 h-3.5" />
        <ShieldCheck v-else class="w-3.5 h-3.5" />
        <Transition name="fade">
          <span v-if="!sidebarCollapsed">{{ portalTitle }}</span>
        </Transition>
      </div>

      <!-- Navigation -->
      <nav class="flex flex-col gap-1">
        <button
          v-for="link in navLinks"
          :key="link.name"
          class="flex items-center gap-3 py-3 px-3.5 rounded-xs text-sm font-medium text-text-secondary bg-transparent border-none cursor-pointer transition-all duration-200 text-left w-full whitespace-nowrap hover:bg-bg-elevated hover:text-text-primary"
          :class="[
            currentRoute === link.name ? 'bg-[rgba(var(--accent-primary-rgb),0.1)] text-accent-primary hover:bg-[rgba(var(--accent-primary-rgb),0.15)]' : '',
            sidebarCollapsed ? 'justify-center px-3' : ''
          ]"
          @click="navigateTo(link.name)"
          :title="sidebarCollapsed ? link.label : undefined"
        >
          <component :is="link.icon" class="w-[18px] h-[18px] shrink-0" />
          <Transition name="fade">
            <span v-if="!sidebarCollapsed" class="overflow-hidden">{{ link.label }}</span>
          </Transition>
        </button>
      </nav>

      <!-- Spacer -->
      <div class="flex-1" />

      <!-- User section -->
      <div class="flex flex-col gap-3 pt-4 border-t border-border-subtle mt-4">
        <ThemeSwitcher class="hidden lg:flex" />

        <button 
          class="flex items-center gap-3 py-3 px-3.5 rounded-xs text-sm font-medium text-text-muted bg-transparent border border-border-subtle cursor-pointer transition-all duration-200 w-full hover:text-accent-red hover:border-accent-red hover:bg-[rgba(var(--accent-red-rgb),0.05)]"
          :class="sidebarCollapsed ? 'justify-center px-3' : ''"
          @click="logout" 
          :title="sidebarCollapsed ? 'Logout' : undefined"
        >
          <LogOut class="w-4 h-4" />
          <Transition name="fade">
            <span v-if="!sidebarCollapsed">Logout</span>
          </Transition>
        </button>
      </div>
    </aside>

    <!-- Main content -->
    <main class="flex-1 p-8 min-w-0 max-w-full relative z-10 max-lg:ml-0 max-lg:pt-[60px]">
      <RouterView />
    </main>

    <!-- Toast notifications -->
    <Toast :toasts="toasts" @dismiss="dismiss" />
  </div>
</template>

<style scoped>
  @reference "../../style.css";
/* Background pattern - needs pseudo-element */
.bg-pattern {
  @apply fixed inset-0 pointer-events-none z-0 opacity-[0.12];
  background-image: 
    linear-gradient(var(--border-subtle) 1px, transparent 1px),
    linear-gradient(90deg, var(--border-subtle) 1px, transparent 1px);
  background-size: 60px 60px;
}

/* Background gradients */
.bg-gradient-primary {
  background: radial-gradient(ellipse at 30% 20%, rgba(var(--accent-primary-rgb), 0.08) 0%, transparent 50%),
              radial-gradient(ellipse at 70% 80%, rgba(var(--accent-primary-rgb), 0.05) 0%, transparent 50%);
}

.bg-gradient-secondary {
  background: radial-gradient(ellipse at 30% 20%, rgba(var(--accent-secondary-rgb), 0.08) 0%, transparent 50%),
              radial-gradient(ellipse at 70% 80%, rgba(var(--accent-secondary-rgb), 0.05) 0%, transparent 50%);
}

/* Mobile header */
.mobile-header {
  @apply hidden fixed top-0 left-0 right-0 h-[60px] bg-bg-surface border-b border-border-subtle px-4 items-center gap-4 z-[100];
}

.mobile-overlay {
  @apply hidden fixed inset-0 bg-black/50 z-[199];
}

@media (max-width: 1023px) {
  .mobile-header {
    @apply flex;
  }
  
  .mobile-overlay {
    @apply block;
  }
  
  .collapse-btn {
    @apply hidden;
  }
}

/* Sidebar */
.sidebar {
  @apply w-[260px] bg-bg-surface border-r border-border-subtle flex flex-col p-5 transition-[width] duration-300 shrink-0 relative z-10;
}

.sidebar--collapsed {
  @apply w-20 px-3;
}

@media (max-width: 1023px) {
  .sidebar {
    @apply fixed -left-[280px] top-0 bottom-0 z-[200] transition-[left] duration-300;
  }
  
  .sidebar--mobile-open {
    @apply left-0;
  }
}
</style>
