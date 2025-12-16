<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { 
  PageHeader, 
  DataTable, 
  StatCard,
  SearchInput,
  Pagination,
  StatusBadge
} from '@/components/ui'
import { useAuditLogs } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import { ShieldCheck, ShieldX, Activity, AlertTriangle, RefreshCw } from 'lucide-vue-next'

const { events, stats, total, loading, error, fetchEvents, fetchStats } = useAuditLogs()
const { formatDate } = useFormatters()

// Pagination
const currentPage = ref(1)
const pageSize = 50

// Filters
const searchQuery = ref('')
const filterSuccess = ref<boolean | null>(null)

// Table columns
const columns = [
  { key: 'timestamp', label: 'Time', sortable: true, width: '180px' },
  { key: 'authType', label: 'Auth Type', width: '100px' },
  { key: 'success', label: 'Result', width: '100px' },
  { key: 'userIdentity', label: 'User / Key' },
  { key: 'sourceIp', label: 'Source IP', width: '140px' },
  { key: 'failureReason', label: 'Details' },
]

const filteredEvents = computed(() => {
  let result = events.value
  
  if (filterSuccess.value !== null) {
    result = result.filter(e => e.success === filterSuccess.value)
  }
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(e => 
      e.userIdentity?.toLowerCase().includes(query) ||
      e.sourceIp?.toLowerCase().includes(query)
    )
  }
  
  return result
})

onMounted(async () => {
  await Promise.all([
    fetchEvents({ limit: pageSize, offset: 0 }),
    fetchStats()
  ])
})

async function loadPage(page: number) {
  currentPage.value = page
  await fetchEvents({
    limit: pageSize,
    offset: (page - 1) * pageSize
  })
}

function handleRefresh() {
  loadPage(currentPage.value)
  fetchStats()
}

function getAuthTypeLabel(type: string): string {
  switch (type) {
    case 'basic': return 'Basic'
    case 'api_key': return 'API Key'
    case 'oidc': return 'OIDC'
    default: return type
  }
}
</script>

<template>
  <div class="audit-page">
    <PageHeader 
      title="Audit Logs" 
      description="View authentication attempts and security events"
    >
      <template #actions>
        <button class="btn btn-secondary" @click="handleRefresh" :disabled="loading">
          <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': loading }" />
          Refresh
        </button>
      </template>
    </PageHeader>

    <!-- Stats Grid -->
    <div class="stats-grid" v-if="stats">
      <StatCard
        label="Total Attempts"
        :value="stats.totalAttempts"
        :icon="Activity"
        color="primary"
      />
      <StatCard
        label="Successful"
        :value="stats.successCount"
        :icon="ShieldCheck"
        color="secondary"
      />
      <StatCard
        label="Failed"
        :value="stats.failureCount"
        :icon="ShieldX"
        color="red"
      />
      <StatCard
        label="Failed Today"
        :value="stats.failuresToday"
        :icon="AlertTriangle"
        color="amber"
      />
    </div>

    <!-- Toolbar -->
    <div class="toolbar">
      <SearchInput v-model="searchQuery" placeholder="Search by user or IP..." />
      <div class="filter-buttons">
        <button 
          class="filter-btn"
          :class="{ 'filter-btn--active': filterSuccess === null }"
          @click="filterSuccess = null"
        >
          All
        </button>
        <button 
          class="filter-btn filter-btn--success"
          :class="{ 'filter-btn--active': filterSuccess === true }"
          @click="filterSuccess = true"
        >
          Success
        </button>
        <button 
          class="filter-btn filter-btn--failure"
          :class="{ 'filter-btn--active': filterSuccess === false }"
          @click="filterSuccess = false"
        >
          Failed
        </button>
      </div>
    </div>

    <!-- Error -->
    <div v-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="filteredEvents"
      :loading="loading"
      empty-title="No audit events"
      empty-description="There are no authentication events to display."
      row-key="id"
    >
      <template #cell-timestamp="{ value }">
        {{ formatDate(value) }}
      </template>
      
      <template #cell-authType="{ value }">
        <span class="auth-type-badge">
          {{ getAuthTypeLabel(value) }}
        </span>
      </template>
      
      <template #cell-success="{ value }">
        <StatusBadge 
          :status="value ? 'success' : 'error'" 
          :label="value ? 'Success' : 'Failed'"
          size="sm"
        />
      </template>
      
      <template #cell-userIdentity="{ value }">
        <code v-if="value" class="user-identity">{{ value }}</code>
        <span v-else class="text-muted">—</span>
      </template>
      
      <template #cell-sourceIp="{ value }">
        <code class="source-ip">{{ value }}</code>
      </template>
      
      <template #cell-failureReason="{ value, row }">
        <span v-if="!row.success && value" class="failure-reason">{{ value }}</span>
        <span v-else class="text-muted">—</span>
      </template>
    </DataTable>

    <!-- Pagination -->
    <Pagination
      v-model="currentPage"
      :total="total"
      :page-size="pageSize"
      @update:model-value="loadPage"
    />
  </div>
</template>

<style scoped>
.audit-page {
  max-width: 1400px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.filter-buttons {
  display: flex;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  overflow: hidden;
}

.filter-btn {
  padding: 0.5rem 1rem;
  font-size: 0.8125rem;
  font-weight: 500;
  border: none;
  background: var(--bg-surface);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.filter-btn:not(:last-child) {
  border-right: 1px solid var(--border-subtle);
}

.filter-btn:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
}

.filter-btn--active {
  background: rgba(var(--accent-primary-rgb), 0.1);
  color: var(--accent-primary);
}

.filter-btn--success.filter-btn--active {
  background: rgba(var(--accent-secondary-rgb), 0.1);
  color: var(--accent-secondary);
}

.filter-btn--failure.filter-btn--active {
  background: rgba(var(--accent-red-rgb), 0.1);
  color: var(--accent-red);
}

.auth-type-badge {
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  background: var(--bg-elevated);
  color: var(--text-secondary);
}

.user-identity {
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  color: var(--text-primary);
}

.source-ip {
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  color: var(--text-secondary);
}

.failure-reason {
  font-size: 0.8125rem;
  color: var(--accent-red);
}

.text-muted {
  color: var(--text-muted);
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
