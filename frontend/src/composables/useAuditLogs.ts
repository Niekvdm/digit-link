import { ref, onMounted } from 'vue'
import { useApi } from './useApi'
import type {
  AuditEvent,
  AuditEventsResponse,
  AuthStats
} from '@/types/api'

export function useAuditLogs() {
  const { get } = useApi()

  const events = ref<AuditEvent[]>([])
  const stats = ref<AuthStats | null>(null)
  const total = ref(0)
  const loading = ref(false)
  const statsLoading = ref(false)
  const error = ref<string | null>(null)

  async function loadEvents(options: {
    orgId?: string
    appId?: string
    limit?: number
    offset?: number
  } = {}) {
    loading.value = true
    error.value = null

    try {
      const params = new URLSearchParams()
      if (options.orgId) params.set('org', options.orgId)
      if (options.appId) params.set('app', options.appId)
      if (options.limit) params.set('limit', options.limit.toString())
      if (options.offset) params.set('offset', options.offset.toString())

      const queryString = params.toString()
      const url = `/admin/audit${queryString ? '?' + queryString : ''}`
      
      const data = await get<AuditEventsResponse>(url)
      events.value = data.events || []
      total.value = data.total || 0
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load audit events'
    } finally {
      loading.value = false
    }
  }

  async function loadStats() {
    statsLoading.value = true

    try {
      const data = await get<AuthStats>('/admin/audit/stats')
      stats.value = data
    } catch (err) {
      console.error('Failed to load audit stats:', err)
    } finally {
      statsLoading.value = false
    }
  }

  onMounted(() => {
    loadEvents()
    loadStats()
  })

  return {
    events,
    stats,
    total,
    loading,
    statsLoading,
    error,
    loadEvents,
    loadStats,
    refresh: () => {
      loadEvents()
      loadStats()
    }
  }
}
