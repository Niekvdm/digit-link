<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { 
  PageHeader, 
  DataTable, 
  Modal, 
  ConfirmDialog,
  SearchInput
} from '@/components/ui'
import { useWhitelist, useApplications } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import type { AddWhitelistRequest } from '@/types/api'
import { Plus, Trash2, ShieldCheck, Building2, AppWindow } from 'lucide-vue-next'

const { orgWhitelist, appWhitelists, loading, error, fetchOrgWhitelist, addOrgEntry, removeOrgEntry, addAppEntry, removeAppEntry } = useWhitelist()
const { applications, fetchAll: fetchApps } = useApplications()
const { formatDate } = useFormatters()

// View mode
type ViewMode = 'org' | 'app'
const viewMode = ref<ViewMode>('org')
const selectedAppId = ref('')

// Search
const searchQuery = ref('')

const currentWhitelist = computed(() => {
  if (viewMode.value === 'org') {
    return orgWhitelist.value
  }
  return selectedAppId.value ? (appWhitelists.value[selectedAppId.value] || []) : []
})

const filteredWhitelist = computed(() => {
  if (!searchQuery.value) return currentWhitelist.value
  const query = searchQuery.value.toLowerCase()
  return currentWhitelist.value.filter((entry: { ipRange: string; description?: string }) => 
    entry.ipRange.toLowerCase().includes(query) ||
    entry.description?.toLowerCase().includes(query)
  )
})

// Modals
const showAddModal = ref(false)
const showDeleteConfirm = ref(false)

// Form state
const formIpRange = ref('')
const formDescription = ref('')
const formLoading = ref(false)
const formError = ref('')
const selectedEntry = ref<{ id: string; ipRange: string } | null>(null)

// Table columns
const columns = [
  { key: 'ipRange', label: 'IP Range', sortable: true },
  { key: 'description', label: 'Description' },
  { key: 'createdAt', label: 'Created', sortable: true, width: '160px' },
]

onMounted(async () => {
  await Promise.all([fetchOrgWhitelist(), fetchApps()])
})

// Add
function openAddModal() {
  formIpRange.value = ''
  formDescription.value = ''
  formError.value = ''
  showAddModal.value = true
}

async function handleAdd() {
  if (!formIpRange.value.trim()) {
    formError.value = 'IP range is required'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    const data: AddWhitelistRequest = {
      ipRange: formIpRange.value.trim(),
      description: formDescription.value.trim() || undefined
    }
    
    if (viewMode.value === 'app' && selectedAppId.value) {
      await addAppEntry(selectedAppId.value, data)
    } else {
      await addOrgEntry(data)
    }
    
    showAddModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to add whitelist entry'
  } finally {
    formLoading.value = false
  }
}

// Delete
function openDeleteConfirm(entry: { id: string; ipRange: string }) {
  selectedEntry.value = entry
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!selectedEntry.value) return
  
  try {
    if (viewMode.value === 'app') {
      await removeAppEntry(selectedEntry.value.id)
    } else {
      await removeOrgEntry(selectedEntry.value.id)
    }
    showDeleteConfirm.value = false
  } catch (e) {
    console.error('Delete failed:', e)
  }
}
</script>

