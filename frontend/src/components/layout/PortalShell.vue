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
  <div class="portal-shell">
    <!-- Animated background -->
    <div class="bg-pattern" />
    <div class="bg-gradient" :class="`bg-gradient--${accentColor}`" />
    
    <!-- Decorative corners -->
    <div class="corner corner-tl" />
    <div class="corner corner-br" />

    <!-- Mobile header -->
    <header class="mobile-header">
      <button class="mobile-menu-btn" @click="toggleMobileMenu">
        <Menu v-if="!mobileMenuOpen" class="w-5 h-5" />
        <X v-else class="w-5 h-5" />
      </button>
      <div class="mobile-brand">
        <div class="brand-icon" :class="`brand-icon--${accentColor}`">
          <div class="brand-icon-inner" />
        </div>
        <span class="brand-name">digit-link</span>
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
      <div class="sidebar-brand">
        <RouterLink to="/" class="brand-link">
          <div class="brand-icon" :class="`brand-icon--${accentColor}`">
            <div class="brand-icon-inner" />
          </div>
          <Transition name="fade">
            <span v-if="!sidebarCollapsed" class="brand-name">digit-link</span>
          </Transition>
        </RouterLink>
        <button class="collapse-btn" @click="toggleSidebar">
          <ChevronLeft v-if="!sidebarCollapsed" class="w-4 h-4" />
          <ChevronRight v-else class="w-4 h-4" />
        </button>
      </div>

      <!-- Portal type badge -->
      <div class="sidebar-badge" :class="`sidebar-badge--${accentColor}`">
        <Building2 v-if="!isAdmin" class="w-3.5 h-3.5" />
        <ShieldCheck v-else class="w-3.5 h-3.5" />
        <Transition name="fade">
          <span v-if="!sidebarCollapsed">{{ portalTitle }}</span>
        </Transition>
      </div>

      <!-- Navigation -->
      <nav class="sidebar-nav">
        <button
          v-for="link in navLinks"
          :key="link.name"
          class="nav-item"
          :class="{ 'nav-item--active': currentRoute === link.name }"
          @click="navigateTo(link.name)"
          :title="sidebarCollapsed ? link.label : undefined"
        >
          <component :is="link.icon" class="nav-icon" />
          <Transition name="fade">
            <span v-if="!sidebarCollapsed" class="nav-label">{{ link.label }}</span>
          </Transition>
        </button>
      </nav>

      <!-- Spacer -->
      <div class="sidebar-spacer" />

      <!-- User section -->
      <div class="sidebar-user">
        <ThemeSwitcher class="hidden lg:flex" />
        <div class="user-info" v-if="!sidebarCollapsed">
          <div class="user-avatar" :class="`user-avatar--${accentColor}`">
            {{ username?.charAt(0).toUpperCase() || 'U' }}
          </div>
          <span class="user-name">{{ username }}</span>
        </div>
        <button class="logout-btn" @click="logout" :title="sidebarCollapsed ? 'Logout' : undefined">
          <LogOut class="w-4 h-4" />
          <Transition name="fade">
            <span v-if="!sidebarCollapsed">Logout</span>
          </Transition>
        </button>
      </div>
    </aside>

    <!-- Main content -->
    <main class="main-content" :class="{ 'main-content--expanded': sidebarCollapsed }">
      <RouterView />
    </main>

    <!-- Toast notifications -->
    <Toast :toasts="toasts" @dismiss="dismiss" />
  </div>
</template>

<style scoped>
.portal-shell {
  min-height: 100vh;
  display: flex;
  background: var(--bg-deep);
  position: relative;
  overflow: hidden;
}

/* Background effects */
.bg-pattern {
  position: fixed;
  inset: 0;
  background-image: 
    linear-gradient(var(--border-subtle) 1px, transparent 1px),
    linear-gradient(90deg, var(--border-subtle) 1px, transparent 1px);
  background-size: 60px 60px;
  opacity: 0.12;
  pointer-events: none;
  z-index: 0;
}

.bg-gradient {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  transition: background 0.6s ease;
}

.bg-gradient--primary {
  background: radial-gradient(ellipse at 30% 20%, rgba(var(--accent-primary-rgb), 0.08) 0%, transparent 50%),
              radial-gradient(ellipse at 70% 80%, rgba(var(--accent-primary-rgb), 0.05) 0%, transparent 50%);
}

.bg-gradient--secondary {
  background: radial-gradient(ellipse at 30% 20%, rgba(var(--accent-secondary-rgb), 0.08) 0%, transparent 50%),
              radial-gradient(ellipse at 70% 80%, rgba(var(--accent-secondary-rgb), 0.05) 0%, transparent 50%);
}

/* Decorative corners */
.corner {
  position: fixed;
  width: 100px;
  height: 100px;
  pointer-events: none;
  z-index: 1;
}

.corner::before,
.corner::after {
  content: '';
  position: absolute;
  background: var(--accent-primary);
  opacity: 0.25;
  transition: background 0.3s ease;
}

