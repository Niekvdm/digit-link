<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { 
  PageHeader, 
  DataTable, 
  Modal, 
  ConfirmDialog,
  SearchInput,
  PolicyEditor,
  StatusBadge
} from '@/components/ui'
import { useOrganizations } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import type { Organization, SetPolicyRequest } from '@/types/api'
import { Plus, Settings, Trash2, Pencil, Building2 } from 'lucide-vue-next'

const { organizations, loading, error, fetchAll, create, update, remove, getPolicy, setPolicy } = useOrganizations()
const { formatDate } = useFormatters()

// Search
const searchQuery = ref('')

const filteredOrganizations = computed(() => {
  if (!searchQuery.value) return organizations.value
  const query = searchQuery.value.toLowerCase()
  return organizations.value.filter(org => 
    org.name.toLowerCase().includes(query)
  )
})

// Modals
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showPolicyModal = ref(false)
const showDeleteConfirm = ref(false)

// Form state
const formName = ref('')
const formLoading = ref(false)
const formError = ref('')
const selectedOrg = ref<Organization | null>(null)
const currentPolicy = ref<SetPolicyRequest | null>(null)

// Table columns
const columns = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'appCount', label: 'Applications', sortable: true, width: '120px' },
  { key: 'hasPolicy', label: 'Auth Policy', width: '120px' },
  { key: 'createdAt', label: 'Created', sortable: true, width: '160px' },
]

onMounted(() => {
  fetchAll()
})

// Create
function openCreateModal() {
  formName.value = ''
  formError.value = ''
  showCreateModal.value = true
}

async function handleCreate() {
  if (!formName.value.trim()) {
    formError.value = 'Name is required'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    await create(formName.value.trim())
    showCreateModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to create organization'
  } finally {
    formLoading.value = false
  }
}

// Edit
function openEditModal(org: Organization) {
  selectedOrg.value = org
  formName.value = org.name
  formError.value = ''
  showEditModal.value = true
}

async function handleUpdate() {
  if (!selectedOrg.value || !formName.value.trim()) {
    formError.value = 'Name is required'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    await update(selectedOrg.value.id, formName.value.trim())
    showEditModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to update organization'
  } finally {
    formLoading.value = false
  }
}

// Policy
async function openPolicyModal(org: Organization) {
  selectedOrg.value = org
  formError.value = ''
  
  try {
    const policy = await getPolicy(org.id)
    currentPolicy.value = policy as SetPolicyRequest | null
  } catch (e) {
    currentPolicy.value = null
  }
  
  showPolicyModal.value = true
}

async function handleSetPolicy(policy: SetPolicyRequest) {
  if (!selectedOrg.value) return
  
  formLoading.value = true
  formError.value = ''
  
  try {
    await setPolicy(selectedOrg.value.id, policy)
    showPolicyModal.value = false
    await fetchAll() // Refresh to update hasPolicy
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to save policy'
  } finally {
    formLoading.value = false
  }
}

// Delete
function openDeleteConfirm(org: Organization) {
  selectedOrg.value = org
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!selectedOrg.value) return
  
  try {
    await remove(selectedOrg.value.id)
    showDeleteConfirm.value = false
  } catch (e) {
    console.error('Delete failed:', e)
  }
}
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      title="Organizations" 
      description="Manage organizations and their authentication policies"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          New Organization
        </button>
      </template>
    </PageHeader>

    <!-- Search -->
    <div class="mb-6">
      <SearchInput v-model="searchQuery" placeholder="Search organizations..." />
    </div>

    <!-- Error -->
    <div v-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="filteredOrganizations"
      :loading="loading"
      empty-title="No organizations"
      empty-description="Create your first organization to get started."
      row-key="id"
    >
      <template #cell-name="{ row }">
        <div class="flex items-center gap-3">
          <div class="w-8 h-8 rounded-xs bg-[rgba(var(--accent-primary-rgb),0.1)] text-accent-primary flex items-center justify-center">
            <Building2 class="w-4 h-4" />
          </div>
          <span>{{ row.name }}</span>
        </div>
      </template>
      
      <template #cell-hasPolicy="{ value }">
        <StatusBadge 
          :status="value ? 'active' : 'inactive'" 
          :label="value ? 'Configured' : 'Not Set'"
          size="sm"
        />
      </template>
      
      <template #cell-createdAt="{ value }">
        {{ formatDate(value) }}
      </template>
      
      <template #actions="{ row }">
        <div class="flex items-center gap-1">
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-bg-elevated hover:text-text-primary"
            title="Configure Policy" 
            @click.stop="openPolicyModal(row)"
          >
            <Settings class="w-4 h-4" />
          </button>
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-bg-elevated hover:text-text-primary"
            title="Edit" 
            @click.stop="openEditModal(row)"
          >
            <Pencil class="w-4 h-4" />
          </button>
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-[rgba(var(--accent-red-rgb),0.1)] hover:text-accent-red disabled:opacity-30 disabled:cursor-not-allowed"
            title="Delete" 
            @click.stop="openDeleteConfirm(row)"
            :disabled="row.appCount > 0"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </template>
      
      <template #emptyAction>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          Create Organization
        </button>
      </template>
    </DataTable>

    <!-- Create Modal -->
    <Modal v-model="showCreateModal" title="New Organization">
      <form @submit.prevent="handleCreate" class="flex flex-col gap-4">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="org-name">Organization Name</label>
          <input
            id="org-name"
            v-model="formName"
            type="text"
            class="form-input"
            placeholder="Enter organization name"
            autofocus
          />
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showCreateModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button class="btn btn-primary" @click="handleCreate" :disabled="formLoading || !formName.trim()">
          {{ formLoading ? 'Creating...' : 'Create' }}
        </button>
      </template>
    </Modal>

    <!-- Edit Modal -->
    <Modal v-model="showEditModal" title="Edit Organization">
      <form @submit.prevent="handleUpdate" class="flex flex-col gap-4">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="edit-org-name">Organization Name</label>
          <input
            id="edit-org-name"
            v-model="formName"
            type="text"
            class="form-input"
            placeholder="Enter organization name"
            autofocus
          />
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showEditModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button class="btn btn-primary" @click="handleUpdate" :disabled="formLoading || !formName.trim()">
          {{ formLoading ? 'Saving...' : 'Save Changes' }}
        </button>
      </template>
    </Modal>

    <!-- Policy Modal -->
    <Modal v-model="showPolicyModal" :title="`Auth Policy: ${selectedOrg?.name}`" size="lg">
      <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
      <PolicyEditor 
        :initial-policy="currentPolicy"
        @submit="handleSetPolicy"
        @cancel="showPolicyModal = false"
      />
    </Modal>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="Delete Organization"
      :message="`Are you sure you want to delete '${selectedOrg?.name}'? This action cannot be undone.`"
      confirm-text="Delete"
      variant="danger"
      @confirm="handleDelete"
    />
  </div>
</template>
