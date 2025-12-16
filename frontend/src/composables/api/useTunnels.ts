import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import { usePortalContext } from '@/composables/usePortalContext'
import type { Tunnel, TunnelsResponse } from '@/types/api'

interface TunnelRecord {
  id: string
  subdomain: string
  accountId: string
  appId?: string
  orgId?: string
  createdAt: string
  lastActive?: string
  bytesSent?: number
  bytesReceived?: number
}

interface TunnelsListResponse {
  active: Tunnel[]
  records: TunnelRecord[]
}

export function useTunnels() {
  const api = useApi()
  const { isAdmin } = usePortalContext()
  
  const tunnels = ref<Tunnel[]>([])
  const records = ref<TunnelRecord[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    
    try {
      const endpoint = isAdmin.value ? '/admin/tunnels' : '/org/tunnels'
      const res = await api.get<TunnelsListResponse>(endpoint)
      tunnels.value = res.active
      records.value = res.records || []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load tunnels'
    } finally {
      loading.value = false
    }
  }

  return {
    tunnels: readonly(tunnels),
    records: readonly(records),
    loading: readonly(loading),
    error: readonly(error),
    fetchAll
  }
}
