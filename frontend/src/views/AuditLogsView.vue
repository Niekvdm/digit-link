<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useAuditLogs, useOrganizations, useApplications } from '@/composables'
import AppLayout from '@/components/layout/AppLayout.vue'
import { StatCard, LoadingState, EmptyState } from '@/components/shared'
import type { AuditEvent } from '@/types/api'
import {
  ScrollText,
  RefreshCw,
  CheckCircle,
  XCircle,
  Key,
  KeyRound,
  Globe,
  User,
  Building2,
  AppWindow,
  Filter,
  ChevronLeft,
  ChevronRight,
  Activity,
  ShieldAlert,
  Shield,
  Fingerprint
} from 'lucide-vue-next'

const { events, stats, total, loading, statsLoading, loadEvents, loadStats, refresh } = useAuditLogs()
const { organizations } = useOrganizations()
const { applications } = useApplications()

// Filters
const filterOrgId = ref<string>('')
const filterAppId = ref<string>('')

// Pagination
const currentPage = ref(1)
const pageSize = 20

const totalPages = computed(() => Math.ceil(total.value / pageSize))

const filteredApps = computed(() => {
  if (!filterOrgId.value) return applications.value
  return applications.value.filter(app => app.orgId === filterOrgId.value)
})

function formatTimestamp(timestamp: string) {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  // Less than 1 minute
  if (diff < 60000) return 'Just now'
  // Less than 1 hour
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  // Less than 24 hours
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}h ago`
  // Otherwise show date
  return date.toLocaleString()
}

function getAuthTypeIcon(authType: string) {
  switch (authType) {
    case 'basic': return Key
    case 'api_key': return KeyRound
    case 'oidc': return Globe
    default: return Shield
  }
}

function getAuthTypeLabel(authType: string): string {
  switch (authType) {
    case 'basic': return 'Basic'
    case 'api_key': return 'API Key'
    case 'oidc': return 'OIDC'
    default: return authType
  }
}

function getOrgName(orgId?: string): string {
  if (!orgId) return '-'
  const org = organizations.value.find(o => o.id === orgId)
  return org?.name || orgId.slice(0, 8) + '...'
}

function getAppSubdomain(appId?: string): string {
  if (!appId) return '-'
  const app = applications.value.find(a => a.id === appId)
  return app?.subdomain || appId.slice(0, 8) + '...'
}

async function handleFilter() {
  currentPage.value = 1
  await loadEvents({
    orgId: filterOrgId.value || undefined,
    appId: filterAppId.value || undefined,
    limit: pageSize,
    offset: 0
  })
}

async function goToPage(page: number) {
  if (page < 1 || page > totalPages.value) return
  currentPage.value = page
  await loadEvents({
    orgId: filterOrgId.value || undefined,
    appId: filterAppId.value || undefined,
    limit: pageSize,
    offset: (page - 1) * pageSize
  })
}

const successRate = computed(() => {
  if (!stats.value || stats.value.totalAttempts === 0) return 0
  return Math.round((stats.value.successCount / stats.value.totalAttempts) * 100)
})

// Clear app filter when org changes
watch(filterOrgId, () => {
  filterAppId.value = ''
  handleFilter()
})

watch(filterAppId, () => {
  handleFilter()
})
</script>

<template>
  <AppLayout>
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
      <h1 class="font-[var(--font-display)] text-3xl font-semibold">Audit Log</h1>
      <button
        class="btn btn-secondary"
        :disabled="loading || statsLoading"
        @click="refresh"
      >
        <RefreshCw
          class="w-4 h-4"
          :class="{ 'animate-spin': loading || statsLoading }"
        />
        Refresh
      </button>
    </div>

    <!-- Stats Grid -->
    <div class="grid grid-cols-2 lg:grid-cols-5 gap-4 mb-8">
      <StatCard
        label="Total Attempts"
        :value="stats?.totalAttempts ?? 0"
        :icon="Activity"
        :loading="statsLoading"
      />
      <StatCard
        label="Success Rate"
        :value="`${successRate}%`"
        :icon="CheckCircle"
        :loading="statsLoading"
      />
      <StatCard
        label="Failures Today"
        :value="stats?.failuresToday ?? 0"
        :icon="ShieldAlert"
        :loading="statsLoading"
      />
      <StatCard
        label="Unique IPs"
        :value="stats?.uniqueIps ?? 0"
        :icon="Fingerprint"
        :loading="statsLoading"
      />
      <StatCard
        label="Total Failures"
        :value="stats?.failureCount ?? 0"
        :icon="XCircle"
        :loading="statsLoading"
      />
    </div>

    <!-- Filter Bar -->
    <div class="mb-6 flex items-center gap-4">
      <Filter class="w-4 h-4 text-[var(--text-muted)]" />
      <div class="flex items-center gap-2">
        <Building2 class="w-4 h-4 text-[var(--text-muted)]" />
        <select
          v-model="filterOrgId"
          class="form-input py-2 pr-8 min-w-[180px]"
        >
          <option value="">All Organizations</option>
          <option v-for="org in organizations" :key="org.id" :value="org.id">
            {{ org.name }}
          </option>
        </select>
      </div>

      <div class="flex items-center gap-2">
        <AppWindow class="w-4 h-4 text-[var(--text-muted)]" />
        <select
          v-model="filterAppId"
          class="form-input py-2 pr-8 min-w-[180px]"
        >
          <option value="">All Applications</option>
          <option v-for="app in filteredApps" :key="app.id" :value="app.id">
            {{ app.subdomain }}
          </option>
        </select>
      </div>
    </div>

    <!-- Events Card -->
    <div class="card">
      <div class="card-header">
        <h2 class="card-title">
          <ScrollText class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          Authentication Events
          <span class="text-sm font-normal text-[var(--text-muted)] ml-2">
            ({{ total }} total)
          </span>
        </h2>
      </div>
      <div class="card-body p-0">
        <LoadingState v-if="loading && !events.length" message="Loading events..." />

        <EmptyState
          v-else-if="!events.length"
          :icon="ScrollText"
          title="No audit events found"
        />

        <div v-else>
          <!-- Table -->
          <div class="overflow-x-auto">
            <table class="w-full">
              <thead>
                <tr class="border-b border-[var(--border-subtle)]">
                  <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-[var(--text-muted)]">
                    Status
                  </th>
                  <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-[var(--text-muted)]">
                    Time
                  </th>
                  <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-[var(--text-muted)]">
                    Auth Type
                  </th>
                  <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-[var(--text-muted)]">
                    Organization
                  </th>
                  <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-[var(--text-muted)]">
                    Application
                  </th>
                  <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-[var(--text-muted)]">
                    Source IP
                  </th>
                  <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-[var(--text-muted)]">
                    Identity
                  </th>
                  <th class="text-left px-4 py-3 text-xs font-medium uppercase tracking-wider text-[var(--text-muted)]">
                    Details
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="event in events"
                  :key="event.id"
                  class="border-b border-[var(--border-subtle)] hover:bg-[var(--bg-elevated)] transition-colors"
                >
                  <!-- Status -->
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-2">
                      <CheckCircle
                        v-if="event.success"
                        class="w-4 h-4 text-[var(--accent-emerald)]"
                      />
                      <XCircle
                        v-else
                        class="w-4 h-4 text-[var(--accent-red)]"
                      />
                    </div>
                  </td>

                  <!-- Time -->
                  <td class="px-4 py-3">
                    <span class="text-sm text-[var(--text-secondary)]">
                      {{ formatTimestamp(event.timestamp) }}
                    </span>
                  </td>

                  <!-- Auth Type -->
                  <td class="px-4 py-3">
                    <div class="flex items-center gap-1.5">
                      <component
                        :is="getAuthTypeIcon(event.authType)"
                        class="w-3.5 h-3.5 text-[var(--text-muted)]"
                      />
                      <span class="text-sm">{{ getAuthTypeLabel(event.authType) }}</span>
                    </div>
                  </td>

                  <!-- Organization -->
                  <td class="px-4 py-3">
                    <span class="text-sm text-[var(--text-secondary)]">
                      {{ getOrgName(event.orgId) }}
                    </span>
                  </td>

                  <!-- Application -->
                  <td class="px-4 py-3">
                    <span class="text-sm font-mono text-[var(--accent-copper)]">
                      {{ getAppSubdomain(event.appId) }}
                    </span>
                  </td>

                  <!-- Source IP -->
                  <td class="px-4 py-3">
                    <code class="text-xs font-mono text-[var(--text-muted)]">
                      {{ event.sourceIp }}
                    </code>
                  </td>

                  <!-- Identity -->
                  <td class="px-4 py-3">
                    <div v-if="event.userIdentity" class="flex items-center gap-1.5">
                      <User class="w-3 h-3 text-[var(--text-muted)]" />
                      <span class="text-sm text-[var(--text-secondary)] truncate max-w-[150px]">
                        {{ event.userIdentity }}
                      </span>
                    </div>
                    <span v-else class="text-[var(--text-muted)]">-</span>
                  </td>

                  <!-- Details -->
                  <td class="px-4 py-3">
                    <span
                      v-if="event.failureReason"
                      class="text-xs text-[var(--accent-red)]"
                    >
                      {{ event.failureReason }}
                    </span>
                    <span
                      v-else-if="event.keyId"
                      class="text-xs text-[var(--text-muted)]"
                    >
                      Key: {{ event.keyId.slice(0, 8) }}...
                    </span>
                    <span v-else class="text-[var(--text-muted)]">-</span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Pagination -->
          <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t border-[var(--border-subtle)]">
            <div class="text-sm text-[var(--text-muted)]">
              Page {{ currentPage }} of {{ totalPages }}
            </div>
            <div class="flex items-center gap-2">
              <button
                class="btn btn-secondary btn-sm"
                :disabled="currentPage === 1"
                @click="goToPage(currentPage - 1)"
              >
                <ChevronLeft class="w-4 h-4" />
                Previous
              </button>
              <button
                class="btn btn-secondary btn-sm"
                :disabled="currentPage === totalPages"
                @click="goToPage(currentPage + 1)"
              >
                Next
                <ChevronRight class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>
