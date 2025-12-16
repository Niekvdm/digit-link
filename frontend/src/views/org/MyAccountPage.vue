<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { 
  PageHeader, 
  Modal, 
  StatusBadge,
  LoadingSpinner
} from '@/components/ui'
import { useOrgAccounts } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import { 
  User,
  Save, 
  Lock, 
  Crown, 
  UserCheck,
  Key,
  CheckCircle
} from 'lucide-vue-next'

const { myAccount, loading, error, fetchMyAccount, updateMyAccount, setMyPassword } = useOrgAccounts()
const { formatDate } = useFormatters()

// Form state
const formUsername = ref('')
const formPassword = ref('')
const formCurrentPassword = ref('')
const editingUsername = ref(false)
const saving = ref(false)
const formError = ref('')
const successMessage = ref('')

// Modals
const showPasswordModal = ref(false)

onMounted(async () => {
  try {
    await fetchMyAccount()
    if (myAccount.value) {
      formUsername.value = myAccount.value.username
    }
  } catch {
    // Error handled by composable
  }
})

// Username editing
function startEditingUsername() {
  formUsername.value = myAccount.value?.username || ''
  editingUsername.value = true
}

function cancelEditingUsername() {
  formUsername.value = myAccount.value?.username || ''
  editingUsername.value = false
}

async function saveUsername() {
  if (!formUsername.value.trim()) {
    formError.value = 'Username is required'
    return
  }
  
  saving.value = true
  formError.value = ''
  successMessage.value = ''
  
  try {
    await updateMyAccount(formUsername.value.trim())
    editingUsername.value = false
    successMessage.value = 'Username updated successfully'
    setTimeout(() => { successMessage.value = '' }, 5000)
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to update username'
  } finally {
    saving.value = false
  }
}

// Password
function openPasswordModal() {
  formPassword.value = ''
  formCurrentPassword.value = ''
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
  successMessage.value = ''
  
  try {
    await setMyPassword(formPassword.value)
    showPasswordModal.value = false
    successMessage.value = 'Password updated successfully'
    setTimeout(() => { successMessage.value = '' }, 5000)
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to set password'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="max-w-[800px]">
    <PageHeader 
      title="My Account" 
      description="Manage your personal account settings"
    />

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-20">
      <LoadingSpinner />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Success message -->
    <div 
      v-if="successMessage" 
      class="flex items-center gap-3 py-4 px-5 bg-[rgba(var(--accent-secondary-rgb),0.1)] border border-[rgba(var(--accent-secondary-rgb),0.3)] rounded-xs text-[0.9375rem] text-accent-secondary mb-6"
    >
      <CheckCircle class="w-5 h-5" />
      {{ successMessage }}
    </div>

    <!-- Content -->
    <template v-if="myAccount">
      <!-- Account Info Card -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden mb-6">
        <div class="flex gap-4 p-6 border-b border-border-subtle bg-bg-elevated">
          <div 
            class="w-12 h-12 rounded-xs flex items-center justify-center shrink-0"
            :class="myAccount.isOrgAdmin 
              ? 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber'
              : 'bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary'"
          >
            <Crown v-if="myAccount.isOrgAdmin" class="w-6 h-6" />
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
                <h2 class="text-lg font-semibold text-text-primary m-0">{{ myAccount.username }}</h2>
                <button 
                  class="text-xs text-accent-secondary hover:underline cursor-pointer bg-transparent border-none"
                  @click="startEditingUsername"
                >
                  Edit
                </button>
              </template>
            </div>
            <div class="flex items-center gap-4 text-sm text-text-secondary">
              <StatusBadge 
                :status="myAccount.active ? 'active' : 'inactive'" 
                size="sm"
              />
              <span 
                class="text-xs font-medium py-1 px-2 rounded"
                :class="myAccount.isOrgAdmin 
                  ? 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber' 
                  : 'bg-bg-elevated text-text-secondary'"
              >
                {{ myAccount.isOrgAdmin ? 'Org Admin' : 'User' }}
              </span>
            </div>
          </div>
        </div>

        <div class="p-6 grid grid-cols-2 gap-6">
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-text-muted mb-1">Account ID</dt>
            <dd class="font-mono text-sm text-text-secondary">{{ myAccount.id }}</dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-text-muted mb-1">Created</dt>
            <dd class="text-sm text-text-secondary">{{ formatDate(myAccount.createdAt) }}</dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-text-muted mb-1">Last Used</dt>
            <dd class="text-sm text-text-secondary">{{ myAccount.lastUsed ? formatDate(myAccount.lastUsed) : 'Never' }}</dd>
          </div>
          <div>
            <dt class="text-xs font-medium uppercase tracking-wider text-text-muted mb-1">Auth Type</dt>
            <dd class="flex items-center gap-2 text-sm text-text-secondary">
              <Key v-if="!myAccount.hasPassword" class="w-4 h-4" />
              <Lock v-else class="w-4 h-4" />
              {{ myAccount.hasPassword ? 'Password' : 'Token Only' }}
            </dd>
          </div>
        </div>
      </div>

      <!-- Password Section -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden mb-6">
        <div class="flex gap-4 p-6 border-b border-border-subtle bg-bg-elevated">
          <div class="w-10 h-10 rounded-xs bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary flex items-center justify-center shrink-0">
            <Lock class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-text-primary m-0">Password</h3>
            <p class="text-sm text-text-secondary mt-1 mb-0">
              {{ myAccount.hasPassword ? 'Change your password' : 'Set a password to enable password-based login' }}
            </p>
          </div>
        </div>
        <div class="p-6">
          <button class="btn btn-primary" @click="openPasswordModal">
            <Lock class="w-4 h-4" />
            {{ myAccount.hasPassword ? 'Change Password' : 'Set Password' }}
          </button>
        </div>
      </div>

      <!-- Role Info -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
        <div class="flex gap-4 p-6 border-b border-border-subtle bg-bg-elevated">
          <div 
            class="w-10 h-10 rounded-xs flex items-center justify-center shrink-0"
            :class="myAccount.isOrgAdmin 
              ? 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber'
              : 'bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary'"
          >
            <Crown v-if="myAccount.isOrgAdmin" class="w-5 h-5" />
            <User v-else class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-text-primary m-0">Your Role</h3>
            <p class="text-sm text-text-secondary mt-1 mb-0">
              {{ myAccount.isOrgAdmin 
                ? 'You are an Organization Admin with full access to manage accounts and settings.' 
                : 'You are a standard user in this organization.' 
              }}
            </p>
          </div>
        </div>
        <div class="p-6 text-sm text-text-secondary">
          <p class="m-0" v-if="myAccount.isOrgAdmin">
            As an Org Admin, you can:
          </p>
          <ul v-if="myAccount.isOrgAdmin" class="mt-2 mb-0 pl-5 space-y-1">
            <li>Create and manage organization accounts</li>
            <li>Promote or demote other users to Org Admin</li>
            <li>Enable or disable accounts</li>
            <li>Permanently delete accounts</li>
          </ul>
          <p class="m-0" v-else>
            Contact your organization administrator if you need additional permissions or access.
          </p>
        </div>
      </div>
    </template>

    <!-- Password Modal -->
    <Modal v-model="showPasswordModal" :title="myAccount?.hasPassword ? 'Change Password' : 'Set Password'">
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
          {{ saving ? 'Saving...' : (myAccount?.hasPassword ? 'Change Password' : 'Set Password') }}
        </button>
      </template>
    </Modal>
  </div>
</template>
