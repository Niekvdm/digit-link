<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useAPIKeys, useOrganizations, useApplications } from '@/composables'
import AppLayout from '@/components/layout/AppLayout.vue'
import { BaseModal, LoadingState, EmptyState } from '@/components/shared'
import type { APIKey } from '@/types/api'
import {
  KeyRound,
  Plus,
  RefreshCw,
  Trash2,
  Copy,
  Check,
  AlertTriangle,
  Building2,
  AppWindow,
  Clock,
  Shield
} from 'lucide-vue-next'

const { keys, loading, loadKeys, createKey, revokeKey } = useAPIKeys()
const { organizations, loading: orgsLoading } = useOrganizations()
const { applications, loading: appsLoading } = useApplications()

// Filters
const selectedOrgId = ref<string>('')
const selectedAppId = ref<string>('')

// Create modal
const showCreateModal = ref(false)
const newOrgId = ref('')
const newAppId = ref('')
const newDescription = ref('')
const newExpiresIn = ref<number | undefined>(undefined)
const createLoading = ref(false)

// New key display modal
const showNewKeyModal = ref(false)
const newRawKey = ref('')
const keyCopied = ref(false)

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingKey = ref<APIKey | null>(null)

const filteredApps = computed(() => {
  if (!selectedOrgId.value) return []
  return applications.value.filter(app => app.orgId === selectedOrgId.value)
})

const createFilteredApps = computed(() => {
  if (!newOrgId.value) return []
  return applications.value.filter(app => app.orgId === newOrgId.value)
})

function formatDate(timestamp?: string) {
  if (!timestamp) return 'Never'
  return new Date(timestamp).toLocaleDateString()
}

function formatDateTime(timestamp?: string) {
  if (!timestamp) return 'Never'
  return new Date(timestamp).toLocaleString()
}

function isExpired(key: APIKey): boolean {
  if (!key.expiresAt) return false
  return new Date(key.expiresAt) < new Date()
}

function isExpiringSoon(key: APIKey): boolean {
  if (!key.expiresAt) return false
  const expires = new Date(key.expiresAt)
  const now = new Date()
  const daysUntilExpiry = (expires.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  return daysUntilExpiry > 0 && daysUntilExpiry <= 7
}

function getKeyStatusClass(key: APIKey): string {
  if (isExpired(key)) return 'bg-[rgba(201,108,108,0.15)] text-[var(--accent-red)]'
  if (isExpiringSoon(key)) return 'bg-[rgba(212,168,75,0.15)] text-[var(--accent-amber)]'
  return 'bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)]'
}

function getKeyStatus(key: APIKey): string {
  if (isExpired(key)) return 'Expired'
  if (isExpiringSoon(key)) return 'Expiring Soon'
  if (key.expiresAt) return `Expires ${formatDate(key.expiresAt)}`
  return 'Never Expires'
}

async function handleLoadKeys() {
  await loadKeys(selectedOrgId.value || undefined, selectedAppId.value || undefined)
}

function openCreateModal() {
  newOrgId.value = selectedOrgId.value || organizations.value[0]?.id || ''
  newAppId.value = ''
  newDescription.value = ''
  newExpiresIn.value = undefined
  showCreateModal.value = true
}

async function handleCreate() {
  if (!newOrgId.value) return

  createLoading.value = true
  const result = await createKey({
    orgId: newOrgId.value,
    appId: newAppId.value || undefined,
    description: newDescription.value.trim(),
    expiresIn: newExpiresIn.value
  })
  createLoading.value = false

  if (!result.success) {
    alert(result.error || 'Failed to create API key')
    return
  }

  showCreateModal.value = false

  // Show the new key
  if (result.rawKey) {
    newRawKey.value = result.rawKey
    showNewKeyModal.value = true
  }

  // Refresh list
  await handleLoadKeys()
}

function openDeleteConfirm(key: APIKey) {
  deletingKey.value = key
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deletingKey.value) return

  const result = await revokeKey(
    deletingKey.value.id,
    selectedOrgId.value || undefined,
    selectedAppId.value || undefined
  )

  if (!result.success) {
    alert(result.error || 'Failed to revoke API key')
  }

  showDeleteConfirm.value = false
  deletingKey.value = null
}

