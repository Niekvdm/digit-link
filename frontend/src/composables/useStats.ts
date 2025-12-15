import { ref, onMounted, onUnmounted } from 'vue'
import { useApi } from './useApi'
import type { Stats } from '@/types/api'

export function useStats(autoRefresh = true, refreshInterval = 30000) {
  const { get } = useApi()
  
  const stats = ref<Stats | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  
  let intervalId: ReturnType<typeof setInterval> | null = null

  async function loadStats() {
    loading.value = true
    error.value = null
    
    try {
      stats.value = await get<Stats>('/admin/stats')
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load stats'
    } finally {
      loading.value = false
    }
  }

  function startAutoRefresh() {
    if (intervalId) return
    intervalId = setInterval(loadStats, refreshInterval)
  }

  function stopAutoRefresh() {
    if (intervalId) {
      clearInterval(intervalId)
      intervalId = null
    }
  }

  onMounted(() => {
    loadStats()
    if (autoRefresh) {
      startAutoRefresh()
    }
  })

  onUnmounted(() => {
    stopAutoRefresh()
  })

  return {
    stats,
    loading,
    error,
    refresh: loadStats,
    startAutoRefresh,
    stopAutoRefresh
  }
}
