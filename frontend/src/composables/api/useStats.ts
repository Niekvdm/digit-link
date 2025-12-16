import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import type { Stats } from '@/types/api'

export function useStats() {
  const api = useApi()
  
  const stats = ref<Stats | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchStats(isOrgPortal = false) {
    loading.value = true
    error.value = null
    
    try {
      const endpoint = isOrgPortal ? '/org/stats' : '/admin/stats'
      stats.value = await api.get<Stats>(endpoint)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load stats'
    } finally {
      loading.value = false
    }
  }

  return {
    stats: readonly(stats),
    loading: readonly(loading),
    error: readonly(error),
    fetchStats
  }
}
