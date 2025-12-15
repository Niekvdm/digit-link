<script setup lang="ts">
import { ref, computed } from 'vue'
import { useOrganizations } from '@/composables'
import AppLayout from '@/components/layout/AppLayout.vue'
import { BaseModal, LoadingState, EmptyState } from '@/components/shared'
import type { Organization, OrgAuthPolicy, AuthType, SetPolicyRequest } from '@/types/api'
import {
  Building2,
  Plus,
  RefreshCw,
  Pencil,
  Trash2,
  Shield,
  ShieldCheck,
  ShieldOff,
  Key,
  KeyRound,
  Globe,
  X,
  Check,
  ChevronDown,
  AlertTriangle
} from 'lucide-vue-next'

const {
  organizations,
  loading,
  refresh,
  createOrganization,
  updateOrganization,
  deleteOrganization,
  getPolicy,
  setPolicy
} = useOrganizations()

// Create/Edit modal
const showCreateModal = ref(false)
const editingOrg = ref<Organization | null>(null)
const orgName = ref('')
const createLoading = ref(false)

// Policy modal
const showPolicyModal = ref(false)
const policyOrg = ref<Organization | null>(null)
const policyLoading = ref(false)
const currentPolicy = ref<OrgAuthPolicy | null>(null)

// Policy form
const policyAuthType = ref<AuthType>('basic')
const basicUsername = ref('')
const basicPassword = ref('')
const oidcIssuerUrl = ref('')
const oidcClientId = ref('')
const oidcClientSecret = ref('')
const oidcScopes = ref('openid,email,profile')
const oidcAllowedDomains = ref('')

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingOrg = ref<Organization | null>(null)

const isEditing = computed(() => !!editingOrg.value)
const modalTitle = computed(() => isEditing.value ? 'Edit Organization' : 'Create Organization')

function formatDate(timestamp: string) {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleDateString()
}

function openCreateModal() {
  editingOrg.value = null
  orgName.value = ''
  showCreateModal.value = true
}

function openEditModal(org: Organization) {
  editingOrg.value = org
  orgName.value = org.name
  showCreateModal.value = true
}

async function handleSave() {
  if (!orgName.value.trim()) return

  createLoading.value = true

  if (isEditing.value && editingOrg.value) {
    const result = await updateOrganization(editingOrg.value.id, orgName.value.trim())
    if (!result.success) {
      alert(result.error || 'Failed to update organization')
    }
  } else {
    const result = await createOrganization({ name: orgName.value.trim() })
    if (!result.success) {
      alert(result.error || 'Failed to create organization')
    }
  }

  createLoading.value = false
  showCreateModal.value = false
}

function openDeleteConfirm(org: Organization) {
  deletingOrg.value = org
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deletingOrg.value) return

  const result = await deleteOrganization(deletingOrg.value.id)
  if (!result.success) {
    alert(result.error || 'Failed to delete organization')
  }

  showDeleteConfirm.value = false
  deletingOrg.value = null
}

async function openPolicyModal(org: Organization) {
  policyOrg.value = org
  policyLoading.value = true
  showPolicyModal.value = true

  // Reset form
  policyAuthType.value = 'basic'
  basicUsername.value = ''
  basicPassword.value = ''
  oidcIssuerUrl.value = ''
  oidcClientId.value = ''
  oidcClientSecret.value = ''
  oidcScopes.value = 'openid,email,profile'
  oidcAllowedDomains.value = ''

  // Load existing policy
  const policy = await getPolicy(org.id)
  currentPolicy.value = policy

  if (policy) {
    policyAuthType.value = policy.authType
    if (policy.authType === 'oidc') {
      oidcIssuerUrl.value = policy.oidcIssuerUrl || ''
      oidcClientId.value = policy.oidcClientId || ''
      oidcScopes.value = policy.oidcScopes?.join(',') || 'openid,email,profile'
      oidcAllowedDomains.value = policy.oidcAllowedDomains?.join(',') || ''
    }
  }

  policyLoading.value = false
}

async function handleSavePolicy() {
  if (!policyOrg.value) return

  policyLoading.value = true

  const policy: SetPolicyRequest = {
    authType: policyAuthType.value
  }

  if (policyAuthType.value === 'basic') {
    if (!basicUsername.value || !basicPassword.value) {
      alert('Username and password are required for Basic auth')
      policyLoading.value = false
      return
    }
    policy.basicUsername = basicUsername.value
    policy.basicPassword = basicPassword.value
  } else if (policyAuthType.value === 'oidc') {
    if (!oidcIssuerUrl.value || !oidcClientId.value) {
      alert('Issuer URL and Client ID are required for OIDC')
      policyLoading.value = false
      return
    }
    policy.oidcIssuerUrl = oidcIssuerUrl.value
    policy.oidcClientId = oidcClientId.value
    policy.oidcClientSecret = oidcClientSecret.value || undefined
    policy.oidcScopes = oidcScopes.value.split(',').map(s => s.trim()).filter(Boolean)
    policy.oidcAllowedDomains = oidcAllowedDomains.value
      ? oidcAllowedDomains.value.split(',').map(s => s.trim()).filter(Boolean)
      : undefined
  }

  const result = await setPolicy(policyOrg.value.id, policy)
  policyLoading.value = false

  if (!result.success) {
    alert(result.error || 'Failed to set policy')
    return
  }

  showPolicyModal.value = false
}

