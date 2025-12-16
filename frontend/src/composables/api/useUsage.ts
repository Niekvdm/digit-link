import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import type { 
  OrgUsageResponse, 
  UsageHistoryResponse, 
  UsageSummaryResponse 
} from '@/types/api'

export function useUsage() {
  const api = useApi()
  
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function getOrgUsage(orgId: string, isOrgPortal = false): Promise<OrgUsageResponse | null> {
    loading.value = true
    error.value = null
    
    try {
      const endpoint = isOrgPortal ? '/org/usage' : `/admin/organizations/${orgId}/usage`
      return await api.get<OrgUsageResponse>(endpoint)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load usage'
      return null
    } finally {
      loading.value = false
    }
  }

  async function getUsageHistory(
    orgId: string | null, 
    period: 'hourly' | 'daily' | 'monthly' = 'daily',
    days = 30,
    isOrgPortal = false
  ): Promise<UsageHistoryResponse | null> {
    loading.value = true
    error.value = null
    
    try {
      const endpoint = isOrgPortal 
        ? `/org/usage/history?period=${period}&days=${days}`
        : `/admin/organizations/${orgId}/usage?period=${period}&days=${days}`
      return await api.get<UsageHistoryResponse>(endpoint)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load usage history'
      return null
    } finally {
      loading.value = false
    }
  }

  async function getUsageSummary(): Promise<UsageSummaryResponse | null> {
    loading.value = true
    error.value = null
    
    try {
      return await api.get<UsageSummaryResponse>('/admin/usage/summary')
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load usage summary'
      return null
    } finally {
      loading.value = false
    }
  }

  async function resetOrgUsage(orgId: string): Promise<boolean> {
    loading.value = true
    error.value = null
    
    try {
      await api.post(`/admin/organizations/${orgId}/usage/reset`)
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to reset usage'
      return false
    } finally {
      loading.value = false
    }
  }

  return {
    loading: readonly(loading),
    error: readonly(error),
    getOrgUsage,
    getUsageHistory,
    getUsageSummary,
    resetOrgUsage
  }
}
