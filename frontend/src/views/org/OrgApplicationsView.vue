<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import OrgLayout from '@/components/layout/OrgLayout.vue'
import { BaseModal, LoadingState, EmptyState } from '@/components/shared'
import type { Application, AuthMode, AuthType, TunnelStats, AppAuthPolicy, SetPolicyRequest } from '@/types/api'
import {
  AppWindow,
  Plus,
  RefreshCw,
  Pencil,
  Trash2,
  Shield,
  ShieldCheck,
  ShieldBan,
  Check,
  AlertTriangle,
  Circle,
  Activity,
  Key,
  Globe,
  Settings
} from 'lucide-vue-next'

interface OrgApplication extends Application {
  isActive?: boolean
  activeTunnelCount?: number
  stats?: TunnelStats
}

const router = useRouter()
const authStore = useAuthStore()

const applications = ref<OrgApplication[]>([])
const loading = ref(false)

// Create modal
const showCreateModal = ref(false)
const newSubdomain = ref('')
const newName = ref('')
const createLoading = ref(false)

// Edit modal
const showEditModal = ref(false)
const editingApp = ref<OrgApplication | null>(null)
const editName = ref('')
const editSubdomain = ref('')
const editAuthMode = ref<AuthMode>('inherit')
const editLoading = ref(false)

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingApp = ref<OrgApplication | null>(null)

// Policy configuration modal
const showPolicyModal = ref(false)
const policyApp = ref<OrgApplication | null>(null)
const currentPolicy = ref<AppAuthPolicy | null>(null)
const policyAuthType = ref<AuthType>('basic')
const basicUsername = ref('')
const basicPassword = ref('')
const oidcIssuerUrl = ref('')
const oidcClientId = ref('')
const oidcClientSecret = ref('')
const oidcScopes = ref('openid,email,profile')
const oidcAllowedDomains = ref('')
const policyLoading = ref(false)

async function loadApplications() {
  loading.value = true
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
  } finally {
    loading.value = false
  }
}

function formatDate(timestamp: string) {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleDateString()
}

function getAuthModeLabel(mode: AuthMode): string {
  switch (mode) {
    case 'inherit': return 'Inherit from Org'
    case 'disabled': return 'Disabled'
    case 'custom': return 'Custom'
    default: return mode
  }
}

function getAuthModeIcon(mode: AuthMode) {
  switch (mode) {
    case 'inherit': return Shield
    case 'disabled': return ShieldBan
    case 'custom': return ShieldCheck
    default: return Shield
  }
}

function getAuthModeColor(mode: AuthMode): string {
  switch (mode) {
    case 'inherit': return 'text-[var(--accent-blue)]'
    case 'disabled': return 'text-[var(--text-muted)]'
    case 'custom': return 'text-[var(--accent-emerald)]'
    default: return 'text-[var(--text-secondary)]'
  }
}

function navigateToDetail(appId: string) {
  router.push(`/portal/applications/${appId}`)
}

function openCreateModal() {
  newSubdomain.value = ''
  newName.value = ''
  showCreateModal.value = true
}

async function handleCreate() {
  if (!newSubdomain.value.trim()) return

  createLoading.value = true
  try {
    const response = await fetch('/org/applications', {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        subdomain: newSubdomain.value.trim().toLowerCase(),
        name: newName.value.trim()
      })
    })

    const data = await response.json()
    if (!response.ok || !data.success) {
      alert(data.error || 'Failed to create application')
      return
    }

    showCreateModal.value = false
    await loadApplications()
  } catch (err) {
    alert('Failed to create application')
  } finally {
    createLoading.value = false
  }
}

function openEditModal(app: OrgApplication) {
  editingApp.value = app
  editName.value = app.name
  editSubdomain.value = app.subdomain
  editAuthMode.value = app.authMode
  showEditModal.value = true
}

async function handleUpdate() {
  if (!editingApp.value) return

  editLoading.value = true
  try {
    const response = await fetch(`/org/applications/${editingApp.value.id}`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        name: editName.value.trim(),
        subdomain: editSubdomain.value.trim().toLowerCase(),
        authMode: editAuthMode.value
      })
    })

    const data = await response.json()
    if (!response.ok || !data.success) {
      alert(data.error || 'Failed to update application')
      return
    }

    showEditModal.value = false
    await loadApplications()
  } catch (err) {
    alert('Failed to update application')
  } finally {
    editLoading.value = false
  }
}

