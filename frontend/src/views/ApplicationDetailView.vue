<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useApi } from '@/composables/useApi'
import AppLayout from '@/components/layout/AppLayout.vue'
import { LoadingState, EmptyState } from '@/components/shared'
import type { Application, TunnelStats } from '@/types/api'
import {
  AppWindow,
  ArrowLeft,
  RefreshCw,
  Cable,
  Clock,
  Activity,
  BarChart3,
  ArrowDownToLine,
  ArrowUpFromLine,
  Shield,
  ShieldCheck,
  ShieldBan,
  Building2,
  Circle,
  ExternalLink
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const { get } = useApi()

const appId = computed(() => route.params.id as string)
const application = ref<Application | null>(null)
const stats = ref<TunnelStats | null>(null)
const activeTunnels = ref<Array<{ subdomain: string; url: string; createdAt: string }>>([])
const loading = ref(false)
const refreshInterval = ref<number | null>(null)

async function loadApplication() {
  loading.value = true
  try {
    const data = await get<{ application: Application }>(`/admin/applications/${appId.value}`)
    application.value = data.application
    stats.value = data.application.stats || null
  } catch (err) {
    console.error('Failed to load application:', err)
  } finally {
    loading.value = false
  }
}

async function loadTunnels() {
  try {
    const data = await get<{ active: Array<{ subdomain: string; url: string; createdAt: string }> }>(`/admin/applications/${appId.value}/tunnels`)
    activeTunnels.value = data.active || []
  } catch (err) {
    console.error('Failed to load tunnels:', err)
  }
}

async function refresh() {
  await Promise.all([loadApplication(), loadTunnels()])
}

function formatBytes(bytes?: number): string {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

function formatDate(timestamp: string) {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleDateString()
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

function getAuthModeLabel(mode: string): string {
  switch (mode) {
    case 'inherit': return 'Inherit from Org'
    case 'disabled': return 'Disabled'
    case 'custom': return 'Custom'
    default: return mode
  }
}

function getAuthModeIcon(mode: string) {
  switch (mode) {
    case 'inherit': return Shield
    case 'disabled': return ShieldBan
    case 'custom': return ShieldCheck
    default: return Shield
  }
}

function goBack() {
  router.push('/applications')
}

onMounted(async () => {
  await refresh()
  // Auto-refresh every 10 seconds
  refreshInterval.value = window.setInterval(() => {
    loadTunnels()
  }, 10000)
})

onUnmounted(() => {
  if (refreshInterval.value) {
    clearInterval(refreshInterval.value)
  }
})
</script>

<template>
  <AppLayout>
    <!-- Back Button -->
    <button
      class="flex items-center gap-2 text-sm text-[var(--text-secondary)] hover:text-[var(--text-primary)] mb-6 transition-colors"
      @click="goBack"
    >
      <ArrowLeft class="w-4 h-4" />
      Back to Applications
    </button>

    <LoadingState v-if="loading && !application" message="Loading application..." />

    <template v-else-if="application">
      <!-- Page Header -->
      <div class="flex items-start justify-between mb-8">
        <div class="flex items-center gap-4">
          <div class="relative">
            <div class="w-14 h-14 rounded-xl bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)] flex items-center justify-center">
              <AppWindow class="w-7 h-7" />
            </div>
            <!-- Active Indicator -->
            <div
              v-if="application.isActive"
              class="absolute -top-1 -right-1 w-4 h-4 bg-[var(--accent-emerald)] rounded-full border-2 border-[var(--bg-surface)] flex items-center justify-center"
            >
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-[var(--accent-emerald)] opacity-75"></span>
            </div>
          </div>
          <div>
            <div class="flex items-center gap-3 mb-1">
              <h1 class="font-[var(--font-display)] text-3xl font-semibold font-mono text-[var(--accent-copper)]">
                {{ application.subdomain }}
              </h1>
              <span
                v-if="application.isActive"
                class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)]"
              >
                <Circle class="w-2 h-2 fill-current" />
                {{ application.activeTunnelCount }} Active
              </span>
            </div>
            <div class="flex items-center gap-4 text-sm text-[var(--text-secondary)]">
              <span v-if="application.name">{{ application.name }}</span>
              <span class="flex items-center gap-1.5">
                <Building2 class="w-3.5 h-3.5" />
                {{ application.orgName || 'Unknown Organization' }}
              </span>
              <span class="flex items-center gap-1.5">
                <component :is="getAuthModeIcon(application.authMode)" class="w-3.5 h-3.5" />
                {{ getAuthModeLabel(application.authMode) }}
              </span>
            </div>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <div v-if="application.isActive" class="live-indicator">
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
      <div class="grid grid-cols-4 gap-4 mb-6">
        <div class="p-5 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-xl">
          <div class="flex items-center gap-3 mb-3">
            <div class="w-10 h-10 rounded-lg bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)] flex items-center justify-center">
              <Activity class="w-5 h-5" />
            </div>
          </div>
          <div class="font-mono text-3xl font-semibold text-[var(--text-primary)] mb-1">
            {{ activeTunnels.length }}
          </div>
          <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">Active Now</div>
        </div>

        <div class="p-5 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-xl">
          <div class="flex items-center gap-3 mb-3">
            <div class="w-10 h-10 rounded-lg bg-[rgba(93,123,191,0.15)] text-[var(--accent-blue)] flex items-center justify-center">
              <BarChart3 class="w-5 h-5" />
            </div>
          </div>
          <div class="font-mono text-3xl font-semibold text-[var(--text-primary)] mb-1">
            {{ stats?.totalConnections ?? 0 }}
          </div>
          <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">Total Connections</div>
        </div>

        <div class="p-5 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-xl">
          <div class="flex items-center gap-3 mb-3">
            <div class="w-10 h-10 rounded-lg bg-[rgba(201,149,108,0.15)] text-[var(--accent-copper)] flex items-center justify-center">
              <ArrowUpFromLine class="w-5 h-5" />
            </div>
          </div>
          <div class="font-mono text-3xl font-semibold text-[var(--text-primary)] mb-1">
            {{ formatBytes(stats?.bytesSent) }}
          </div>
          <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">Data Sent</div>
        </div>

        <div class="p-5 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-xl">
          <div class="flex items-center gap-3 mb-3">
            <div class="w-10 h-10 rounded-lg bg-[rgba(178,131,191,0.15)] text-[var(--accent-purple)] flex items-center justify-center">
              <ArrowDownToLine class="w-5 h-5" />
            </div>
          </div>
          <div class="font-mono text-3xl font-semibold text-[var(--text-primary)] mb-1">
            {{ formatBytes(stats?.bytesReceived) }}
          </div>
          <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">Data Received</div>
        </div>
      </div>

      <!-- Active Tunnels Card -->
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">
            <Cable class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
            Active Tunnels
          </h2>
          <span class="text-xs text-[var(--text-muted)]">
            Auto-refreshes every 10s
          </span>
        </div>
        <div class="card-body">
          <EmptyState
            v-if="!activeTunnels.length"
            :icon="Cable"
            title="No active tunnels"
            description="Tunnels will appear here when clients connect to this application"
          />

          <div v-else class="grid gap-4" style="grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));">
            <div
              v-for="tunnel in activeTunnels"
              :key="tunnel.subdomain"
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
                      class="hover:underline inline-flex items-center gap-1"
                    >
                      {{ tunnel.url }}
                      <ExternalLink class="w-3 h-3" />
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

      <!-- Application Info Card -->
      <div class="card mt-6">
        <div class="card-header">
          <h2 class="card-title">
            <AppWindow class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
            Application Details
          </h2>
        </div>
        <div class="card-body">
          <div class="grid grid-cols-2 gap-6">
            <div>
              <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide mb-1">Subdomain</div>
              <div class="font-mono text-[var(--accent-copper)]">{{ application.subdomain }}</div>
            </div>
            <div>
              <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide mb-1">Display Name</div>
              <div>{{ application.name || '-' }}</div>
            </div>
            <div>
              <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide mb-1">Organization</div>
              <div>{{ application.orgName || '-' }}</div>
            </div>
            <div>
              <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide mb-1">Created</div>
              <div>{{ formatDate(application.createdAt) }}</div>
            </div>
            <div>
              <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide mb-1">Auth Mode</div>
              <div class="flex items-center gap-2">
                <component :is="getAuthModeIcon(application.authMode)" class="w-4 h-4" />
                {{ getAuthModeLabel(application.authMode) }}
              </div>
            </div>
            <div>
              <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide mb-1">Has Policy</div>
              <div>{{ application.hasPolicy ? 'Yes' : 'No' }}</div>
            </div>
          </div>
        </div>
      </div>
    </template>

    <EmptyState
      v-else
      :icon="AppWindow"
      title="Application not found"
      description="The application you're looking for doesn't exist or has been deleted"
    />
  </AppLayout>
</template>
