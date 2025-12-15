<script setup lang="ts">
import { computed } from 'vue'
import { useStats, useTunnels } from '@/composables'
import AppLayout from '@/components/layout/AppLayout.vue'
import { StatCard, LoadingState, EmptyState } from '@/components/shared'
import { 
  Cable, 
  Users, 
  ShieldCheck, 
  Activity,
  RefreshCw,
  UserPlus,
  PlusCircle,
  Eye,
  ChevronRight
} from 'lucide-vue-next'

const { stats, loading: statsLoading } = useStats()
const { tunnels, loading: tunnelsLoading, refresh: refreshTunnels } = useTunnels()

const displayTunnels = computed(() => tunnels.value.slice(0, 5))
const hasMoreTunnels = computed(() => tunnels.value.length > 5)

function formatTime(timestamp: string) {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  if (diff < 60000) return 'Just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  return date.toLocaleDateString()
}

const quickActions = [
  { to: '/accounts', icon: UserPlus, title: 'Create Account', desc: 'Add a new tunnel user' },
  { to: '/whitelist', icon: PlusCircle, title: 'Add IP to Whitelist', desc: 'Allow new IP addresses' },
  { to: '/tunnels', icon: Eye, title: 'View All Tunnels', desc: 'Monitor active connections' },
]
</script>

<template>
  <AppLayout>
    <!-- Page Header -->
    <div class="mb-8">
      <h1 class="font-[var(--font-display)] text-3xl font-semibold mb-2">Dashboard</h1>
      <p class="text-sm text-[var(--text-secondary)]">System overview and quick actions</p>
    </div>

    <!-- Stats Grid -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
      <StatCard
        label="Active Tunnels"
        :value="stats?.activeTunnels ?? 0"
        :icon="Cable"
        :loading="statsLoading"
      />
      <StatCard
        label="Active Accounts"
        :value="stats?.activeAccounts ?? 0"
        :icon="Users"
        :loading="statsLoading"
      />
      <StatCard
        label="Whitelisted IPs"
        :value="stats?.whitelistEntries ?? 0"
        :icon="ShieldCheck"
        :loading="statsLoading"
      />
      <StatCard
        label="Total Connections"
        :value="stats?.totalTunnels ?? 0"
        :icon="Activity"
        :loading="statsLoading"
      />
    </div>

    <!-- Two Column Layout -->
    <div class="grid lg:grid-cols-2 gap-6">
      <!-- Active Tunnels Card -->
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">
            <Cable class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
            Active Tunnels
          </h2>
          <button 
            class="btn btn-secondary btn-sm"
            :disabled="tunnelsLoading"
            @click="refreshTunnels"
          >
            <RefreshCw 
              class="w-3.5 h-3.5" 
              :class="{ 'animate-spin': tunnelsLoading }"
            />
            Refresh
          </button>
        </div>
        <div class="card-body">
          <LoadingState v-if="tunnelsLoading && !tunnels.length" message="Loading tunnels..." />
          
          <EmptyState 
            v-else-if="!tunnels.length"
            :icon="Cable"
            title="No active tunnels"
          />

          <template v-else>
            <div class="flex flex-col gap-3">
              <div
                v-for="tunnel in displayTunnels"
                :key="tunnel.id"
                class="flex items-center gap-4 p-4 bg-[var(--bg-deep)] rounded-lg hover:bg-[var(--bg-elevated)] transition-colors"
              >
                <div class="tunnel-status" />
                <div class="flex-1 min-w-0">
                  <div class="font-mono text-sm font-medium mb-1 truncate">
                    {{ tunnel.subdomain }}
                  </div>
                  <div class="text-xs text-[var(--text-muted)] truncate">
                    {{ tunnel.url }}
                  </div>
                </div>
                <div class="text-right text-xs text-[var(--text-secondary)]">
                  {{ formatTime(tunnel.createdAt) }}
                </div>
              </div>
            </div>

            <RouterLink
              v-if="hasMoreTunnels"
              to="/tunnels"
              class="block text-center mt-4 text-sm text-[var(--accent-copper)] hover:underline"
            >
              View all {{ tunnels.length }} tunnels →
            </RouterLink>
          </template>
        </div>
      </div>

      <!-- Quick Actions Card -->
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">
            <Activity class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
            Quick Actions
          </h2>
        </div>
        <div class="card-body">
          <div class="flex flex-col gap-3">
            <RouterLink
              v-for="action in quickActions"
              :key="action.to"
              :to="action.to"
              class="flex items-center gap-4 p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg hover:border-[var(--accent-copper)] hover:bg-[var(--bg-elevated)] transition-all no-underline text-[var(--text-primary)]"
            >
              <component :is="action.icon" class="w-5 h-5 text-[var(--accent-copper)]" />
              <div class="flex-1">
                <div class="font-medium text-sm mb-0.5">{{ action.title }}</div>
                <div class="text-xs text-[var(--text-muted)]">{{ action.desc }}</div>
              </div>
              <ChevronRight class="w-4 h-4 text-[var(--text-muted)]" />
            </RouterLink>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>