<template>
  <div class="whitelist-page">
    <PageHeader 
      title="IP Whitelist" 
      description="Control which IP addresses can access your tunnels"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openAddModal">
          <Plus class="w-4 h-4" />
          Add IP Range
        </button>
      </template>
    </PageHeader>

    <!-- View Mode Tabs -->
    <div class="view-tabs">
      <button 
        class="view-tab"
        :class="{ 'view-tab--active': viewMode === 'org' }"
        @click="viewMode = 'org'"
      >
        <Building2 class="w-4 h-4" />
        Organization
      </button>
      <button 
        class="view-tab"
        :class="{ 'view-tab--active': viewMode === 'app' }"
        @click="viewMode = 'app'"
      >
        <AppWindow class="w-4 h-4" />
        Application
      </button>
    </div>

    <!-- App Selector (when in app mode) -->
    <div v-if="viewMode === 'app'" class="app-selector">
      <select v-model="selectedAppId" class="form-input">
        <option value="" disabled>Select an application</option>
        <option v-for="app in applications" :key="app.id" :value="app.id">
          {{ app.name }} ({{ app.subdomain }})
        </option>
      </select>
    </div>

    <!-- Info banner -->
    <div class="info-banner">
      <ShieldCheck class="w-5 h-5" />
      <div>
        <strong>{{ viewMode === 'org' ? 'Organization Whitelist' : 'Application Whitelist' }}</strong>
        <p v-if="viewMode === 'org'">
          These IP ranges can access all tunnels in your organization.
        </p>
        <p v-else>
          These IP ranges can only access the selected application's tunnels.
        </p>
      </div>
    </div>

    <!-- Search -->
    <div class="toolbar">
      <SearchInput v-model="searchQuery" placeholder="Search IP ranges..." />
    </div>

    <!-- Error -->
    <div v-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- No app selected message -->
    <div v-if="viewMode === 'app' && !selectedAppId" class="select-app-message">
      <AppWindow class="w-12 h-12" />
      <h3>Select an Application</h3>
      <p>Choose an application to manage its IP whitelist.</p>
    </div>

    <!-- Table -->
    <DataTable
      v-else
      :columns="columns"
      :data="filteredWhitelist"
      :loading="loading"
      empty-title="No whitelist entries"
      empty-description="Add IP ranges to allow tunnel connections."
      row-key="id"
    >
      <template #cell-ipRange="{ value }">
        <div class="ip-range">
          <ShieldCheck class="w-4 h-4" />
          <code>{{ value }}</code>
        </div>
      </template>
      
      <template #cell-description="{ value }">
        {{ value || '—' }}
      </template>
      
      <template #cell-createdAt="{ value }">
        {{ formatDate(value) }}
      </template>
      
      <template #actions="{ row }">
        <div class="action-buttons">
          <button 
            class="icon-btn icon-btn--danger" 
            title="Remove" 
            @click.stop="openDeleteConfirm(row)"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </template>
      
      <template #emptyAction>
        <button class="btn btn-primary" @click="openAddModal">
          <Plus class="w-4 h-4" />
          Add IP Range
        </button>
      </template>
    </DataTable>

    <!-- Add Modal -->
    <Modal v-model="showAddModal" title="Add IP Range">
      <form @submit.prevent="handleAdd" class="form">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="form-group">
          <label class="form-label" for="ip-range">IP Range</label>
          <input
            id="ip-range"
            v-model="formIpRange"
            type="text"
            class="form-input form-input-mono"
            placeholder="e.g., 192.168.1.0/24 or 10.0.0.1"
            autofocus
          />
          <p class="form-hint">Enter a single IP or CIDR range</p>
        </div>
        
        <div class="form-group">
          <label class="form-label" for="ip-desc">
            Description
            <span class="form-label-optional">(optional)</span>
          </label>
          <input
            id="ip-desc"
            v-model="formDescription"
            type="text"
            class="form-input"
            placeholder="e.g., Office network"
          />
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showAddModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button 
          class="btn btn-primary" 
          @click="handleAdd" 
          :disabled="formLoading || !formIpRange.trim()"
        >
          {{ formLoading ? 'Adding...' : 'Add Entry' }}
        </button>
      </template>
    </Modal>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="Remove IP Range"
      :message="`Are you sure you want to remove '${selectedEntry?.ipRange}' from the whitelist?`"
      confirm-text="Remove"
      variant="danger"
      @confirm="handleDelete"
    />
  </div>
</template>

<style scoped>
.whitelist-page {
  max-width: 1000px;
}

.view-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 1.5rem;
}

.view-tab {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.view-tab:hover {
  background: var(--bg-elevated);
  border-color: var(--border-accent);
}

.view-tab--active {
  background: rgba(var(--accent-secondary-rgb), 0.1);
  border-color: var(--accent-secondary);
  color: var(--accent-secondary);
}

.app-selector {
  margin-bottom: 1.5rem;
}

.app-selector .form-input {
  max-width: 400px;
}

.info-banner {
  display: flex;
  gap: 1rem;
  padding: 1.25rem 1.5rem;
  background: rgba(var(--accent-secondary-rgb), 0.1);
  border: 1px solid rgba(var(--accent-secondary-rgb), 0.2);
  border-radius: 12px;
  margin-bottom: 1.5rem;
  color: var(--accent-secondary);
}

.info-banner strong {
  display: block;
  color: var(--text-primary);
  margin-bottom: 0.25rem;
}

.info-banner p {
  font-size: 0.875rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.5;
}

.toolbar {
  margin-bottom: 1.5rem;
}

.select-app-message {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 4rem 2rem;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  color: var(--text-muted);
}

.select-app-message h3 {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 1rem 0 0.5rem;
}

.select-app-message p {
  font-size: 0.9375rem;
  margin: 0;
  max-width: 280px;
}

.ip-range {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  color: var(--accent-secondary);
}

.ip-range code {
  font-family: var(--font-mono);
  font-size: 0.875rem;
}

.action-buttons {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.icon-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s ease;
}

.icon-btn:hover:not(:disabled) {
  background: var(--bg-elevated);
  color: var(--text-primary);
}

.icon-btn--danger:hover:not(:disabled) {
  background: rgba(var(--accent-red-rgb), 0.1);
  color: var(--accent-red);
}

.form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-label-optional {
  font-weight: 400;
  text-transform: none;
  letter-spacing: normal;
  color: var(--text-muted);
}

.mb-4 {
  margin-bottom: 1rem;
}
</style>
