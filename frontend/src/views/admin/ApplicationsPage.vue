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
</script>

<template>
  <div class="applications-page">
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
    <div class="toolbar">
      <SearchInput v-model="searchQuery" placeholder="Search applications..." />
      <select v-model="filterOrgId" class="form-input filter-select">
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
        <div class="app-subdomain">
          <div class="app-icon">
            <AppWindow class="w-4 h-4" />
          </div>
          <code class="subdomain-code">{{ row.subdomain }}</code>
          <ExternalLink class="external-icon" />
        </div>
      </template>
      
      <template #cell-authMode="{ value }">
        <span class="auth-mode" :class="`auth-mode--${value}`">
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
        <div class="action-buttons">
          <button 
            class="icon-btn icon-btn--danger" 
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
      <form @submit.prevent="handleCreate" class="form">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="form-group">
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
        
        <div class="form-group">
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
        
        <div class="form-group">
          <label class="form-label" for="app-name">
            Display Name
            <span class="form-label-optional">(optional)</span>
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

<style scoped>
.applications-page {
  max-width: 1200px;
}

.toolbar {
  display: flex;
  gap: 1rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.filter-select {
  width: auto;
  min-width: 180px;
}

.app-subdomain {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.app-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: rgba(var(--accent-secondary-rgb), 0.1);
  color: var(--accent-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.subdomain-code {
  font-family: var(--font-mono);
  font-size: 0.875rem;
  color: var(--accent-secondary);
}

.external-icon {
  width: 14px;
  height: 14px;
  color: var(--text-muted);
  opacity: 0;
  transition: opacity 0.15s ease;
}

.app-subdomain:hover .external-icon {
  opacity: 1;
}

.auth-mode {
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.auth-mode--inherit {
  background: var(--bg-elevated);
  color: var(--text-secondary);
}

.auth-mode--disabled {
  background: rgba(var(--accent-amber-rgb), 0.1);
  color: var(--accent-amber);
}

.auth-mode--custom {
  background: rgba(var(--accent-primary-rgb), 0.1);
  color: var(--accent-primary);
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
