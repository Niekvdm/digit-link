<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useAuthStore } from '@/stores/authStore'
import OrgLayout from '@/components/layout/OrgLayout.vue'
import { BaseModal, LoadingState, EmptyState } from '@/components/shared'
import type { APIKey } from '@/types/api'
import {
  KeyRound,
  Plus,
  RefreshCw,
  Trash2,
  AlertTriangle,
  Copy,
  Check,
  AppWindow
} from 'lucide-vue-next'

interface Application {
  id: string
  subdomain: string
  name: string
}

const authStore = useAuthStore()

const apiKeys = ref<APIKey[]>([])
const applications = ref<Application[]>([])
const loading = ref(false)

// Create modal
const showCreateModal = ref(false)
const createAppId = ref('')
const createDescription = ref('')
const createExpiresIn = ref<number | undefined>(undefined)
const createLoading = ref(false)

// New key display
const showNewKeyModal = ref(false)
const newRawKey = ref('')
const keyCopied = ref(false)

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingKey = ref<APIKey | null>(null)

async function loadAPIKeys() {
  loading.value = true
  try {
    const response = await fetch('/org/api-keys', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const data = await response.json()
      apiKeys.value = data.keys || []
    }
  } catch (err) {
    console.error('Failed to load API keys:', err)
  } finally {
    loading.value = false
  }
}

async function loadApplications() {
  try {
    const response = await fetch('/org/applications', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const data = await response.json()
      applications.value = data.applications || []
    }
  } catch (err) {
    console.error('Failed to load applications:', err)
  }
}

function formatDate(timestamp?: string) {
  if (!timestamp) return '-'
  return new Date(timestamp).toLocaleDateString()
}

function getAppName(appId?: string): string {
  if (!appId) return 'Organization-wide'
  const app = applications.value.find(a => a.id === appId)
  return app ? (app.name || app.subdomain) : appId
}

function getKeyType(key: APIKey): string {
  return key.appId ? 'App-specific' : 'Account'
}

function openCreateModal() {
  createAppId.value = ''
  createDescription.value = ''
  createExpiresIn.value = undefined
  showCreateModal.value = true
}

async function handleCreate() {
  createLoading.value = true
  try {
    const body: Record<string, unknown> = {
      description: createDescription.value.trim()
    }
    if (createAppId.value) {
      body.appId = createAppId.value
    }
    if (createExpiresIn.value && createExpiresIn.value > 0) {
      body.expiresIn = createExpiresIn.value
    }

    const response = await fetch('/org/api-keys', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(body)
    })

    const data = await response.json()
    if (!response.ok || !data.success) {
      alert(data.error || 'Failed to create API key')
      return
    }

    showCreateModal.value = false
    
    // Show the new key
    newRawKey.value = data.rawKey
    keyCopied.value = false
    showNewKeyModal.value = true
    
    await loadAPIKeys()
  } catch (err) {
    alert('Failed to create API key')
  } finally {
    createLoading.value = false
  }
}

async function copyKey() {
  try {
    await navigator.clipboard.writeText(newRawKey.value)
    keyCopied.value = true
    setTimeout(() => {
      keyCopied.value = false
    }, 2000)
  } catch {
    alert('Failed to copy to clipboard')
  }
}

