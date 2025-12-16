<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { 
  Modal, 
  ConfirmDialog,
  StatusBadge,
  TokenReveal,
  LoadingSpinner
} from '@/components/ui'
import { useAccounts, useOrganizations } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import { 
  ArrowLeft, 
  Save, 
  RotateCcw, 
  Lock, 
  Crown, 
  Shield,
  UserCheck,
  Building2,
  Trash2,
  AlertTriangle,
  Key
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const { currentAccount, loading, error, fetchOne, updateUsername, setPassword, regenerateToken, setOrganization, setOrgAdmin, activate, remove, hardDelete } = useAccounts()
const { organizations, fetchAll: fetchOrgs } = useOrganizations()
const { formatDate } = useFormatters()

const accountId = computed(() => route.params.accountId as string)

// Form state
const formUsername = ref('')
const formPassword = ref('')
const formOrgId = ref('')
const editingUsername = ref(false)
const saving = ref(false)
const formError = ref('')

// Modals
const showPasswordModal = ref(false)
const showOrgModal = ref(false)
const showTokenModal = ref(false)
const showDeactivateConfirm = ref(false)
const showHardDeleteConfirm = ref(false)
const hardDeleteConfirmText = ref('')
const generatedToken = ref('')

onMounted(async () => {
  try {
    await Promise.all([fetchOne(accountId.value), fetchOrgs()])
    if (currentAccount.value) {
      formUsername.value = currentAccount.value.username
      formOrgId.value = currentAccount.value.orgId || ''
    }
  } catch {
    router.push({ name: 'admin-accounts' })
  }
})

function goBack() {
  router.push({ name: 'admin-accounts' })
}

// Username editing
function startEditingUsername() {
  formUsername.value = currentAccount.value?.username || ''
  editingUsername.value = true
}

function cancelEditingUsername() {
  formUsername.value = currentAccount.value?.username || ''
  editingUsername.value = false
}

async function saveUsername() {
  if (!formUsername.value.trim()) {
    formError.value = 'Username is required'
    return
  }
  
  saving.value = true
  formError.value = ''
  
  try {
    await updateUsername(accountId.value, formUsername.value.trim())
    editingUsername.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to update username'
  } finally {
    saving.value = false
  }
}

// Password
function openPasswordModal() {
  formPassword.value = ''
  formError.value = ''
  showPasswordModal.value = true
}

async function handleSetPassword() {
  if (formPassword.value.length < 8) {
    formError.value = 'Password must be at least 8 characters'
    return
  }
  
  saving.value = true
  formError.value = ''
  
  try {
    await setPassword(accountId.value, formPassword.value)
    showPasswordModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to set password'
  } finally {
    saving.value = false
  }
}

// Organization
function openOrgModal() {
  formOrgId.value = currentAccount.value?.orgId || ''
  showOrgModal.value = true
}

async function handleSetOrg() {
  saving.value = true
  
  try {
    await setOrganization(accountId.value, formOrgId.value)
    showOrgModal.value = false
    // Refresh to get updated org name
    await fetchOne(accountId.value)
  } catch (e) {
    console.error('Failed to set organization:', e)
  } finally {
    saving.value = false
  }
}

// Token regeneration
async function handleRegenerateToken() {
  try {
    const token = await regenerateToken(accountId.value)
    if (token) {
      generatedToken.value = token
      showTokenModal.value = true
    }
  } catch (e) {
    console.error('Failed to regenerate token:', e)
  }
}

// Org Admin toggle
async function toggleOrgAdmin() {
  if (currentAccount.value?.isAdmin) return
  if (!currentAccount.value?.orgId) return
  
  try {
    await setOrgAdmin(accountId.value, !currentAccount.value?.isOrgAdmin)
  } catch (e) {
    console.error('Failed to toggle org admin:', e)
  }
}

// Activate/Deactivate
async function handleActivate() {
  try {
    await activate(accountId.value)
  } catch (e) {
    console.error('Failed to activate:', e)
  }
}

async function handleDeactivate() {
  try {
    await remove(accountId.value)
    showDeactivateConfirm.value = false
  } catch (e) {
    console.error('Failed to deactivate:', e)
  }
}

// Hard delete
async function handleHardDelete() {
  if (hardDeleteConfirmText.value !== currentAccount.value?.username) {
    return
  }
  
  try {
    await hardDelete(accountId.value)
    router.push({ name: 'admin-accounts' })
  } catch (e) {
    console.error('Failed to delete:', e)
  }
}

// Helper to get role display
const roleDisplay = computed(() => {
  if (!currentAccount.value) return { label: 'User', class: 'bg-bg-elevated text-text-secondary' }
  if (currentAccount.value.isAdmin) return { label: 'Admin', class: 'bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary' }
  if (currentAccount.value.isOrgAdmin) return { label: 'Org Admin', class: 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber' }
  return { label: 'User', class: 'bg-bg-elevated text-text-secondary' }
})
</script>

<template>
  <div class="max-w-[800px]">
    <!-- Header -->
    <div class="flex items-center gap-4 mb-8">
      <button 
        class="w-10 h-10 flex items-center justify-center border border-border-subtle rounded-xs bg-transparent text-text-secondary cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary"
        @click="goBack"
      >
        <ArrowLeft class="w-5 h-5" />
      </button>
      <div class="flex-1">
        <h1 class="text-2xl font-display font-semibold text-text-primary m-0">Account Details</h1>
        <p class="text-sm text-text-secondary mt-1 mb-0">Manage account settings and permissions</p>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-20">
      <LoadingSpinner />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Content -->
    <template v-else-if="currentAccount">
      <!-- Account Info Card -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden mb-6">
        <div class="flex gap-4 p-6 border-b border-border-subtle bg-bg-elevated">
          <div 
            class="w-12 h-12 rounded-xs flex items-center justify-center shrink-0"
            :class="currentAccount.isAdmin 
              ? 'bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary'
              : currentAccount.isOrgAdmin 
                ? 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber'
                : 'bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary'"
          >
            <Shield v-if="currentAccount.isAdmin" class="w-6 h-6" />
            <Crown v-else-if="currentAccount.isOrgAdmin" class="w-6 h-6" />
            <UserCheck v-else class="w-6 h-6" />
          </div>
          <div class="flex-1">
            <div class="flex items-center gap-3 mb-1">
              <template v-if="editingUsername">
                <input 
                  v-model="formUsername" 
                  type="text" 
                  class="form-input py-1.5 text-lg font-semibold"
                  @keyup.enter="saveUsername"
                  @keyup.escape="cancelEditingUsername"
                />
                <button class="btn btn-sm btn-primary" @click="saveUsername" :disabled="saving">
                  <Save class="w-4 h-4" />
                </button>
                <button class="btn btn-sm btn-secondary" @click="cancelEditingUsername">
                  Cancel
                </button>
              </template>
              <template v-else>
                <h2 class="text-lg font-semibold text-text-primary m-0">{{ currentAccount.username }}</h2>
                <button 
                  class="text-xs text-accent-primary hover:underline cursor-pointer bg-transparent border-none"
                  @click="startEditingUsername"
                >
                  Edit
                </button>
              </template>
            </div>
            <div class="flex items-center gap-4 text-sm text-text-secondary">
              <StatusBadge 
                :status="currentAccount.active ? 'active' : 'inactive'" 
                size="sm"
              />
              <span 
                class="text-xs font-medium py-1 px-2 rounded"
                :class="roleDisplay.class"
              >
                {{ roleDisplay.label }}
              </span>
            </div>
          </div>
        </div>

        <div class="p-6 grid grid-cols-2 gap-6">
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-text-muted mb-1">Account ID</dt>
            <dd class="font-mono text-sm text-text-secondary">{{ currentAccount.id }}</dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-text-muted mb-1">Created</dt>
            <dd class="text-sm text-text-secondary">{{ formatDate(currentAccount.createdAt) }}</dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-text-muted mb-1">Last Used</dt>
            <dd class="text-sm text-text-secondary">{{ currentAccount.lastUsed ? formatDate(currentAccount.lastUsed) : 'Never' }}</dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-text-muted mb-1">Auth Type</dt>
            <dd class="flex items-center gap-2 text-sm text-text-secondary">
              <Key v-if="!currentAccount.hasPassword" class="w-4 h-4" />
              <Lock v-else class="w-4 h-4" />
              {{ currentAccount.hasPassword ? 'Password' : 'Token Only' }}
            </dd>
          </div>
        </div>
      </div>

      <!-- Organization Section -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden mb-6" v-if="!currentAccount.isAdmin">
        <div class="p-4 border-b border-border-subtle bg-bg-elevated flex items-center justify-between">
          <h3 class="text-base font-semibold text-text-primary m-0 flex items-center gap-2">
            <Building2 class="w-5 h-5" />
            Organization
          </h3>
          <button class="btn btn-sm btn-secondary" @click="openOrgModal">
            Change
          </button>
        </div>
        <div class="p-6">
          <p class="text-sm text-text-primary m-0" v-if="currentAccount.orgName">
            <strong>{{ currentAccount.orgName }}</strong>
          </p>
          <p class="text-sm text-text-secondary m-0" v-else>
            No organization assigned
          </p>
          
          <!-- Org Admin Toggle -->
          <div class="flex items-center justify-between mt-4 pt-4 border-t border-border-subtle" v-if="currentAccount.orgId">
            <div class="flex items-center gap-4">
              <Crown class="w-5 h-5 text-accent-amber" />
              <div>
                <p class="text-sm font-medium text-text-primary m-0">Organization Admin</p>
                <p class="text-xs text-text-secondary mt-0.5 mb-0">Can manage all accounts in this organization</p>
              </div>
            </div>
            <button 
              class="relative w-12 h-6 rounded-full transition-colors duration-200 cursor-pointer border-none"
              :class="currentAccount.isOrgAdmin ? 'bg-accent-amber' : 'bg-bg-deep'"
              @click="toggleOrgAdmin"
            >
              <span 
                class="absolute top-0.5 w-5 h-5 rounded-full bg-white shadow transition-transform duration-200"
                :class="currentAccount.isOrgAdmin ? 'left-[26px]' : 'left-0.5'"
              />
            </button>
          </div>
        </div>
      </div>

      <!-- Authentication Section -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden mb-6">
        <div class="p-4 border-b border-border-subtle bg-bg-elevated">
          <h3 class="text-base font-semibold text-text-primary m-0">Authentication</h3>
        </div>
        <div class="p-6 flex flex-wrap gap-3">
          <button class="btn btn-secondary" @click="openPasswordModal">
            <Lock class="w-4 h-4" />
            {{ currentAccount.hasPassword ? 'Change Password' : 'Set Password' }}
          </button>
          <button class="btn btn-secondary" @click="handleRegenerateToken">
            <RotateCcw class="w-4 h-4" />
            Regenerate Token
          </button>
        </div>
      </div>

      <!-- Status Section -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden mb-6">
        <div class="p-4 border-b border-border-subtle bg-bg-elevated">
          <h3 class="text-base font-semibold text-text-primary m-0">Account Status</h3>
        </div>
        <div class="p-6">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-text-primary m-0">
                Account is currently <strong>{{ currentAccount.active ? 'active' : 'inactive' }}</strong>
              </p>
              <p class="text-sm text-text-secondary mt-1 mb-0">
                {{ currentAccount.active ? 'User can access the system' : 'User cannot access the system' }}
              </p>
            </div>
            <button 
              v-if="currentAccount.active"
              class="btn btn-danger"
              @click="showDeactivateConfirm = true"
            >
              Deactivate
            </button>
            <button 
              v-else
              class="btn btn-primary"
              @click="handleActivate"
            >
              Activate
            </button>
          </div>
        </div>
      </div>

      <!-- Danger Zone -->
      <div class="bg-bg-surface border border-accent-red/30 rounded-xs overflow-hidden">
        <div class="p-4 border-b border-accent-red/30 bg-[rgba(var(--accent-red-rgb),0.05)]">
          <h3 class="text-base font-semibold text-accent-red m-0 flex items-center gap-2">
            <AlertTriangle class="w-5 h-5" />
            Danger Zone
          </h3>
        </div>
        <div class="p-6">
          <div class="flex items-center justify-between">
            <div>
              <p class="text-sm text-text-primary m-0 font-medium">Permanently delete this account</p>
              <p class="text-sm text-text-secondary mt-1 mb-0">
                This action cannot be undone. All associated data will be permanently removed.
              </p>
            </div>
            <button 
              class="btn btn-danger"
              @click="showHardDeleteConfirm = true"
            >
              <Trash2 class="w-4 h-4" />
              Delete Forever
            </button>
          </div>
        </div>
      </div>
    </template>

    <!-- Password Modal -->
    <Modal v-model="showPasswordModal" title="Set Password">
      <form @submit.prevent="handleSetPassword" class="flex flex-col gap-5">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="new-password">New Password</label>
          <input
            id="new-password"
            v-model="formPassword"
            type="password"
            class="form-input"
            placeholder="Enter new password"
            autocomplete="new-password"
          />
          <p class="form-hint">Minimum 8 characters</p>
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showPasswordModal = false" :disabled="saving">
          Cancel
        </button>
        <button 
          class="btn btn-primary" 
          @click="handleSetPassword" 
          :disabled="saving || formPassword.length < 8"
        >
          {{ saving ? 'Saving...' : 'Set Password' }}
        </button>
      </template>
    </Modal>

    <!-- Organization Modal -->
    <Modal v-model="showOrgModal" title="Set Organization">
      <form @submit.prevent="handleSetOrg" class="flex flex-col gap-5">
        <div class="flex flex-col gap-2">
          <label class="form-label" for="set-org">Organization</label>
          <select
            id="set-org"
            v-model="formOrgId"
            class="form-input"
          >
            <option value="">No organization</option>
            <option v-for="org in organizations" :key="org.id" :value="org.id">
              {{ org.name }}
            </option>
          </select>
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showOrgModal = false" :disabled="saving">
          Cancel
        </button>
        <button class="btn btn-primary" @click="handleSetOrg" :disabled="saving">
          {{ saving ? 'Saving...' : 'Save' }}
        </button>
      </template>
    </Modal>

    <!-- Token Modal -->
    <Modal v-model="showTokenModal" title="New Token Generated">
      <TokenReveal 
        :value="generatedToken"
        label="API Token"
        show-warning
        warning-text="Save this token securely. It will only be shown once!"
      />
      
      <template #footer>
        <button class="btn btn-primary" @click="showTokenModal = false">
          Done
        </button>
      </template>
    </Modal>

    <!-- Deactivate Confirmation -->
    <ConfirmDialog
      v-model="showDeactivateConfirm"
      title="Deactivate Account"
      :message="`Are you sure you want to deactivate '${currentAccount?.username}'? They will no longer be able to access the system.`"
      confirm-text="Deactivate"
      variant="danger"
      @confirm="handleDeactivate"
    />

    <!-- Hard Delete Confirmation -->
    <Modal v-model="showHardDeleteConfirm" title="Permanently Delete Account">
      <div class="flex flex-col gap-4">
        <div class="flex items-start gap-3 p-4 bg-[rgba(var(--accent-red-rgb),0.1)] border border-accent-red/30 rounded-xs">
          <AlertTriangle class="w-5 h-5 text-accent-red shrink-0 mt-0.5" />
          <div class="text-sm text-text-primary">
            <p class="m-0 font-medium">This action is irreversible!</p>
            <p class="m-0 mt-2 text-text-secondary">
              Permanently deleting this account will remove all associated data including API keys and activity logs.
            </p>
          </div>
        </div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label">
            Type <strong class="text-accent-red">{{ currentAccount?.username }}</strong> to confirm
          </label>
          <input
            v-model="hardDeleteConfirmText"
            type="text"
            class="form-input"
            :placeholder="currentAccount?.username"
          />
        </div>
      </div>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showHardDeleteConfirm = false">
          Cancel
        </button>
        <button 
          class="btn btn-danger" 
          @click="handleHardDelete"
          :disabled="hardDeleteConfirmText !== currentAccount?.username"
        >
          <Trash2 class="w-4 h-4" />
          Delete Forever
        </button>
      </template>
    </Modal>
  </div>
</template>
