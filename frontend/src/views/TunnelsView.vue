<script setup lang="ts">
import { computed } from 'vue'
import { useStats, useTunnels } from '@/composables'
import AppLayout from '@/components/layout/AppLayout.vue'
import { LoadingState, EmptyState } from '@/components/shared'
import { Cable, RefreshCw, Clock } from 'lucide-vue-next'

const { stats } = useStats()
const { tunnels, loading, refresh } = useTunnels(true, 10000)

const activeTunnelsCount = computed(() => tunnels.value.length)

function formatBytes(bytes?: number) {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatDuration(timestamp: string) {
  if (!timestamp) return 'Unknown'
  const start = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - start.getTime()

  if (diff < 60000) return 'Just connected'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  if (diff < 86400000) {
    const hours = Math.floor(diff / 3600000)
    const mins = Math.floor((diff % 3600000) / 60000)
    return `${hours}h ${mins}m`
  }
  return `${Math.floor(diff / 86400000)}d ago`
}
</script>

<template>
  <AppLayout>
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="font-[var(--font-display)] text-3xl font-semibold mb-1">Active Tunnels</h1>
        <p class="text-sm text-[var(--text-secondary)]">Monitor currently connected tunnel clients</p>
      </div>
      <div class="flex items-center gap-4">
        <div class="live-indicator">
          <div class="live-dot" />
          Live
        </div>
        <button 
          class="btn btn-secondary"
          :disabled="loading"
          @click="refresh"
        >
          <RefreshCw 
            class="w-4 h-4" 
            :class="{ 'animate-spin': loading }"
          />
          Refresh
        </button>
      </div>
    </div>

    <!-- Stats Row -->
    <div class="flex gap-8 p-4 bg-[var(--bg-deep)] rounded-lg mb-6">
      <div class="text-center">
        <div class="font-mono text-2xl font-medium text-[var(--text-primary)]">
          {{ activeTunnelsCount }}
        </div>
        <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">Active Now</div>
      </div>
      <div class="text-center">
        <div class="font-mono text-2xl font-medium text-[var(--text-primary)]">
          {{ stats?.totalTunnels ?? 0 }}
        </div>
        <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">Total Connections</div>
      </div>
      <div class="text-center">
        <div class="font-mono text-2xl font-medium text-[var(--text-primary)]">
          {{ formatBytes(stats?.totalBytesSent) }}
        </div>
        <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">Data Sent</div>
      </div>
      <div class="text-center">
        <div class="font-mono text-2xl font-medium text-[var(--text-primary)]">
          {{ formatBytes(stats?.totalBytesReceived) }}
        </div>
        <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">Data Received</div>
      </div>
    </div>

    <!-- Tunnels Card -->
    <div class="card">
      <div class="card-header">
        <h2 class="card-title">
          <Cable class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          Connected Tunnels
        </h2>
        <span class="text-xs text-[var(--text-muted)]">
          Auto-refreshes every 10s
        </span>
      </div>
      <div class="card-body">
        <LoadingState v-if="loading && !tunnels.length" message="Loading tunnels..." />
        
        <EmptyState 
          v-else-if="!tunnels.length"
          :icon="Cable"
          title="No active tunnels"
          description="Tunnels will appear here when clients connect"
        />

        <div v-else class="grid gap-4" style="grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));">
          <div
            v-for="tunnel in tunnels"
            :key="tunnel.id"
            class="bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-xl p-5 hover:border-[var(--border-accent)] transition-colors"
          >
            <!-- Header -->
            <div class="flex items-start gap-3 mb-4">
              <div class="tunnel-status mt-1.5 flex-shrink-0" />
              <div class="flex-1 min-w-0">
                <div class="font-mono font-medium mb-1 break-all">
                  {{ tunnel.subdomain }}
                </div>
                <div class="text-xs text-[var(--accent-blue)] break-all">
                  <a 
                    :href="tunnel.url" 
                    target="_blank"
                    class="hover:underline"
                  >
                    {{ tunnel.url }}
                  </a>
                </div>
              </div>
            </div>

            <!-- Meta -->
            <div class="flex flex-wrap gap-4 pt-4 border-t border-[var(--border-subtle)] text-xs text-[var(--text-muted)]">
              <div class="flex items-center gap-1.5">
                <Clock class="w-3 h-3" />
                {{ formatDuration(tunnel.createdAt) }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>
