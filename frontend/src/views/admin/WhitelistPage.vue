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
  <div class="max-w-[1000px]">
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
    <div class="flex gap-4 py-5 px-6 bg-[rgba(var(--accent-blue-rgb),0.1)] border border-[rgba(var(--accent-blue-rgb),0.2)] rounded-xs mb-6 text-accent-blue">
      <Globe class="w-5 h-5 shrink-0" />
      <div>
        <strong class="block text-text-primary mb-1">Global Whitelist</strong>
        <p class="text-sm text-text-secondary m-0 leading-relaxed">These IP ranges are allowed to connect to all tunnels system-wide. Organization and application-specific whitelists can be managed from their respective settings.</p>
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
            class="form-input font-mono"
            placeholder="e.g., 192.168.1.0/24 or 10.0.0.1"
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
