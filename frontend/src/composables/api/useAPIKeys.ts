import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import { usePortalContext } from '@/composables/usePortalContext'
import type { 
  APIKey, 
  APIKeysResponse,
  CreateAPIKeyRequest,
  CreateAPIKeyResponse,
  DeleteResponse
} from '@/types/api'

export function useAPIKeys() {
  const api = useApi()
  const { isAdmin } = usePortalContext()
  
  const apiKeys = ref<APIKey[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  function getEndpoint(path: string = '') {
    const base = isAdmin.value ? '/admin' : '/org'
    return `${base}/api-keys${path}`
  }

  async function fetchByOrg(orgId: string) {
    loading.value = true
    error.value = null
    
    try {
      const endpoint = isAdmin.value 
        ? `/admin/api-keys?org=${orgId}` 
        : '/org/api-keys'
      const res = await api.get<APIKeysResponse>(endpoint)
      apiKeys.value = res.keys
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load API keys'
    } finally {
      loading.value = false
    }
  }

  async function fetchByApp(appId: string) {
    loading.value = true
    error.value = null
    
    try {
      const endpoint = isAdmin.value 
        ? `/admin/api-keys?app=${appId}` 
        : `/org/api-keys?app=${appId}`
      const res = await api.get<APIKeysResponse>(endpoint)
      apiKeys.value = res.keys
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load API keys'
    } finally {
      loading.value = false
    }
  }

  async function create(data: CreateAPIKeyRequest) {
    const res = await api.post<CreateAPIKeyResponse>(getEndpoint(), data)
    if (res.success && res.key) {
      apiKeys.value = [...apiKeys.value, res.key]
      return { key: res.key, rawKey: res.rawKey! }
    }
    throw new Error(res.error || 'Failed to create API key')
  }

  async function remove(keyId: string) {
    await api.del<DeleteResponse>(getEndpoint(`/${keyId}`))
    apiKeys.value = apiKeys.value.filter(key => key.id !== keyId)
  }

  return {
    apiKeys: readonly(apiKeys),
    loading: readonly(loading),
    error: readonly(error),
    fetchByOrg,
    fetchByApp,
    create,
    remove
  }
}
