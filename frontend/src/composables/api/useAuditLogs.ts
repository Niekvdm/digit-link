import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import type { AuditEvent, AuditEventsResponse, AuthStats } from '@/types/api'

export function useAuditLogs() {
  const api = useApi()
  
  const events = ref<AuditEvent[]>([])
  const stats = ref<AuthStats | null>(null)
  const total = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchEvents(options: {
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
      if (options.limit) params.set('limit', String(options.limit))
      if (options.offset) params.set('offset', String(options.offset))
      
      const queryString = params.toString()
      const endpoint = `/admin/audit${queryString ? `?${queryString}` : ''}`
      
      const res = await api.get<AuditEventsResponse>(endpoint)
      events.value = res.events
      total.value = res.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load audit logs'
    } finally {
      loading.value = false
    }
  }

  async function fetchStats() {
    try {
      stats.value = await api.get<AuthStats>('/admin/audit/stats')
    } catch (e) {
      console.error('Failed to fetch audit stats:', e)
    }
  }

  return {
    events: readonly(events),
    stats: readonly(stats),
    total: readonly(total),
    loading: readonly(loading),
    error: readonly(error),
    fetchEvents,
    fetchStats
  }
}
