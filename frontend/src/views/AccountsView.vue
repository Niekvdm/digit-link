<script setup lang="ts">
import { ref } from 'vue'
import { useAccounts } from '@/composables'
import AppLayout from '@/components/layout/AppLayout.vue'
import { BaseModal, LoadingState, EmptyState } from '@/components/shared'
import { 
  Users, 
  UserPlus, 
  RefreshCw, 
  Key, 
  UserX,
  Copy,
  Check,
  AlertTriangle
} from 'lucide-vue-next'

const { 
  accounts, 
  loading, 
  refresh,
  createAccount,
  regenerateToken,
  deactivateAccount
} = useAccounts()

// Create modal
const showCreateModal = ref(false)
const newUsername = ref('')
const newIsAdmin = ref(false)
const createLoading = ref(false)

// Token modal
const showTokenModal = ref(false)
const tokenUsername = ref('')
const currentToken = ref('')
const tokenCopied = ref(false)

function formatDate(timestamp?: string) {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleDateString()
}

function getInitial(username: string) {
  return username.charAt(0).toUpperCase()
}

function openCreateModal() {
  newUsername.value = ''
  newIsAdmin.value = false
  showCreateModal.value = true
}

async function handleCreate() {
  if (!newUsername.value.trim()) return
  
  createLoading.value = true
  const result = await createAccount({
    username: newUsername.value.trim(),
    isAdmin: newIsAdmin.value
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

async function handleDeactivate(accountId: string) {
  if (!confirm('Deactivate this account? They will no longer be able to create tunnels.')) return
  
  const result = await deactivateAccount(accountId)
  
  if (!result.success) {
    alert(result.error || 'Failed to deactivate account')
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
              </div>
              <div class="text-xs text-[var(--text-muted)]">
                Created {{ formatDate(account.createdAt) }}
                <template v-if="account.lastUsed">
                  · Last used {{ formatDate(account.lastUsed) }}
                </template>
              </div>
            </div>

            <!-- Actions -->
            <div class="flex gap-2 flex-shrink-0">
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
