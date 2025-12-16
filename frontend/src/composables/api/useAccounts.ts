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

export function useAccounts() {
  const api = useApi()
  
  const accounts = ref<Account[]>([])
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
    }
    return res
  }

  async function setPassword(accountId: string, password: string) {
    await api.put(`/admin/accounts/${accountId}/password`, { password })
    accounts.value = accounts.value.map(acc => 
      acc.id === accountId ? { ...acc, hasPassword: true } : acc
    )
  }

  return {
    accounts: readonly(accounts),
    loading: readonly(loading),
    error: readonly(error),
    fetchAll,
    create,
    remove,
    activate,
    regenerateToken,
    setOrganization,
    setPassword
  }
}
