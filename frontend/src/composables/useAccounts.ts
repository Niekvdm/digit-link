import { ref, onMounted } from 'vue'
import { useApi } from './useApi'
import type { 
  Account, 
  AccountsResponse, 
  CreateAccountRequest, 
  CreateAccountResponse,
  RegenerateTokenResponse,
  SetAccountOrgRequest,
  SetAccountOrgResponse,
  SetAccountPasswordRequest,
  SetAccountPasswordResponse,
  DeleteResponse 
} from '@/types/api'

export function useAccounts() {
  const { get, post, put, del } = useApi()
  
  const accounts = ref<Account[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadAccounts() {
    loading.value = true
    error.value = null
    
    try {
      const data = await get<AccountsResponse>('/admin/accounts')
      accounts.value = data.accounts || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load accounts'
    } finally {
      loading.value = false
    }
  }

  async function createAccount(request: CreateAccountRequest): Promise<CreateAccountResponse> {
    try {
      const response = await post<CreateAccountResponse>('/admin/accounts', request)
      if (response.success) {
        await loadAccounts()
      }
      return response
    } catch (err) {
      return { 
        success: false, 
        error: err instanceof Error ? err.message : 'Failed to create account' 
      }
    }
  }

  async function regenerateToken(accountId: string): Promise<RegenerateTokenResponse> {
    try {
      const response = await post<RegenerateTokenResponse>(
        `/admin/accounts/${accountId}/regenerate`
      )
      return response
    } catch (err) {
      return { 
        success: false, 
        error: err instanceof Error ? err.message : 'Failed to regenerate token' 
      }
    }
  }

  async function setAccountOrganization(accountId: string, orgId: string): Promise<SetAccountOrgResponse> {
    try {
      const response = await put<SetAccountOrgResponse>(
        `/admin/accounts/${accountId}/organization`,
        { orgId } as SetAccountOrgRequest
      )
      if (response.success) {
        await loadAccounts()
      }
      return response
    } catch (err) {
      return { 
        success: false, 
        error: err instanceof Error ? err.message : 'Failed to update account organization' 
      }
    }
  }

  async function setAccountPassword(accountId: string, password: string): Promise<SetAccountPasswordResponse> {
    try {
      const response = await put<SetAccountPasswordResponse>(
        `/admin/accounts/${accountId}/password`,
        { password } as SetAccountPasswordRequest
      )
      if (response.success) {
        await loadAccounts()
      }
      return response
    } catch (err) {
      return { 
        success: false, 
        error: err instanceof Error ? err.message : 'Failed to set account password' 
      }
    }
  }

  async function deactivateAccount(accountId: string): Promise<DeleteResponse> {
    try {
      const response = await del<DeleteResponse>(`/admin/accounts/${accountId}`)
      if (response.success) {
        await loadAccounts()
      }
      return response
    } catch (err) {
      return { 
        success: false, 
        error: err instanceof Error ? err.message : 'Failed to deactivate account' 
      }
    }
  }

  async function activateAccount(accountId: string): Promise<DeleteResponse> {
    try {
      const response = await post<DeleteResponse>(`/admin/accounts/${accountId}/activate`)
      if (response.success) {
        await loadAccounts()
      }
      return response
    } catch (err) {
      return { 
        success: false, 
        error: err instanceof Error ? err.message : 'Failed to activate account' 
      }
    }
  }

  onMounted(() => {
    loadAccounts()
  })

  return {
    accounts,
    loading,
    error,
    refresh: loadAccounts,
    createAccount,
    regenerateToken,
    setAccountOrganization,
    setAccountPassword,
    deactivateAccount,
    activateAccount
  }
}
