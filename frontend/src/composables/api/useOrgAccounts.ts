import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import type { Account } from '@/types/api'

interface OrgAccountsResponse {
  accounts: Account[]
}

interface OrgAccountResponse {
  account: Account
}

interface CreateOrgAccountRequest {
  username: string
  password?: string
  isOrgAdmin?: boolean
}

interface CreateOrgAccountResponse {
  success: boolean
  account?: Account
  token?: string
  error?: string
}

interface RegenerateTokenResponse {
  success: boolean
  token?: string
  error?: string
}

export function useOrgAccounts() {
  const api = useApi()
  
  const accounts = ref<Account[]>([])
  const currentAccount = ref<Account | null>(null)
  const myAccount = ref<Account | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  // ============================================
  // Self-management (all org users)
  // ============================================

  async function fetchMyAccount() {
    loading.value = true
    error.value = null
    
    try {
      const res = await api.get<OrgAccountResponse>('/org/accounts/me')
      myAccount.value = res.account
      return res.account
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load account'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateMyAccount(username: string) {
    await api.put('/org/accounts/me', { username })
    if (myAccount.value) {
      myAccount.value = { ...myAccount.value, username }
    }
  }

  async function setMyPassword(password: string) {
    await api.put('/org/accounts/me/password', { password })
    if (myAccount.value) {
      myAccount.value = { ...myAccount.value, hasPassword: true }
    }
  }

  // ============================================
  // Org Admin functions
  // ============================================

  async function fetchAll() {
    loading.value = true
    error.value = null
    
    try {
      const res = await api.get<OrgAccountsResponse>('/org/accounts')
      accounts.value = res.accounts
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load accounts'
    } finally {
      loading.value = false
    }
  }

  async function fetchOne(accountId: string) {
    loading.value = true
    error.value = null
    
    try {
      const res = await api.get<OrgAccountResponse>(`/org/accounts/${accountId}`)
      currentAccount.value = res.account
      return res.account
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load account'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function create(data: CreateOrgAccountRequest) {
    const res = await api.post<CreateOrgAccountResponse>('/org/accounts', data)
    if (res.success) {
      await fetchAll()
      return res.token
    }
    throw new Error(res.error || 'Failed to create account')
  }

  async function updateAccount(accountId: string, username: string) {
    await api.put(`/org/accounts/${accountId}`, { username })
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, username } : acc
    )
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = { ...currentAccount.value, username }
    }
  }

  async function setPassword(accountId: string, password: string) {
    await api.put(`/org/accounts/${accountId}/password`, { password })
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, hasPassword: true } : acc
    )
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = { ...currentAccount.value, hasPassword: true }
    }
  }

  async function regenerateToken(accountId: string) {
    const res = await api.post<RegenerateTokenResponse>(`/org/accounts/${accountId}/regenerate`)
    if (res.success) {
      return res.token
    }
    throw new Error(res.error || 'Failed to regenerate token')
  }

  async function setOrgAdmin(accountId: string, isOrgAdmin: boolean) {
    await api.put(`/org/accounts/${accountId}/org-admin`, { isOrgAdmin })
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, isOrgAdmin } : acc
    )
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = { ...currentAccount.value, isOrgAdmin }
    }
  }

  async function activate(accountId: string) {
    await api.post(`/org/accounts/${accountId}/activate`)
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, active: true } : acc
    )
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = { ...currentAccount.value, active: true }
    }
  }

  async function deactivate(accountId: string) {
    await api.del(`/org/accounts/${accountId}`)
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, active: false } : acc
    )
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = { ...currentAccount.value, active: false }
    }
  }

  async function hardDelete(accountId: string) {
    await api.del(`/org/accounts/${accountId}/hard`)
    accounts.value = accounts.value.filter(acc => acc.id !== accountId)
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = null
    }
  }

  return {
    // State
    accounts: readonly(accounts),
    currentAccount: readonly(currentAccount),
    myAccount: readonly(myAccount),
    loading: readonly(loading),
    error: readonly(error),
    
    // Self-management
    fetchMyAccount,
    updateMyAccount,
    setMyPassword,
    
    // Org admin
    fetchAll,
    fetchOne,
    create,
    updateAccount,
    setPassword,
    regenerateToken,
    setOrgAdmin,
    activate,
    deactivate,
    hardDelete
  }
}
