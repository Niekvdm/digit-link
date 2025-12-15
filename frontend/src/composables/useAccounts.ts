import { ref, onMounted } from 'vue'
import { useApi } from './useApi'
import type { 
  Account, 
  AccountsResponse, 
  CreateAccountRequest, 
  CreateAccountResponse,
  RegenerateTokenResponse,
  DeleteResponse 
} from '@/types/api'

export function useAccounts() {
  const { get, post, del } = useApi()
  
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
    deactivateAccount
  }
}