function openDeleteConfirm(key: APIKey) {
  deletingKey.value = key
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deletingKey.value) return

  try {
    const response = await fetch(`/org/api-keys/${deletingKey.value.id}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })

    const data = await response.json()
    if (!response.ok || !data.success) {
      alert(data.error || 'Failed to delete API key')
      return
    }

    showDeleteConfirm.value = false
    deletingKey.value = null
    await loadAPIKeys()
  } catch (err) {
    alert('Failed to delete API key')
  }
}

const accountKeys = computed(() => apiKeys.value.filter(k => !k.appId))
const appKeys = computed(() => apiKeys.value.filter(k => k.appId))

onMounted(async () => {
  await Promise.all([loadAPIKeys(), loadApplications()])
})
</script>

<template>
  <OrgLayout>
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="font-[var(--font-display)] text-3xl font-semibold mb-1">API Keys</h1>
        <p class="text-sm text-[var(--text-secondary)]">Manage API keys for tunnel authentication</p>
      </div>
      <button class="btn btn-primary" @click="openCreateModal">
        <Plus class="w-4 h-4" />
        Create API Key
      </button>
    </div>

    <!-- Account Keys -->
    <div class="card mb-6">
      <div class="card-header">
        <h2 class="card-title">
          <KeyRound class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          Account API Keys
        </h2>
        <button
          class="btn btn-secondary btn-sm"
          :disabled="loading"
          @click="loadAPIKeys"
        >
          <RefreshCw
            class="w-3.5 h-3.5"
            :class="{ 'animate-spin': loading }"
          />
          Refresh
        </button>
      </div>
      <div class="card-body">
        <LoadingState v-if="loading && !apiKeys.length" message="Loading API keys..." />

        <EmptyState
          v-else-if="!accountKeys.length"
          :icon="KeyRound"
          title="No account API keys"
          description="Account keys allow random subdomain generation for any application"
        />

        <div v-else class="flex flex-col gap-2">
          <div
            v-for="key in accountKeys"
            :key="key.id"
            class="flex items-center justify-between p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg"
          >
            <div class="flex items-center gap-4">
              <div class="w-10 h-10 rounded-lg bg-[rgba(201,149,108,0.15)] text-[var(--accent-copper)] flex items-center justify-center">
                <KeyRound class="w-5 h-5" />
              </div>
              <div>
                <div class="flex items-center gap-2 mb-1">
                  <code class="text-sm font-mono text-[var(--accent-copper)]">{{ key.keyPrefix }}...</code>
                  <span class="px-2 py-0.5 rounded text-xs bg-[rgba(93,123,191,0.15)] text-[var(--accent-blue)]">Account</span>
                </div>
                <div class="text-xs text-[var(--text-muted)]">
                  {{ key.description || 'No description' }}
                </div>
              </div>
            </div>
            <div class="flex items-center gap-4">
              <div class="text-right text-xs text-[var(--text-muted)]">
                <div>Created {{ formatDate(key.createdAt) }}</div>
                <div v-if="key.expiresAt">Expires {{ formatDate(key.expiresAt) }}</div>
                <div v-if="key.lastUsed">Last used {{ formatDate(key.lastUsed) }}</div>
              </div>
              <button
                class="btn btn-danger btn-sm"
                @click="openDeleteConfirm(key)"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- App Keys -->
    <div class="card">
      <div class="card-header">
        <h2 class="card-title">
          <AppWindow class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          Application API Keys
        </h2>
      </div>
      <div class="card-body">
        <EmptyState
          v-if="!appKeys.length"
          :icon="AppWindow"
          title="No application API keys"
          description="App keys are restricted to specific application subdomains"
        />

        <div v-else class="flex flex-col gap-2">
          <div
            v-for="key in appKeys"
            :key="key.id"
            class="flex items-center justify-between p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg"
          >
            <div class="flex items-center gap-4">
              <div class="w-10 h-10 rounded-lg bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)] flex items-center justify-center">
                <KeyRound class="w-5 h-5" />
              </div>
              <div>
                <div class="flex items-center gap-2 mb-1">
                  <code class="text-sm font-mono text-[var(--accent-copper)]">{{ key.keyPrefix }}...</code>
                  <span class="px-2 py-0.5 rounded text-xs bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)]">App</span>
                </div>
                <div class="text-xs text-[var(--text-muted)]">
                  <span class="text-[var(--text-secondary)]">{{ getAppName(key.appId) }}</span>
                  <span v-if="key.description"> · {{ key.description }}</span>
                </div>
              </div>
            </div>
            <div class="flex items-center gap-4">
              <div class="text-right text-xs text-[var(--text-muted)]">
                <div>Created {{ formatDate(key.createdAt) }}</div>
                <div v-if="key.expiresAt">Expires {{ formatDate(key.expiresAt) }}</div>
                <div v-if="key.lastUsed">Last used {{ formatDate(key.lastUsed) }}</div>
              </div>
              <button
                class="btn btn-danger btn-sm"
                @click="openDeleteConfirm(key)"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Modal -->
    <BaseModal
      :show="showCreateModal"
      title="Create API Key"
      @close="showCreateModal = false"
    >
      <form @submit.prevent="handleCreate" class="space-y-4">
        <div>
          <label class="form-label" for="createAppId">Key Type</label>
          <select
            id="createAppId"
            v-model="createAppId"
            class="form-input"
          >
            <option value="">Account Key (any app)</option>
            <option v-for="app in applications" :key="app.id" :value="app.id">
              App Key: {{ app.name || app.subdomain }}
            </option>
          </select>
          <p class="form-hint">
            Account keys work with any app. App keys are restricted to a specific subdomain.
          </p>
        </div>

        <div>
          <label class="form-label" for="createDescription">Description</label>
          <input
            id="createDescription"
            v-model="createDescription"
            type="text"
            class="form-input"
            placeholder="Development machine"
          />
        </div>

        <div>
          <label class="form-label" for="createExpiresIn">Expires In (days)</label>
          <input
            id="createExpiresIn"
            v-model.number="createExpiresIn"
            type="number"
            min="1"
            class="form-input"
            placeholder="Never (leave empty)"
          />
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showCreateModal = false">
          Cancel
        </button>
        <button
          class="btn btn-primary"
          :class="{ 'btn-loading': createLoading }"
          :disabled="createLoading"
          @click="handleCreate"
        >
          <span class="btn-text flex items-center gap-2">
            <Plus class="w-4 h-4" />
            Create Key
          </span>
        </button>
      </template>
    </BaseModal>

    <!-- New Key Display Modal -->
    <BaseModal
      :show="showNewKeyModal"
      title="API Key Created"
      @close="showNewKeyModal = false"
    >
      <div class="warning-box mb-4">
        <AlertTriangle class="w-5 h-5 text-[var(--accent-copper)] flex-shrink-0" />
        <div class="text-sm text-[var(--text-secondary)]">
          Make sure to copy your API key now. You won't be able to see it again!
        </div>
      </div>

      <div class="p-4 bg-[var(--bg-deep)] rounded-lg border border-[var(--border-subtle)]">
        <div class="flex items-center justify-between gap-4">
          <code class="text-sm font-mono text-[var(--accent-copper)] break-all">{{ newRawKey }}</code>
          <button
            class="btn btn-secondary btn-sm flex-shrink-0"
            @click="copyKey"
          >
            <Check v-if="keyCopied" class="w-4 h-4 text-[var(--accent-emerald)]" />
            <Copy v-else class="w-4 h-4" />
          </button>
        </div>
      </div>

      <template #footer>
        <button class="btn btn-primary" @click="showNewKeyModal = false">
          Done
        </button>
      </template>
    </BaseModal>

    <!-- Delete Confirmation Modal -->
    <BaseModal
      :show="showDeleteConfirm"
      title="Delete API Key"
      @close="showDeleteConfirm = false"
    >
      <div class="warning-box mb-4">
        <AlertTriangle class="w-5 h-5 text-[var(--accent-red)] flex-shrink-0" />
        <div class="text-sm text-[var(--text-secondary)]">
          Are you sure you want to delete the API key 
          <strong class="text-[var(--accent-copper)] font-mono">{{ deletingKey?.keyPrefix }}...</strong>?
          Any clients using this key will lose access.
        </div>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showDeleteConfirm = false">
          Cancel
        </button>
        <button class="btn btn-danger" @click="handleDelete">
          <Trash2 class="w-4 h-4" />
          Delete
        </button>
      </template>
    </BaseModal>
  </OrgLayout>
</template>
