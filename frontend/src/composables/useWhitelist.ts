import { ref, onMounted } from 'vue'
import { useApi } from './useApi'
import type { 
  WhitelistEntry, 
  WhitelistResponse, 
  AddWhitelistRequest,
  AddWhitelistResponse,
  DeleteResponse 
} from '@/types/api'

export function useWhitelist() {
  const { get, post, del } = useApi()
  
  const entries = ref<WhitelistEntry[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadWhitelist() {
    loading.value = true
    error.value = null
    
    try {
      const data = await get<WhitelistResponse>('/admin/whitelist')
      entries.value = data.entries || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load whitelist'
    } finally {
      loading.value = false
    }
  }

  async function addEntry(request: AddWhitelistRequest): Promise<AddWhitelistResponse> {
    try {
      const response = await post<AddWhitelistResponse>('/admin/whitelist', request)
      if (response.success) {
        await loadWhitelist()
      }
      return response
    } catch (err) {
      return { 
        success: false, 
        error: err instanceof Error ? err.message : 'Failed to add IP' 
      }
    }
  }

  async function removeEntry(entryId: string): Promise<DeleteResponse> {
    try {
      const response = await del<DeleteResponse>(`/admin/whitelist/${entryId}`)
      if (response.success) {
        await loadWhitelist()
      }
      return response
    } catch (err) {
      return { 
        success: false, 
        error: err instanceof Error ? err.message : 'Failed to remove IP' 
      }
    }
  }

  onMounted(() => {
    loadWhitelist()
  })

  return {
    entries,
    loading,
    error,
    refresh: loadWhitelist,
    addEntry,
    removeEntry
  }
}
