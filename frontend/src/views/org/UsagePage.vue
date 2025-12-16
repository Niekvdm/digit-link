<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { PageHeader, StatCard, LoadingSpinner } from '@/components/ui'
import { useUsage } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import { 
  BarChart3, 
  Clock, 
  Cable, 
  Activity,
  TrendingUp,
  AlertTriangle
} from 'lucide-vue-next'
import type { OrgUsageResponse, UsageHistoryResponse } from '@/types/api'

const { getOrgUsage, getUsageHistory, loading, error } = useUsage()
const { formatDate } = useFormatters()

const usage = ref<OrgUsageResponse | null>(null)
const history = ref<UsageHistoryResponse | null>(null)
const selectedPeriod = ref<'hourly' | 'daily' | 'monthly'>('daily')

onMounted(async () => {
  await Promise.all([
    loadUsage(),
    loadHistory()
  ])
})

async function loadUsage() {
  usage.value = await getOrgUsage('', true)
}

async function loadHistory() {
  history.value = await getUsageHistory(null, selectedPeriod.value, 30, true)
}

async function changePeriod(period: 'hourly' | 'daily' | 'monthly') {
  selectedPeriod.value = period
  await loadHistory()
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

// Quota meter helpers
function getQuotaColor(percent: number): string {
  if (percent >= 90) return 'bg-accent-red'
  if (percent >= 80) return 'bg-accent-amber'
  return 'bg-accent-secondary'
}

function getQuotaTextColor(percent: number): string {
  if (percent >= 90) return 'text-accent-red'
  if (percent >= 80) return 'text-accent-amber'
  return 'text-accent-secondary'
}
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      title="Usage" 
      description="Monitor your organization's usage and quota limits"
    />

    <!-- Loading -->
    <div v-if="loading && !usage" class="flex items-center justify-center py-20">
      <LoadingSpinner />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <template v-else-if="usage">
      <!-- Period Info -->
      <div class="text-sm text-text-secondary mb-6">
        Current billing period: {{ formatDate(usage.periodStart) }} - {{ formatDate(usage.periodEnd) }}
      </div>

      <!-- Stats Overview -->
      <div class="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-5 mb-10">
        <StatCard
          label="Bandwidth Used"
          :value="formatBytes(usage.usage.bandwidthBytes)"
          :icon="BarChart3"
          color="primary"
        />
        <StatCard
          label="Tunnel Hours"
          :value="`${usage.usage.tunnelHours} hrs`"
          :icon="Clock"
          color="secondary"
        />
        <StatCard
          label="Requests"
          :value="formatNumber(usage.usage.requestCount)"
          :icon="Activity"
          color="blue"
        />
        <StatCard
          label="Current Tunnels"
          :value="usage.usage.currentConcurrent"
          :icon="Cable"
          color="amber"
        />
      </div>

      <!-- Quota Meters -->
      <div v-if="usage.quotas" class="bg-bg-surface border border-border-subtle rounded-xs p-6 mb-10">
        <h2 class="font-display text-xl font-semibold text-text-primary m-0 mb-6">Quota Usage</h2>
        
        <div class="grid gap-6">
          <!-- Bandwidth -->
          <div v-if="usage.quotas.bandwidth" class="flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <BarChart3 class="w-4 h-4 text-text-muted" />
                <span class="text-sm font-medium text-text-primary">Bandwidth</span>
              </div>
              <span 
                class="text-sm font-medium"
                :class="getQuotaTextColor(usage.quotas.bandwidth.percent)"
              >
                {{ Math.round(usage.quotas.bandwidth.percent) }}%
              </span>
            </div>
            <div class="w-full h-3 bg-bg-elevated rounded-full overflow-hidden">
              <div 
                class="h-full rounded-full transition-all duration-500"
                :class="getQuotaColor(usage.quotas.bandwidth.percent)"
                :style="{ width: `${Math.min(usage.quotas.bandwidth.percent, 100)}%` }"
              />
            </div>
            <div class="flex justify-between text-xs text-text-muted">
              <span>{{ formatBytes(usage.quotas.bandwidth.used) }} used</span>
              <span>{{ formatBytes(usage.quotas.bandwidth.limit) }} limit</span>
            </div>
          </div>

          <!-- Tunnel Hours -->
          <div v-if="usage.quotas.tunnelHours" class="flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Clock class="w-4 h-4 text-text-muted" />
                <span class="text-sm font-medium text-text-primary">Tunnel Hours</span>
              </div>
              <span 
                class="text-sm font-medium"
                :class="getQuotaTextColor(usage.quotas.tunnelHours.percent)"
              >
                {{ Math.round(usage.quotas.tunnelHours.percent) }}%
              </span>
            </div>
            <div class="w-full h-3 bg-bg-elevated rounded-full overflow-hidden">
              <div 
                class="h-full rounded-full transition-all duration-500"
                :class="getQuotaColor(usage.quotas.tunnelHours.percent)"
                :style="{ width: `${Math.min(usage.quotas.tunnelHours.percent, 100)}%` }"
              />
            </div>
            <div class="flex justify-between text-xs text-text-muted">
              <span>{{ usage.quotas.tunnelHours.used }} hrs used</span>
              <span>{{ usage.quotas.tunnelHours.limit }} hrs limit</span>
            </div>
          </div>

          <!-- Concurrent Tunnels -->
          <div v-if="usage.quotas.concurrentTunnels" class="flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Cable class="w-4 h-4 text-text-muted" />
                <span class="text-sm font-medium text-text-primary">Concurrent Tunnels</span>
              </div>
              <span 
                class="text-sm font-medium"
                :class="getQuotaTextColor(usage.quotas.concurrentTunnels.percent)"
              >
                {{ Math.round(usage.quotas.concurrentTunnels.percent) }}%
              </span>
            </div>
            <div class="w-full h-3 bg-bg-elevated rounded-full overflow-hidden">
              <div 
                class="h-full rounded-full transition-all duration-500"
                :class="getQuotaColor(usage.quotas.concurrentTunnels.percent)"
                :style="{ width: `${Math.min(usage.quotas.concurrentTunnels.percent, 100)}%` }"
              />
            </div>
            <div class="flex justify-between text-xs text-text-muted">
              <span>{{ usage.quotas.concurrentTunnels.current }} active</span>
              <span>{{ usage.quotas.concurrentTunnels.limit }} max</span>
            </div>
          </div>

          <!-- Requests -->
          <div v-if="usage.quotas.requests" class="flex flex-col gap-2">
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <Activity class="w-4 h-4 text-text-muted" />
                <span class="text-sm font-medium text-text-primary">Requests</span>
              </div>
              <span 
                class="text-sm font-medium"
                :class="getQuotaTextColor(usage.quotas.requests.percent)"
              >
                {{ Math.round(usage.quotas.requests.percent) }}%
              </span>
            </div>
            <div class="w-full h-3 bg-bg-elevated rounded-full overflow-hidden">
              <div 
                class="h-full rounded-full transition-all duration-500"
                :class="getQuotaColor(usage.quotas.requests.percent)"
                :style="{ width: `${Math.min(usage.quotas.requests.percent, 100)}%` }"
              />
            </div>
            <div class="flex justify-between text-xs text-text-muted">
              <span>{{ formatNumber(usage.quotas.requests.used) }} used</span>
              <span>{{ formatNumber(usage.quotas.requests.limit) }} limit</span>
            </div>
          </div>
        </div>

        <!-- Warning for high usage -->
        <div 
          v-if="usage.quotas.bandwidth?.percent >= 80 || 
                usage.quotas.tunnelHours?.percent >= 80 || 
                usage.quotas.requests?.percent >= 80"
          class="mt-6 p-4 bg-[rgba(var(--accent-amber-rgb),0.1)] border border-accent-amber/30 rounded-xs flex items-start gap-3"
        >
          <AlertTriangle class="w-5 h-5 text-accent-amber shrink-0 mt-0.5" />
          <div>
            <p class="text-sm font-medium text-text-primary m-0">Approaching quota limit</p>
            <p class="text-sm text-text-secondary mt-1 mb-0">
              One or more quotas are above 80%. Consider upgrading your plan to avoid service interruptions.
            </p>
          </div>
        </div>
      </div>

      <!-- No Plan Message -->
      <div 
        v-else 
        class="bg-bg-surface border border-border-subtle rounded-xs p-6 mb-10 text-center"
      >
        <TrendingUp class="w-12 h-12 text-text-muted mx-auto mb-4" />
        <h3 class="text-lg font-semibold text-text-primary mb-2">No Plan Assigned</h3>
        <p class="text-sm text-text-secondary">
          Your organization doesn't have a plan assigned. Contact your administrator to enable quota tracking.
        </p>
      </div>

      <!-- Usage History -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs p-6">
        <div class="flex items-center justify-between mb-6">
          <h2 class="font-display text-xl font-semibold text-text-primary m-0">Usage History</h2>
          
          <div class="flex gap-1 p-1 bg-bg-deep rounded-xs">
            <button
              v-for="period in ['hourly', 'daily', 'monthly'] as const"
              :key="period"
              class="px-3 py-1.5 text-sm rounded transition-all duration-200"
              :class="selectedPeriod === period 
                ? 'bg-bg-surface text-text-primary shadow-sm' 
                : 'text-text-secondary hover:text-text-primary'"
              @click="changePeriod(period)"
            >
              {{ period.charAt(0).toUpperCase() + period.slice(1) }}
            </button>
          </div>
        </div>

        <div v-if="history?.history?.length" class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-border-subtle">
                <th class="text-left py-3 px-4 font-medium text-text-secondary">Period</th>
                <th class="text-right py-3 px-4 font-medium text-text-secondary">Bandwidth</th>
                <th class="text-right py-3 px-4 font-medium text-text-secondary">Tunnel Hours</th>
                <th class="text-right py-3 px-4 font-medium text-text-secondary">Requests</th>
                <th class="text-right py-3 px-4 font-medium text-text-secondary">Peak Concurrent</th>
              </tr>
            </thead>
            <tbody>
              <tr 
                v-for="snap in history.history" 
                :key="snap.id"
                class="border-b border-border-subtle/50 hover:bg-bg-elevated/50"
              >
                <td class="py-3 px-4 text-text-primary">
                  {{ formatDate(snap.periodStart) }}
                </td>
                <td class="py-3 px-4 text-right text-text-secondary">
                  {{ formatBytes(snap.bandwidthBytes) }}
                </td>
                <td class="py-3 px-4 text-right text-text-secondary">
                  {{ Math.round(snap.tunnelSeconds / 3600) }} hrs
                </td>
                <td class="py-3 px-4 text-right text-text-secondary">
                  {{ formatNumber(snap.requestCount) }}
                </td>
                <td class="py-3 px-4 text-right text-text-secondary">
                  {{ snap.peakConcurrentTunnels }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else class="text-center py-12 text-text-muted">
          No usage history available for this period.
        </div>
      </div>
    </template>
  </div>
</template>
