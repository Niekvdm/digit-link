import { computed } from 'vue'
import { useAuthStore } from '@/stores/authStore'

/**
 * Provides context about the current portal user
 * Used by unified views to adapt behavior for admin vs org users
 */
export function usePortalContext() {
  const authStore = useAuthStore()

  /**
   * Whether the current user is an admin
   */
  const isAdmin = computed(() => authStore.isAdmin)

  /**
   * Whether the current user is an organization user (not admin)
   */
  const isOrgUser = computed(() => authStore.isOrgUser)

  /**
   * The organization ID for org users (undefined for admins)
   */
  const currentOrgId = computed(() => authStore.orgId)

  /**
   * The organization name for org users (undefined for admins)
   */
  const currentOrgName = computed(() => authStore.orgName)

  /**
   * The current username
   */
  const username = computed(() => authStore.username)

  /**
   * Whether the user is authenticated
   */
  const isAuthenticated = computed(() => authStore.isAuthenticated)

  /**
   * Get the appropriate layout component name based on user type
   */
  const layoutName = computed(() => isAdmin.value ? 'AppLayout' : 'OrgLayout')

  /**
   * Get the base path for navigation based on user type
   * Admin users use root paths, org users use /portal prefix for org-specific routes
   */
  const basePath = computed(() => isAdmin.value ? '' : '/portal')

  /**
   * Get the dashboard route name based on user type
   */
  const dashboardRoute = computed(() => isAdmin.value ? 'dashboard' : 'org-dashboard')

  return {
    isAdmin,
    isOrgUser,
    currentOrgId,
    currentOrgName,
    username,
    isAuthenticated,
    layoutName,
    basePath,
    dashboardRoute
  }
}
