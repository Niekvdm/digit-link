<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { 
  PageHeader, 
  DataTable, 
  Modal, 
  ConfirmDialog,
  SearchInput,
  StatusBadge,
  TokenReveal
} from '@/components/ui'
import { useAccounts, useOrganizations } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import type { Account, CreateAccountRequest } from '@/types/api'
import { Plus, Trash2, RotateCcw, UserCheck, Shield, Building2, Key, Lock, Crown } from 'lucide-vue-next'

const router = useRouter()
const { accounts, loading, error, fetchAll, create, remove, activate, regenerateToken, setOrganization, setPassword } = useAccounts()
const { organizations, fetchAll: fetchOrgs } = useOrganizations()
const { formatDate } = useFormatters()

// Search & filter
const searchQuery = ref('')
const showInactive = ref(false)

const filteredAccounts = computed(() => {
  let result = accounts.value
  
  if (!showInactive.value) {
    result = result.filter(acc => acc.active)
  }
  
  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase()
    result = result.filter(acc => 
      acc.username.toLowerCase().includes(query) ||
      acc.orgName?.toLowerCase().includes(query)
    )
  }
  
  return result
})

// Modals
const showCreateModal = ref(false)
const showTokenModal = ref(false)
const showOrgModal = ref(false)
const showPasswordModal = ref(false)
const showDeleteConfirm = ref(false)

// Form state
const formUsername = ref('')
const formPassword = ref('')
const formIsAdmin = ref(false)
const formOrgId = ref('')
const formLoading = ref(false)
const formError = ref('')
const selectedAccount = ref<Account | null>(null)
const generatedToken = ref('')
const newPassword = ref('')

// Table columns
const columns = [
  { key: 'username', label: 'Username', sortable: true },
  { key: 'isAdmin', label: 'Role', width: '125px' },
  { key: 'orgName', label: 'Organization', sortable: true },
  { key: 'active', label: 'Status', width: '100px' },
  { key: 'hasPassword', label: 'Auth', width: '100px' },
  { key: 'lastUsed', label: 'Last Used', sortable: true, width: '140px' },
]

onMounted(async () => {
  await Promise.all([fetchAll(), fetchOrgs()])
})

// Navigate to detail
function viewAccount(account: Account) {
  router.push({ name: 'admin-account-detail', params: { accountId: account.id } })
}

// Create
function openCreateModal() {
  formUsername.value = ''
  formPassword.value = ''
  formIsAdmin.value = false
  formOrgId.value = ''
  formError.value = ''
  showCreateModal.value = true
}

async function handleCreate() {
  if (!formUsername.value.trim()) {
    formError.value = 'Username is required'
    return
  }
  
  if (formPassword.value && formPassword.value.length < 8) {
    formError.value = 'Password must be at least 8 characters'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    const data: CreateAccountRequest = {
      username: formUsername.value.trim(),
      isAdmin: formIsAdmin.value
    }
    if (formPassword.value) {
      data.password = formPassword.value
    }
    if (formOrgId.value) {
      data.orgId = formOrgId.value
    }
    
    const token = await create(data)
    if (token) {
      generatedToken.value = token
      showCreateModal.value = false
      showTokenModal.value = true
    }
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to create account'
  } finally {
    formLoading.value = false
  }
}

// Regenerate Token
async function handleRegenerateToken(account: Account) {
  selectedAccount.value = account
  
  try {
    const token = await regenerateToken(account.id)
    if (token) {
      generatedToken.value = token
      showTokenModal.value = true
    }
  } catch (e) {
    console.error('Failed to regenerate token:', e)
  }
}

// Set Organization
function openOrgModal(account: Account) {
  selectedAccount.value = account
  formOrgId.value = account.orgId || ''
  showOrgModal.value = true
}

async function handleSetOrg() {
  if (!selectedAccount.value) return
  
  formLoading.value = true
  
  try {
    await setOrganization(selectedAccount.value.id, formOrgId.value)
    showOrgModal.value = false
  } catch (e) {
    console.error('Failed to set organization:', e)
  } finally {
    formLoading.value = false
  }
}

// Set Password
function openPasswordModal(account: Account) {
  selectedAccount.value = account
  newPassword.value = ''
  formError.value = ''
  showPasswordModal.value = true
}

