<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { PageHeader, StatCard, PlanBadge, QuotaMeter } from '@/components/ui'
import { useStats, useUsage } from '@/composables/api'
import { usePortalContext } from '@/composables/usePortalContext'
import type { OrgUsageResponse } from '@/types/api'
import { 
  Cable, 
  AppWindow, 
  ShieldCheck, 
  ArrowUpRight,
  Activity,
  KeyRound,
  Settings,
  Package,
  CreditCard,
  BarChart3
} from 'lucide-vue-next'

const router = useRouter()
const { currentOrgName } = usePortalContext()
const { stats, loading, fetchStats } = useStats()
const { getOrgUsage, loading: usageLoading } = useUsage()

const usage = ref<OrgUsageResponse | null>(null)

onMounted(async () => {
  fetchStats(true) // true = org portal
  usage.value = await getOrgUsage('', true) // true = org portal
})

function navigateTo(name: string) {
  router.push({ name })
}

const actionCards = [
  { 
    name: 'org-applications', 
    icon: AppWindow, 
    title: 'Applications', 
    desc: 'Manage your applications and subdomains',
    colorClass: 'bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary'
  },
  { 
    name: 'org-api-keys', 
    icon: KeyRound, 
    title: 'API Keys', 
    desc: 'Create and manage API keys for your apps',
    colorClass: 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber'
  },
  { 
    name: 'org-whitelist', 
    icon: ShieldCheck, 
    title: 'IP Whitelist', 
    desc: 'Control which IPs can access your tunnels',
    colorClass: 'bg-[rgba(var(--accent-blue-rgb),0.15)] text-accent-blue'
  },
  { 
    name: 'org-settings', 
    icon: Settings, 
    title: 'Settings', 
    desc: 'Configure authentication policies',
    colorClass: 'bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary'
  },
]
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      :title="`Welcome, ${currentOrgName}`" 
      description="Organization portal overview"
    />

    <!-- Plan & Usage Summary -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5 mb-8">
      <!-- Current Plan Card -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs p-5">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-sm font-semibold text-text-secondary uppercase tracking-wide flex items-center gap-2">
            <Package class="w-4 h-4" />
            Your Plan
          </h3>
          <button 
            class="text-xs text-accent-primary hover:underline"
            @click="navigateTo('org-billing')"
          >
            View Details
          </button>
        </div>
        <div class="flex items-center gap-3 mb-4">
          <PlanBadge :plan-name="usage?.plan?.name" size="lg" />
        </div>
        <div v-if="usage?.plan" class="space-y-2 text-sm">
          <div class="flex justify-between text-text-secondary">
            <span>Concurrent Tunnels</span>
            <span class="font-mono text-text-primary">{{ usage.plan.concurrentTunnelsMax || '∞' }}</span>
          </div>
          <div class="flex justify-between text-text-secondary">
            <span>Overage</span>
            <span class="font-mono text-text-primary">{{ usage.plan.overageAllowedPercent }}%</span>
          </div>
        </div>
        <div v-else class="text-sm text-text-muted">
          No plan assigned. Contact your administrator.
        </div>
      </div>

      <!-- Bandwidth Usage -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs p-5">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-sm font-semibold text-text-secondary uppercase tracking-wide flex items-center gap-2">
            <BarChart3 class="w-4 h-4" />
            Bandwidth
          </h3>
        </div>
        <template v-if="usageLoading">
          <div class="h-16 bg-bg-elevated rounded animate-pulse" />
        </template>
        <template v-else-if="usage">
          <QuotaMeter
            label=""
            :used="usage.usage.bandwidthBytes"
            :limit="usage.plan?.bandwidthBytesMonthly"
            unit="bytes"
            size="lg"
            show-percentage
          />
          <p class="text-xs text-text-muted mt-3">This billing period</p>
        </template>
      </div>

      <!-- Request Usage -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs p-5">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-sm font-semibold text-text-secondary uppercase tracking-wide flex items-center gap-2">
            <Activity class="w-4 h-4" />
            Requests
          </h3>
        </div>
        <template v-if="usageLoading">
          <div class="h-16 bg-bg-elevated rounded animate-pulse" />
        </template>
        <template v-else-if="usage">
          <QuotaMeter
            label=""
            :used="usage.usage.requestCount"
            :limit="usage.plan?.requestsMonthly"
            unit="count"
            size="lg"
            show-percentage
          />
          <p class="text-xs text-text-muted mt-3">This billing period</p>
        </template>
      </div>
    </div>

    <!-- Stats Grid -->
    <div class="grid grid-cols-[repeat(auto-fit,minmax(220px,1fr))] gap-5 mb-10">
      <StatCard
        label="Active Tunnels"
        :value="stats?.activeTunnels ?? 0"
        :icon="Cable"
        color="secondary"
        :loading="loading"
      />
      <StatCard
        label="Applications"
        :value="stats?.applicationCount ?? 0"
        :icon="AppWindow"
        color="primary"
        :loading="loading"
      />
      <StatCard
        label="Whitelist Entries"
        :value="stats?.whitelistEntries ?? 0"
        :icon="ShieldCheck"
        color="blue"
        :loading="loading"
      />
      <StatCard
        label="Total Connections"
        :value="stats?.totalConnections ?? 0"
        :icon="Activity"
        color="amber"
        :loading="loading"
      />
    </div>

    <!-- Quick Actions -->
    <div>
      <h2 class="font-display text-xl font-semibold text-text-primary m-0 mb-5">Quick Actions</h2>
      <div class="grid grid-cols-[repeat(auto-fit,minmax(280px,1fr))] gap-4">
        <!-- Billing Card (new) -->
        <button 
          class="flex items-center gap-4 py-5 px-6 bg-bg-surface border border-border-subtle rounded-xs cursor-pointer transition-all duration-200 text-left w-full hover:border-border-accent hover:bg-bg-elevated group"
          @click="navigateTo('org-billing')"
        >
          <div class="w-12 h-12 rounded-xs flex items-center justify-center shrink-0 bg-[rgba(168,85,247,0.15)] text-[rgb(168,85,247)]">
            <CreditCard class="w-6 h-6" />
          </div>
          <div class="flex-1 min-w-0">
            <h3 class="text-base font-semibold text-text-primary m-0 mb-1">Billing & Usage</h3>
            <p class="text-[0.8125rem] text-text-secondary m-0">View your plan, usage, and billing history</p>
          </div>
          <ArrowUpRight class="w-[18px] h-[18px] text-text-muted transition-all duration-200 shrink-0 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 group-hover:text-[rgb(168,85,247)]" />
        </button>
        
        <button 
          v-for="card in actionCards"
          :key="card.name"
          class="flex items-center gap-4 py-5 px-6 bg-bg-surface border border-border-subtle rounded-xs cursor-pointer transition-all duration-200 text-left w-full hover:border-border-accent hover:bg-bg-elevated group"
          @click="navigateTo(card.name)"
        >
          <div 
            class="w-12 h-12 rounded-xs flex items-center justify-center shrink-0"
            :class="card.colorClass"
          >
            <component :is="card.icon" class="w-6 h-6" />
          </div>
          <div class="flex-1 min-w-0">
            <h3 class="text-base font-semibold text-text-primary m-0 mb-1">{{ card.title }}</h3>
            <p class="text-[0.8125rem] text-text-secondary m-0">{{ card.desc }}</p>
          </div>
          <ArrowUpRight class="w-[18px] h-[18px] text-text-muted transition-all duration-200 shrink-0 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 group-hover:text-accent-secondary" />
        </button>
      </div>
    </div>
  </div>
</template>
