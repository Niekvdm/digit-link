import { ref } from 'vue'
import { useApi } from './useApi'
import type {
  APIKey,
  APIKeysResponse,
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,
  DeleteResponse
} from '@/types/api'

export function useAPIKeys() {
  const { get, post, del } = useApi()

  const keys = ref<APIKey[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadKeys(orgId?: string, appId?: string) {
    if (!orgId && !appId) {
      keys.value = []
      return
    }

    loading.value = true
    error.value = null

    try {
      const params = appId ? `?app=${appId}` : `?org=${orgId}`
      const data = await get<APIKeysResponse>(`/admin/api-keys${params}`)
      keys.value = data.keys || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load API keys'
    } finally {
      loading.value = false
    }
  }

  async function createKey(request: CreateAPIKeyRequest): Promise<CreateAPIKeyResponse> {
    try {
      const response = await post<CreateAPIKeyResponse>('/admin/api-keys', request)
      if (response.success) {
        await loadKeys(request.orgId, request.appId)
      }
      return response
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : 'Failed to create API key'
      }
    }
  }

  async function revokeKey(keyId: string, orgId?: string, appId?: string): Promise<DeleteResponse> {
    try {
      const response = await del<DeleteResponse>(`/admin/api-keys/${keyId}`)
      if (response.success && (orgId || appId)) {
        await loadKeys(orgId, appId)
      }
      return response
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : 'Failed to revoke API key'
      }
    }
  }

  return {
    keys,
    loading,
    error,
    loadKeys,
    createKey,
    revokeKey
  }
}