function copyKey() {
  navigator.clipboard.writeText(newRawKey.value)
  keyCopied.value = true
  setTimeout(() => { keyCopied.value = false }, 2000)
}

function closeNewKeyModal() {
  showNewKeyModal.value = false
  newRawKey.value = ''
}

// Load keys when filters change
watch([selectedOrgId, selectedAppId], () => {
  if (selectedOrgId.value || selectedAppId.value) {
    handleLoadKeys()
  } else {
    // Clear keys if no filter
  }
}, { immediate: false })

// Clear app filter when org changes
watch(selectedOrgId, () => {
  selectedAppId.value = ''
})
</script>

<template>
  <AppLayout>
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
      <h1 class="font-[var(--font-display)] text-3xl font-semibold">API Keys</h1>
      <button
        class="btn btn-primary"
        @click="openCreateModal"
        :disabled="!organizations.length"
      >
        <Plus class="w-4 h-4" />
        Create API Key
      </button>
    </div>

    <!-- Filter Bar -->
    <div class="mb-6 flex items-center gap-4">
      <div class="flex items-center gap-2">
        <Building2 class="w-4 h-4 text-[var(--text-muted)]" />
        <select
          v-model="selectedOrgId"
          class="form-input py-2 pr-8 min-w-[200px]"
        >
          <option value="">Select Organization</option>
          <option v-for="org in organizations" :key="org.id" :value="org.id">
            {{ org.name }}
          </option>
        </select>
      </div>

      <div v-if="selectedOrgId" class="flex items-center gap-2">
        <AppWindow class="w-4 h-4 text-[var(--text-muted)]" />
        <select
          v-model="selectedAppId"
          class="form-input py-2 pr-8 min-w-[200px]"
        >
          <option value="">All Apps (Org-level keys)</option>
          <option v-for="app in filteredApps" :key="app.id" :value="app.id">
            {{ app.subdomain }}{{ app.name ? ` - ${app.name}` : '' }}
          </option>
        </select>
      </div>
    </div>

    <!-- Keys Card -->
    <div class="card">
      <div class="card-header">
        <h2 class="card-title">
          <KeyRound class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          API Keys
        </h2>
        <button
          class="btn btn-secondary btn-sm"
          :disabled="loading || !selectedOrgId"
          @click="handleLoadKeys"
        >
          <RefreshCw
            class="w-3.5 h-3.5"
            :class="{ 'animate-spin': loading }"
          />
          Refresh
        </button>
      </div>
      <div class="card-body">
        <div v-if="!selectedOrgId" class="py-8 text-center text-[var(--text-muted)]">
          <Shield class="w-12 h-12 mx-auto mb-3 opacity-30" />
          <p>Select an organization to view API keys</p>
        </div>

        <LoadingState v-else-if="loading && !keys.length" message="Loading API keys..." />

        <EmptyState
          v-else-if="!keys.length"
          :icon="KeyRound"
          title="No API keys found"
        />

        <div v-else class="flex flex-col gap-3">
          <div
            v-for="key in keys"
            :key="key.id"
            class="flex items-center gap-4 p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg hover:border-[var(--border-accent)] transition-colors"
            :class="{ 'opacity-60': isExpired(key) }"
          >
            <!-- Icon -->
            <div class="w-10 h-10 rounded-lg bg-[rgba(108,159,201,0.15)] text-[var(--accent-blue)] flex items-center justify-center">
              <KeyRound class="w-5 h-5" />
            </div>

            <!-- Info -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <code class="font-mono font-medium text-[var(--accent-amber)]">{{ key.keyPrefix }}...</code>
                <span
                  class="text-[0.625rem] font-semibold uppercase tracking-wide px-2 py-0.5 rounded"
                  :class="getKeyStatusClass(key)"
                >
                  {{ getKeyStatus(key) }}
                </span>
                <span
                  v-if="key.appId"
                  class="text-[0.625rem] font-semibold uppercase tracking-wide px-2 py-0.5 rounded bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)]"
                >
                  App-specific
                </span>
              </div>
              <div class="text-sm text-[var(--text-secondary)] mb-1">
                {{ key.description || 'No description' }}
              </div>
              <div class="flex items-center gap-3 text-xs text-[var(--text-muted)]">
                <span>Created {{ formatDate(key.createdAt) }}</span>
                <span v-if="key.lastUsed" class="flex items-center gap-1">
                  <Clock class="w-3 h-3" />
                  Last used {{ formatDateTime(key.lastUsed) }}
                </span>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex gap-2 flex-shrink-0">
              <button
                class="btn btn-danger btn-sm"
                @click="openDeleteConfirm(key)"
              >
                <Trash2 class="w-3.5 h-3.5" />
                Revoke
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
          <label class="form-label" for="newOrgId">Organization</label>
          <select
            id="newOrgId"
            v-model="newOrgId"
            class="form-input"
            required
          >
            <option v-for="org in organizations" :key="org.id" :value="org.id">
              {{ org.name }}
            </option>
          </select>
        </div>

        <div>
          <label class="form-label" for="newAppId">Application (optional)</label>
          <select
            id="newAppId"
            v-model="newAppId"
            class="form-input"
            :disabled="!newOrgId"
          >
            <option value="">Org-level key (all apps)</option>
            <option v-for="app in createFilteredApps" :key="app.id" :value="app.id">
              {{ app.subdomain }}{{ app.name ? ` - ${app.name}` : '' }}
            </option>
          </select>
          <p class="form-hint">Leave empty for an org-level key that works with all apps</p>
        </div>

        <div>
          <label class="form-label" for="newDescription">Description</label>
          <input
            id="newDescription"
            v-model="newDescription"
            type="text"
            class="form-input"
            placeholder="CI/CD Pipeline, Backend Service, etc."
          />
        </div>

        <div>
          <label class="form-label" for="newExpiresIn">Expiration (optional)</label>
          <select
            id="newExpiresIn"
            v-model="newExpiresIn"
            class="form-input"
          >
            <option :value="undefined">Never expires</option>
            <option :value="7">7 days</option>
            <option :value="30">30 days</option>
            <option :value="90">90 days</option>
            <option :value="180">180 days</option>
            <option :value="365">1 year</option>
          </select>
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showCreateModal = false">
          Cancel
        </button>
        <button
          class="btn btn-primary"
          :class="{ 'btn-loading': createLoading }"
          :disabled="createLoading || !newOrgId"
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
      @close="closeNewKeyModal"
    >
      <div class="warning-box mb-4">
        <AlertTriangle class="w-5 h-5 text-[var(--accent-amber)] flex-shrink-0" />
        <div class="text-sm text-[var(--text-secondary)]">
          Copy this key now. You won't be able to see it again!
        </div>
      </div>

      <div class="token-box">
        <div class="flex items-start gap-2">
          <code class="flex-1 token-value break-all">{{ newRawKey }}</code>
          <button
            class="p-1.5 text-[var(--text-secondary)] hover:text-[var(--accent-copper)] transition-colors flex-shrink-0"
            @click="copyKey"
          >
            <Check v-if="keyCopied" class="w-4 h-4 text-[var(--accent-emerald)]" />
            <Copy v-else class="w-4 h-4" />
          </button>
        </div>
      </div>

      <div class="mt-4 p-3 bg-[var(--bg-elevated)] rounded-lg text-sm">
        <div class="font-medium text-[var(--text-primary)] mb-1">Usage</div>
        <code class="text-xs text-[var(--text-muted)]">
          curl -H "X-API-Key: {{ newRawKey.slice(0, 8) }}..." https://myapp.tunnel.digit.zone
        </code>
      </div>

      <template #footer>
        <button class="btn btn-primary" @click="closeNewKeyModal">
          Done
        </button>
      </template>
    </BaseModal>

    <!-- Delete Confirmation Modal -->
    <BaseModal
      :show="showDeleteConfirm"
      title="Revoke API Key"
      @close="showDeleteConfirm = false"
    >
      <div class="warning-box mb-4">
        <AlertTriangle class="w-5 h-5 text-[var(--accent-red)] flex-shrink-0" />
        <div class="text-sm text-[var(--text-secondary)]">
          Are you sure you want to revoke the key starting with <code class="font-mono text-[var(--accent-amber)]">{{ deletingKey?.keyPrefix }}...</code>?
          Any applications using this key will lose access immediately.
        </div>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showDeleteConfirm = false">
          Cancel
        </button>
        <button class="btn btn-danger" @click="handleDelete">
          <Trash2 class="w-4 h-4" />
          Revoke Key
        </button>
      </template>
    </BaseModal>
  </AppLayout>
</template>
