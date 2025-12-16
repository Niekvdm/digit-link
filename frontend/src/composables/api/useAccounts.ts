import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import type { 
  Account, 
  AccountsResponse,
  CreateAccountRequest,
  CreateAccountResponse,
  RegenerateTokenResponse,
  SetAccountOrgResponse,
  DeleteResponse
} from '@/types/api'

interface AccountResponse {
  account: Account
}

export function useAccounts() {
  const api = useApi()
  
  const accounts = ref<Account[]>([])
  const currentAccount = ref<Account | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    
    try {
      const res = await api.get<AccountsResponse>('/admin/accounts')
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
      const res = await api.get<AccountResponse>(`/admin/accounts/${accountId}`)
      currentAccount.value = res.account
      return res.account
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load account'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function create(data: CreateAccountRequest) {
    const res = await api.post<CreateAccountResponse>('/admin/accounts', data)
    if (res.success) {
      await fetchAll() // Refresh list
      return res.token
    }
    throw new Error(res.error || 'Failed to create account')
  }

  async function remove(accountId: string) {
    await api.del<DeleteResponse>(`/admin/accounts/${accountId}`)
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, active: false } : acc
    )
  }

  async function activate(accountId: string) {
    await api.post(`/admin/accounts/${accountId}/activate`)
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, active: true } : acc
    )
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = { ...currentAccount.value, active: true }
    }
  }

  async function regenerateToken(accountId: string) {
    const res = await api.post<RegenerateTokenResponse>(`/admin/accounts/${accountId}/regenerate`)
    if (res.success) {
      return res.token
    }
    throw new Error(res.error || 'Failed to regenerate token')
  }

  async function setOrganization(accountId: string, orgId: string) {
    const res = await api.put<SetAccountOrgResponse>(`/admin/accounts/${accountId}/organization`, { orgId })
    if (res.success) {
      accounts.value = accounts.value.map(acc => 
        acc.id === accountId ? { ...acc, orgId: res.orgId, orgName: res.orgName } : acc
      )
      if (currentAccount.value?.id === accountId) {
        currentAccount.value = { ...currentAccount.value, orgId: res.orgId, orgName: res.orgName }
      }
    }
    return res
  }

  async function setPassword(accountId: string, password: string) {
    await api.put(`/admin/accounts/${accountId}/password`, { password })
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, hasPassword: true } : acc
    )
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = { ...currentAccount.value, hasPassword: true }
    }
  }

  async function updateUsername(accountId: string, username: string) {
    await api.put(`/admin/accounts/${accountId}/username`, { username })
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, username } : acc
    )
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = { ...currentAccount.value, username }
    }
  }

  async function setOrgAdmin(accountId: string, isOrgAdmin: boolean) {
    await api.put(`/admin/accounts/${accountId}/org-admin`, { isOrgAdmin })
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, isOrgAdmin } : acc
    )
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = { ...currentAccount.value, isOrgAdmin }
    }
  }

  async function hardDelete(accountId: string) {
    await api.del(`/admin/accounts/${accountId}/hard`)
    accounts.value = accounts.value.filter(acc => acc.id !== accountId)
    if (currentAccount.value?.id === accountId) {
      currentAccount.value = null
    }
  }

  return {
    accounts: readonly(accounts),
    currentAccount: readonly(currentAccount),
    loading: readonly(loading),
    error: readonly(error),
    fetchAll,
    fetchOne,
    create,
    remove,
    activate,
    regenerateToken,
    setOrganization,
    setPassword,
    updateUsername,
    setOrgAdmin,
    hardDelete
  }
}
