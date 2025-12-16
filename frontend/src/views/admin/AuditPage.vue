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

function getFilterBtnClass(value: boolean | null, type: 'all' | 'success' | 'failure') {
  const isActive = filterSuccess.value === value
  const base = 'py-2 px-4 text-[0.8125rem] font-medium border-none cursor-pointer transition-all duration-150'
  
  if (!isActive) {
    return `${base} bg-bg-surface text-text-secondary hover:bg-bg-elevated hover:text-text-primary`
  }
  
  if (type === 'success') {
    return `${base} bg-[rgba(var(--accent-secondary-rgb),0.1)] text-accent-secondary`
  }
  if (type === 'failure') {
    return `${base} bg-[rgba(var(--accent-red-rgb),0.1)] text-accent-red`
  }
  return `${base} bg-[rgba(var(--accent-primary-rgb),0.1)] text-accent-primary`
}
</script>

<template>
  <div class="max-w-[1400px]">
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
    <div class="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-4 mb-6" v-if="stats">
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
    <div class="flex items-center gap-4 mb-6 flex-wrap">
      <SearchInput v-model="searchQuery" placeholder="Search by user or IP..." />
      <div class="flex border border-border-subtle rounded-xs overflow-hidden">
        <button 
          :class="getFilterBtnClass(null, 'all')"
          @click="filterSuccess = null"
        >
          All
        </button>
        <button 
          class="border-l border-border-subtle"
          :class="getFilterBtnClass(true, 'success')"
          @click="filterSuccess = true"
        >
          Success
        </button>
        <button 
          class="border-l border-border-subtle"
          :class="getFilterBtnClass(false, 'failure')"
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
        <span class="text-xs font-medium py-1 px-2 rounded bg-bg-elevated text-text-secondary">
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
        <code v-if="value" class="font-mono text-[0.8125rem] text-text-primary">{{ value }}</code>
        <span v-else class="text-text-muted">—</span>
      </template>
      
      <template #cell-sourceIp="{ value }">
        <code class="font-mono text-[0.8125rem] text-text-secondary">{{ value }}</code>
      </template>
      
      <template #cell-failureReason="{ value, row }">
        <span v-if="!row.success && value" class="text-[0.8125rem] text-accent-red">{{ value }}</span>
        <span v-else class="text-text-muted">—</span>
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
