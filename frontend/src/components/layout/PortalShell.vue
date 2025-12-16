<script setup lang="ts">
import { computed, ref, onMounted, onUnmounted } from 'vue'
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
  X,
  ChevronUp
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { isAdmin, isOrgAdmin, username, currentOrgName } = usePortalContext()
const { toasts, dismiss } = useToast()

// Sidebar state
const sidebarCollapsed = ref(false)
const mobileMenuOpen = ref(false)
const profileMenuOpen = ref(false)
const profileFooterRef = ref<HTMLElement | null>(null)

// User initials for avatar
const userInitials = computed(() => {
  if (!username.value) return '?'
  return username.value
    .split(/[\s._-]+/)
    .map((part: string) => part[0]?.toUpperCase() || '')
    .slice(0, 2)
    .join('')
})

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

const orgLinks = computed(() => {
  const links = [
    { name: 'org-dashboard', label: 'Dashboard', icon: LayoutDashboard },
    { name: 'org-applications', label: 'Applications', icon: AppWindow },
    { name: 'org-api-keys', label: 'API Keys', icon: KeyRound },
    { name: 'org-whitelist', label: 'Whitelist', icon: ShieldCheck },
  ]
  
  // Only show Accounts link for org admins
  if (isOrgAdmin.value) {
    links.push({ name: 'org-accounts', label: 'Accounts', icon: Users })
  }
  
  // Add Settings for all users (My Account moved to profile footer)
  links.push({ name: 'org-settings', label: 'Settings', icon: Settings })
  
  return links
})

const navLinks = computed(() => isAdmin.value ? adminLinks : orgLinks.value)

