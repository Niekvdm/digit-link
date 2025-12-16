import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'

// Lazy-load views for better performance
const UnifiedLoginView = () => import('@/views/UnifiedLoginView.vue')
const SetupView = () => import('@/views/SetupView.vue')
const PortalShell = () => import('@/components/layout/PortalShell.vue')

// Admin pages
const AdminDashboard = () => import('@/views/admin/DashboardPage.vue')
const AdminOrganizations = () => import('@/views/admin/OrganizationsPage.vue')
const AdminApplications = () => import('@/views/admin/ApplicationsPage.vue')
const AdminApplicationDetail = () => import('@/views/admin/ApplicationDetailPage.vue')
const AdminAccounts = () => import('@/views/admin/AccountsPage.vue')
const AdminAccountDetail = () => import('@/views/admin/AccountDetailPage.vue')
const AdminMyAccount = () => import('@/views/admin/MyAccountPage.vue')
const AdminAPIKeys = () => import('@/views/admin/APIKeysPage.vue')
const AdminWhitelist = () => import('@/views/admin/WhitelistPage.vue')
const AdminTunnels = () => import('@/views/admin/TunnelsPage.vue')
const AdminAudit = () => import('@/views/admin/AuditPage.vue')
const AdminPlans = () => import('@/views/admin/PlansPage.vue')
const AdminUsage = () => import('@/views/admin/UsagePage.vue')

// Org pages
const OrgDashboard = () => import('@/views/org/DashboardPage.vue')
const OrgApplications = () => import('@/views/org/ApplicationsPage.vue')
const OrgApplicationDetail = () => import('@/views/org/ApplicationDetailPage.vue')
const OrgAccounts = () => import('@/views/org/AccountsPage.vue')
const OrgAccountDetail = () => import('@/views/org/AccountDetailPage.vue')
const OrgMyAccount = () => import('@/views/org/MyAccountPage.vue')
const OrgAPIKeys = () => import('@/views/org/APIKeysPage.vue')
const OrgWhitelist = () => import('@/views/org/WhitelistPage.vue')
const OrgSettings = () => import('@/views/org/SettingsPage.vue')
const OrgUsage = () => import('@/views/org/UsagePage.vue')

const routes: RouteRecordRaw[] = [
  // Public routes
  {
    path: '/',
    name: 'login',
    component: UnifiedLoginView,
    meta: { requiresAuth: false }
  },
  {
    path: '/setup',
    name: 'setup',
    component: SetupView,
    meta: { requiresAuth: false }
  },

  // Admin Portal
  {
    path: '/admin',
    component: PortalShell,
    meta: { requiresAuth: true, role: 'admin' },
    children: [
      {
        path: '',
        name: 'admin-dashboard',
        component: AdminDashboard
      },
      {
        path: 'organizations',
        name: 'admin-organizations',
        component: AdminOrganizations
      },
      {
        path: 'applications',
        name: 'admin-applications',
        component: AdminApplications
      },
      {
        path: 'applications/:appId',
        name: 'admin-application-detail',
        component: AdminApplicationDetail,
        props: true
      },
      {
        path: 'accounts',
        name: 'admin-accounts',
        component: AdminAccounts
      },
      {
        path: 'accounts/:accountId',
        name: 'admin-account-detail',
        component: AdminAccountDetail,
        props: true
      },
      {
        path: 'my-account',
        name: 'admin-my-account',
        component: AdminMyAccount
      },
      {
        path: 'api-keys',
        name: 'admin-api-keys',
        component: AdminAPIKeys
      },
      {
        path: 'whitelist',
        name: 'admin-whitelist',
        component: AdminWhitelist
      },
      {
        path: 'tunnels',
        name: 'admin-tunnels',
        component: AdminTunnels
      },
      {
        path: 'audit',
        name: 'admin-audit',
        component: AdminAudit
      },
      {
        path: 'plans',
        name: 'admin-plans',
        component: AdminPlans
      },
      {
        path: 'usage',
        name: 'admin-usage',
        component: AdminUsage
      }
    ]
  },

  // Organization Portal
  {
    path: '/portal',
    component: PortalShell,
    meta: { requiresAuth: true, role: 'org' },
    children: [
      {
        path: '',
        name: 'org-dashboard',
        component: OrgDashboard
      },
      {
        path: 'applications',
        name: 'org-applications',
        component: OrgApplications
      },
      {
        path: 'applications/:appId',
        name: 'org-application-detail',
        component: OrgApplicationDetail,
        props: true
      },
      {
        path: 'api-keys',
        name: 'org-api-keys',
        component: OrgAPIKeys
      },
      {
        path: 'whitelist',
        name: 'org-whitelist',
        component: OrgWhitelist
      },
      {
        path: 'accounts',
        name: 'org-accounts',
        component: OrgAccounts
      },
      {
        path: 'accounts/:accountId',
        name: 'org-account-detail',
        component: OrgAccountDetail,
        props: true
      },
      {
        path: 'my-account',
        name: 'org-my-account',
        component: OrgMyAccount
      },
      {
        path: 'settings',
        name: 'org-settings',
        component: OrgSettings
      },
      {
        path: 'usage',
        name: 'org-usage',
        component: OrgUsage
      }
    ]
  },

  // Legacy redirects for backwards compatibility
  {
    path: '/dashboard',
    redirect: '/admin'
  },
  {
    path: '/organizations',
    redirect: '/admin/organizations'
  },
  {
    path: '/applications',
    redirect: '/admin/applications'
  },
  {
    path: '/accounts',
    redirect: '/admin/accounts'
  },
  {
    path: '/tunnels',
    redirect: '/admin/tunnels'
  },
  {
    path: '/audit',
    redirect: '/admin/audit'
  },
  {
    path: '/whitelist',
    redirect: '/admin/whitelist'
  },
  {
    path: '/api-keys',
    redirect: '/admin/api-keys'
  },

  // Catch-all
  {
    path: '/:pathMatch(.*)*',
    redirect: '/'
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guard
router.beforeEach(async (to, _from, next) => {
  const authStore = useAuthStore()
  const requiresAuth = to.meta.requiresAuth as boolean
  const requiredRole = to.meta.role as 'admin' | 'org' | undefined

  // Public routes
  if (!requiresAuth) {
    if (to.name === 'login' && authStore.isAuthenticated) {
      // Redirect to appropriate dashboard if already authenticated
      const isValid = await authStore.validateToken()
      if (isValid) {
        next(authStore.isOrgUser ? { name: 'org-dashboard' } : { name: 'admin-dashboard' })
        return
      }
    }
    next()
    return
  }

  // Protected routes - check authentication
  if (!authStore.isAuthenticated) {
    next({ name: 'login' })
    return
  }

  // Check role requirements
  if (requiredRole === 'admin' && !authStore.isAdmin) {
    // Org user trying to access admin routes
    next({ name: 'org-dashboard' })
    return
  }

  if (requiredRole === 'org' && !authStore.isOrgUser) {
    // Admin user trying to access org routes
    next({ name: 'admin-dashboard' })
    return
  }

  next()
})

export default router