.corner-tl { top: 2rem; left: 2rem; }
.corner-tl::before { top: 0; left: 0; width: 50px; height: 2px; }
.corner-tl::after { top: 0; left: 0; width: 2px; height: 50px; }

.corner-br { bottom: 2rem; right: 2rem; }
.corner-br::before { bottom: 0; right: 0; width: 50px; height: 2px; }
.corner-br::after { bottom: 0; right: 0; width: 2px; height: 50px; }

/* Mobile header */
.mobile-header {
  display: none;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  height: 60px;
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border-subtle);
  padding: 0 1rem;
  align-items: center;
  gap: 1rem;
  z-index: 100;
}

.mobile-menu-btn {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.mobile-menu-btn:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
}

.mobile-brand {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.mobile-overlay {
  display: none;
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 199;
}

@media (max-width: 1023px) {
  .mobile-header {
    display: flex;
  }
  
  .mobile-overlay {
    display: block;
  }
  
  .main-content {
    margin-left: 0 !important;
    padding-top: 60px;
  }
  
  .sidebar {
    position: fixed;
    left: -280px;
    top: 0;
    bottom: 0;
    z-index: 200;
    transition: left 0.3s ease;
  }
  
  .sidebar--mobile-open {
    left: 0;
  }
  
  .collapse-btn {
    display: none !important;
  }
}

/* Sidebar */
.sidebar {
  width: 260px;
  background: var(--bg-surface);
  border-right: 1px solid var(--border-subtle);
  display: flex;
  flex-direction: column;
  padding: 1.25rem;
  transition: width 0.3s ease;
  flex-shrink: 0;
  position: relative;
  z-index: 10;
}

.sidebar--collapsed {
  width: 80px;
  padding: 1.25rem 0.75rem;
}

/* Brand */
.sidebar-brand {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1.5rem;
}

.brand-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  text-decoration: none;
  color: var(--text-primary);
}

.brand-icon {
  width: 36px;
  height: 36px;
  border: 2px solid var(--accent-primary);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  transition: border-color 0.3s ease;
}

.brand-icon--secondary {
  border-color: var(--accent-secondary);
}

.brand-icon-inner {
  width: 14px;
  height: 14px;
  background: var(--accent-primary);
  border-radius: 4px;
  transform: rotate(45deg);
  transition: background 0.3s ease;
}

.brand-icon--secondary .brand-icon-inner {
  background: var(--accent-secondary);
}

.brand-name {
  font-family: var(--font-display);
  font-size: 1.25rem;
  font-weight: 600;
  white-space: nowrap;
}

.collapse-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.collapse-btn:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
}

/* Badge */
.sidebar-badge {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 0.875rem;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 500;
  margin-bottom: 1.5rem;
  white-space: nowrap;
  overflow: hidden;
}

.sidebar-badge--primary {
  background: rgba(var(--accent-primary-rgb), 0.1);
  color: var(--accent-primary);
}

.sidebar-badge--secondary {
  background: rgba(var(--accent-secondary-rgb), 0.1);
  color: var(--accent-secondary);
}

.sidebar--collapsed .sidebar-badge {
  justify-content: center;
  padding: 0.625rem;
}

/* Navigation */
.sidebar-nav {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 0.875rem;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-secondary);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: left;
  width: 100%;
  white-space: nowrap;
}

.sidebar--collapsed .nav-item {
  justify-content: center;
  padding: 0.75rem;
}

.nav-item:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
}

.nav-item--active {
  background: rgba(var(--accent-primary-rgb), 0.1);
  color: var(--accent-primary);
}

.nav-item--active:hover {
  background: rgba(var(--accent-primary-rgb), 0.15);
}

.nav-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}

.nav-label {
  overflow: hidden;
}

/* Spacer */
.sidebar-spacer {
  flex: 1;
}

/* User section */
.sidebar-user {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-subtle);
  margin-top: 1rem;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 0;
}

.user-avatar {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.875rem;
  flex-shrink: 0;
}

.user-avatar--primary {
  background: rgba(var(--accent-primary-rgb), 0.2);
  color: var(--accent-primary);
}

.user-avatar--secondary {
  background: rgba(var(--accent-secondary-rgb), 0.2);
  color: var(--accent-secondary);
}

.user-name {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.logout-btn {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 0.875rem;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-muted);
  background: transparent;
  border: 1px solid var(--border-subtle);
  cursor: pointer;
  transition: all 0.2s ease;
  width: 100%;
}

.sidebar--collapsed .logout-btn {
  justify-content: center;
  padding: 0.75rem;
}

.logout-btn:hover {
  color: var(--accent-red);
  border-color: var(--accent-red);
  background: rgba(var(--accent-red-rgb), 0.05);
}

/* Main content */
.main-content {
  flex: 1;
  margin-left: 0;
  padding: 2rem;
  min-width: 0;
  max-width: 100%;
  position: relative;
  z-index: 10;
}

.main-content--expanded {
  margin-left: 0;
}

/* Transitions */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
