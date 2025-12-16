<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { 
  PageHeader, 
  DataTable, 
  Modal, 
  ConfirmDialog,
  SearchInput,
  TokenReveal
} from '@/components/ui'
import { useAPIKeys, useOrganizations, useApplications } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import type { APIKey, CreateAPIKeyRequest } from '@/types/api'
import { Plus, Trash2, Key, Building2, AppWindow } from 'lucide-vue-next'

const { apiKeys, loading, error, fetchByOrg, create, remove } = useAPIKeys()
const { organizations, fetchAll: fetchOrgs } = useOrganizations()
const { applications, fetchAll: fetchApps } = useApplications()
const { formatDate } = useFormatters()

// Filters
const searchQuery = ref('')
const filterOrgId = ref('')

// Computed: filtered apps for selected org
const filteredApps = computed(() => {
  if (!filterOrgId.value) return []
  return applications.value.filter(app => app.orgId === filterOrgId.value)
})

// Computed: apps filtered by formOrgId for the create modal
const formApps = computed(() => {
  if (!formOrgId.value) return []
  return applications.value.filter(app => app.orgId === formOrgId.value)
})

const filteredKeys = computed(() => {
  if (!searchQuery.value) return apiKeys.value
  const query = searchQuery.value.toLowerCase()
  return apiKeys.value.filter(key => 
    key.description.toLowerCase().includes(query) ||
    key.keyPrefix.toLowerCase().includes(query)
  )
})

// Modals
const showCreateModal = ref(false)
const showTokenModal = ref(false)
const showDeleteConfirm = ref(false)

// Form state
const formOrgId = ref('')
const formAppId = ref('')
const formDescription = ref('')
const formExpiresIn = ref<number | undefined>(undefined)
const formLoading = ref(false)
const formError = ref('')
const selectedKey = ref<APIKey | null>(null)
const generatedRawKey = ref('')

// Table columns
const columns = [
  { key: 'keyPrefix', label: 'Key Prefix', width: '120px' },
  { key: 'description', label: 'Description', sortable: true },
  { key: 'scope', label: 'Scope', width: '150px' },
  { key: 'createdAt', label: 'Created', sortable: true, width: '140px' },
  { key: 'lastUsed', label: 'Last Used', sortable: true, width: '140px' },
  { key: 'expiresAt', label: 'Expires', sortable: true, width: '140px' },
]

onMounted(async () => {
  await Promise.all([fetchOrgs(), fetchApps()])
})

// Load keys when org filter changes
async function handleOrgChange() {
  if (filterOrgId.value) {
    await fetchByOrg(filterOrgId.value)
  } else {
    // Clear keys if no org selected (would need to implement fetchAll in API)
  }
}

// Create
function openCreateModal() {
  formOrgId.value = filterOrgId.value
  formAppId.value = ''
  formDescription.value = ''
  formExpiresIn.value = undefined
  formError.value = ''
  showCreateModal.value = true
}

async function handleCreate() {
  if (!formOrgId.value || !formDescription.value.trim()) {
    formError.value = 'Organization and description are required'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    const data: CreateAPIKeyRequest = {
      orgId: formOrgId.value,
      description: formDescription.value.trim()
    }
    if (formAppId.value) {
      data.appId = formAppId.value
    }
    if (formExpiresIn.value) {
      data.expiresIn = formExpiresIn.value
    }
    
    const result = await create(data)
    generatedRawKey.value = result.rawKey
    showCreateModal.value = false
    showTokenModal.value = true
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to create API key'
  } finally {
    formLoading.value = false
  }
}

// Delete
function openDeleteConfirm(key: APIKey) {
  selectedKey.value = key
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!selectedKey.value) return
  
  try {
    await remove(selectedKey.value.id)
    showDeleteConfirm.value = false
  } catch (e) {
    console.error('Delete failed:', e)
  }
}

function getKeyScope(key: APIKey): string {
  if (key.appId) {
    const app = applications.value.find(a => a.id === key.appId)
    return app?.name || 'App-specific'
  }
  const org = organizations.value.find(o => o.id === key.orgId)
  return org?.name || 'Org-wide'
}
</script>

