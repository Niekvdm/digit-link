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
import { useAPIKeys, useApplications } from '@/composables/api'
import { usePortalContext } from '@/composables/usePortalContext'
import { useFormatters } from '@/composables/useFormatters'
import type { APIKey, CreateAPIKeyRequest } from '@/types/api'
import { Plus, Trash2, Key, AppWindow, Building2 } from 'lucide-vue-next'

const { currentOrgId } = usePortalContext()
const { apiKeys, loading, error, fetchByOrg, create, remove } = useAPIKeys()
const { applications, fetchAll: fetchApps } = useApplications()
const { formatDate } = useFormatters()

// Search
const searchQuery = ref('')

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
  await Promise.all([
    currentOrgId.value ? fetchByOrg(currentOrgId.value) : Promise.resolve(),
    fetchApps()
  ])
})

// Create
function openCreateModal() {
  formAppId.value = ''
  formDescription.value = ''
  formExpiresIn.value = undefined
  formError.value = ''
  showCreateModal.value = true
}

async function handleCreate() {
  if (!formDescription.value.trim()) {
    formError.value = 'Description is required'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    const data: CreateAPIKeyRequest = {
      orgId: currentOrgId.value || '',
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
  return 'All Applications'
}
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      title="API Keys" 
      description="Manage API keys for tunnel authentication"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          New API Key
        </button>
      </template>
    </PageHeader>

    <!-- Search -->
    <div class="mb-6">
      <SearchInput v-model="searchQuery" placeholder="Search keys..." />
    </div>

    <!-- Error -->
    <div v-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="filteredKeys"
      :loading="loading"
      empty-title="No API keys"
      empty-description="Create an API key to authenticate your tunnel connections."
      row-key="id"
    >
      <template #cell-keyPrefix="{ value }">
        <code class="font-mono text-[0.8125rem] text-accent-amber bg-[rgba(var(--accent-amber-rgb),0.1)] py-1 px-2 rounded">{{ value }}...</code>
      </template>
      
      <template #cell-scope="{ row }">
        <div class="flex items-center gap-2 text-text-secondary">
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
        <span v-if="value" :class="new Date(value) < new Date() ? 'text-accent-amber' : ''">
          {{ formatDate(value) }}
        </span>
        <span v-else class="text-text-muted">Never</span>
      </template>
      
      <template #actions="{ row }">
        <div class="flex items-center gap-1">
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-[rgba(var(--accent-red-rgb),0.1)] hover:text-accent-red"
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
      <form @submit.prevent="handleCreate" class="flex flex-col gap-5">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="key-app">
            Application
            <span class="font-normal normal-case tracking-normal text-text-muted">(optional)</span>
          </label>
          <select
            id="key-app"
            v-model="formAppId"
            class="form-input"
          >
            <option value="">All applications (org-wide)</option>
            <option 
              v-for="app in applications" 
              :key="app.id" 
              :value="app.id"
            >
              {{ app.name }} ({{ app.subdomain }})
            </option>
          </select>
          <p class="form-hint">Leave empty for org-wide access, or select an app for app-specific key</p>
        </div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="key-desc">Description</label>
          <input
            id="key-desc"
            v-model="formDescription"
            type="text"
            class="form-input"
            placeholder="e.g., CI/CD Pipeline"
            autofocus
          />
        </div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="key-expires">
            Expires In
            <span class="font-normal normal-case tracking-normal text-text-muted">(optional)</span>
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
          :disabled="formLoading || !formDescription.trim()"
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
