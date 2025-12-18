<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { 
  PageHeader, 
  Modal, 
  StatusBadge,
  LoadingSpinner
} from '@/components/ui'
import { useAccounts } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import { 
  Save, 
  Lock, 
  Key,
  CheckCircle,
  Shield,
  Loader2,
  AlertCircle
} from 'lucide-vue-next'

const { myAccount, loading, error, fetchMyAccount, setMyPassword, setupMyTOTP, enableMyTOTP, disableMyTOTP } = useAccounts()
const { formatDate } = useFormatters()

// Form state
const formPassword = ref('')
const saving = ref(false)
const formError = ref('')
const successMessage = ref('')

// Modals
const showPasswordModal = ref(false)
const showTOTPSetupModal = ref(false)
const showTOTPDisableConfirm = ref(false)

// TOTP setup state
const totpSecret = ref('')
const totpUrl = ref('')
const totpCode = ref('')
const totpLoading = ref(false)
const totpError = ref('')
const disableTotpCode = ref('')

onMounted(async () => {
  try {
    await fetchMyAccount()
  } catch {
    // Error handled by composable
  }
})

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

// TOTP Setup
async function openTOTPSetupModal() {
  totpSecret.value = ''
  totpUrl.value = ''
  totpCode.value = ''
  totpError.value = ''
  totpLoading.value = true
  showTOTPSetupModal.value = true
  
  try {
    const { secret, url } = await setupMyTOTP()
    totpSecret.value = secret
    totpUrl.value = url
  } catch (e) {
    totpError.value = e instanceof Error ? e.message : 'Failed to setup TOTP'
  } finally {
    totpLoading.value = false
  }
}

async function handleEnableTOTP() {
  if (!totpCode.value || totpCode.value.length !== 6) {
    totpError.value = 'Please enter a valid 6-digit code'
    return
  }
  
  totpLoading.value = true
  totpError.value = ''
  
  try {
    await enableMyTOTP(totpCode.value)
    showTOTPSetupModal.value = false
    successMessage.value = 'Two-factor authentication enabled successfully'
    setTimeout(() => { successMessage.value = '' }, 5000)
  } catch (e) {
    totpError.value = e instanceof Error ? e.message : 'Failed to enable TOTP'
    totpCode.value = ''
  } finally {
    totpLoading.value = false
  }
}

// Auto-submit TOTP when 6 digits entered
watch(totpCode, (val) => {
  if (val.length === 6 && showTOTPSetupModal.value) {
    handleEnableTOTP()
  }
})

function openTOTPDisableModal() {
  disableTotpCode.value = ''
  totpError.value = ''
  showTOTPDisableConfirm.value = true
}

async function handleDisableTOTP() {
  if (!disableTotpCode.value || disableTotpCode.value.length !== 6) {
    totpError.value = 'Please enter a valid 6-digit code'
    return
  }
  
  totpLoading.value = true
  totpError.value = ''
  
  try {
    await disableMyTOTP(disableTotpCode.value)
    showTOTPDisableConfirm.value = false
    successMessage.value = 'Two-factor authentication disabled'
    setTimeout(() => { successMessage.value = '' }, 5000)
  } catch (e) {
    totpError.value = e instanceof Error ? e.message : 'Failed to disable TOTP'
    disableTotpCode.value = ''
  } finally {
    totpLoading.value = false
  }
}

function copySecret() {
  navigator.clipboard.writeText(totpSecret.value)
  successMessage.value = 'Secret copied to clipboard'
  setTimeout(() => { successMessage.value = '' }, 2000)
}
</script>

