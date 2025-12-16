<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { 
  PageHeader, 
  DataTable, 
  Modal, 
  ConfirmDialog,
  SearchInput
} from '@/components/ui'
import { useWhitelist } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import type { AddWhitelistRequest } from '@/types/api'
import { Plus, Trash2, Globe, ShieldCheck } from 'lucide-vue-next'

const { globalWhitelist, loading, error, fetchGlobal, addGlobal, removeGlobal } = useWhitelist()
const { formatDate } = useFormatters()

// Search
const searchQuery = ref('')

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

onMounted(() => {
  fetchGlobal()
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
    await addGlobal(data)
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
    await removeGlobal(selectedEntry.value.id)
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
      description="Manage global IP whitelist for tunnel connections"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openAddModal">
          <Plus class="w-4 h-4" />
          Add IP Range
        </button>
      </template>
    </PageHeader>

    <!-- Info banner -->
    <div class="info-banner">
      <Globe class="w-5 h-5" />
      <div>
        <strong>Global Whitelist</strong>
        <p>These IP ranges are allowed to connect to all tunnels system-wide. Organization and application-specific whitelists can be managed from their respective settings.</p>
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

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="globalWhitelist"
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

.info-banner {
  display: flex;
  gap: 1rem;
  padding: 1.25rem 1.5rem;
  background: rgba(var(--accent-blue-rgb), 0.1);
  border: 1px solid rgba(var(--accent-blue-rgb), 0.2);
  border-radius: 12px;
  margin-bottom: 1.5rem;
  color: var(--accent-blue);
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
