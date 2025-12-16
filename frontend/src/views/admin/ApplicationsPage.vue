<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { 
  PageHeader, 
  DataTable, 
  Modal, 
  ConfirmDialog,
  SearchInput,
  StatusBadge
} from '@/components/ui'
import { useApplications, useOrganizations } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import type { Application, CreateApplicationRequest } from '@/types/api'
import { Plus, Trash2, AppWindow, ExternalLink } from 'lucide-vue-next'

const router = useRouter()
const { applications, loading, error, fetchAll, create, remove } = useApplications()
const { organizations, fetchAll: fetchOrgs } = useOrganizations()
const { formatDate } = useFormatters()

// Search & filter
const searchQuery = ref('')
const filterOrgId = ref('')

const filteredApplications = computed(() => {
  let result = applications.value
  
  if (filterOrgId.value) {
    result = result.filter(app => app.orgId === filterOrgId.value)
  }
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(app => 
      app.name.toLowerCase().includes(query) ||
      app.subdomain.toLowerCase().includes(query)
    )
  }
  
  return result
})

// Modals
const showCreateModal = ref(false)
const showDeleteConfirm = ref(false)

// Form state
const formOrgId = ref('')
const formSubdomain = ref('')
const formName = ref('')
const formLoading = ref(false)
const formError = ref('')
const selectedApp = ref<Application | null>(null)

// Table columns
const columns = [
  { key: 'subdomain', label: 'Subdomain', sortable: true },
  { key: 'name', label: 'Name', sortable: true },
  { key: 'orgName', label: 'Organization', sortable: true },
  { key: 'authMode', label: 'Auth', width: '100px' },
  { key: 'isActive', label: 'Status', width: '100px' },
  { key: 'createdAt', label: 'Created', sortable: true, width: '140px' },
]

onMounted(async () => {
  await Promise.all([fetchAll(), fetchOrgs()])
})

// Navigate to detail
function viewApplication(app: Application) {
  router.push({ name: 'admin-application-detail', params: { appId: app.id } })
}

// Create
function openCreateModal() {
  formOrgId.value = ''
  formSubdomain.value = ''
  formName.value = ''
  formError.value = ''
  showCreateModal.value = true
}

async function handleCreate() {
  if (!formOrgId.value || !formSubdomain.value.trim()) {
    formError.value = 'Organization and subdomain are required'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    const data: CreateApplicationRequest = {
      orgId: formOrgId.value,
      subdomain: formSubdomain.value.trim().toLowerCase(),
      name: formName.value.trim() || formSubdomain.value.trim()
    }
    await create(data)
    showCreateModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to create application'
  } finally {
    formLoading.value = false
  }
}

// Delete
function openDeleteConfirm(app: Application) {
  selectedApp.value = app
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!selectedApp.value) return
  
  try {
    await remove(selectedApp.value.id)
    showDeleteConfirm.value = false
  } catch (e) {
    console.error('Delete failed:', e)
  }
}

function getAuthModeLabel(mode: string): string {
  switch (mode) {
    case 'inherit': return 'Inherit'
    case 'disabled': return 'Disabled'
    case 'custom': return 'Custom'
    default: return mode
  }
}

const authModeClasses: Record<string, string> = {
  inherit: 'bg-bg-elevated text-text-secondary',
  disabled: 'bg-[rgba(var(--accent-amber-rgb),0.1)] text-accent-amber',
  custom: 'bg-[rgba(var(--accent-primary-rgb),0.1)] text-accent-primary'
}
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      title="Applications" 
      description="Manage applications and their subdomains"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          New Application
        </button>
      </template>
    </PageHeader>

    <!-- Toolbar -->
    <div class="flex gap-4 mb-6 flex-wrap">
      <SearchInput v-model="searchQuery" placeholder="Search applications..." />
      <select v-model="filterOrgId" class="form-input w-auto min-w-[180px]">
        <option value="">All Organizations</option>
        <option v-for="org in organizations" :key="org.id" :value="org.id">
          {{ org.name }}
        </option>
      </select>
    </div>

    <!-- Error -->
    <div v-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="filteredApplications"
      :loading="loading"
      empty-title="No applications"
      empty-description="Create your first application to get started."
      row-key="id"
      @row-click="viewApplication"
    >
      <template #cell-subdomain="{ row }">
        <div class="flex items-center gap-3 group">
          <div class="w-8 h-8 rounded-xs bg-[rgba(var(--accent-secondary-rgb),0.1)] text-accent-secondary flex items-center justify-center">
            <AppWindow class="w-4 h-4" />
          </div>
          <code class="font-mono text-sm text-accent-secondary">{{ row.subdomain }}</code>
          <ExternalLink class="w-3.5 h-3.5 text-text-muted opacity-0 transition-opacity duration-150 group-hover:opacity-100" />
        </div>
      </template>
      
      <template #cell-authMode="{ value }">
        <span 
          class="text-xs font-medium py-1 px-2 rounded"
          :class="authModeClasses[value] || authModeClasses.inherit"
        >
          {{ getAuthModeLabel(value) }}
        </span>
      </template>
      
      <template #cell-isActive="{ value }">
        <StatusBadge 
          :status="value ? 'active' : 'inactive'" 
          :label="value ? 'Active' : 'Inactive'"
          size="sm"
        />
      </template>
      
      <template #cell-createdAt="{ value }">
        {{ formatDate(value) }}
      </template>
      
      <template #actions="{ row }">
        <div class="flex items-center gap-1">
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-[rgba(var(--accent-red-rgb),0.1)] hover:text-accent-red"
            title="Delete" 
            @click.stop="openDeleteConfirm(row)"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </template>
      
      <template #emptyAction>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          Create Application
        </button>
      </template>
    </DataTable>

    <!-- Create Modal -->
    <Modal v-model="showCreateModal" title="New Application">
      <form @submit.prevent="handleCreate" class="flex flex-col gap-5">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="app-org">Organization</label>
          <select
            id="app-org"
            v-model="formOrgId"
            class="form-input"
            required
          >
            <option value="" disabled>Select organization</option>
            <option v-for="org in organizations" :key="org.id" :value="org.id">
              {{ org.name }}
            </option>
          </select>
        </div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="app-subdomain">Subdomain</label>
          <input
            id="app-subdomain"
            v-model="formSubdomain"
            type="text"
            class="form-input form-input-mono"
            placeholder="my-app"
            pattern="[a-z0-9-]+"
          />
          <p class="form-hint">Only lowercase letters, numbers, and hyphens</p>
        </div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="app-name">
            Display Name
            <span class="font-normal normal-case tracking-normal text-text-muted">(optional)</span>
          </label>
          <input
            id="app-name"
            v-model="formName"
            type="text"
            class="form-input"
            placeholder="My Application"
          />
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showCreateModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button 
          class="btn btn-primary" 
          @click="handleCreate" 
          :disabled="formLoading || !formOrgId || !formSubdomain.trim()"
        >
          {{ formLoading ? 'Creating...' : 'Create' }}
        </button>
      </template>
    </Modal>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="Delete Application"
      :message="`Are you sure you want to delete '${selectedApp?.name}'? This will also delete all associated API keys and policies.`"
      confirm-text="Delete"
      variant="danger"
      @confirm="handleDelete"
    />
  </div>
</template>