<template>
  <div class="max-w-[800px]">
    <PageHeader 
      title="My Account" 
      description="Manage your administrator account settings"
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
          <div class="w-12 h-12 rounded-xs flex items-center justify-center shrink-0 bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary">
            <Shield class="w-6 h-6" />
          </div>
          <div class="flex-1">
            <div class="flex items-center gap-3 mb-1">
              <h2 class="text-lg font-semibold text-text-primary m-0">{{ myAccount.username }}</h2>
            </div>
            <div class="flex items-center gap-4 text-sm text-text-secondary">
              <span class="text-xs font-medium py-1 px-2 rounded bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary">
                Administrator
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
          <div class="w-10 h-10 rounded-xs bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary flex items-center justify-center shrink-0">
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

      <!-- Two-Factor Authentication Section -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden mb-6">
        <div class="flex gap-4 p-6 border-b border-border-subtle bg-bg-elevated">
          <div 
            class="w-10 h-10 rounded-xs flex items-center justify-center shrink-0"
            :class="myAccount.totpEnabled 
              ? 'bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary'
              : 'bg-bg-deep text-text-muted'"
          >
            <Shield class="w-5 h-5" />
          </div>
          <div class="flex-1">
            <div class="flex items-center gap-3">
              <h3 class="text-base font-semibold text-text-primary m-0">Two-Factor Authentication</h3>
              <span 
                v-if="myAccount.totpEnabled"
                class="text-xs font-medium py-1 px-2 rounded bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary"
              >
                Enabled
              </span>
              <span 
                v-else
                class="text-xs font-medium py-1 px-2 rounded bg-bg-elevated text-text-muted"
              >
                Disabled
              </span>
            </div>
            <p class="text-sm text-text-secondary mt-1 mb-0">
              {{ myAccount.totpEnabled 
                ? 'Your account is protected with two-factor authentication' 
                : 'Add an extra layer of security to your account' 
              }}
            </p>
          </div>
        </div>
        <div class="p-6">
          <button 
            v-if="!myAccount.totpEnabled"
            class="btn btn-primary" 
            @click="openTOTPSetupModal"
          >
            <Shield class="w-4 h-4" />
            Setup Two-Factor Authentication
          </button>
          <button 
            v-else
            class="btn btn-danger" 
            @click="openTOTPDisableModal"
          >
            <Shield class="w-4 h-4" />
            Disable Two-Factor Authentication
          </button>
        </div>
      </div>

      <!-- Role Info -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
        <div class="flex gap-4 p-6 border-b border-border-subtle bg-bg-elevated">
          <div class="w-10 h-10 rounded-xs flex items-center justify-center shrink-0 bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary">
            <Shield class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-base font-semibold text-text-primary m-0">Your Role</h3>
            <p class="text-sm text-text-secondary mt-1 mb-0">
              You are a System Administrator with full access to all features.
            </p>
          </div>
        </div>
        <div class="p-6 text-sm text-text-secondary">
          <p class="m-0">As a System Administrator, you can:</p>
          <ul class="mt-2 mb-0 pl-5 space-y-1">
            <li>Manage all organizations and applications</li>
            <li>Create and manage user accounts</li>
            <li>Configure global whitelist rules</li>
            <li>Monitor all tunnel connections</li>
            <li>View audit logs and system activity</li>
          </ul>
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

    <!-- TOTP Setup Modal -->
    <Modal v-model="showTOTPSetupModal" title="Setup Two-Factor Authentication">
      <div class="flex flex-col gap-6">
        <!-- Error message -->
        <div 
          v-if="totpError" 
          class="flex items-start gap-2.5 py-3.5 px-4 bg-[rgba(var(--accent-red-rgb),0.1)] border border-[rgba(var(--accent-red-rgb),0.3)] rounded-xs text-sm text-accent-red"
        >
          <AlertCircle class="w-4 h-4 shrink-0" />
          <span>{{ totpError }}</span>
        </div>

        <!-- Loading state -->
        <div v-if="totpLoading && !totpSecret" class="flex items-center justify-center py-8">
          <Loader2 class="w-8 h-8 text-text-muted animate-spin" />
        </div>

        <template v-else-if="totpSecret">
          <p class="text-sm text-text-secondary text-center leading-relaxed m-0">
            Scan the QR code below with your authenticator app (Google Authenticator, Authy, etc.)
          </p>
          
          <!-- QR Code -->
          <div class="flex justify-center">
            <div class="w-[180px] h-[180px] bg-white rounded-xs flex items-center justify-center overflow-hidden">
              <img 
                v-if="totpUrl"
                :src="`https://api.qrserver.com/v1/create-qr-code/?size=160x160&data=${encodeURIComponent(totpUrl)}`"
                alt="TOTP QR Code"
                class="w-[160px] h-[160px]"
              />
            </div>
          </div>

          <!-- Manual entry secret -->
          <div class="text-center p-4 bg-bg-deep border border-dashed border-border-accent rounded-xs">
            <p class="text-xs text-text-muted mb-2 m-0">Can't scan? Enter this code manually:</p>
            <div class="flex items-center justify-center gap-2">
              <code class="font-mono text-[0.8125rem] text-accent-amber tracking-wider break-all">
                {{ totpSecret }}
              </code>
              <button 
                class="p-1.5 text-text-muted hover:text-text-secondary transition-colors bg-transparent border-none cursor-pointer"
                @click="copySecret"
                title="Copy secret"
              >
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <rect x="9" y="9" width="13" height="13" rx="2" ry="2" stroke-width="2"/>
                  <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" stroke-width="2"/>
                </svg>
              </button>
            </div>
          </div>

          <!-- TOTP code input -->
          <div>
            <label class="form-label" for="totp-code">Verification Code</label>
            <input
              id="totp-code"
              v-model="totpCode"
              type="text"
              inputmode="numeric"
              pattern="[0-9]*"
              maxlength="6"
              class="form-input text-center font-mono text-2xl tracking-[0.5em] py-4"
              placeholder="000000"
              autocomplete="one-time-code"
              :disabled="totpLoading"
            />
            <p class="form-hint text-center">Enter the 6-digit code from your authenticator app</p>
          </div>
        </template>
      </div>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showTOTPSetupModal = false" :disabled="totpLoading">
          Cancel
        </button>
        <button 
          class="btn btn-primary" 
          @click="handleEnableTOTP" 
          :disabled="totpLoading || totpCode.length !== 6"
        >
          <template v-if="totpLoading">
            <Loader2 class="w-4 h-4 animate-spin" />
            Verifying...
          </template>
          <template v-else>
            Enable Two-Factor Auth
          </template>
        </button>
      </template>
    </Modal>

    <!-- TOTP Disable Confirmation Modal -->
    <Modal v-model="showTOTPDisableConfirm" title="Disable Two-Factor Authentication">
      <div class="flex flex-col gap-5">
        <!-- Error message -->
        <div 
          v-if="totpError" 
          class="flex items-start gap-2.5 py-3.5 px-4 bg-[rgba(var(--accent-red-rgb),0.1)] border border-[rgba(var(--accent-red-rgb),0.3)] rounded-xs text-sm text-accent-red"
        >
          <AlertCircle class="w-4 h-4 shrink-0" />
          <span>{{ totpError }}</span>
        </div>

        <div class="flex items-start gap-3 p-4 bg-[rgba(var(--accent-amber-rgb),0.1)] border border-[rgba(var(--accent-amber-rgb),0.3)] rounded-xs">
          <AlertCircle class="w-5 h-5 text-accent-amber shrink-0 mt-0.5" />
          <div class="text-sm text-text-primary">
            <p class="m-0 font-medium">This will reduce your account security</p>
            <p class="m-0 mt-2 text-text-secondary">
              As an administrator, it's strongly recommended to keep two-factor authentication enabled.
            </p>
          </div>
        </div>

        <div>
          <label class="form-label" for="disable-totp-code">Current Authentication Code</label>
          <input
            id="disable-totp-code"
            v-model="disableTotpCode"
            type="text"
            inputmode="numeric"
            pattern="[0-9]*"
            maxlength="6"
            class="form-input text-center font-mono text-2xl tracking-[0.5em] py-4"
            placeholder="000000"
            autocomplete="one-time-code"
            :disabled="totpLoading"
          />
          <p class="form-hint text-center">Enter the 6-digit code to confirm</p>
        </div>
      </div>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showTOTPDisableConfirm = false" :disabled="totpLoading">
          Cancel
        </button>
        <button 
          class="btn btn-danger" 
          @click="handleDisableTOTP" 
          :disabled="totpLoading || disableTotpCode.length !== 6"
        >
          <template v-if="totpLoading">
            <Loader2 class="w-4 h-4 animate-spin" />
            Disabling...
          </template>
          <template v-else>
            Disable Two-Factor Auth
          </template>
        </button>
      </template>
    </Modal>
  </div>
</template>


