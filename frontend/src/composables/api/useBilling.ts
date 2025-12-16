import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import { useUsage } from './useUsage'
import type { BillingInfo, Invoice, PaymentMethod, OrgUsageResponse, Plan } from '@/types/api'

/**
 * Composable for billing and subscription data
 * Currently uses existing usage APIs - billing-specific APIs to be added in future
 */
export function useBilling() {
  const api = useApi()
  const { getOrgUsage } = useUsage()
  
  const loading = ref(false)
  const error = ref<string | null>(null)

  /**
   * Get billing info for the current organization
   * Currently combines usage data - full billing integration to come
   */
  async function getBillingInfo(): Promise<BillingInfo | null> {
    loading.value = true
    error.value = null
    
    try {
      const usage = await getOrgUsage('', true)
      
      if (!usage) {
        return null
      }

      // Construct plan from usage response if available
      const currentPlan: Plan | undefined = usage.plan ? {
        id: '',
        name: usage.plan.name,
        bandwidthBytesMonthly: usage.plan.bandwidthBytesMonthly,
        tunnelHoursMonthly: usage.plan.tunnelHoursMonthly,
        concurrentTunnelsMax: usage.plan.concurrentTunnelsMax,
        requestsMonthly: usage.plan.requestsMonthly,
        overageAllowedPercent: usage.plan.overageAllowedPercent,
        gracePeriodHours: usage.plan.gracePeriodHours,
        createdAt: '',
        updatedAt: ''
      } : undefined

      return {
        currentPlan,
        usage,
        billingHistory: [], // Placeholder - no billing history yet
        paymentMethod: undefined // Placeholder - managed by admin
      }
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load billing info'
      return null
    } finally {
      loading.value = false
    }
  }

  /**
   * Get invoice history (placeholder - returns empty array)
   */
  async function getInvoices(): Promise<Invoice[]> {
    // Placeholder for future billing integration
    return []
  }

  /**
   * Get payment method (placeholder - returns undefined)
   */
  async function getPaymentMethod(): Promise<PaymentMethod | undefined> {
    // Placeholder for future billing integration
    return undefined
  }

  return {
    loading: readonly(loading),
    error: readonly(error),
    getBillingInfo,
    getInvoices,
    getPaymentMethod
  }
}