function openDeleteConfirm(app: OrgApplication) {
  deletingApp.value = app
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deletingApp.value) return

  try {
    const response = await fetch(`/org/applications/${deletingApp.value.id}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })

    const data = await response.json()
    if (!response.ok || !data.success) {
      alert(data.error || 'Failed to delete application')
      return
    }

    showDeleteConfirm.value = false
    deletingApp.value = null
    await loadApplications()
  } catch (err) {
    alert('Failed to delete application')
  }
}

// Policy configuration functions
function getAuthTypeIcon(authType: AuthType) {
  switch (authType) {
    case 'basic': return Key
    case 'api_key': return Key
    case 'oidc': return Globe
    default: return Key
  }
}

async function openPolicyModal(app: OrgApplication) {
  policyApp.value = app
  policyAuthType.value = 'basic'
  basicUsername.value = ''
  basicPassword.value = ''
  oidcIssuerUrl.value = ''
  oidcClientId.value = ''
  oidcClientSecret.value = ''
  oidcScopes.value = 'openid,email,profile'
  oidcAllowedDomains.value = ''
  currentPolicy.value = null

  // Load existing policy if any
  try {
    const response = await fetch(`/org/applications/${app.id}/policy`, {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const data = await response.json()
      if (data.policy) {
        currentPolicy.value = data.policy
        policyAuthType.value = data.policy.authType
        if (data.policy.authType === 'oidc') {
          oidcIssuerUrl.value = data.policy.oidcIssuerUrl || ''
          oidcClientId.value = data.policy.oidcClientId || ''
          oidcScopes.value = data.policy.oidcScopes?.join(',') || 'openid,email,profile'
          oidcAllowedDomains.value = data.policy.oidcAllowedDomains?.join(',') || ''
        }
      }
    }
  } catch (err) {
    console.error('Failed to load policy:', err)
  }

  showPolicyModal.value = true
}

async function handleSavePolicy() {
  if (!policyApp.value) return

  const policy: SetPolicyRequest = {
    authType: policyAuthType.value
  }

  if (policyAuthType.value === 'basic') {
    if (!basicUsername.value || !basicPassword.value) {
      alert('Username and password are required for basic auth')
      return
    }
    policy.basicUsername = basicUsername.value
    policy.basicPassword = basicPassword.value
  } else if (policyAuthType.value === 'oidc') {
    if (!oidcIssuerUrl.value || !oidcClientId.value) {
      alert('Issuer URL and Client ID are required for OIDC')
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

  policyLoading.value = true
  try {
    const response = await fetch(`/org/applications/${policyApp.value.id}/policy`, {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(policy)
    })

    const data = await response.json()
    if (!response.ok || !data.success) {
      alert(data.error || 'Failed to save policy')
      return
    }

    showPolicyModal.value = false
    await loadApplications()
  } catch (err) {
    alert('Failed to save policy')
  } finally {
    policyLoading.value = false
  }
}

onMounted(() => {
  loadApplications()
})
</script>

<template>
  <OrgLayout>
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
      <h1 class="font-[var(--font-display)] text-3xl font-semibold">Applications</h1>
      <button class="btn btn-primary" @click="openCreateModal">
        <Plus class="w-4 h-4" />
        Create Application
      </button>
    </div>

    <!-- Applications Card -->
    <div class="card">
      <div class="card-header">
        <h2 class="card-title">
          <AppWindow class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          Your Applications
        </h2>
        <button
          class="btn btn-secondary btn-sm"
          :disabled="loading"
          @click="loadApplications"
        >
          <RefreshCw
            class="w-3.5 h-3.5"
            :class="{ 'animate-spin': loading }"
          />
          Refresh
        </button>
      </div>
      <div class="card-body">
        <LoadingState v-if="loading && !applications.length" message="Loading applications..." />

        <EmptyState
          v-else-if="!applications.length"
          :icon="AppWindow"
          title="No applications yet"
          description="Create your first application to start using tunnels"
        />

        <div v-else class="flex flex-col gap-3">
          <div
            v-for="app in applications"
            :key="app.id"
            class="flex items-center gap-4 p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg hover:border-[var(--border-accent)] transition-colors cursor-pointer"
            @click="navigateToDetail(app.id)"
          >
            <!-- Icon with Active Status -->
            <div class="relative">
              <div class="w-10 h-10 rounded-lg bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)] flex items-center justify-center">
                <AppWindow class="w-5 h-5" />
              </div>
              <!-- Active Indicator -->
              <div 
                v-if="app.isActive"
                class="absolute -top-1 -right-1 w-3.5 h-3.5 bg-[var(--accent-emerald)] rounded-full border-2 border-[var(--bg-deep)] flex items-center justify-center"
              >
                <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-[var(--accent-emerald)] opacity-75"></span>
              </div>
            </div>

            <!-- Info -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <span class="font-mono font-medium text-[var(--accent-copper)]">{{ app.subdomain }}</span>
                <span v-if="app.name" class="text-[var(--text-secondary)]">· {{ app.name }}</span>
                <span 
                  v-if="app.isActive"
                  class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)]"
                >
                  <Circle class="w-2 h-2 fill-current" />
                  {{ app.activeTunnelCount }} Active
                </span>
              </div>
              <div class="flex items-center gap-3 text-xs text-[var(--text-muted)]">
                <span class="flex items-center gap-1" :class="getAuthModeColor(app.authMode)">
                  <component :is="getAuthModeIcon(app.authMode)" class="w-3 h-3" />
                  {{ getAuthModeLabel(app.authMode) }}
                </span>
                <span v-if="app.stats" class="flex items-center gap-1">
                  <Activity class="w-3 h-3" />
                  {{ app.stats.totalConnections }} connections
                </span>
                <span>Created {{ formatDate(app.createdAt) }}</span>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex gap-2 flex-shrink-0" @click.stop>
              <button
                v-if="app.authMode === 'custom'"
                class="btn btn-secondary btn-sm"
                @click="openPolicyModal(app)"
              >
                <Settings class="w-3.5 h-3.5" />
                Policy
              </button>
              <button
                class="btn btn-secondary btn-sm"
                @click="openEditModal(app)"
              >
                <Pencil class="w-3.5 h-3.5" />
                Edit
              </button>
              <button
                class="btn btn-danger btn-sm"
                @click="openDeleteConfirm(app)"
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
      title="Create Application"
      @close="showCreateModal = false"
    >
      <form @submit.prevent="handleCreate" class="space-y-4">
        <div>
          <label class="form-label" for="newSubdomain">Subdomain</label>
          <div class="flex items-center">
            <input
              id="newSubdomain"
              v-model="newSubdomain"
              type="text"
              class="form-input form-input-mono rounded-r-none"
              placeholder="myapp"
              pattern="[a-z0-9-]+"
              required
            />
            <span class="px-3 py-[0.875rem] bg-[var(--bg-elevated)] border border-l-0 border-[var(--border-subtle)] rounded-r-lg text-sm text-[var(--text-muted)]">
              .tunnel.digit.zone
            </span>
          </div>
          <p class="form-hint">Lowercase letters, numbers, and hyphens only</p>
        </div>

        <div>
          <label class="form-label" for="newName">Display Name (optional)</label>
          <input
            id="newName"
            v-model="newName"
            type="text"
            class="form-input"
            placeholder="My Application"
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
          :disabled="createLoading || !newSubdomain.trim()"
          @click="handleCreate"
        >
          <span class="btn-text flex items-center gap-2">
            <Plus class="w-4 h-4" />
            Create
          </span>
        </button>
      </template>
    </BaseModal>

    <!-- Edit Modal -->
    <BaseModal
      :show="showEditModal"
      title="Edit Application"
      @close="showEditModal = false"
    >
      <form @submit.prevent="handleUpdate" class="space-y-4">
        <div>
          <label class="form-label" for="editSubdomain">Subdomain</label>
          <div class="flex items-center">
            <input
              id="editSubdomain"
              v-model="editSubdomain"
              type="text"
              class="form-input form-input-mono rounded-r-none"
              placeholder="myapp"
              pattern="[a-z0-9-]+"
            />
            <span class="px-3 py-[0.875rem] bg-[var(--bg-elevated)] border border-l-0 border-[var(--border-subtle)] rounded-r-lg text-sm text-[var(--text-muted)]">
              .tunnel.digit.zone
            </span>
          </div>
        </div>

        <div>
          <label class="form-label" for="editName">Display Name</label>
          <input
            id="editName"
            v-model="editName"
            type="text"
            class="form-input"
            placeholder="My Application"
          />
        </div>

        <div>
          <label class="form-label">Authentication Mode</label>
          <div class="grid grid-cols-3 gap-2">
            <button
              v-for="mode in (['inherit', 'disabled', 'custom'] as AuthMode[])"
              :key="mode"
              type="button"
              class="p-3 rounded-lg border text-center transition-all"
              :class="editAuthMode === mode
                ? 'border-[var(--accent-emerald)] bg-[rgba(74,159,126,0.1)] text-[var(--accent-emerald)]'
                : 'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:border-[var(--border-accent)]'"
              @click="editAuthMode = mode"
            >
              <component :is="getAuthModeIcon(mode)" class="w-5 h-5 mx-auto mb-1" />
              <div class="text-xs font-medium">{{ getAuthModeLabel(mode) }}</div>
            </button>
          </div>
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showEditModal = false">
          Cancel
        </button>
        <button
          class="btn btn-primary"
          :class="{ 'btn-loading': editLoading }"
          :disabled="editLoading"
          @click="handleUpdate"
        >
          <span class="btn-text flex items-center gap-2">
            <Check class="w-4 h-4" />
            Save
          </span>
        </button>
      </template>
    </BaseModal>

    <!-- Delete Confirmation Modal -->
    <BaseModal
      :show="showDeleteConfirm"
      title="Delete Application"
      @close="showDeleteConfirm = false"
    >
      <div class="warning-box mb-4">
        <AlertTriangle class="w-5 h-5 text-[var(--accent-red)] flex-shrink-0" />
        <div class="text-sm text-[var(--text-secondary)]">
          Are you sure you want to delete <strong class="text-[var(--accent-copper)] font-mono">{{ deletingApp?.subdomain }}</strong>?
          This will remove the subdomain reservation and any associated policies.
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

    <!-- Policy Configuration Modal -->
    <BaseModal
      :show="showPolicyModal"
      title="Configure Auth Policy"
      @close="showPolicyModal = false"
    >
      <div class="space-y-4">
        <div class="text-sm text-[var(--text-secondary)] mb-4">
          Configure authentication for <strong class="text-[var(--accent-copper)] font-mono">{{ policyApp?.subdomain }}</strong>
        </div>

        <!-- Auth Type Selection -->
        <div>
          <label class="form-label">Authentication Type</label>
          <div class="grid grid-cols-3 gap-2">
            <button
              v-for="type in (['basic', 'api_key', 'oidc'] as AuthType[])"
              :key="type"
              type="button"
              class="p-3 rounded-lg border text-center transition-all"
              :class="policyAuthType === type
                ? 'border-[var(--accent-emerald)] bg-[rgba(74,159,126,0.1)] text-[var(--accent-emerald)]'
                : 'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:border-[var(--border-accent)]'"
              @click="policyAuthType = type"
            >
              <component :is="getAuthTypeIcon(type)" class="w-5 h-5 mx-auto mb-1" />
              <div class="text-xs font-medium">{{ type === 'api_key' ? 'API Key' : type === 'oidc' ? 'OIDC' : 'Basic' }}</div>
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
              placeholder="Enter username (min 8 chars)"
            />
          </div>
          <div>
            <label class="form-label" for="basicPassword">Password</label>
            <input
              id="basicPassword"
              v-model="basicPassword"
              type="password"
              class="form-input"
              placeholder="Enter password (min 8 chars)"
            />
          </div>
        </div>

        <!-- API Key Info -->
        <div v-else-if="policyAuthType === 'api_key'" class="p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg">
          <p class="text-sm text-[var(--text-secondary)]">
            API Key authentication uses the X-API-Key header. Create API keys in the API Keys section.
          </p>
        </div>

        <!-- OIDC Fields -->
        <div v-else-if="policyAuthType === 'oidc'" class="space-y-4">
          <div>
            <label class="form-label" for="oidcIssuerUrl">Issuer URL</label>
            <input
              id="oidcIssuerUrl"
              v-model="oidcIssuerUrl"
              type="url"
              class="form-input"
              placeholder="https://accounts.google.com"
            />
          </div>
          <div>
            <label class="form-label" for="oidcClientId">Client ID</label>
            <input
              id="oidcClientId"
              v-model="oidcClientId"
              type="text"
              class="form-input"
              placeholder="your-client-id"
            />
          </div>
          <div>
            <label class="form-label" for="oidcClientSecret">Client Secret</label>
            <input
              id="oidcClientSecret"
              v-model="oidcClientSecret"
              type="password"
              class="form-input"
              placeholder="your-client-secret"
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
            <p class="form-hint">Comma-separated list of allowed email domains</p>
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
            <Check class="w-4 h-4" />
            Save Policy
          </span>
        </button>
      </template>
    </BaseModal>
  </OrgLayout>
</template>
