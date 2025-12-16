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
  <div class="max-w-[1200px]">
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
    <div class="flex items-center gap-4 py-5 px-6 bg-bg-surface border border-border-subtle rounded-xs mb-6 text-accent-secondary">
      <Cable class="w-6 h-6" />
      <div class="flex-1">
        <span class="font-display text-[1.75rem] font-semibold text-text-primary mr-2">{{ tunnels.length }}</span>
        <span class="text-[0.9375rem] text-text-secondary">Active Connections</span>
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
        <div class="flex items-center gap-3">
          <div class="tunnel-status" />
          <code class="font-mono text-sm text-accent-secondary">{{ value }}</code>
        </div>
      </template>
      
      <template #cell-url="{ value }">
        <a 
          :href="value as string" 
          target="_blank" 
          class="flex items-center gap-2 text-text-secondary text-sm no-underline transition-colors duration-150 hover:text-accent-primary"
        >
          {{ value }}
          <ExternalLink class="w-3.5 h-3.5" />
        </a>
      </template>
      
      <template #cell-createdAt="{ value }">
        {{ formatDate(value as string) }}
      </template>
    </DataTable>
  </div>
</template>

<style>
  @reference "../../style.css";
	.tunnel-status {
	@apply w-2.5 h-2.5 rounded-full bg-accent-secondary;
	box-shadow: 0 0 8px var(--accent-secondary);
	}
</style>
