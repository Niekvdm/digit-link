<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { PageHeader, DataTable, StatCard, LoadingSpinner } from '@/components/ui'
import UsageChart from '@/components/charts/UsageChart.vue'
import { useUsage, usePlans } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import { BarChart3, Building2, TrendingUp, AlertTriangle, Package, Download } from 'lucide-vue-next'
import type { UsageSnapshot } from '@/types/api'
import type { Dataset } from '@/components/charts/UsageChart.vue'

const router = useRouter()
const { getUsageSummary, getUsageHistory, loading, error } = useUsage()
const { plans, fetchAll: fetchPlans } = usePlans()
const { formatDate } = useFormatters()

// Timeframe selection state
const selectedTimeframe = ref<'daily' | 'weekly' | 'monthly'>('daily')

// Metric selection state
type MetricType = 'bandwidth' | 'requests' | 'tunnel_hours'
const selectedMetric = ref<MetricType>('bandwidth')

// Chart data state
const chartLoading = ref(false)
const chartError = ref<string | null>(null)
const allChartDatasets = ref<Dataset[]>([])

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

// Map timeframe to API period and chart time unit
const chartTimeUnit = computed<'hour' | 'day' | 'week' | 'month'>(() => {
  switch (selectedTimeframe.value) {
    case 'daily': return 'day'
    case 'weekly': return 'week'
    case 'monthly': return 'month'
    default: return 'day'
  }
})

// Filter datasets based on selected metric
const chartDatasets = computed<Dataset[]>(() => {
  const metricLabelMap: Record<MetricType, string> = {
    bandwidth: 'Bandwidth',
    requests: 'Requests',
    tunnel_hours: 'Tunnel Hours'
  }
  const targetLabel = metricLabelMap[selectedMetric.value]
  return allChartDatasets.value.filter(d => d.label === targetLabel)
})

// Get Y-axis formatter based on selected metric
const chartYAxisFormatter = computed(() => {
  switch (selectedMetric.value) {
    case 'bandwidth':
      return formatBytes
    case 'requests':
      return formatNumber
    case 'tunnel_hours':
      return (hours: number) => `${formatNumber(hours)} hrs`
    default:
      return formatNumber
  }
})

// Get Y-axis label based on selected metric
const chartYAxisLabel = computed(() => {
  switch (selectedMetric.value) {
    case 'bandwidth':
      return 'Bandwidth'
    case 'requests':
      return 'Requests'
    case 'tunnel_hours':
      return 'Hours'
    default:
      return ''
  }
})

// Metric options for selector
const metricOptions = [
  { value: 'bandwidth' as MetricType, label: 'Bandwidth' },
  { value: 'requests' as MetricType, label: 'Requests' },
  { value: 'tunnel_hours' as MetricType, label: 'Tunnel Hours' }
]

// Get API period and days based on timeframe
function getChartParams(): { period: 'hourly' | 'daily' | 'monthly'; days: number } {
  switch (selectedTimeframe.value) {
    case 'daily':
      return { period: 'hourly', days: 7 }
    case 'weekly':
      return { period: 'daily', days: 30 }
    case 'monthly':
      return { period: 'daily', days: 90 }
    default:
      return { period: 'daily', days: 30 }
  }
}

// Transform UsageSnapshot[] to Chart.js dataset format
function transformToChartData(history: UsageSnapshot[]): Dataset[] {
  if (!history.length) return []

  // Aggregate data across all organizations by timestamp
  const aggregatedData = new Map<string, {
    bandwidthBytes: number
    tunnelSeconds: number
    requestCount: number
  }>()

  history.forEach(snapshot => {
    const key = snapshot.periodStart
    const existing = aggregatedData.get(key) || {
      bandwidthBytes: 0,
      tunnelSeconds: 0,
      requestCount: 0
    }
    aggregatedData.set(key, {
      bandwidthBytes: existing.bandwidthBytes + snapshot.bandwidthBytes,
      tunnelSeconds: existing.tunnelSeconds + snapshot.tunnelSeconds,
      requestCount: existing.requestCount + snapshot.requestCount
    })
  })

  // Sort by date and create data points
  const sortedEntries = Array.from(aggregatedData.entries())
    .sort((a, b) => new Date(a[0]).getTime() - new Date(b[0]).getTime())

  return [
    {
      label: 'Bandwidth',
      data: sortedEntries.map(([date, data]) => ({
        x: new Date(date),
        y: data.bandwidthBytes
      }))
    },
    {
      label: 'Requests',
      data: sortedEntries.map(([date, data]) => ({
        x: new Date(date),
        y: data.requestCount
      }))
    },
    {
      label: 'Tunnel Hours',
      data: sortedEntries.map(([date, data]) => ({
        x: new Date(date),
        y: Math.round(data.tunnelSeconds / 3600)
      }))
    }
  ]
}

// Load chart data from API
async function loadChartData() {
  chartLoading.value = true
  chartError.value = null

  try {
    const { period, days } = getChartParams()
    // Pass null for orgId to get aggregate data for all orgs
    const result = await getUsageHistory(null, period, days)

    if (result) {
      allChartDatasets.value = transformToChartData(result.history)
    } else {
      allChartDatasets.value = []
    }
  } catch (e) {
    chartError.value = e instanceof Error ? e.message : 'Failed to load chart data'
    allChartDatasets.value = []
  } finally {
    chartLoading.value = false
  }
}

// Watch for timeframe changes and reload chart data
watch(selectedTimeframe, () => {
  loadChartData()
})

