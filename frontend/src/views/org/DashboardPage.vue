<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { PageHeader, StatCard } from '@/components/ui'
import { useStats } from '@/composables/api'
import { usePortalContext } from '@/composables/usePortalContext'
import { 
  Cable, 
  AppWindow, 
  ShieldCheck, 
  ArrowUpRight,
  Activity,
  KeyRound,
  Settings
} from 'lucide-vue-next'

const router = useRouter()
const { currentOrgName } = usePortalContext()
const { stats, loading, fetchStats } = useStats()

onMounted(() => {
  fetchStats(true) // true = org portal
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