<template>
  <div class="api-keys-page">
    <PageHeader 
      title="API Keys" 
      description="Manage API keys for tunnel authentication"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreateModal" :disabled="!filterOrgId">
          <Plus class="w-4 h-4" />
          New API Key
        </button>
      </template>
    </PageHeader>

    <!-- Toolbar -->
    <div class="toolbar">
      <select 
        v-model="filterOrgId" 
        class="form-input filter-select"
        @change="handleOrgChange"
      >
        <option value="" disabled>Select organization</option>
        <option v-for="org in organizations" :key="org.id" :value="org.id">
          {{ org.name }}
        </option>
      </select>
      <SearchInput 
        v-model="searchQuery" 
        placeholder="Search keys..." 
        :disabled="!filterOrgId"
      />
    </div>

    <!-- No org selected message -->
    <div v-if="!filterOrgId" class="select-org-message">
      <Key class="w-12 h-12" />
      <h3>Select an Organization</h3>
      <p>Choose an organization to view and manage its API keys.</p>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Table -->
    <DataTable
      v-else
      :columns="columns"
      :data="filteredKeys"
      :loading="loading"
      empty-title="No API keys"
      empty-description="Create an API key for this organization."
      row-key="id"
    >
      <template #cell-keyPrefix="{ value }">
        <code class="key-prefix">{{ value }}...</code>
      </template>
      
      <template #cell-scope="{ row }">
        <div class="key-scope">
          <AppWindow v-if="row.appId" class="w-4 h-4" />
          <Building2 v-else class="w-4 h-4" />
          <span>{{ getKeyScope(row) }}</span>
        </div>
      </template>
      
      <template #cell-createdAt="{ value }">
        {{ formatDate(value) }}
      </template>
      
      <template #cell-lastUsed="{ value }">
        {{ value ? formatDate(value) : 'Never' }}
      </template>
      
      <template #cell-expiresAt="{ value }">
        <span v-if="value" :class="{ 'text-warning': new Date(value) < new Date() }">
          {{ formatDate(value) }}
        </span>
        <span v-else class="text-muted">Never</span>
      </template>
      
      <template #actions="{ row }">
        <div class="action-buttons">
          <button 
            class="icon-btn icon-btn--danger" 
            title="Revoke" 
            @click.stop="openDeleteConfirm(row)"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </template>
      
      <template #emptyAction>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          Create API Key
        </button>
      </template>
    </DataTable>

    <!-- Create Modal -->
    <Modal v-model="showCreateModal" title="New API Key">
      <form @submit.prevent="handleCreate" class="form">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="form-group">
          <label class="form-label" for="key-org">Organization</label>
          <select
            id="key-org"
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
          <label class="form-label" for="key-app">
            Application
            <span class="form-label-optional">(optional)</span>
          </label>
          <select
            id="key-app"
            v-model="formAppId"
            class="form-input"
            :disabled="!formOrgId"
          >
            <option value="">All applications (org-wide)</option>
            <option 
              v-for="app in formApps" 
              :key="app.id" 
              :value="app.id"
            >
              {{ app.name }} ({{ app.subdomain }})
            </option>
          </select>
          <p class="form-hint">Leave empty for org-wide access, or select an app for app-specific key</p>
        </div>
        
        <div class="form-group">
          <label class="form-label" for="key-desc">Description</label>
          <input
            id="key-desc"
            v-model="formDescription"
            type="text"
            class="form-input"
            placeholder="e.g., CI/CD Pipeline"
          />
        </div>
        
        <div class="form-group">
          <label class="form-label" for="key-expires">
            Expires In
            <span class="form-label-optional">(optional)</span>
          </label>
          <select
            id="key-expires"
            v-model="formExpiresIn"
            class="form-input"
          >
            <option :value="undefined">Never</option>
            <option :value="7">7 days</option>
            <option :value="30">30 days</option>
            <option :value="90">90 days</option>
            <option :value="365">1 year</option>
          </select>
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showCreateModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button 
          class="btn btn-primary" 
          @click="handleCreate" 
          :disabled="formLoading || !formOrgId || !formDescription.trim()"
        >
          {{ formLoading ? 'Creating...' : 'Create Key' }}
        </button>
      </template>
    </Modal>

    <!-- Token Modal -->
    <Modal v-model="showTokenModal" title="API Key Created">
      <TokenReveal 
        :value="generatedRawKey"
        label="API Key"
        show-warning
        warning-text="This key will only be shown once. Copy and store it securely!"
      />
      
      <template #footer>
        <button class="btn btn-primary" @click="showTokenModal = false">
          Done
        </button>
      </template>
    </Modal>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="Revoke API Key"
      :message="`Are you sure you want to revoke key '${selectedKey?.keyPrefix}...'? This action cannot be undone.`"
      confirm-text="Revoke"
      variant="danger"
      @confirm="handleDelete"
    />
  </div>
</template>

<style scoped>
.api-keys-page {
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
  min-width: 200px;
}

.select-org-message {
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

.select-org-message h3 {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 1rem 0 0.5rem;
}

.select-org-message p {
  font-size: 0.9375rem;
  margin: 0;
  max-width: 280px;
}

.key-prefix {
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  color: var(--accent-amber);
  background: rgba(var(--accent-amber-rgb), 0.1);
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.key-scope {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--text-secondary);
}

.text-warning {
  color: var(--accent-amber);
}

.text-muted {
  color: var(--text-muted);
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
