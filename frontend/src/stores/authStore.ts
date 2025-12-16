import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

const TOKEN_KEY = 'digit-link-token'
const USER_TYPE_KEY = 'digit-link-user-type'
const ORG_ID_KEY = 'digit-link-org-id'
const ORG_NAME_KEY = 'digit-link-org-name'
const USERNAME_KEY = 'digit-link-username'

export type UserType = 'admin' | 'org'

export const useAuthStore = defineStore('auth', () => {
  // State
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))
  const userType = ref<UserType>((localStorage.getItem(USER_TYPE_KEY) as UserType) || 'admin')
  const orgId = ref<string | null>(localStorage.getItem(ORG_ID_KEY))
  const orgName = ref<string | null>(localStorage.getItem(ORG_NAME_KEY))
  const username = ref<string | null>(localStorage.getItem(USERNAME_KEY))

  // Getters
  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => userType.value === 'admin')
  const isOrgUser = computed(() => userType.value === 'org')

  // Actions
  function setToken(
    newToken: string,
    type: UserType = 'admin',
    organizationId?: string,
    organizationName?: string,
    user?: string
  ) {
    token.value = newToken
    userType.value = type
    localStorage.setItem(TOKEN_KEY, newToken)
    localStorage.setItem(USER_TYPE_KEY, type)
    if (organizationId) {
      orgId.value = organizationId
      localStorage.setItem(ORG_ID_KEY, organizationId)
    } else {
      orgId.value = null
      localStorage.removeItem(ORG_ID_KEY)
    }
    if (organizationName) {
      orgName.value = organizationName
      localStorage.setItem(ORG_NAME_KEY, organizationName)
    } else {
      orgName.value = null
      localStorage.removeItem(ORG_NAME_KEY)
    }
    if (user) {
      username.value = user
      localStorage.setItem(USERNAME_KEY, user)
    } else {
      username.value = null
      localStorage.removeItem(USERNAME_KEY)
    }
  }

  function clearToken() {
    token.value = null
    userType.value = 'admin'
    orgId.value = null
    orgName.value = null
    username.value = null
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_TYPE_KEY)
    localStorage.removeItem(ORG_ID_KEY)
    localStorage.removeItem(ORG_NAME_KEY)
    localStorage.removeItem(USERNAME_KEY)
  }

  async function validateToken(): Promise<boolean> {
    if (!token.value) return false

    try {
      // Use appropriate endpoint and header based on user type
      const endpoint = userType.value === 'org' ? '/org/stats' : '/admin/stats'
      const headers: Record<string, string> = userType.value === 'org' 
        ? { 'Authorization': `Bearer ${token.value}` }
        : { 'X-Admin-Token': token.value }
      
      const response = await fetch(endpoint, { headers })
      
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
    orgName,
    username,
    isAuthenticated,
    isAdmin,
    isOrgUser,
    setToken,
    clearToken,
    validateToken
  }
})
