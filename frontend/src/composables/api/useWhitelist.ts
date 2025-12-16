import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import { usePortalContext } from '@/composables/usePortalContext'
import type { 
  WhitelistEntry, 
  WhitelistResponse,
  AddWhitelistRequest,
  AddWhitelistResponse,
  DeleteResponse
} from '@/types/api'

interface OrgWhitelistEntry extends WhitelistEntry {
  orgId: string
}

interface AppWhitelistEntry extends WhitelistEntry {
  appId: string
}

interface OrgWhitelistResponse {
  orgWhitelist: OrgWhitelistEntry[]
  appWhitelists: Record<string, AppWhitelistEntry[]>
}

export function useWhitelist() {
  const api = useApi()
  const { isAdmin } = usePortalContext()
  
  // Global whitelist (admin only)
  const globalWhitelist = ref<WhitelistEntry[]>([])
  
  // Org-level whitelists (admin sees all, org sees own)
  const orgWhitelist = ref<OrgWhitelistEntry[]>([])
  
  // App-level whitelists
  const appWhitelists = ref<Record<string, AppWhitelistEntry[]>>({})
  
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Fetch global whitelist (admin only)
  async function fetchGlobal() {
    if (!isAdmin.value) return
    
    loading.value = true
    error.value = null
    
    try {
      const res = await api.get<WhitelistResponse>('/admin/whitelist')
      globalWhitelist.value = res.entries
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load whitelist'
    } finally {
      loading.value = false
    }
  }

  // Fetch org whitelist (org portal or admin for specific org)
  async function fetchOrgWhitelist(orgId?: string) {
    loading.value = true
    error.value = null
    
    try {
      if (isAdmin.value && orgId) {
        // Admin fetching all org whitelists
        const res = await api.get<{ entries: OrgWhitelistEntry[] }>('/admin/org-whitelists')
        orgWhitelist.value = res.entries.filter(e => e.orgId === orgId)
      } else {
        // Org portal - fetches own whitelist
        const res = await api.get<OrgWhitelistResponse>('/org/whitelist')
        orgWhitelist.value = res.orgWhitelist
        appWhitelists.value = res.appWhitelists
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load whitelist'
    } finally {
      loading.value = false
    }
  }

  // Add to global whitelist
  async function addGlobal(data: AddWhitelistRequest) {
    const res = await api.post<AddWhitelistResponse>('/admin/whitelist', data)
    if (res.success) {
      await fetchGlobal()
    }
  }

  // Remove from global whitelist
  async function removeGlobal(entryId: string) {
    await api.del<DeleteResponse>(`/admin/whitelist/${entryId}`)
    globalWhitelist.value = globalWhitelist.value.filter(e => e.id !== entryId)
  }

  // Add to org whitelist (org portal)
  async function addOrgEntry(data: AddWhitelistRequest) {
    const res = await api.post<AddWhitelistResponse>('/org/whitelist', data)
    if (res.success) {
      await fetchOrgWhitelist()
    }
  }

  // Remove from org whitelist
  async function removeOrgEntry(entryId: string) {
    await api.del<DeleteResponse>(`/org/whitelist/${entryId}`)
    orgWhitelist.value = orgWhitelist.value.filter(e => e.id !== entryId)
  }

  // Add to app whitelist
  async function addAppEntry(appId: string, data: AddWhitelistRequest) {
    const res = await api.post<AddWhitelistResponse>('/org/app-whitelist', { 
      appId, 
      ...data 
    })
    if (res.success) {
      await fetchOrgWhitelist()
    }
  }

  // Remove from app whitelist
  async function removeAppEntry(entryId: string) {
    await api.del<DeleteResponse>(`/org/app-whitelist/${entryId}`)
    // Update local state
    for (const appId in appWhitelists.value) {
      const entries = appWhitelists.value[appId]
      if (entries) {
        appWhitelists.value[appId] = entries.filter(e => e.id !== entryId)
      }
    }
  }

  return {
    globalWhitelist: readonly(globalWhitelist),
    orgWhitelist: readonly(orgWhitelist),
    appWhitelists: readonly(appWhitelists),
    loading: readonly(loading),
    error: readonly(error),
    fetchGlobal,
    fetchOrgWhitelist,
    addGlobal,
    removeGlobal,
    addOrgEntry,
    removeOrgEntry,
    addAppEntry,
    removeAppEntry
  }
}
