<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { PageHeader, DataTable, StatCard, LoadingSpinner } from '@/components/ui'
import { useUsage, usePlans } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import { BarChart3, Building2, TrendingUp, AlertTriangle, Package } from 'lucide-vue-next'

const router = useRouter()
const { getUsageSummary, loading, error } = useUsage()
const { plans, fetchAll: fetchPlans } = usePlans()
const { formatDate } = useFormatters()

const summary = ref<{
  organizations: Array<{
    orgId: string
    orgName: string
    planId?: string
    planName?: string
    bandwidthBytes: number
    tunnelSeconds: number
    requestCount: number
    peakConcurrentTunnels: number
    limits?: {
      bandwidthBytesMonthly?: number
      tunnelHoursMonthly?: number
      concurrentTunnelsMax?: number
      requestsMonthly?: number
    }
  }>
  periodStart: string
  periodEnd: string
} | null>(null)

// Table columns
const columns = [
  { key: 'orgName', label: 'Organization', sortable: true },
  { key: 'planName', label: 'Plan', width: '120px' },
  { key: 'bandwidthBytes', label: 'Bandwidth', width: '140px', sortable: true },
  { key: 'tunnelHours', label: 'Tunnel Hours', width: '120px', sortable: true },
  { key: 'requestCount', label: 'Requests', width: '120px', sortable: true },
  { key: 'peakConcurrentTunnels', label: 'Peak Concurrent', width: '130px' },
]

// Computed stats
const totalBandwidth = computed(() => 
  summary.value?.organizations.reduce((sum, org) => sum + org.bandwidthBytes, 0) || 0
)

const totalRequests = computed(() =>
  summary.value?.organizations.reduce((sum, org) => sum + org.requestCount, 0) || 0
)

const orgsNearLimit = computed(() => {
  if (!summary.value) return 0
  return summary.value.organizations.filter(org => {
    if (!org.limits?.bandwidthBytesMonthly) return false
    const percent = (org.bandwidthBytes / org.limits.bandwidthBytesMonthly) * 100
    return percent >= 80
  }).length
})

const tableData = computed(() => {
  if (!summary.value) return []
  return summary.value.organizations.map(org => ({
    ...org,
    tunnelHours: Math.round(org.tunnelSeconds / 3600)
  }))
})

onMounted(async () => {
  await Promise.all([
    fetchPlans(),
    loadSummary()
  ])
})

async function loadSummary() {
  const result = await getUsageSummary()
  if (result) {
    summary.value = result
  }
}

function viewOrg(org: { orgId: string }) {
  router.push({ 
    name: 'admin-organizations',
    query: { focus: org.orgId }
  })
}

// Format helpers
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

function formatNumber(n: number): string {
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return n.toString()
}

function getUsagePercent(value: number, limit?: number): number | null {
  if (!limit) return null
  return Math.round((value / limit) * 100)
}
</script>

