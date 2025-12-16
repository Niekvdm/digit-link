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
</script>

<template>
  <div class="dashboard">
    <PageHeader 
      title="Dashboard" 
      description="System overview and key metrics"
    />

    <!-- Stats Grid -->
    <div class="stats-grid">
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
    <div class="quick-actions">
      <h2 class="section-title">Quick Actions</h2>
      <div class="actions-grid">
        <button class="action-card" @click="navigateTo('admin-organizations')">
          <div class="action-icon action-icon--primary">
            <Building2 class="w-6 h-6" />
          </div>
          <div class="action-content">
            <h3 class="action-title">Organizations</h3>
            <p class="action-desc">Manage organizations and their policies</p>
          </div>
          <ArrowUpRight class="action-arrow" />
        </button>
        
        <button class="action-card" @click="navigateTo('admin-applications')">
          <div class="action-icon action-icon--secondary">
            <AppWindow class="w-6 h-6" />
          </div>
          <div class="action-content">
            <h3 class="action-title">Applications</h3>
            <p class="action-desc">Configure applications and subdomains</p>
          </div>
          <ArrowUpRight class="action-arrow" />
        </button>
        
        <button class="action-card" @click="navigateTo('admin-accounts')">
          <div class="action-icon action-icon--amber">
            <Users class="w-6 h-6" />
          </div>
          <div class="action-content">
            <h3 class="action-title">Accounts</h3>
            <p class="action-desc">Manage user accounts and permissions</p>
          </div>
          <ArrowUpRight class="action-arrow" />
        </button>
        
        <button class="action-card" @click="navigateTo('admin-tunnels')">
          <div class="action-icon action-icon--blue">
            <Cable class="w-6 h-6" />
          </div>
          <div class="action-content">
            <h3 class="action-title">Tunnels</h3>
            <p class="action-desc">Monitor active tunnel connections</p>
          </div>
          <ArrowUpRight class="action-arrow" />
        </button>
      </div>
    </div>

    <!-- Traffic Stats -->
    <div v-if="stats?.totalBytesSent || stats?.totalBytesReceived" class="traffic-section">
      <h2 class="section-title">Traffic Overview</h2>
      <div class="traffic-stats">
        <div class="traffic-stat">
          <span class="traffic-label">Total Data Sent</span>
          <span class="traffic-value">{{ formatBytes(stats.totalBytesSent ?? 0) }}</span>
        </div>
        <div class="traffic-stat">
          <span class="traffic-label">Total Data Received</span>
          <span class="traffic-value">{{ formatBytes(stats.totalBytesReceived ?? 0) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}
</script>

<style scoped>
.dashboard {
  max-width: 1200px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 1.25rem;
  margin-bottom: 2.5rem;
}

.section-title {
  font-family: var(--font-display);
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 1.25rem;
}

.actions-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1rem;
}

.action-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem 1.5rem;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: left;
  width: 100%;
}

.action-card:hover {
  border-color: var(--border-accent);
  background: var(--bg-elevated);
}

.action-card:hover .action-arrow {
  transform: translate(2px, -2px);
  color: var(--accent-primary);
}

.action-icon {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.action-icon--primary {
  background: rgba(var(--accent-primary-rgb), 0.15);
  color: var(--accent-primary);
}

.action-icon--secondary {
  background: rgba(var(--accent-secondary-rgb), 0.15);
  color: var(--accent-secondary);
}

.action-icon--amber {
  background: rgba(var(--accent-amber-rgb), 0.15);
  color: var(--accent-amber);
}

.action-icon--blue {
  background: rgba(var(--accent-blue-rgb), 0.15);
  color: var(--accent-blue);
}

.action-content {
  flex: 1;
  min-width: 0;
}

.action-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.25rem;
}

.action-desc {
  font-size: 0.8125rem;
  color: var(--text-secondary);
  margin: 0;
}

.action-arrow {
  width: 18px;
  height: 18px;
  color: var(--text-muted);
  transition: all 0.2s ease;
  flex-shrink: 0;
}

.quick-actions {
  margin-bottom: 2.5rem;
}

.traffic-section {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  padding: 1.5rem;
}

.traffic-section .section-title {
  margin-bottom: 1rem;
}

.traffic-stats {
  display: flex;
  gap: 2rem;
  flex-wrap: wrap;
}

.traffic-stat {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.traffic-label {
  font-size: 0.8125rem;
  color: var(--text-secondary);
}

.traffic-value {
  font-family: var(--font-mono);
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
}
</style>
