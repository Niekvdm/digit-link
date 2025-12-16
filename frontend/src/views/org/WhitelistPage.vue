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
  <div class="max-w-[1200px]">
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
    <div class="flex gap-2 mb-6">
      <button 
        class="flex items-center gap-2 py-3 px-5 bg-bg-surface border rounded-xs text-sm font-medium cursor-pointer transition-all duration-200"
        :class="viewMode === 'org' 
          ? 'bg-[rgba(var(--accent-secondary-rgb),0.1)] border-accent-secondary text-accent-secondary' 
          : 'border-border-subtle text-text-secondary hover:bg-bg-elevated hover:border-border-accent'"
        @click="viewMode = 'org'"
      >
        <Building2 class="w-4 h-4" />
        Organization
      </button>
      <button 
        class="flex items-center gap-2 py-3 px-5 bg-bg-surface border rounded-xs text-sm font-medium cursor-pointer transition-all duration-200"
        :class="viewMode === 'app' 
          ? 'bg-[rgba(var(--accent-secondary-rgb),0.1)] border-accent-secondary text-accent-secondary' 
          : 'border-border-subtle text-text-secondary hover:bg-bg-elevated hover:border-border-accent'"
        @click="viewMode = 'app'"
      >
        <AppWindow class="w-4 h-4" />
        Application
      </button>
    </div>

    <!-- App Selector (when in app mode) -->
    <div v-if="viewMode === 'app'" class="mb-6">
      <select v-model="selectedAppId" class="form-input max-w-[400px]">
        <option value="" disabled>Select an application</option>
        <option v-for="app in applications" :key="app.id" :value="app.id">
          {{ app.name }} ({{ app.subdomain }})
        </option>
      </select>
    </div>

    <!-- Info banner -->
    <div class="flex gap-4 py-5 px-6 bg-[rgba(var(--accent-secondary-rgb),0.1)] border border-[rgba(var(--accent-secondary-rgb),0.2)] rounded-xs mb-6 text-accent-secondary">
      <ShieldCheck class="w-5 h-5 shrink-0 mt-0.5" />
      <div>
        <strong class="block text-text-primary mb-1">{{ viewMode === 'org' ? 'Organization Whitelist' : 'Application Whitelist' }}</strong>
        <p v-if="viewMode === 'org'" class="text-sm text-text-secondary m-0 leading-relaxed">
          These IP ranges can access all tunnels in your organization.
        </p>
        <p v-else class="text-sm text-text-secondary m-0 leading-relaxed">
          These IP ranges can only access the selected application's tunnels.
        </p>
      </div>
    </div>

    <!-- Search -->
    <div class="mb-6">
      <SearchInput v-model="searchQuery" placeholder="Search IP ranges..." />
    </div>

    <!-- Error -->
    <div v-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- No app selected message -->
    <div 
      v-if="viewMode === 'app' && !selectedAppId" 
      class="flex flex-col items-center justify-center text-center py-16 px-8 bg-bg-surface border border-border-subtle rounded-xs text-text-muted"
    >
      <AppWindow class="w-12 h-12" />
      <h3 class="text-lg font-semibold text-text-primary mt-4 mb-2">Select an Application</h3>
      <p class="text-[0.9375rem] m-0 max-w-[280px]">Choose an application to manage its IP whitelist.</p>
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
        <div class="flex items-center gap-2.5 text-accent-secondary">
          <ShieldCheck class="w-4 h-4" />
          <code class="font-mono text-sm">{{ value }}</code>
        </div>
      </template>
      
      <template #cell-description="{ value }">
        {{ value || '—' }}
      </template>
      
      <template #cell-createdAt="{ value }">
        {{ formatDate(value) }}
      </template>
      
      <template #actions="{ row }">
        <div class="flex items-center gap-1">
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-[rgba(var(--accent-red-rgb),0.1)] hover:text-accent-red"
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
      <form @submit.prevent="handleAdd" class="flex flex-col gap-5">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
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
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="ip-desc">
            Description
            <span class="font-normal normal-case tracking-normal text-text-muted">(optional)</span>
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
