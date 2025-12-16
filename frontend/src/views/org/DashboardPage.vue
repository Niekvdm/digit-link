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
</script>

<template>
  <div class="dashboard">
    <PageHeader 
      :title="`Welcome, ${currentOrgName}`" 
      description="Organization portal overview"
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
    <div class="quick-actions">
      <h2 class="section-title">Quick Actions</h2>
      <div class="actions-grid">
        <button class="action-card" @click="navigateTo('org-applications')">
          <div class="action-icon action-icon--secondary">
            <AppWindow class="w-6 h-6" />
          </div>
          <div class="action-content">
            <h3 class="action-title">Applications</h3>
            <p class="action-desc">Manage your applications and subdomains</p>
          </div>
          <ArrowUpRight class="action-arrow" />
        </button>
        
        <button class="action-card" @click="navigateTo('org-api-keys')">
          <div class="action-icon action-icon--amber">
            <KeyRound class="w-6 h-6" />
          </div>
          <div class="action-content">
            <h3 class="action-title">API Keys</h3>
            <p class="action-desc">Create and manage API keys for your apps</p>
          </div>
          <ArrowUpRight class="action-arrow" />
        </button>
        
        <button class="action-card" @click="navigateTo('org-whitelist')">
          <div class="action-icon action-icon--blue">
            <ShieldCheck class="w-6 h-6" />
          </div>
          <div class="action-content">
            <h3 class="action-title">IP Whitelist</h3>
            <p class="action-desc">Control which IPs can access your tunnels</p>
          </div>
          <ArrowUpRight class="action-arrow" />
        </button>
        
        <button class="action-card" @click="navigateTo('org-settings')">
          <div class="action-icon action-icon--primary">
            <Settings class="w-6 h-6" />
          </div>
          <div class="action-content">
            <h3 class="action-title">Settings</h3>
            <p class="action-desc">Configure authentication policies</p>
          </div>
          <ArrowUpRight class="action-arrow" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  max-width: 1200px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
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
  color: var(--accent-secondary);
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
</style>