async function handleSetPassword() {
  if (!selectedAccount.value) return
  
  if (newPassword.value.length < 8) {
    formError.value = 'Password must be at least 8 characters'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    await setPassword(selectedAccount.value.id, newPassword.value)
    showPasswordModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to set password'
  } finally {
    formLoading.value = false
  }
}

// Activate/Deactivate
async function handleActivate(account: Account) {
  try {
    await activate(account.id)
  } catch (e) {
    console.error('Failed to activate account:', e)
  }
}

// Delete (deactivate)
function openDeleteConfirm(account: Account) {
  selectedAccount.value = account
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!selectedAccount.value) return
  
  try {
    await remove(selectedAccount.value.id)
    showDeleteConfirm.value = false
  } catch (e) {
    console.error('Delete failed:', e)
  }
}
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      title="Accounts" 
      description="Manage user accounts and access permissions"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          New Account
        </button>
      </template>
    </PageHeader>

    <!-- Toolbar -->
    <div class="flex items-center gap-6 mb-6 flex-wrap">
      <SearchInput v-model="searchQuery" placeholder="Search accounts..." />
      <label class="flex items-center gap-2 text-sm text-text-secondary cursor-pointer">
        <input type="checkbox" v-model="showInactive" class="accent-accent-primary" />
        <span>Show inactive</span>
      </label>
    </div>

    <!-- Error -->
    <div v-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="filteredAccounts"
      :loading="loading"
      empty-title="No accounts"
      empty-description="Create your first account to get started."
      row-key="id"
      @row-click="viewAccount"
    >
      <template #cell-username="{ row }">
        <div class="flex items-center gap-3">
          <div 
            class="w-8 h-8 rounded-xs flex items-center justify-center"
            :class="row.isAdmin 
              ? 'bg-[rgba(var(--accent-primary-rgb),0.1)] text-accent-primary' 
              : row.isOrgAdmin
                ? 'bg-[rgba(var(--accent-amber-rgb),0.1)] text-accent-amber'
                : 'bg-[rgba(var(--accent-secondary-rgb),0.1)] text-accent-secondary'"
          >
            <Shield v-if="row.isAdmin" class="w-4 h-4" />
            <Crown v-else-if="row.isOrgAdmin" class="w-4 h-4" />
            <Building2 v-else-if="row.orgId" class="w-4 h-4" />
            <UserCheck v-else class="w-4 h-4" />
          </div>
          <span>{{ row.username }}</span>
        </div>
      </template>
      
      <template #cell-isAdmin="{ row }">
        <span 
          class="text-xs font-medium py-1 px-2 rounded"
          :class="row.isAdmin 
            ? 'bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary' 
            : row.isOrgAdmin
              ? 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber'
              : 'bg-bg-elevated text-text-secondary'"
        >
          {{ row.isAdmin ? 'Admin' : row.isOrgAdmin ? 'Org Admin' : 'User' }}
        </span>
      </template>
      
      <template #cell-orgName="{ value }">
        {{ value || '—' }}
      </template>
      
      <template #cell-active="{ value }">
        <StatusBadge 
          :status="value ? 'active' : 'inactive'" 
          size="sm"
        />
      </template>
      
      <template #cell-hasPassword="{ value }">
        <span class="flex items-center gap-1.5 text-[0.8125rem] text-text-secondary">
          <Key v-if="!value" class="w-3.5 h-3.5" title="Token Auth" />
          <Lock v-else class="w-3.5 h-3.5" title="Password Auth" />
          {{ value ? 'Password' : 'Token' }}
        </span>
      </template>
      
      <template #cell-lastUsed="{ value }">
        {{ value ? formatDate(value as string) : 'Never' }}
      </template>
      
      <template #actions="{ row }">
        <div class="flex items-center gap-1">
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-bg-elevated hover:text-text-primary" 
            title="Regenerate Token" 
            @click.stop="handleRegenerateToken(row)"
          >
            <RotateCcw class="w-4 h-4" />
          </button>
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-bg-elevated hover:text-text-primary disabled:opacity-30 disabled:cursor-not-allowed" 
            title="Set Organization" 
            @click.stop="openOrgModal(row)"
            :disabled="row.isAdmin"
          >
            <Building2 class="w-4 h-4" />
          </button>
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-bg-elevated hover:text-text-primary" 
            title="Set Password" 
            @click.stop="openPasswordModal(row)"
          >
            <Lock class="w-4 h-4" />
          </button>
          <button 
            v-if="!row.active"
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-[rgba(var(--accent-secondary-rgb),0.1)] hover:text-accent-secondary" 
            title="Activate" 
            @click.stop="handleActivate(row)"
          >
            <UserCheck class="w-4 h-4" />
          </button>
          <button 
            v-else
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-[rgba(var(--accent-red-rgb),0.1)] hover:text-accent-red" 
            title="Deactivate" 
            @click.stop="openDeleteConfirm(row)"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </template>
      
      <template #emptyAction>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          Create Account
        </button>
      </template>
    </DataTable>

    <!-- Create Modal -->
    <Modal v-model="showCreateModal" title="New Account">
      <form @submit.prevent="handleCreate" class="flex flex-col gap-5">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="acc-username">Username</label>
          <input
            id="acc-username"
            v-model="formUsername"
            type="text"
            class="form-input"
            placeholder="Enter username"
            autofocus
          />
        </div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="acc-password">
            Password
            <span class="font-normal normal-case tracking-normal text-text-muted">(optional)</span>
          </label>
          <input
            id="acc-password"
            v-model="formPassword"
            type="password"
            class="form-input"
            placeholder="Leave blank for token auth"
            autocomplete="new-password"
          />
          <p class="form-hint">If set, user can login with password. Min 8 characters.</p>
        </div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="acc-org">
            Organization
            <span class="font-normal normal-case tracking-normal text-text-muted">(optional)</span>
          </label>
          <select
            id="acc-org"
            v-model="formOrgId"
            class="form-input"
            :disabled="formIsAdmin"
          >
            <option value="">No organization</option>
            <option v-for="org in organizations" :key="org.id" :value="org.id">
              {{ org.name }}
            </option>
          </select>
        </div>
        
        <label class="flex items-center gap-4 p-4 bg-bg-deep border border-border-subtle rounded-xs cursor-pointer transition-colors duration-200 has-[:checked]:border-accent-primary has-[:checked]:bg-[rgba(var(--accent-primary-rgb),0.05)]">
          <input type="checkbox" v-model="formIsAdmin" class="accent-accent-primary w-[18px] h-[18px]" />
          <div class="flex items-center gap-3 flex-1 text-text-secondary">
            <Shield class="w-5 h-5" />
            <div>
              <strong class="block text-text-primary text-[0.9375rem]">Administrator</strong>
              <p class="text-[0.8125rem] mt-1 mb-0 text-text-muted">Full access to all system features</p>
            </div>
          </div>
        </label>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showCreateModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button 
          class="btn btn-primary" 
          @click="handleCreate" 
          :disabled="formLoading || !formUsername.trim()"
        >
          {{ formLoading ? 'Creating...' : 'Create Account' }}
        </button>
      </template>
    </Modal>

    <!-- Token Modal -->
    <Modal v-model="showTokenModal" title="Account Token">
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
        <button class="btn btn-secondary" @click="showOrgModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button class="btn btn-primary" @click="handleSetOrg" :disabled="formLoading">
          {{ formLoading ? 'Saving...' : 'Save' }}
        </button>
      </template>
    </Modal>

    <!-- Password Modal -->
    <Modal v-model="showPasswordModal" title="Set Password">
      <form @submit.prevent="handleSetPassword" class="flex flex-col gap-5">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="new-password">New Password</label>
          <input
            id="new-password"
            v-model="newPassword"
            type="password"
            class="form-input"
            placeholder="Enter new password"
            autocomplete="new-password"
          />
          <p class="form-hint">Minimum 8 characters</p>
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showPasswordModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button 
          class="btn btn-primary" 
          @click="handleSetPassword" 
          :disabled="formLoading || newPassword.length < 8"
        >
          {{ formLoading ? 'Saving...' : 'Set Password' }}
        </button>
      </template>
    </Modal>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="Deactivate Account"
      :message="`Are you sure you want to deactivate '${selectedAccount?.username}'? They will no longer be able to access the system.`"
      confirm-text="Deactivate"
      variant="danger"
      @confirm="handleDelete"
    />
  </div>
</template>
