<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/authStore'
import OrgLayout from '@/components/layout/OrgLayout.vue'
import { LoadingState } from '@/components/shared'
import {
  AppWindow,
  Activity,
  Cable,
  ArrowUpFromLine,
  ArrowDownToLine,
  ShieldCheck
} from 'lucide-vue-next'

const authStore = useAuthStore()

interface OrgStats {
  orgId: string
  applicationCount: number
  whitelistEntries: number
  activeTunnels: number
  totalConnections: number
  totalBytesSent: number
  totalBytesReceived: number
  liveTunnels: number
}

const stats = ref<OrgStats | null>(null)
const loading = ref(true)

async function loadStats() {
  loading.value = true
  try {
    const response = await fetch('/org/stats', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      stats.value = await response.json()
    }
  } catch (err) {
    console.error('Failed to load stats:', err)
  } finally {
    loading.value = false
  }
}

function formatBytes(bytes?: number): string {
  if (!bytes) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

onMounted(() => {
  loadStats()
})
</script>

<template>
  <OrgLayout>
    <!-- Page Header -->
    <div class="mb-8">
      <h1 class="font-[var(--font-display)] text-3xl font-semibold mb-1">Dashboard</h1>
      <p class="text-sm text-[var(--text-secondary)]">Overview of your organization's tunnel usage</p>
    </div>

    <LoadingState v-if="loading" message="Loading statistics..." />

    <template v-else-if="stats">
      <!-- Stats Grid -->
      <div class="grid grid-cols-2 lg:grid-cols-3 gap-4 mb-8">
        <!-- Applications -->
        <div class="card">
          <div class="card-body">
            <div class="flex items-center gap-3 mb-4">
              <div class="w-12 h-12 rounded-xl bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)] flex items-center justify-center">
                <AppWindow class="w-6 h-6" />
              </div>
            </div>
            <div class="font-mono text-4xl font-semibold text-[var(--text-primary)] mb-1">
              {{ stats.applicationCount }}
            </div>
            <div class="text-sm text-[var(--text-muted)]">Applications</div>
          </div>
        </div>

        <!-- Active Tunnels -->
        <div class="card">
          <div class="card-body">
            <div class="flex items-center gap-3 mb-4">
              <div class="w-12 h-12 rounded-xl bg-[rgba(93,123,191,0.15)] text-[var(--accent-blue)] flex items-center justify-center">
                <Cable class="w-6 h-6" />
              </div>
            </div>
            <div class="font-mono text-4xl font-semibold text-[var(--text-primary)] mb-1">
              {{ stats.liveTunnels }}
            </div>
            <div class="text-sm text-[var(--text-muted)]">Active Tunnels</div>
          </div>
        </div>

        <!-- Whitelist Entries -->
        <div class="card">
          <div class="card-body">
            <div class="flex items-center gap-3 mb-4">
              <div class="w-12 h-12 rounded-xl bg-[rgba(201,149,108,0.15)] text-[var(--accent-copper)] flex items-center justify-center">
                <ShieldCheck class="w-6 h-6" />
              </div>
            </div>
            <div class="font-mono text-4xl font-semibold text-[var(--text-primary)] mb-1">
              {{ stats.whitelistEntries }}
            </div>
            <div class="text-sm text-[var(--text-muted)]">Whitelist Entries</div>
          </div>
        </div>
      </div>

      <!-- Transfer Stats -->
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">
            <Activity class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
            Transfer Statistics
          </h2>
        </div>
        <div class="card-body">
          <div class="grid grid-cols-3 gap-8">
            <div class="text-center">
              <div class="font-mono text-3xl font-semibold text-[var(--text-primary)] mb-1">
                {{ stats.totalConnections }}
              </div>
              <div class="text-sm text-[var(--text-muted)]">Total Connections</div>
            </div>
            <div class="text-center">
              <div class="flex items-center justify-center gap-2 mb-1">
                <ArrowUpFromLine class="w-5 h-5 text-[var(--accent-copper)]" />
                <span class="font-mono text-3xl font-semibold text-[var(--text-primary)]">
                  {{ formatBytes(stats.totalBytesSent) }}
                </span>
              </div>
              <div class="text-sm text-[var(--text-muted)]">Data Sent</div>
            </div>
            <div class="text-center">
              <div class="flex items-center justify-center gap-2 mb-1">
                <ArrowDownToLine class="w-5 h-5 text-[var(--accent-purple)]" />
                <span class="font-mono text-3xl font-semibold text-[var(--text-primary)]">
                  {{ formatBytes(stats.totalBytesReceived) }}
                </span>
              </div>
              <div class="text-sm text-[var(--text-muted)]">Data Received</div>
            </div>
          </div>
        </div>
      </div>
    </template>
  </OrgLayout>
</template>
