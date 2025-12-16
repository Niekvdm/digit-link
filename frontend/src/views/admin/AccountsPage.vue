<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
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
import { Plus, Trash2, RotateCcw, UserCheck, Shield, Building2, Key, Lock } from 'lucide-vue-next'

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
  { key: 'isAdmin', label: 'Role', width: '100px' },
  { key: 'orgName', label: 'Organization', sortable: true },
  { key: 'active', label: 'Status', width: '100px' },
  { key: 'hasPassword', label: 'Auth', width: '100px' },
  { key: 'lastUsed', label: 'Last Used', sortable: true, width: '140px' },
]

onMounted(async () => {
  await Promise.all([fetchAll(), fetchOrgs()])
})

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
  <div class="accounts-page">
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
    <div class="toolbar">
      <SearchInput v-model="searchQuery" placeholder="Search accounts..." />
      <label class="checkbox-label">
        <input type="checkbox" v-model="showInactive" />
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
    >
      <template #cell-username="{ row }">
        <div class="account-name">
          <div class="account-icon" :class="row.isAdmin ? 'account-icon--admin' : 'account-icon--user'">
            <Shield v-if="row.isAdmin" class="w-4 h-4" />
            <Building2 v-else-if="row.orgId" class="w-4 h-4" />
            <UserCheck v-else class="w-4 h-4" />
          </div>
          <span>{{ row.username }}</span>
        </div>
      </template>
      
      <template #cell-isAdmin="{ value }">
        <span class="role-badge" :class="value ? 'role-badge--admin' : 'role-badge--user'">
          {{ value ? 'Admin' : 'User' }}
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
        <span class="auth-type">
          <Key v-if="!value" class="w-3.5 h-3.5" title="Token Auth" />
          <Lock v-else class="w-3.5 h-3.5" title="Password Auth" />
          {{ value ? 'Password' : 'Token' }}
        </span>
      </template>
      
      <template #cell-lastUsed="{ value }">
        {{ value ? formatDate(value) : 'Never' }}
      </template>
      
      <template #actions="{ row }">
        <div class="action-buttons">
          <button 
            class="icon-btn" 
            title="Regenerate Token" 
            @click.stop="handleRegenerateToken(row)"
          >
            <RotateCcw class="w-4 h-4" />
          </button>
          <button 
            class="icon-btn" 
            title="Set Organization" 
            @click.stop="openOrgModal(row)"
            :disabled="row.isAdmin"
          >
            <Building2 class="w-4 h-4" />
          </button>
          <button 
            class="icon-btn" 
            title="Set Password" 
            @click.stop="openPasswordModal(row)"
          >
            <Lock class="w-4 h-4" />
          </button>
          <button 
            v-if="!row.active"
            class="icon-btn icon-btn--success" 
            title="Activate" 
            @click.stop="handleActivate(row)"
          >
            <UserCheck class="w-4 h-4" />
          </button>
          <button 
            v-else
            class="icon-btn icon-btn--danger" 
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
      <form @submit.prevent="handleCreate" class="form">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="form-group">
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
        
        <div class="form-group">
          <label class="form-label" for="acc-password">
            Password
            <span class="form-label-optional">(optional)</span>
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
        
        <div class="form-group">
          <label class="form-label" for="acc-org">
            Organization
            <span class="form-label-optional">(optional)</span>
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
        
        <label class="checkbox-card">
          <input type="checkbox" v-model="formIsAdmin" />
          <div class="checkbox-content">
            <Shield class="w-5 h-5" />
            <div>
              <strong>Administrator</strong>
              <p>Full access to all system features</p>
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
      <form @submit.prevent="handleSetOrg" class="form">
        <div class="form-group">
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
      <form @submit.prevent="handleSetPassword" class="form">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="form-group">
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

<style scoped>
.accounts-page {
  max-width: 1200px;
}

.toolbar {
  display: flex;
  align-items: center;
  gap: 1.5rem;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.checkbox-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  color: var(--text-secondary);
  cursor: pointer;
}

.checkbox-label input {
  accent-color: var(--accent-primary);
}

.account-name {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.account-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.account-icon--admin {
  background: rgba(var(--accent-primary-rgb), 0.1);
  color: var(--accent-primary);
}

.account-icon--user {
  background: rgba(var(--accent-secondary-rgb), 0.1);
  color: var(--accent-secondary);
}

.role-badge {
  font-size: 0.75rem;
  font-weight: 500;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
}

.role-badge--admin {
  background: rgba(var(--accent-primary-rgb), 0.15);
  color: var(--accent-primary);
}

.role-badge--user {
  background: var(--bg-elevated);
  color: var(--text-secondary);
}

.auth-type {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  font-size: 0.8125rem;
  color: var(--text-secondary);
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

.icon-btn--success:hover:not(:disabled) {
  background: rgba(var(--accent-secondary-rgb), 0.1);
  color: var(--accent-secondary);
}

.icon-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
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

.checkbox-card {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1rem 1.25rem;
  background: var(--bg-deep);
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  cursor: pointer;
  transition: border-color 0.2s ease;
}

.checkbox-card:has(input:checked) {
  border-color: var(--accent-primary);
  background: rgba(var(--accent-primary-rgb), 0.05);
}

.checkbox-card input {
  accent-color: var(--accent-primary);
  width: 18px;
  height: 18px;
}

.checkbox-content {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex: 1;
  color: var(--text-secondary);
}

.checkbox-content strong {
  display: block;
  color: var(--text-primary);
  font-size: 0.9375rem;
}

.checkbox-content p {
  font-size: 0.8125rem;
  margin: 0.25rem 0 0;
  color: var(--text-muted);
}

.mb-4 {
  margin-bottom: 1rem;
}
</style>
