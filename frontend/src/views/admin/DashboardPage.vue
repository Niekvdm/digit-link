<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { PageHeader, StatCard } from '@/components/ui'
import { useStats } from '@/composables/api'
import { 
  Cable, 
  Users, 
  ShieldCheck, 
  Building2, 
  AppWindow, 
  ArrowUpRight,
  Activity
} from 'lucide-vue-next'

const router = useRouter()
const { stats, loading, fetchStats } = useStats()

onMounted(() => {
  fetchStats()
})

function navigateTo(name: string) {
  router.push({ name })
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

const actionCards = [
  { 
    name: 'admin-organizations', 
    icon: Building2, 
    title: 'Organizations', 
    desc: 'Manage organizations and their policies',
    colorClass: 'bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary'
  },
  { 
    name: 'admin-applications', 
    icon: AppWindow, 
    title: 'Applications', 
    desc: 'Configure applications and subdomains',
    colorClass: 'bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary'
  },
  { 
    name: 'admin-accounts', 
    icon: Users, 
    title: 'Accounts', 
    desc: 'Manage user accounts and permissions',
    colorClass: 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber'
  },
  { 
    name: 'admin-tunnels', 
    icon: Cable, 
    title: 'Tunnels', 
    desc: 'Monitor active tunnel connections',
    colorClass: 'bg-[rgba(var(--accent-blue-rgb),0.15)] text-accent-blue'
  },
]
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      title="Dashboard" 
      description="System overview and key metrics"
    />

    <!-- Stats Grid -->
    <div class="grid grid-cols-[repeat(auto-fit,minmax(240px,1fr))] gap-5 mb-10">
      <StatCard
        label="Active Tunnels"
        :value="stats?.activeTunnels ?? 0"
        :icon="Cable"
        color="secondary"
        :loading="loading"
      />
      <StatCard
        label="Active Accounts"
        :value="stats?.activeAccounts ?? 0"
        :icon="Users"
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
        label="Total Tunnels"
        :value="stats?.totalTunnels ?? 0"
        :icon="Activity"
        color="amber"
        :loading="loading"
      />
    </div>

    <!-- Quick Actions -->
    <div class="mb-10">
      <h2 class="font-display text-xl font-semibold text-text-primary m-0 mb-5">Quick Actions</h2>
      <div class="grid grid-cols-[repeat(auto-fit,minmax(280px,1fr))] gap-4">
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
          <ArrowUpRight class="w-[18px] h-[18px] text-text-muted transition-all duration-200 shrink-0 group-hover:translate-x-0.5 group-hover:-translate-y-0.5 group-hover:text-accent-primary" />
        </button>
      </div>
    </div>

    <!-- Traffic Stats -->
    <div 
      v-if="stats?.totalBytesSent || stats?.totalBytesReceived" 
      class="bg-bg-surface border border-border-subtle rounded-xs p-6"
    >
      <h2 class="font-display text-xl font-semibold text-text-primary m-0 mb-4">Traffic Overview</h2>
      <div class="flex gap-8 flex-wrap">
        <div class="flex flex-col gap-1">
          <span class="text-[0.8125rem] text-text-secondary">Total Data Sent</span>
          <span class="font-mono text-xl font-semibold text-text-primary">{{ formatBytes(stats.totalBytesSent ?? 0) }}</span>
        </div>
        <div class="flex flex-col gap-1">
          <span class="text-[0.8125rem] text-text-secondary">Total Data Received</span>
          <span class="font-mono text-xl font-semibold text-text-primary">{{ formatBytes(stats.totalBytesReceived ?? 0) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
