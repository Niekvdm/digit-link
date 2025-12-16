<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { 
  PageHeader, 
  PlanCard,
  PlanBadge,
  QuotaMeter,
  PlanSelector,
  DataTable,
  LoadingSpinner
} from '@/components/ui'
import { useUsage } from '@/composables/api'
import { useApi } from '@/composables/useApi'
import { useFormatters } from '@/composables/useFormatters'
import type { OrgUsageResponse, Plan, UsageSnapshot } from '@/types/api'
import { 
  CreditCard, 
  Package, 
  BarChart3, 
  History,
  Wallet,
  FileText,
  Mail
} from 'lucide-vue-next'

const api = useApi()
const { getOrgUsage, getUsageHistory, loading: usageLoading } = useUsage()
const { formatDate } = useFormatters()

const usage = ref<OrgUsageResponse | null>(null)
const plans = ref<Plan[]>([])
const history = ref<UsageSnapshot[]>([])
const loading = ref(true)

const currentPlan = computed(() => {
  if (!usage.value?.plan) return null
  return {
    id: '',
    name: usage.value.plan.name,
    bandwidthBytesMonthly: usage.value.plan.bandwidthBytesMonthly,
    tunnelHoursMonthly: usage.value.plan.tunnelHoursMonthly,
    concurrentTunnelsMax: usage.value.plan.concurrentTunnelsMax,
    requestsMonthly: usage.value.plan.requestsMonthly,
    overageAllowedPercent: usage.value.plan.overageAllowedPercent,
    gracePeriodHours: usage.value.plan.gracePeriodHours,
    createdAt: '',
    updatedAt: ''
  } as Plan
})

// Billing history columns (placeholder)
const historyColumns = [
  { key: 'date', label: 'Date', width: '120px' },
  { key: 'description', label: 'Description' },
  { key: 'amount', label: 'Amount', width: '120px' },
  { key: 'status', label: 'Status', width: '100px' },
]

// Empty billing data (placeholder)
const billingHistory: never[] = []

onMounted(async () => {
  loading.value = true
  try {
    const [usageData, plansData, historyData] = await Promise.all([
      getOrgUsage('', true),
      api.get<{ plans: Plan[] }>('/api/plans').catch(() => ({ plans: [] })),
      getUsageHistory(null, 'daily', 30, true)
    ])
    
    usage.value = usageData
    plans.value = plansData.plans || []
    history.value = historyData?.history || []
  } finally {
    loading.value = false
  }
})

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      title="Billing & Subscription" 
      description="Manage your plan, view usage, and billing information"
    />

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-20">
      <LoadingSpinner />
    </div>

    <template v-else>
      <!-- Current Plan & Usage Grid -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
        <!-- Current Plan -->
        <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
          <div class="flex items-center gap-3 p-5 border-b border-border-subtle bg-bg-elevated">
            <Package class="w-5 h-5 text-accent-primary" />
            <h2 class="text-lg font-semibold text-text-primary">Current Plan</h2>
          </div>
          <div class="p-5">
            <template v-if="currentPlan">
              <PlanCard :plan="currentPlan" current />
            </template>
            <template v-else>
              <div class="text-center py-8">
                <Package class="w-12 h-12 mx-auto mb-3 text-text-muted opacity-40" />
                <p class="text-text-secondary mb-2">No plan assigned</p>
                <p class="text-sm text-text-muted">Contact your administrator to set up a subscription.</p>
              </div>
            </template>
          </div>
        </div>

        <!-- Usage This Period -->
        <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
          <div class="flex items-center justify-between p-5 border-b border-border-subtle bg-bg-elevated">
            <div class="flex items-center gap-3">
              <BarChart3 class="w-5 h-5 text-accent-secondary" />
              <h2 class="text-lg font-semibold text-text-primary">Usage This Period</h2>
            </div>
            <span v-if="usage" class="text-xs text-text-muted">
              {{ formatDate(usage.periodStart) }} - {{ formatDate(usage.periodEnd) }}
            </span>
          </div>
          <div class="p-5 space-y-5">
            <template v-if="usage">
              <QuotaMeter
                label="Bandwidth"
                :used="usage.usage.bandwidthBytes"
                :limit="usage.plan?.bandwidthBytesMonthly"
                unit="bytes"
                show-percentage
              />
              <QuotaMeter
                label="Tunnel Hours"
                :used="usage.usage.tunnelSeconds"
                :limit="usage.plan?.tunnelHoursMonthly ? usage.plan.tunnelHoursMonthly * 3600 : undefined"
                unit="hours"
                show-percentage
              />
              <QuotaMeter
                label="Requests"
                :used="usage.usage.requestCount"
                :limit="usage.plan?.requestsMonthly"
                unit="count"
                show-percentage
              />
              <QuotaMeter
                label="Concurrent Tunnels"
                :used="usage.usage.currentConcurrent"
                :limit="usage.plan?.concurrentTunnelsMax"
                unit="concurrent"
              />
            </template>
            <template v-else>
              <div class="text-center py-8 text-text-muted">
                <BarChart3 class="w-10 h-10 mx-auto mb-3 opacity-40" />
                <p>No usage data available</p>
              </div>
            </template>
          </div>
        </div>
      </div>

      <!-- Plan Comparison -->
      <div v-if="plans.length > 0" class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden mb-8">
        <div class="flex items-center gap-3 p-5 border-b border-border-subtle bg-bg-elevated">
          <CreditCard class="w-5 h-5 text-[rgb(168,85,247)]" />
          <h2 class="text-lg font-semibold text-text-primary">Available Plans</h2>
        </div>
        <div class="p-5">
          <PlanSelector 
            :plans="plans" 
            :current-plan-id="currentPlan?.id"
            disabled
            show-contact-message
          />
        </div>
      </div>

      <!-- Billing History (Placeholder) -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden mb-8">
        <div class="flex items-center gap-3 p-5 border-b border-border-subtle bg-bg-elevated">
          <History class="w-5 h-5 text-accent-amber" />
          <h2 class="text-lg font-semibold text-text-primary">Billing History</h2>
        </div>
        <div class="p-5">
          <div class="text-center py-12">
            <FileText class="w-12 h-12 mx-auto mb-3 text-text-muted opacity-40" />
            <p class="text-text-secondary mb-2">No billing history available</p>
            <p class="text-sm text-text-muted">Your invoices and payment history will appear here.</p>
          </div>
        </div>
      </div>

      <!-- Payment Method (Placeholder) -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
        <div class="flex items-center gap-3 p-5 border-b border-border-subtle bg-bg-elevated">
          <Wallet class="w-5 h-5 text-accent-blue" />
          <h2 class="text-lg font-semibold text-text-primary">Payment Method</h2>
        </div>
        <div class="p-5">
          <div class="flex items-center justify-between p-4 bg-bg-elevated rounded-xs border border-border-subtle">
            <div class="flex items-center gap-4">
              <div class="w-12 h-8 bg-bg-surface rounded flex items-center justify-center border border-border-subtle">
                <CreditCard class="w-5 h-5 text-text-muted" />
              </div>
              <div>
                <p class="text-text-primary font-medium">Managed by Administrator</p>
                <p class="text-sm text-text-secondary">Billing is handled at the organization level</p>
              </div>
            </div>
            <a 
              href="mailto:billing@example.com" 
              class="flex items-center gap-2 text-sm text-accent-primary hover:underline"
            >
              <Mail class="w-4 h-4" />
              Contact Billing
            </a>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>
