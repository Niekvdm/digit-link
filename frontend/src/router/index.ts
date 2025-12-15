import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    name: 'login',
    component: () => import('@/views/LoginView.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/org-login',
    name: 'org-login',
    component: () => import('@/views/OrgLoginView.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/setup',
    name: 'setup',
    component: () => import('@/views/SetupView.vue'),
    meta: { requiresAuth: false }
  },
  // Admin routes
  {
    path: '/dashboard',
    name: 'dashboard',
    component: () => import('@/views/DashboardView.vue'),
    meta: { requiresAuth: true, userType: 'admin' }
  },
  {
    path: '/organizations',
    name: 'organizations',
    component: () => import('@/views/OrganizationsView.vue'),
    meta: { requiresAuth: true, userType: 'admin' }
  },
  {
    path: '/applications',
    name: 'applications',
    component: () => import('@/views/ApplicationsView.vue'),
    meta: { requiresAuth: true, userType: 'admin' }
  },
  {
    path: '/applications/:id',
    name: 'application-detail',
    component: () => import('@/views/ApplicationDetailView.vue'),
    meta: { requiresAuth: true, userType: 'admin' }
  },
  {
    path: '/api-keys',
    name: 'api-keys',
    component: () => import('@/views/APIKeysView.vue'),
    meta: { requiresAuth: true, userType: 'admin' }
  },
  {
    path: '/accounts',
    name: 'accounts',
    component: () => import('@/views/AccountsView.vue'),
    meta: { requiresAuth: true, userType: 'admin' }
  },
  {
    path: '/tunnels',
    name: 'tunnels',
    component: () => import('@/views/TunnelsView.vue'),
    meta: { requiresAuth: true, userType: 'admin' }
  },
  {
    path: '/whitelist',
    name: 'whitelist',
    component: () => import('@/views/WhitelistView.vue'),
    meta: { requiresAuth: true, userType: 'admin' }
  },
  {
    path: '/audit',
    name: 'audit',
    component: () => import('@/views/AuditLogsView.vue'),
    meta: { requiresAuth: true, userType: 'admin' }
  },
  // Org Portal routes
  {
    path: '/portal',
    name: 'org-dashboard',
    component: () => import('@/views/org/OrgDashboardView.vue'),
    meta: { requiresAuth: true, userType: 'org' }
  },
  {
    path: '/portal/applications',
    name: 'org-applications',
    component: () => import('@/views/org/OrgApplicationsView.vue'),
    meta: { requiresAuth: true, userType: 'org' }
  },
  {
    path: '/portal/applications/:id',
    name: 'org-application-detail',
    component: () => import('@/views/org/OrgApplicationDetailView.vue'),
    meta: { requiresAuth: true, userType: 'org' }
  },
  {
    path: '/portal/whitelist',
    name: 'org-whitelist',
    component: () => import('@/views/org/OrgWhitelistView.vue'),
    meta: { requiresAuth: true, userType: 'org' }
  },
  {
    path: '/portal/api-keys',
    name: 'org-api-keys',
    component: () => import('@/views/org/OrgAPIKeysView.vue'),
    meta: { requiresAuth: true, userType: 'org' }
  },
  {
    path: '/portal/settings',
    name: 'org-settings',
    component: () => import('@/views/org/OrgSettingsView.vue'),
    meta: { requiresAuth: true, userType: 'org' }
  },
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
  const requiredUserType = to.meta.userType as string | undefined

  if (requiresAuth && !authStore.isAuthenticated) {
    // Redirect to appropriate login based on where user was trying to go
    if (requiredUserType === 'org') {
      next({ name: 'org-login' })
    } else {
      next({ name: 'login' })
    }
  } else if (requiresAuth && requiredUserType) {
    // Check user type matches
    if (requiredUserType === 'admin' && !authStore.isAdmin) {
      // Org user trying to access admin pages
      next({ name: 'org-dashboard' })
    } else if (requiredUserType === 'org' && !authStore.isOrgUser) {
      // Admin user trying to access org pages
      next({ name: 'dashboard' })
    } else {
      next()
    }
  } else if ((to.name === 'login' || to.name === 'org-login') && authStore.isAuthenticated) {
    // Redirect to appropriate dashboard if already authenticated
    const isValid = await authStore.validateToken()
    if (isValid) {
      if (authStore.isOrgUser) {
        next({ name: 'org-dashboard' })
      } else {
        next({ name: 'dashboard' })
      }
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router
