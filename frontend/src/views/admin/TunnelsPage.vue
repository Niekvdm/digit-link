<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { PageHeader, DataTable, StatusBadge } from '@/components/ui'
import { useTunnels } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import { Cable, RefreshCw, ExternalLink } from 'lucide-vue-next'

const { tunnels, loading, error, fetchAll } = useTunnels()
const { formatDate } = useFormatters()

const refreshInterval = ref<ReturnType<typeof setInterval> | null>(null)

// Table columns
const columns = [
  { key: 'subdomain', label: 'Subdomain', sortable: true },
  { key: 'url', label: 'URL' },
  { key: 'createdAt', label: 'Connected At', sortable: true, width: '180px' },
]

onMounted(() => {
  fetchAll()
  // Auto-refresh every 10 seconds
  refreshInterval.value = setInterval(fetchAll, 10000)
})

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
})

function handleRefresh() {
  fetchAll()
}
</script>

<template>
  <div class="tunnels-page">
    <PageHeader 
      title="Active Tunnels" 
      description="Monitor currently active tunnel connections"
    >
      <template #actions>
        <button class="btn btn-secondary" @click="handleRefresh" :disabled="loading">
          <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': loading }" />
          Refresh
        </button>
      </template>
    </PageHeader>

    <!-- Stats banner -->
    <div class="stats-banner">
      <Cable class="w-6 h-6" />
      <div class="stats-content">
        <span class="stats-value">{{ tunnels.length }}</span>
        <span class="stats-label">Active Connections</span>
      </div>
      <div class="live-indicator">
        <span class="live-dot" />
        <span>Live</span>
      </div>
    </div>

    <!-- Error -->
    <div v-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="tunnels"
      :loading="loading"
      empty-title="No active tunnels"
      empty-description="There are no active tunnel connections at this time."
      row-key="subdomain"
    >
      <template #cell-subdomain="{ value }">
        <div class="tunnel-subdomain">
          <div class="tunnel-status" />
          <code>{{ value }}</code>
        </div>
      </template>
      
      <template #cell-url="{ value }">
        <a :href="value" target="_blank" class="tunnel-url">
          {{ value }}
          <ExternalLink class="w-3.5 h-3.5" />
        </a>
      </template>
      
      <template #cell-createdAt="{ value }">
        {{ formatDate(value) }}
      </template>
    </DataTable>
  </div>
</template>

<style scoped>
.tunnels-page {
  max-width: 1000px;
}

.stats-banner {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem 1.5rem;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  margin-bottom: 1.5rem;
  color: var(--accent-secondary);
}

.stats-content {
  flex: 1;
}

.stats-value {
  font-family: var(--font-display);
  font-size: 1.75rem;
  font-weight: 600;
  color: var(--text-primary);
  margin-right: 0.5rem;
}

.stats-label {
  font-size: 0.9375rem;
  color: var(--text-secondary);
}

.live-indicator {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--accent-secondary);
}

.live-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent-secondary);
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.tunnel-subdomain {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.tunnel-status {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--accent-secondary);
  box-shadow: 0 0 8px var(--accent-secondary);
}

.tunnel-subdomain code {
  font-family: var(--font-mono);
  font-size: 0.875rem;
  color: var(--accent-secondary);
}

.tunnel-url {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
  text-decoration: none;
  transition: color 0.15s ease;
}

.tunnel-url:hover {
  color: var(--accent-primary);
}

.animate-spin {
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.mb-4 {
  margin-bottom: 1rem;
}
</style>
