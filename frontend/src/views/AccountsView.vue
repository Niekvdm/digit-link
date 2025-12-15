<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAccounts, useOrganizations } from '@/composables'
import AppLayout from '@/components/layout/AppLayout.vue'
import { BaseModal, LoadingState, EmptyState } from '@/components/shared'
import { 
  Users, 
  UserPlus, 
  RefreshCw, 
  Key, 
  UserX,
  UserCheck,
  Copy,
  Check,
  AlertTriangle,
  Building2,
  Link,
  Unlink,
  Lock
} from 'lucide-vue-next'

const { 
  accounts, 
  loading, 
  refresh,
  createAccount,
  regenerateToken,
  setAccountOrganization,
  setAccountPassword,
  deactivateAccount,
  activateAccount
} = useAccounts()

const { organizations, loading: orgsLoading } = useOrganizations()

// Create modal
const showCreateModal = ref(false)
const newUsername = ref('')
const newPassword = ref('')
const newIsAdmin = ref(false)
const newOrgId = ref('')
const createLoading = ref(false)

// Token modal
const showTokenModal = ref(false)
const tokenUsername = ref('')
const currentToken = ref('')
const tokenCopied = ref(false)

// Link org modal
const showLinkOrgModal = ref(false)
const linkAccountId = ref('')
const linkAccountUsername = ref('')
const linkOrgId = ref('')
const linkLoading = ref(false)

// Password modal
const showPasswordModal = ref(false)
const passwordAccountId = ref('')
const passwordAccountUsername = ref('')
const newAccountPassword = ref('')
const passwordLoading = ref(false)

// Computed: sort organizations for dropdown
const sortedOrganizations = computed(() => 
  [...organizations.value].sort((a, b) => a.name.localeCompare(b.name))
)

function formatDate(timestamp?: string) {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleDateString()
}

function getInitial(username: string) {
  return username.charAt(0).toUpperCase()
}

function openCreateModal() {
  newUsername.value = ''
  newPassword.value = ''
  newIsAdmin.value = false
  newOrgId.value = ''
  showCreateModal.value = true
}

async function handleCreate() {
  if (!newUsername.value.trim()) return
  
  // Validate password if provided
  if (newPassword.value && newPassword.value.length < 8) {
    alert('Password must be at least 8 characters')
    return
  }
  
  createLoading.value = true
  const result = await createAccount({
    username: newUsername.value.trim(),
    password: newPassword.value || undefined,
    isAdmin: newIsAdmin.value,
    orgId: newOrgId.value || undefined
  })
  createLoading.value = false

  if (result.success && result.token) {
    showCreateModal.value = false
    showTokenModal.value = true
    tokenUsername.value = newUsername.value
    currentToken.value = result.token
  } else {
    alert(result.error || 'Failed to create account')
  }
}

async function handleRegenerate(accountId: string, username: string) {
  if (!confirm(`Regenerate token for ${username}? The old token will stop working.`)) return
  
  const result = await regenerateToken(accountId)
  
  if (result.success && result.token) {
    showTokenModal.value = true
    tokenUsername.value = username
    currentToken.value = result.token
  } else {
    alert(result.error || 'Failed to regenerate token')
  }
}

function openLinkOrgModal(accountId: string, username: string, currentOrgId?: string) {
  linkAccountId.value = accountId
  linkAccountUsername.value = username
  linkOrgId.value = currentOrgId || ''
  showLinkOrgModal.value = true
}

async function handleLinkOrg() {
  linkLoading.value = true
  const result = await setAccountOrganization(linkAccountId.value, linkOrgId.value)
  linkLoading.value = false

  if (result.success) {
    showLinkOrgModal.value = false
  } else {
    alert(result.error || 'Failed to update organization')
  }
}

function openPasswordModal(accountId: string, username: string) {
  passwordAccountId.value = accountId
  passwordAccountUsername.value = username
  newAccountPassword.value = ''
  showPasswordModal.value = true
}

