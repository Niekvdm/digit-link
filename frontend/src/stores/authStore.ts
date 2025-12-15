import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

const TOKEN_KEY = 'digit-link-token'

export const useAuthStore = defineStore('auth', () => {
  // State
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))

  // Getters
  const isAuthenticated = computed(() => !!token.value)

  // Actions
  function setToken(newToken: string) {
    token.value = newToken
    localStorage.setItem(TOKEN_KEY, newToken)
  }

  function clearToken() {
    token.value = null
    localStorage.removeItem(TOKEN_KEY)
  }

  async function validateToken(): Promise<boolean> {
    if (!token.value) return false

    try {
      const response = await fetch('/admin/stats', {
        headers: {
          'X-Admin-Token': token.value
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
    isAuthenticated,
    setToken,
    clearToken,
    validateToken
  }
})