const currentRoute = computed(() => {
  const name = route.name as string
  // Handle detail pages showing parent as active
  if (name?.includes('application-detail')) {
    return isAdmin.value ? 'admin-applications' : 'org-applications'
  }
  if (name?.includes('account-detail')) {
    return isAdmin.value ? 'admin-accounts' : 'org-accounts'
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
  profileMenuOpen.value = false
}

function toggleProfileMenu() {
  profileMenuOpen.value = !profileMenuOpen.value
}

function goToMyAccount() {
  router.push({ name: 'org-my-account' })
  profileMenuOpen.value = false
  mobileMenuOpen.value = false
}

// Click outside handler for profile menu
function handleClickOutside(event: MouseEvent) {
  if (profileMenuOpen.value && profileFooterRef.value && !profileFooterRef.value.contains(event.target as Node)) {
    profileMenuOpen.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
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
        class="w-10 h-10 flex items-center justify-center border border-border-subtle rounded-sm bg-transparent text-text-secondary cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary"
        @click="toggleMobileMenu"
      >
        <Menu v-if="!mobileMenuOpen" class="w-5 h-5" />
        <X v-else class="w-5 h-5" />
      </button>
      <div class="flex-1 flex items-center gap-3">
        <div 
          class="w-9 h-9 border-2 rounded-sm flex items-center justify-center shrink-0 transition-colors duration-300"
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
            class="w-9 h-9 border-2 rounded-sm flex items-center justify-center shrink-0 transition-colors duration-300"
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
          class="collapse-btn w-7 h-7 flex items-center justify-center border border-border-subtle rounded-sm bg-transparent text-text-muted cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary"
          @click="toggleSidebar"
        >
          <ChevronLeft v-if="!sidebarCollapsed" class="w-4 h-4" />
          <ChevronRight v-else class="w-4 h-4" />
        </button>
      </div>

      <!-- Portal type badge -->
      <div 
        class="flex items-center gap-2 py-2.5 px-3.5 rounded-sm text-xs font-medium mb-6 whitespace-nowrap overflow-hidden"
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
          class="flex items-center gap-3 py-3 px-3.5 rounded-sm text-sm font-medium text-text-secondary bg-transparent border-none cursor-pointer transition-all duration-200 text-left w-full whitespace-nowrap hover:bg-bg-elevated hover:text-text-primary"
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

      <!-- Profile Footer -->
      <div ref="profileFooterRef" class="profile-footer" :class="{ 'profile-footer--collapsed': sidebarCollapsed }">
        <!-- Profile Trigger / Compact view when collapsed -->
        <div class="relative">
          <button 
            class="profile-trigger"
            :class="{ 
              'profile-trigger--collapsed': sidebarCollapsed,
              'profile-trigger--active': profileMenuOpen 
            }"
            @click="toggleProfileMenu"
            :title="sidebarCollapsed ? username || 'Profile' : undefined"
          >
            <!-- Avatar -->
            <div 
              class="profile-avatar"
              :class="accentColor === 'primary' ? 'profile-avatar--primary' : 'profile-avatar--secondary'"
            >
              <span class="profile-avatar-initials">{{ userInitials }}</span>
            </div>
            
            <!-- User Info (hidden when collapsed) -->
            <Transition name="fade">
              <div v-if="!sidebarCollapsed" class="profile-info">
                <span class="profile-username">{{ username || 'User' }}</span>
                <span v-if="currentOrgName && !isAdmin" class="profile-org">{{ currentOrgName }}</span>
                <span v-else-if="isAdmin" class="profile-role">Administrator</span>
              </div>
            </Transition>
            
            <!-- Chevron (hidden when collapsed) -->
            <Transition name="fade">
              <ChevronUp 
                v-if="!sidebarCollapsed" 
                class="profile-chevron"
                :class="{ 'profile-chevron--open': profileMenuOpen }"
              />
            </Transition>
          </button>

          <!-- Profile Dropdown Menu -->
          <Transition name="profile-menu">
            <div 
              v-if="profileMenuOpen" 
              class="profile-menu"
              :class="{ 'profile-menu--collapsed': sidebarCollapsed }"
            >
              <!-- My Account (org users only) -->
              <button 
                v-if="!isAdmin"
                class="profile-menu-item"
                @click="goToMyAccount"
              >
                <Settings class="w-4 h-4" />
                <span>My Account</span>
              </button>
              
              <!-- Theme Switcher -->
              <div class="profile-menu-theme" @click.stop>
                <ThemeSwitcher class="theme-switcher-full" />
              </div>
              
              <!-- Divider -->
              <div class="profile-menu-divider" />
              
              <!-- Logout -->
              <button 
                class="profile-menu-item profile-menu-item--danger"
                @click="logout"
              >
                <LogOut class="w-4 h-4" />
                <span>Logout</span>
              </button>
            </div>
          </Transition>
        </div>
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
  @apply w-[260px] bg-bg-surface border-r border-border-subtle flex flex-col p-5 transition-[width] duration-300 shrink-0 relative z-20;
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

/* Profile Footer */
.profile-footer {
  @apply pt-4 border-t border-border-subtle mt-4 relative z-30;
}

.profile-trigger {
  @apply w-full flex items-center gap-3 p-2.5 rounded-sm bg-bg-elevated/50 border border-transparent 
         cursor-pointer transition-all duration-200 text-left;
}

.profile-trigger:hover {
  @apply bg-bg-elevated border-border-subtle;
}

.profile-trigger--active {
  @apply bg-bg-elevated border-border-accent;
}

.profile-trigger--collapsed {
  @apply justify-center p-2;
}

.profile-avatar {
  @apply w-9 h-9 rounded-sm flex items-center justify-center shrink-0 
         transition-all duration-300 relative overflow-hidden;
}

.profile-avatar::before {
  content: '';
  @apply absolute inset-0 opacity-20;
}

.profile-avatar--primary {
  background: linear-gradient(135deg, rgba(var(--accent-primary-rgb), 0.2), rgba(var(--accent-primary-rgb), 0.1));
  border: 1px solid rgba(var(--accent-primary-rgb), 0.3);
}

.profile-avatar--primary .profile-avatar-initials {
  @apply text-accent-primary;
}

.profile-avatar--secondary {
  background: linear-gradient(135deg, rgba(var(--accent-secondary-rgb), 0.2), rgba(var(--accent-secondary-rgb), 0.1));
  border: 1px solid rgba(var(--accent-secondary-rgb), 0.3);
}

.profile-avatar--secondary .profile-avatar-initials {
  @apply text-accent-secondary;
}

.profile-avatar-initials {
  @apply text-xs font-semibold uppercase tracking-wide;
}

.profile-info {
  @apply flex-1 min-w-0 flex flex-col;
}

.profile-username {
  @apply text-sm font-medium text-text-primary truncate leading-tight;
}

.profile-org,
.profile-role {
  @apply text-[11px] text-text-muted truncate leading-tight mt-0.5;
}

.profile-chevron {
  @apply w-4 h-4 text-text-muted transition-transform duration-200 shrink-0;
}

.profile-chevron--open {
  @apply rotate-180;
}

/* Profile Dropdown Menu */
.profile-menu {
  @apply absolute bottom-full left-0 right-0 mb-2 py-1.5 
         bg-bg-surface border border-border-subtle rounded-sm shadow-lg z-[199]
         min-w-[180px];
}

.profile-menu--collapsed {
  @apply left-0 right-auto min-w-[200px];
}

.profile-menu-item {
  @apply w-full flex items-center gap-3 py-2.5 px-3.5 text-sm text-text-secondary 
         bg-transparent border-none transition-all duration-150 text-left;
  cursor: pointer !important;
}

.profile-menu-item:hover {
  @apply bg-bg-elevated text-text-primary;
}

.profile-menu-item--danger {
  cursor: pointer !important;
}

.profile-menu-item--danger:hover {
  @apply text-accent-red;
  background: rgba(var(--accent-red-rgb), 0.08);
}

.profile-menu-theme {
  @apply px-2 py-1;
}

/* Full-width theme switcher styling */
.theme-switcher-full {
  @apply w-full;
}

.theme-switcher-full :deep(button) {
  @apply w-full justify-start;
}

/* Override theme dropdown to open to the right */
.theme-switcher-full :deep(.absolute) {
  @apply left-0 right-auto;
}

.profile-menu-divider {
  @apply h-px bg-border-subtle my-1.5 mx-3;
}

/* Profile Menu Transition */
.profile-menu-enter-active,
.profile-menu-leave-active {
  @apply transition-all duration-150 ease-out;
}

.profile-menu-enter-from,
.profile-menu-leave-to {
  @apply opacity-0 translate-y-2;
}
</style>