<template>
  <div class="max-w-[1400px]">
    <PageHeader 
      title="Usage Overview" 
      description="Monitor usage across all organizations"
    />

    <!-- Loading -->
    <div v-if="loading && !summary" class="flex items-center justify-center py-20">
      <LoadingSpinner />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <template v-else-if="summary">
      <!-- Period Info -->
      <div class="text-sm text-text-secondary mb-6">
        Current billing period: {{ formatDate(summary.periodStart) }} - {{ formatDate(summary.periodEnd) }}
      </div>

      <!-- Stats Grid -->
      <div class="grid grid-cols-[repeat(auto-fit,minmax(240px,1fr))] gap-5 mb-10">
        <StatCard
          label="Total Bandwidth"
          :value="formatBytes(totalBandwidth)"
          :icon="BarChart3"
          color="primary"
        />
        <StatCard
          label="Total Requests"
          :value="formatNumber(totalRequests)"
          :icon="TrendingUp"
          color="secondary"
        />
        <StatCard
          label="Organizations"
          :value="summary.organizations.length"
          :icon="Building2"
          color="blue"
        />
        <StatCard
          label="Near Limit (>80%)"
          :value="orgsNearLimit"
          :icon="AlertTriangle"
          :color="orgsNearLimit > 0 ? 'amber' : 'blue'"
        />
      </div>

      <!-- Table -->
      <DataTable
        :columns="columns"
        :data="tableData"
        :loading="loading"
        empty-title="No usage data"
        empty-description="Usage data will appear here once organizations start using tunnels."
        row-key="orgId"
        @row-click="viewOrg"
      >
        <template #cell-orgName="{ row }">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded-xs flex items-center justify-center bg-[rgba(var(--accent-secondary-rgb),0.1)] text-accent-secondary">
              <Building2 class="w-4 h-4" />
            </div>
            <span class="font-medium">{{ row.orgName }}</span>
          </div>
        </template>
        
        <template #cell-planName="{ value }">
          <span 
            v-if="value"
            class="text-xs font-medium py-1 px-2 rounded bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary"
          >
            {{ value }}
          </span>
          <span v-else class="text-text-muted">—</span>
        </template>
        
        <template #cell-bandwidthBytes="{ row }">
          <div class="flex flex-col gap-1">
            <span>{{ formatBytes(row.bandwidthBytes) }}</span>
            <div 
              v-if="row.limits?.bandwidthBytesMonthly"
              class="w-full h-1.5 bg-bg-elevated rounded-full overflow-hidden"
            >
              <div 
                class="h-full rounded-full transition-all duration-300"
                :class="[
                  getUsagePercent(row.bandwidthBytes, row.limits.bandwidthBytesMonthly)! >= 90 
                    ? 'bg-accent-red' 
                    : getUsagePercent(row.bandwidthBytes, row.limits.bandwidthBytesMonthly)! >= 80 
                      ? 'bg-accent-amber' 
                      : 'bg-accent-secondary'
                ]"
                :style="{ width: `${Math.min(getUsagePercent(row.bandwidthBytes, row.limits.bandwidthBytesMonthly) || 0, 100)}%` }"
              />
            </div>
          </div>
        </template>
        
        <template #cell-tunnelHours="{ row }">
          <div class="flex flex-col gap-1">
            <span>{{ row.tunnelHours }} hrs</span>
            <div 
              v-if="row.limits?.tunnelHoursMonthly"
              class="w-full h-1.5 bg-bg-elevated rounded-full overflow-hidden"
            >
              <div 
                class="h-full rounded-full transition-all duration-300"
                :class="[
                  getUsagePercent(row.tunnelHours, row.limits.tunnelHoursMonthly)! >= 90 
                    ? 'bg-accent-red' 
                    : getUsagePercent(row.tunnelHours, row.limits.tunnelHoursMonthly)! >= 80 
                      ? 'bg-accent-amber' 
                      : 'bg-accent-secondary'
                ]"
                :style="{ width: `${Math.min(getUsagePercent(row.tunnelHours, row.limits.tunnelHoursMonthly) || 0, 100)}%` }"
              />
            </div>
          </div>
        </template>
        
        <template #cell-requestCount="{ row }">
          <div class="flex flex-col gap-1">
            <span>{{ formatNumber(row.requestCount) }}</span>
            <div 
              v-if="row.limits?.requestsMonthly"
              class="w-full h-1.5 bg-bg-elevated rounded-full overflow-hidden"
            >
              <div 
                class="h-full rounded-full transition-all duration-300"
                :class="[
                  getUsagePercent(row.requestCount, row.limits.requestsMonthly)! >= 90 
                    ? 'bg-accent-red' 
                    : getUsagePercent(row.requestCount, row.limits.requestsMonthly)! >= 80 
                      ? 'bg-accent-amber' 
                      : 'bg-accent-secondary'
                ]"
                :style="{ width: `${Math.min(getUsagePercent(row.requestCount, row.limits.requestsMonthly) || 0, 100)}%` }"
              />
            </div>
          </div>
        </template>
        
        <template #cell-peakConcurrentTunnels="{ value }">
          {{ value }}
        </template>
      </DataTable>
    </template>
  </div>
</template>