function getAuthIcon(hasPolicy: boolean) {
  return hasPolicy ? ShieldCheck : ShieldOff
}

function getAuthTypeIcon(authType: AuthType) {
  switch (authType) {
    case 'basic': return Key
    case 'api_key': return KeyRound
    case 'oidc': return Globe
    default: return Shield
  }
}
</script>

<template>
  <AppLayout>
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
      <h1 class="font-[var(--font-display)] text-3xl font-semibold">Organizations</h1>
      <button class="btn btn-primary" @click="openCreateModal">
        <Plus class="w-4 h-4" />
        Create Organization
      </button>
    </div>

    <!-- Organizations Card -->
    <div class="card">
      <div class="card-header">
        <h2 class="card-title">
          <Building2 class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          All Organizations
        </h2>
        <button
          class="btn btn-secondary btn-sm"
          :disabled="loading"
          @click="refresh"
        >
          <RefreshCw
            class="w-3.5 h-3.5"
            :class="{ 'animate-spin': loading }"
          />
          Refresh
        </button>
      </div>
      <div class="card-body">
        <LoadingState v-if="loading && !organizations.length" message="Loading organizations..." />

        <EmptyState
          v-else-if="!organizations.length"
          :icon="Building2"
          title="No organizations yet"
        />

        <div v-else class="flex flex-col gap-3">
          <div
            v-for="org in organizations"
            :key="org.id"
            class="flex items-center gap-4 p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg hover:border-[var(--border-accent)] transition-colors"
          >
            <!-- Icon -->
            <div class="w-10 h-10 rounded-lg bg-[rgba(201,149,108,0.15)] text-[var(--accent-copper)] flex items-center justify-center">
              <Building2 class="w-5 h-5" />
            </div>

            <!-- Info -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <span class="font-medium truncate">{{ org.name }}</span>
                <component
                  :is="getAuthIcon(org.hasPolicy || false)"
                  class="w-4 h-4"
                  :class="org.hasPolicy ? 'text-[var(--accent-emerald)]' : 'text-[var(--text-muted)]'"
                />
              </div>
              <div class="text-xs text-[var(--text-muted)]">
                {{ org.appCount || 0 }} application{{ (org.appCount || 0) !== 1 ? 's' : '' }}
                · Created {{ formatDate(org.createdAt) }}
              </div>
            </div>

            <!-- Actions -->
            <div class="flex gap-2 flex-shrink-0">
              <button
                class="btn btn-secondary btn-sm"
                @click="openPolicyModal(org)"
              >
                <Shield class="w-3.5 h-3.5" />
                Auth Policy
              </button>
              <button
                class="btn btn-secondary btn-sm"
                @click="openEditModal(org)"
              >
                <Pencil class="w-3.5 h-3.5" />
                Edit
              </button>
              <button
                class="btn btn-danger btn-sm"
                :disabled="(org.appCount || 0) > 0"
                :title="(org.appCount || 0) > 0 ? 'Delete applications first' : 'Delete organization'"
                @click="openDeleteConfirm(org)"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <BaseModal
      :show="showCreateModal"
      :title="modalTitle"
      @close="showCreateModal = false"
    >
      <form @submit.prevent="handleSave">
        <div class="mb-4">
          <label class="form-label" for="orgName">Organization Name</label>
          <input
            id="orgName"
            v-model="orgName"
            type="text"
            class="form-input"
            placeholder="Enter organization name"
            required
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
          :disabled="createLoading || !orgName.trim()"
          @click="handleSave"
        >
          <span class="btn-text flex items-center gap-2">
            <Check class="w-4 h-4" />
            {{ isEditing ? 'Save' : 'Create' }}
          </span>
        </button>
      </template>
    </BaseModal>

    <!-- Delete Confirmation Modal -->
    <BaseModal
      :show="showDeleteConfirm"
      title="Delete Organization"
      @close="showDeleteConfirm = false"
    >
      <div class="warning-box mb-4">
        <AlertTriangle class="w-5 h-5 text-[var(--accent-red)] flex-shrink-0" />
        <div class="text-sm text-[var(--text-secondary)]">
          Are you sure you want to delete <strong class="text-[var(--text-primary)]">{{ deletingOrg?.name }}</strong>?
          This action cannot be undone.
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

    <!-- Auth Policy Modal -->
    <BaseModal
      :show="showPolicyModal"
      :title="`Auth Policy: ${policyOrg?.name}`"
      maxWidth="560px"
      @close="showPolicyModal = false"
    >
      <div v-if="policyLoading" class="py-8">
        <LoadingState message="Loading policy..." />
      </div>

      <div v-else>
        <!-- Auth Type Selector -->
        <div class="mb-6">
          <label class="form-label">Authentication Type</label>
          <div class="grid grid-cols-3 gap-2">
            <button
              v-for="type in (['basic', 'api_key', 'oidc'] as AuthType[])"
              :key="type"
              type="button"
              class="p-3 rounded-lg border text-center transition-all"
              :class="policyAuthType === type
                ? 'border-[var(--accent-copper)] bg-[rgba(201,149,108,0.1)] text-[var(--accent-copper)]'
                : 'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:border-[var(--border-accent)]'"
              @click="policyAuthType = type"
            >
              <component :is="getAuthTypeIcon(type)" class="w-5 h-5 mx-auto mb-1" />
              <div class="text-xs font-medium capitalize">
                {{ type === 'api_key' ? 'API Key' : type === 'oidc' ? 'OIDC' : 'Basic' }}
              </div>
            </button>
          </div>
        </div>

        <!-- Basic Auth Fields -->
        <div v-if="policyAuthType === 'basic'" class="space-y-4">
          <div>
            <label class="form-label" for="basicUsername">Username</label>
            <input
              id="basicUsername"
              v-model="basicUsername"
              type="text"
              class="form-input"
              placeholder="Enter username"
            />
          </div>
          <div>
            <label class="form-label" for="basicPassword">Password</label>
            <input
              id="basicPassword"
              v-model="basicPassword"
              type="password"
              class="form-input"
              placeholder="Enter password"
            />
            <p class="form-hint">Password will be hashed using bcrypt</p>
          </div>
        </div>

        <!-- API Key Info -->
        <div v-else-if="policyAuthType === 'api_key'" class="info-box">
          <KeyRound class="w-5 h-5 flex-shrink-0" />
          <div>
            API key authentication will accept keys via <code class="text-xs bg-[var(--bg-elevated)] px-1 rounded">X-API-Key</code> header
            or <code class="text-xs bg-[var(--bg-elevated)] px-1 rounded">Bearer</code> token. Manage keys in the API Keys section.
          </div>
        </div>

        <!-- OIDC Fields -->
        <div v-else-if="policyAuthType === 'oidc'" class="space-y-4">
          <div>
            <label class="form-label" for="oidcIssuerUrl">Issuer URL</label>
            <input
              id="oidcIssuerUrl"
              v-model="oidcIssuerUrl"
              type="url"
              class="form-input form-input-mono"
              placeholder="https://accounts.google.com"
            />
          </div>
          <div>
            <label class="form-label" for="oidcClientId">Client ID</label>
            <input
              id="oidcClientId"
              v-model="oidcClientId"
              type="text"
              class="form-input form-input-mono"
              placeholder="your-client-id.apps.googleusercontent.com"
            />
          </div>
          <div>
            <label class="form-label" for="oidcClientSecret">Client Secret</label>
            <input
              id="oidcClientSecret"
              v-model="oidcClientSecret"
              type="password"
              class="form-input form-input-mono"
              placeholder="Enter client secret"
            />
          </div>
          <div>
            <label class="form-label" for="oidcScopes">Scopes</label>
            <input
              id="oidcScopes"
              v-model="oidcScopes"
              type="text"
              class="form-input"
              placeholder="openid,email,profile"
            />
            <p class="form-hint">Comma-separated list of scopes</p>
          </div>
          <div>
            <label class="form-label" for="oidcAllowedDomains">Allowed Email Domains (optional)</label>
            <input
              id="oidcAllowedDomains"
              v-model="oidcAllowedDomains"
              type="text"
              class="form-input"
              placeholder="example.com,company.org"
            />
            <p class="form-hint">Leave empty to allow all domains</p>
          </div>
        </div>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showPolicyModal = false">
          Cancel
        </button>
        <button
          class="btn btn-primary"
          :class="{ 'btn-loading': policyLoading }"
          :disabled="policyLoading"
          @click="handleSavePolicy"
        >
          <span class="btn-text flex items-center gap-2">
            <ShieldCheck class="w-4 h-4" />
            Save Policy
          </span>
        </button>
      </template>
    </BaseModal>
  </AppLayout>
</template>