async function handleSetPassword() {
  if (!newAccountPassword.value || newAccountPassword.value.length < 8) {
    alert('Password must be at least 8 characters')
    return
  }
  
  passwordLoading.value = true
  const result = await setAccountPassword(passwordAccountId.value, newAccountPassword.value)
  passwordLoading.value = false

  if (result.success) {
    showPasswordModal.value = false
  } else {
    alert(result.error || 'Failed to set password')
  }
}

async function handleDeactivate(accountId: string) {
  if (!confirm('Deactivate this account? They will no longer be able to create tunnels.')) return
  
  const result = await deactivateAccount(accountId)
  
  if (!result.success) {
    alert(result.error || 'Failed to deactivate account')
  }
}

async function handleActivate(accountId: string) {
  if (!confirm('Reactivate this account?')) return
  
  const result = await activateAccount(accountId)
  
  if (!result.success) {
    alert(result.error || 'Failed to activate account')
  }
}

function copyToken() {
  navigator.clipboard.writeText(currentToken.value)
  tokenCopied.value = true
  setTimeout(() => { tokenCopied.value = false }, 2000)
}

function closeTokenModal() {
  showTokenModal.value = false
  currentToken.value = ''
  tokenUsername.value = ''
}
</script>

<template>
  <AppLayout>
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
      <h1 class="font-[var(--font-display)] text-3xl font-semibold">Accounts</h1>
      <button class="btn btn-primary" @click="openCreateModal">
        <UserPlus class="w-4 h-4" />
        Create Account
      </button>
    </div>

    <!-- Accounts Card -->
    <div class="card">
      <div class="card-header">
        <h2 class="card-title">
          <Users class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          All Accounts
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
        <LoadingState v-if="loading && !accounts.length" message="Loading accounts..." />
        
        <EmptyState 
          v-else-if="!accounts.length"
          :icon="Users"
          title="No accounts yet"
        />

        <div v-else class="flex flex-col gap-3">
          <div
            v-for="account in accounts"
            :key="account.id"
            class="flex items-center gap-4 p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg hover:border-[var(--border-accent)] transition-colors"
          >
            <!-- Avatar -->
            <div class="w-10 h-10 rounded-lg bg-[rgba(201,149,108,0.15)] text-[var(--accent-copper)] flex items-center justify-center font-semibold">
              {{ getInitial(account.username) }}
            </div>

            <!-- Info -->
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <span class="font-medium truncate">{{ account.username }}</span>
                <span 
                  v-if="account.isAdmin"
                  class="text-[0.625rem] font-semibold uppercase tracking-wide px-2 py-0.5 rounded bg-[rgba(201,149,108,0.15)] text-[var(--accent-copper)]"
                >
                  Admin
                </span>
                <span 
                  v-if="!account.active"
                  class="text-[0.625rem] font-semibold uppercase tracking-wide px-2 py-0.5 rounded bg-[rgba(201,108,108,0.15)] text-[var(--accent-red)]"
                >
                  Inactive
                </span>
                <span 
                  v-if="account.hasPassword"
                  class="text-[0.625rem] font-semibold uppercase tracking-wide px-2 py-0.5 rounded bg-[rgba(108,201,150,0.15)] text-[var(--accent-emerald)]"
                  title="Has password set for org portal login"
                >
                  <Lock class="w-2.5 h-2.5 inline -mt-0.5" /> PWD
                </span>
              </div>
              <div class="flex items-center gap-2 text-xs text-[var(--text-muted)]">
                <span>Created {{ formatDate(account.createdAt) }}</span>
                <template v-if="account.lastUsed">
                  <span>·</span>
                  <span>Last used {{ formatDate(account.lastUsed) }}</span>
                </template>
                <template v-if="account.orgName">
                  <span>·</span>
                  <span class="flex items-center gap-1 text-[var(--accent-emerald)]">
                    <Building2 class="w-3 h-3" />
                    {{ account.orgName }}
                  </span>
                </template>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex gap-2 flex-shrink-0">
              <button 
                v-if="account.orgId"
                class="btn btn-secondary btn-sm"
                title="Change or unlink organization"
                @click="openLinkOrgModal(account.id, account.username, account.orgId)"
              >
                <Building2 class="w-3.5 h-3.5" />
                Org
              </button>
              <button 
                v-else
                class="btn btn-secondary btn-sm"
                title="Link to organization"
                @click="openLinkOrgModal(account.id, account.username)"
              >
                <Link class="w-3.5 h-3.5" />
                Link Org
              </button>
              <button 
                class="btn btn-secondary btn-sm"
                :title="account.hasPassword ? 'Change password' : 'Set password for org portal login'"
                @click="openPasswordModal(account.id, account.username)"
              >
                <Lock class="w-3.5 h-3.5" />
                {{ account.hasPassword ? 'Password' : 'Set Pwd' }}
              </button>
              <button 
                class="btn btn-secondary btn-sm"
                @click="handleRegenerate(account.id, account.username)"
              >
                <Key class="w-3.5 h-3.5" />
                Regenerate
              </button>
              <button 
                v-if="account.active"
                class="btn btn-danger btn-sm"
                @click="handleDeactivate(account.id)"
              >
                <UserX class="w-3.5 h-3.5" />
                Deactivate
              </button>
              <button 
                v-else
                class="btn btn-primary btn-sm"
                @click="handleActivate(account.id)"
              >
                <UserCheck class="w-3.5 h-3.5" />
                Activate
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Account Modal -->
    <BaseModal
      :show="showCreateModal"
      title="Create Account"
      @close="showCreateModal = false"
    >
      <form @submit.prevent="handleCreate">
        <div class="mb-4">
          <label class="form-label" for="username">Username</label>
          <input
            id="username"
            v-model="newUsername"
            type="text"
            class="form-input"
            placeholder="Enter username"
            required
          />
        </div>

        <div class="mb-4">
          <label class="form-label" for="org">Organization (optional)</label>
          <select
            id="org"
            v-model="newOrgId"
            class="form-input"
            :disabled="orgsLoading"
          >
            <option value="">No organization</option>
            <option 
              v-for="org in sortedOrganizations" 
              :key="org.id" 
              :value="org.id"
            >
              {{ org.name }}
            </option>
          </select>
          <p class="text-xs text-[var(--text-muted)] mt-1">
            Link this account to an organization to restrict their tunnel access.
          </p>
        </div>

        <div class="mb-4">
          <label class="form-label" for="password">
            Password {{ newOrgId ? '' : '(optional)' }}
          </label>
          <input
            id="password"
            v-model="newPassword"
            type="password"
            class="form-input"
            placeholder="Min 8 characters"
            :required="!!newOrgId"
            minlength="8"
          />
          <p class="text-xs text-[var(--text-muted)] mt-1">
            <template v-if="newOrgId">
              Required for org portal login.
            </template>
            <template v-else>
              Set a password if this user needs to log into the org portal.
            </template>
          </p>
        </div>

        <label class="flex items-center gap-2 text-sm cursor-pointer">
          <input 
            v-model="newIsAdmin" 
            type="checkbox" 
            class="w-4 h-4 accent-[var(--accent-copper)]" 
          />
          Make this an admin account
        </label>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showCreateModal = false">
          Cancel
        </button>
        <button 
          class="btn btn-primary"
          :class="{ 'btn-loading': createLoading }"
          :disabled="createLoading || !newUsername.trim()"
          @click="handleCreate"
        >
          <span class="btn-text flex items-center gap-2">
            <UserPlus class="w-4 h-4" />
            Create
          </span>
        </button>
      </template>
    </BaseModal>

    <!-- Link Organization Modal -->
    <BaseModal
      :show="showLinkOrgModal"
      title="Link to Organization"
      @close="showLinkOrgModal = false"
    >
      <p class="text-sm text-[var(--text-secondary)] mb-4">
        Link <strong class="text-[var(--text-primary)]">{{ linkAccountUsername }}</strong> to an organization.
      </p>

      <div class="mb-4">
        <label class="form-label" for="link-org">Organization</label>
        <select
          id="link-org"
          v-model="linkOrgId"
          class="form-input"
          :disabled="orgsLoading"
        >
          <option value="">No organization (unlink)</option>
          <option 
            v-for="org in sortedOrganizations" 
            :key="org.id" 
            :value="org.id"
          >
            {{ org.name }}
          </option>
        </select>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showLinkOrgModal = false">
          Cancel
        </button>
        <button 
          class="btn btn-primary"
          :class="{ 'btn-loading': linkLoading }"
          :disabled="linkLoading"
          @click="handleLinkOrg"
        >
          <span class="btn-text flex items-center gap-2">
            <template v-if="linkOrgId">
              <Link class="w-4 h-4" />
              Link
            </template>
            <template v-else>
              <Unlink class="w-4 h-4" />
              Unlink
            </template>
          </span>
        </button>
      </template>
    </BaseModal>

    <!-- Set Password Modal -->
    <BaseModal
      :show="showPasswordModal"
      title="Set Password"
      @close="showPasswordModal = false"
    >
      <p class="text-sm text-[var(--text-secondary)] mb-4">
        Set a password for <strong class="text-[var(--text-primary)]">{{ passwordAccountUsername }}</strong> to enable org portal login.
      </p>

      <div class="mb-4">
        <label class="form-label" for="new-password">New Password</label>
        <input
          id="new-password"
          v-model="newAccountPassword"
          type="password"
          class="form-input"
          placeholder="Min 8 characters"
          minlength="8"
          required
        />
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showPasswordModal = false">
          Cancel
        </button>
        <button 
          class="btn btn-primary"
          :class="{ 'btn-loading': passwordLoading }"
          :disabled="passwordLoading || newAccountPassword.length < 8"
          @click="handleSetPassword"
        >
          <span class="btn-text flex items-center gap-2">
            <Lock class="w-4 h-4" />
            Set Password
          </span>
        </button>
      </template>
    </BaseModal>

    <!-- Token Display Modal -->
    <BaseModal
      :show="showTokenModal"
      title="New Token Generated"
      @close="closeTokenModal"
    >
      <p class="text-sm text-[var(--text-secondary)] mb-4">
        A new token has been generated for <strong class="text-[var(--text-primary)]">{{ tokenUsername }}</strong>.
      </p>

      <div class="bg-[var(--bg-deep)] border border-[var(--accent-emerald)] rounded-lg p-4">
        <div class="text-xs font-medium uppercase tracking-wider text-[var(--accent-emerald)] mb-2">
          Access Token
        </div>
        <div class="flex items-center gap-2">
          <code class="flex-1 font-mono text-[0.8125rem] break-all text-[var(--text-primary)]">
            {{ currentToken }}
          </code>
          <button 
            class="p-1 text-[var(--text-secondary)] hover:text-[var(--accent-copper)] transition-colors flex-shrink-0"
            @click="copyToken"
          >
            <Check v-if="tokenCopied" class="w-4 h-4 text-[var(--accent-emerald)]" />
            <Copy v-else class="w-4 h-4" />
          </button>
        </div>
        <div class="flex items-center gap-1.5 mt-3 text-xs text-[var(--accent-copper)]">
          <AlertTriangle class="w-3.5 h-3.5" />
          Save this token now. It won't be shown again.
        </div>
      </div>

      <template #footer>
        <button class="btn btn-primary" @click="closeTokenModal">
          Done
        </button>
      </template>
    </BaseModal>
  </AppLayout>
</template>
