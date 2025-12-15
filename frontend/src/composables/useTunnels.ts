import { ref, onMounted, onUnmounted } from 'vue'
import { useApi } from './useApi'
import type { Tunnel, TunnelsResponse } from '@/types/api'

export function useTunnels(autoRefresh = true, refreshInterval = 10000) {
  const { get } = useApi()
  
  const tunnels = ref<Tunnel[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)
  
  let intervalId: ReturnType<typeof setInterval> | null = null

  async function loadTunnels() {
    loading.value = true
    error.value = null
    
    try {
      const data = await get<TunnelsResponse>('/admin/tunnels')
      tunnels.value = data.active || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load tunnels'
    } finally {
      loading.value = false
    }
  }

  function startAutoRefresh() {
    if (intervalId) return
    intervalId = setInterval(loadTunnels, refreshInterval)
  }

  function stopAutoRefresh() {
    if (intervalId) {
      clearInterval(intervalId)
      intervalId = null
    }
  }

  onMounted(() => {
    loadTunnels()
    if (autoRefresh) {
      startAutoRefresh()
    }
  })

  onUnmounted(() => {
    stopAutoRefresh()
  })

  return {
    tunnels,
    loading,
    error,
    refresh: loadTunnels,
    startAutoRefresh,
    stopAutoRefresh
  }
}
