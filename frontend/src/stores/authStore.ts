import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

const TOKEN_KEY = 'digit-link-token'
const USER_TYPE_KEY = 'digit-link-user-type'
const ORG_ID_KEY = 'digit-link-org-id'

export type UserType = 'admin' | 'org'

export const useAuthStore = defineStore('auth', () => {
  // State
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const userType = ref<UserType>((localStorage.getItem(USER_TYPE_KEY) as UserType) || 'admin')
  const orgId = ref<string | null>(localStorage.getItem(ORG_ID_KEY))

  // Getters
  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => userType.value === 'admin')
  const isOrgUser = computed(() => userType.value === 'org')

  // Actions
  function setToken(newToken: string, type: UserType = 'admin', organizationId?: string) {
    token.value = newToken
    userType.value = type
    localStorage.setItem(TOKEN_KEY, newToken)
    localStorage.setItem(USER_TYPE_KEY, type)
    if (organizationId) {
      orgId.value = organizationId
      localStorage.setItem(ORG_ID_KEY, organizationId)
    }
  }

  function clearToken() {
    token.value = null
    userType.value = 'admin'
    orgId.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_TYPE_KEY)
    localStorage.removeItem(ORG_ID_KEY)
  }

  async function validateToken(): Promise<boolean> {
    if (!token.value) return false

    try {
      // Use appropriate endpoint based on user type
      const endpoint = userType.value === 'org' ? '/org/stats' : '/admin/stats'
      const response = await fetch(endpoint, {
        headers: {
          'Authorization': `Bearer ${token.value}`
        }
      })
      
      if (!response.ok) {
        clearToken()
        return false
      }
      
      return true
    } catch {
      return false
    }
  }

  return {
    token,
    userType,
    orgId,
    isAuthenticated,
    isAdmin,
    isOrgUser,
    setToken,
    clearToken,
    validateToken
  }
})
