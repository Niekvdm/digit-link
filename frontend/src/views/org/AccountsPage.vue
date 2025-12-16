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
import { useOrgAccounts } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import type { Account } from '@/types/api'
import { Plus, Trash2, UserCheck, Crown, Key, Lock } from 'lucide-vue-next'

const router = useRouter()
const { accounts, loading, error, fetchAll, create, deactivate, activate } = useOrgAccounts()
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
      acc.username.toLowerCase().includes(query)
    )
  }
  
  return result
})

// Modals
const showCreateModal = ref(false)
const showTokenModal = ref(false)
const showDeleteConfirm = ref(false)

// Form state
const formUsername = ref('')
const formPassword = ref('')
const formIsOrgAdmin = ref(false)
const formLoading = ref(false)
const formError = ref('')
const selectedAccount = ref<Account | null>(null)
const generatedToken = ref('')

// Table columns
const columns = [
  { key: 'username', label: 'Username', sortable: true },
  { key: 'isOrgAdmin', label: 'Role', width: '125px' },
  { key: 'active', label: 'Status', width: '100px' },
  { key: 'hasPassword', label: 'Auth', width: '100px' },
  { key: 'lastUsed', label: 'Last Used', sortable: true, width: '140px' },
]

onMounted(() => {
  fetchAll()
})

// Navigate to detail
function viewAccount(account: Account) {
  router.push({ name: 'org-account-detail', params: { accountId: account.id } })
}

// Create
function openCreateModal() {
  formUsername.value = ''
  formPassword.value = ''
  formIsOrgAdmin.value = false
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
    const token = await create({
      username: formUsername.value.trim(),
      password: formPassword.value || undefined,
      isOrgAdmin: formIsOrgAdmin.value
    })
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

async function handleDeactivate() {
  if (!selectedAccount.value) return
  
  try {
    await deactivate(selectedAccount.value.id)
    showDeleteConfirm.value = false
  } catch (e) {
    console.error('Deactivate failed:', e)
  }
}
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      title="Accounts" 
      description="Manage organization user accounts"
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
        <input type="checkbox" v-model="showInactive" class="accent-accent-secondary" />
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
      empty-description="Create your first organization account to get started."
      row-key="id"
      @row-click="viewAccount"
    >
      <template #cell-username="{ row }">
        <div class="flex items-center gap-3">
          <div 
            class="w-8 h-8 rounded-xs flex items-center justify-center"
            :class="row.isOrgAdmin 
              ? 'bg-[rgba(var(--accent-amber-rgb),0.1)] text-accent-amber' 
              : 'bg-[rgba(var(--accent-secondary-rgb),0.1)] text-accent-secondary'"
          >
            <Crown v-if="row.isOrgAdmin" class="w-4 h-4" />
            <UserCheck v-else class="w-4 h-4" />
          </div>
          <span>{{ row.username }}</span>
        </div>
      </template>
      
      <template #cell-isOrgAdmin="{ value }">
        <span 
          class="text-xs font-medium py-1 px-2 rounded"
          :class="value 
            ? 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber' 
            : 'bg-bg-elevated text-text-secondary'"
        >
          {{ value ? 'Org Admin' : 'User' }}
        </span>
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
        {{ value ? formatDate(value) : 'Never' }}
      </template>
      
      <template #actions="{ row }">
        <div class="flex items-center gap-1">
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
        
        <label class="flex items-center gap-4 p-4 bg-bg-deep border border-border-subtle rounded-xs cursor-pointer transition-colors duration-200 has-[:checked]:border-accent-amber has-[:checked]:bg-[rgba(var(--accent-amber-rgb),0.05)]">
          <input type="checkbox" v-model="formIsOrgAdmin" class="accent-accent-amber w-[18px] h-[18px]" />
          <div class="flex items-center gap-3 flex-1 text-text-secondary">
            <Crown class="w-5 h-5" />
            <div>
              <strong class="block text-text-primary text-[0.9375rem]">Organization Admin</strong>
              <p class="text-[0.8125rem] mt-1 mb-0 text-text-muted">Can manage all accounts in this organization</p>
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

    <!-- Deactivate Confirmation -->
    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="Deactivate Account"
      :message="`Are you sure you want to deactivate '${selectedAccount?.username}'? They will no longer be able to access the system.`"
      confirm-text="Deactivate"
      variant="danger"
      @confirm="handleDeactivate"
    />
  </div>
</template>