onMounted(async () => {
  await Promise.all([
    fetchPlans(),
    loadSummary(),
    loadChartData()
  ])
})

async function loadSummary() {
  const result = await getUsageSummary()
  if (result) {
    summary.value = result
  }
}

function changeTimeframe(timeframe: 'daily' | 'weekly' | 'monthly') {
  selectedTimeframe.value = timeframe
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

// Export data functions
function getExportData() {
  if (!summary.value) return []
  return summary.value.organizations.map(org => ({
    organization: org.orgName,
    plan: org.planName || '',
    bandwidthBytes: org.bandwidthBytes,
    bandwidthFormatted: formatBytes(org.bandwidthBytes),
    tunnelHours: Math.round(org.tunnelSeconds / 3600),
    requests: org.requestCount,
    peakConcurrentTunnels: org.peakConcurrentTunnels
  }))
}

function downloadFile(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

function exportCSV() {
  const data = getExportData()
  if (!data.length) return

  const headers = ['Organization', 'Plan', 'Bandwidth (bytes)', 'Bandwidth', 'Tunnel Hours', 'Requests', 'Peak Concurrent']
  const rows = data.map(row => [
    `"${row.organization}"`,
    `"${row.plan}"`,
    row.bandwidthBytes,
    `"${row.bandwidthFormatted}"`,
    row.tunnelHours,
    row.requests,
    row.peakConcurrentTunnels
  ].join(','))

  const csv = [headers.join(','), ...rows].join('\n')
  const timestamp = new Date().toISOString().split('T')[0]
  downloadFile(csv, `usage-export-${timestamp}.csv`, 'text/csv')
}

function exportJSON() {
  const data = getExportData()
  if (!data.length) return

  const exportObj = {
    exportDate: new Date().toISOString(),
    period: summary.value ? {
      start: summary.value.periodStart,
      end: summary.value.periodEnd
    } : null,
    organizations: data
  }

  const json = JSON.stringify(exportObj, null, 2)
  const timestamp = new Date().toISOString().split('T')[0]
  downloadFile(json, `usage-export-${timestamp}.json`, 'application/json')
}
</script>

<template>
  <div class="max-w-[1400px]">
    <PageHeader
      title="Usage Overview"
      description="Monitor usage across all organizations"
    >
      <template #actions>
        <button
          class="flex items-center gap-2 px-4 py-2 text-sm font-medium border border-border-subtle rounded-xs bg-bg-surface text-text-primary cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:border-border-accent disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="!summary?.organizations.length"
          @click="exportCSV"
        >
          <Download class="w-4 h-4" />
          Export CSV
        </button>
        <button
          class="flex items-center gap-2 px-4 py-2 text-sm font-medium border border-border-subtle rounded-xs bg-bg-surface text-text-primary cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:border-border-accent disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="!summary?.organizations.length"
          @click="exportJSON"
        >
          <Download class="w-4 h-4" />
          Export JSON
        </button>
      </template>
    </PageHeader>

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

      <!-- Usage Trends Section -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs p-6 mb-10">
        <div class="flex items-center justify-between mb-6 flex-wrap gap-4">
          <h2 class="font-display text-xl font-semibold text-text-primary m-0">Usage Trends</h2>

          <div class="flex items-center gap-4">
            <!-- Metric Selector -->
            <div class="flex items-center gap-2">
              <span class="text-xs text-text-muted uppercase tracking-wider">Metric</span>
              <div class="flex gap-1 p-1 bg-bg-deep rounded-xs">
                <button
                  v-for="metric in metricOptions"
                  :key="metric.value"
                  class="px-3 py-1.5 text-sm rounded transition-all duration-200"
                  :class="selectedMetric === metric.value
                    ? 'bg-bg-surface text-text-primary shadow-sm'
                    : 'text-text-secondary hover:text-text-primary'"
                  @click="selectedMetric = metric.value"
                >
                  {{ metric.label }}
                </button>
              </div>
            </div>

            <!-- Timeframe Selector -->
            <div class="flex items-center gap-2">
              <span class="text-xs text-text-muted uppercase tracking-wider">Period</span>
              <div class="flex gap-1 p-1 bg-bg-deep rounded-xs">
                <button
                  v-for="timeframe in ['daily', 'weekly', 'monthly'] as const"
                  :key="timeframe"
                  class="px-3 py-1.5 text-sm rounded transition-all duration-200"
                  :class="selectedTimeframe === timeframe
                    ? 'bg-bg-surface text-text-primary shadow-sm'
                    : 'text-text-secondary hover:text-text-primary'"
                  @click="changeTimeframe(timeframe)"
                >
                  {{ timeframe.charAt(0).toUpperCase() + timeframe.slice(1) }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Error State -->
        <div v-if="chartError" class="h-64 flex flex-col items-center justify-center text-text-muted border border-dashed border-border-subtle rounded-xs gap-3">
          <span class="text-accent-red">{{ chartError }}</span>
          <button
            class="px-4 py-2 text-sm font-medium text-text-primary bg-bg-elevated hover:bg-bg-deep rounded-xs transition-colors"
            @click="loadChartData"
          >
            Retry
          </button>
        </div>

        <!-- Usage Chart -->
        <UsageChart
          v-else
          :datasets="chartDatasets"
          :loading="chartLoading"
          :use-time-scale="true"
          :time-unit="chartTimeUnit"
          :height="256"
          :show-legend="false"
          :fill="true"
          :y-axis-formatter="chartYAxisFormatter"
          :tooltip-formatter="chartYAxisFormatter"
          :y-axis-label="chartYAxisLabel"
          :show-peak-annotations="true"
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
