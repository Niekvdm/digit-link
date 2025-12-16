<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import { ThemeSwitcher } from '@/components/shared'
import { User, Lock, Key, ArrowRight, ArrowLeft, AlertCircle, Shield, Building2, Loader2, Check } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()

// Form state
type LoginStep = 'username' | 'password' | 'totp' | 'totp-setup'
const currentStep = ref<LoginStep>('username')
const username = ref('')
const password = ref('')
const totpCode = ref('')
const loading = ref(false)
const error = ref('')

// Account metadata from check-account
const accountType = ref<'admin' | 'org' | 'user' | null>(null)
const requiresTOTP = ref(false)
const orgName = ref('')

// TOTP setup state
const totpSecret = ref('')
const totpUrl = ref('')
const pendingToken = ref('')

// Computed
const stepTitle = computed(() => {
  switch (currentStep.value) {
    case 'username': return 'Welcome'
    case 'password': return accountType.value === 'admin' ? 'Administrator' : orgName.value || 'Sign In'
    case 'totp': return 'Verification'
    case 'totp-setup': return 'Security Setup'
  }
})

const stepDescription = computed(() => {
  switch (currentStep.value) {
    case 'username': return 'Enter your username to continue'
    case 'password': return 'Enter your password to authenticate'
    case 'totp': return 'Enter the 6-digit code from your authenticator'
    case 'totp-setup': return 'Set up two-factor authentication for your account'
  }
})

const accentColor = computed(() => {
  return accountType.value === 'admin' ? 'primary' : 'secondary'
})

// Check if already authenticated
onMounted(async () => {
  if (authStore.isAuthenticated) {
    const isValid = await authStore.validateToken()
    if (isValid) {
      redirectToDashboard()
    }
  }
})

// Step 1: Check account exists
async function handleUsernameSubmit() {
  const trimmed = username.value.trim()
  if (!trimmed) {
    error.value = 'Please enter your username'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await fetch('/auth/check-account', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: trimmed })
    })

    const data = await response.json()

    if (!data.exists) {
      error.value = 'Account not found'
      loading.value = false
      return
    }

    accountType.value = data.accountType || 'user'
    requiresTOTP.value = data.requiresTotp || false
    orgName.value = data.orgName || ''
    
    // Transition to password step
    currentStep.value = 'password'
  } catch {
    error.value = 'Connection error. Please try again.'
  } finally {
    loading.value = false
  }
}

// Step 2: Authenticate with password
async function handlePasswordSubmit() {
  if (!password.value) {
    error.value = 'Please enter your password'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await fetch('/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: username.value.trim(),
        password: password.value
      })
    })

    const data = await response.json()

    if (!response.ok || !data.success) {
      error.value = data.error || 'Authentication failed'
      loading.value = false
      return
    }

    // Direct login (no TOTP required)
    if (data.token) {
      completeLogin(data.token, data.accountType, data.orgId, data.orgName, data.isOrgAdmin)
      return
    }

    // TOTP required
    pendingToken.value = data.pendingToken

    if (data.needsSetup) {
      // Fetch TOTP setup info
      await fetchTOTPSetup()
      currentStep.value = 'totp-setup'
    } else if (data.needsTotp) {
      currentStep.value = 'totp'
    }
  } catch {
    error.value = 'Connection error. Please try again.'
  } finally {
    loading.value = false
  }
}

// Fetch TOTP secret for setup
async function fetchTOTPSetup() {
  const response = await fetch(`/auth/totp/setup?token=${pendingToken.value}`)
  const data = await response.json()
  
  if (data.success) {
    totpSecret.value = data.secret
    totpUrl.value = data.url
  } else {
    error.value = data.error || 'Failed to initialize TOTP setup'
  }
}

// Step 3a: Verify TOTP code
async function handleTOTPVerify() {
  if (!totpCode.value || totpCode.value.length !== 6) {
    error.value = 'Please enter a valid 6-digit code'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await fetch('/auth/totp/verify', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pendingToken: pendingToken.value,
        code: totpCode.value
      })
    })

    const data = await response.json()

    if (!response.ok || !data.success) {
      error.value = data.error || 'Invalid code'
      totpCode.value = ''
      loading.value = false
      return
    }

    completeLogin(data.token, data.accountType, data.orgId, data.orgName, data.isOrgAdmin)
  } catch {
    error.value = 'Invalid code'
    totpCode.value = ''
    loading.value = false
  }
}

// Step 3b: Complete TOTP setup
async function handleTOTPSetup() {
  if (!totpCode.value || totpCode.value.length !== 6) {
    error.value = 'Please enter the code from your authenticator app'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await fetch('/auth/totp/setup', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pendingToken: pendingToken.value,
        code: totpCode.value
      })
    })

    const data = await response.json()

    if (!response.ok || !data.success) {
      error.value = data.error || 'Failed to verify code'
      totpCode.value = ''
      loading.value = false
      return
    }

    completeLogin(data.token, data.accountType, data.orgId, data.orgName, data.isOrgAdmin)
  } catch {
    error.value = 'Connection error. Please try again.'
    loading.value = false
  }
}

function completeLogin(token: string, type: string, orgId?: string, orgName?: string, isOrgAdmin?: boolean) {
  const userType = type === 'admin' ? 'admin' : 'org'
  authStore.setToken(token, userType, orgId, orgName, username.value.trim(), isOrgAdmin)
  redirectToDashboard()
}

function redirectToDashboard() {
  if (authStore.isOrgUser) {
    router.push('/portal')
  } else {
    router.push('/dashboard')
  }
}

function goBack() {
  error.value = ''
  totpCode.value = ''
  
  if (currentStep.value === 'password') {
    currentStep.value = 'username'
    password.value = ''
    accountType.value = null
  } else if (currentStep.value === 'totp' || currentStep.value === 'totp-setup') {
    currentStep.value = 'password'
    password.value = ''
    pendingToken.value = ''
  }
}

// Auto-submit TOTP when 6 digits entered
watch(totpCode, (val) => {
  if (val.length === 6) {
    if (currentStep.value === 'totp') {
      handleTOTPVerify()
    } else if (currentStep.value === 'totp-setup') {
      handleTOTPSetup()
    }
  }
})

// Handle form submissions
function handleSubmit() {
  switch (currentStep.value) {
    case 'username':
      handleUsernameSubmit()
      break
    case 'password':
      handlePasswordSubmit()
      break
    case 'totp':
      handleTOTPVerify()
      break
    case 'totp-setup':
      handleTOTPSetup()
      break
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center relative overflow-hidden p-8">
    <!-- Animated background -->
    <div class="bg-pattern" />
    <div 
      class="fixed inset-0 pointer-events-none z-0 transition-[background] duration-600"
      :class="accentColor === 'primary' ? 'bg-gradient-primary' : 'bg-gradient-secondary'"
    />
    
    <!-- Decorative elements -->
    <div class="corner corner-tl" />
    <div class="corner corner-br" />

    <!-- Main login card -->
    <div class="relative z-10 w-full max-w-[400px] animate-fade-in">
      <!-- Logo -->
      <div class="text-center mb-10">
        <div 
          class="relative w-14 h-14 mx-auto mb-5 border-2 rounded-[14px] flex items-center justify-center transition-colors duration-300"
          :class="accentColor === 'primary' ? 'border-accent-primary' : 'border-accent-secondary'"
        >
          <div 
            class="w-5 h-5 rounded rotate-45 transition-colors duration-300"
            :class="accentColor === 'primary' ? 'bg-accent-primary' : 'bg-accent-secondary'"
          />
          <div 
            class="absolute -inset-1 border rounded-2xl opacity-30 transition-colors duration-300"
            :class="accentColor === 'primary' ? 'border-accent-primary' : 'border-accent-secondary'"
          />
        </div>
        <h1 class="font-display text-[2rem] font-semibold tracking-tight mb-1">digit-link</h1>
        <p class="text-sm text-text-secondary">Secure Tunnel Infrastructure</p>
      </div>

      <!-- Login card -->
      <div class="bg-bg-surface border border-border-subtle rounded-2xl overflow-hidden relative">
        <!-- Top accent line -->
        <div 
          class="absolute top-0 left-8 right-8 h-0.5 transition-colors duration-300"
          :class="accentColor === 'primary' ? 'card-accent-primary' : 'card-accent-secondary'"
        />
        
        <!-- Header with back button -->
        <div class="pt-6 px-6 flex items-start gap-3">
          <button 
            v-if="currentStep !== 'username'" 
            class="w-8 h-8 flex items-center justify-center border border-border-subtle rounded-xs bg-transparent text-text-secondary cursor-pointer transition-all duration-200 shrink-0 mt-0.5 hover:bg-bg-elevated hover:text-text-primary hover:border-border-accent disabled:opacity-50 disabled:cursor-not-allowed"
            @click="goBack"
            :disabled="loading"
          >
            <ArrowLeft class="w-4 h-4" />
          </button>
          <div class="flex-1 min-w-0">
            <h2 class="text-xl font-semibold mb-1">{{ stepTitle }}</h2>
            <p class="text-sm text-text-secondary">{{ stepDescription }}</p>
          </div>
          <!-- Account type badge -->
          <div 
            v-if="currentStep !== 'username' && accountType" 
            class="flex items-center gap-1.5 py-1.5 px-2.5 rounded-xs text-xs font-medium shrink-0"
            :class="accentColor === 'primary' ? 'bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary' : 'bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary'"
          >
            <Shield v-if="accountType === 'admin'" class="w-3.5 h-3.5" />
            <Building2 v-else class="w-3.5 h-3.5" />
            <span>{{ accountType === 'admin' ? 'Admin' : 'Org' }}</span>
          </div>
        </div>

        <!-- Form content -->
        <form @submit.prevent="handleSubmit" class="p-6">
          <!-- Error message -->
          <Transition name="shake">
            <div 
              v-if="error" 
              class="flex items-start gap-2.5 py-3.5 px-4 bg-[rgba(var(--accent-red-rgb),0.1)] border border-[rgba(var(--accent-red-rgb),0.3)] rounded-xs mb-5 text-sm text-accent-red"
            >
              <AlertCircle class="w-4 h-4 shrink-0" />
              <span>{{ error }}</span>
            </div>
          </Transition>

          <!-- Step: Username -->
          <Transition name="slide" mode="out-in">
            <div v-if="currentStep === 'username'" key="username" class="mb-6">
              <div class="mb-4">
                <label class="block text-xs font-medium uppercase tracking-wider text-text-secondary mb-2.5" for="username">Username</label>
                <div class="relative">
                  <User class="absolute left-4 top-1/2 -translate-y-1/2 w-[18px] h-[18px] text-text-muted pointer-events-none" />
                  <input
                    id="username"
                    v-model="username"
                    type="text"
                    class="w-full py-3.5 pl-11 pr-4 bg-bg-deep border border-border-subtle rounded-xs font-body text-[0.9375rem] text-text-primary transition-all duration-200 placeholder:text-text-muted focus:outline-none focus:border-accent-primary focus:shadow-[0_0_0_3px_rgba(var(--accent-primary-rgb),0.12)]"
                    placeholder="Enter your username"
                    autocomplete="username"
                    autofocus
                    :disabled="loading"
                  />
                </div>
              </div>
            </div>

            <!-- Step: Password -->
            <div v-else-if="currentStep === 'password'" key="password" class="mb-6">
              <!-- Username display -->
              <div class="flex items-center gap-3 py-3.5 px-4 bg-bg-deep border border-border-subtle rounded-xs mb-5">
                <div 
                  class="w-9 h-9 rounded-xs flex items-center justify-center font-semibold text-sm"
                  :class="accentColor === 'primary' ? 'bg-[rgba(var(--accent-primary-rgb),0.2)] text-accent-primary' : 'bg-[rgba(var(--accent-secondary-rgb),0.2)] text-accent-secondary'"
                >
                  {{ username.charAt(0).toUpperCase() }}
                </div>
                <span class="font-medium text-text-primary">{{ username }}</span>
              </div>

              <div class="mb-4">
                <label class="block text-xs font-medium uppercase tracking-wider text-text-secondary mb-2.5" for="password">Password</label>
                <div class="relative">
                  <Lock class="absolute left-4 top-1/2 -translate-y-1/2 w-[18px] h-[18px] text-text-muted pointer-events-none" />
                  <input
                    id="password"
                    v-model="password"
                    type="password"
                    class="w-full py-3.5 pl-11 pr-4 bg-bg-deep border border-border-subtle rounded-xs font-body text-[0.9375rem] text-text-primary transition-all duration-200 placeholder:text-text-muted focus:outline-none focus:border-accent-primary focus:shadow-[0_0_0_3px_rgba(var(--accent-primary-rgb),0.12)]"
                    placeholder="Enter your password"
                    autocomplete="current-password"
                    autofocus
                    :disabled="loading"
                  />
                </div>
              </div>
            </div>

            <!-- Step: TOTP Verify -->
            <div v-else-if="currentStep === 'totp'" key="totp" class="mb-6">
              <div class="flex justify-center mb-6">
                <div 
                  class="w-16 h-16 rounded-2xl flex items-center justify-center"
                  :class="accentColor === 'primary' ? 'bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary' : 'bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary'"
                >
                  <Key class="w-6 h-6" />
                </div>
              </div>

              <div class="mb-4">
                <label class="block text-xs font-medium uppercase tracking-wider text-text-secondary mb-2.5" for="totp-code">Authentication Code</label>
                <input
                  id="totp-code"
                  v-model="totpCode"
                  type="text"
                  inputmode="numeric"
                  pattern="[0-9]*"
                  maxlength="6"
                  class="w-full py-4 px-4 bg-bg-deep border border-border-subtle rounded-xs font-mono text-2xl text-center tracking-[0.5em] text-text-primary transition-all duration-200 placeholder:text-text-muted focus:outline-none focus:border-accent-primary focus:shadow-[0_0_0_3px_rgba(var(--accent-primary-rgb),0.12)]"
                  placeholder="000000"
                  autocomplete="one-time-code"
                  autofocus
                  :disabled="loading"
                />
                <p class="text-xs text-text-muted mt-2">Enter the 6-digit code from your authenticator app</p>
              </div>
            </div>

            <!-- Step: TOTP Setup -->
            <div v-else-if="currentStep === 'totp-setup'" key="totp-setup" class="mb-6">
              <div class="mb-6">
                <p class="text-sm text-text-secondary text-center mb-5 leading-relaxed">
                  Scan the QR code below with your authenticator app (Google Authenticator, Authy, etc.)
                </p>
                
                <!-- QR Code placeholder -->
                <div class="flex justify-center mb-5">
                  <div 
                    class="w-[180px] h-[180px] bg-bg-deep border-2 rounded-xs flex items-center justify-center overflow-hidden"
                    :class="accentColor === 'primary' ? 'border-[rgba(var(--accent-primary-rgb),0.3)]' : 'border-[rgba(var(--accent-secondary-rgb),0.3)]'"
                  >
                    <img 
                      v-if="totpUrl"
                      :src="`https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=${encodeURIComponent(totpUrl)}`"
                      alt="TOTP QR Code"
                      class="w-[150px] h-[150px] rounded"
                    />
                    <Loader2 v-else class="w-8 h-8 animate-spin" />
                  </div>
                </div>

                <!-- Manual entry secret -->
                <div v-if="totpSecret" class="text-center p-4 bg-bg-deep border border-dashed border-border-accent rounded-xs">
                  <p class="text-xs text-text-muted mb-2">Or enter this code manually:</p>
                  <code class="font-mono text-[0.8125rem] text-accent-amber tracking-wide break-all">{{ totpSecret }}</code>
                </div>
              </div>

              <div class="mb-4">
                <label class="block text-xs font-medium uppercase tracking-wider text-text-secondary mb-2.5" for="setup-code">Verification Code</label>
                <input
                  id="setup-code"
                  v-model="totpCode"
                  type="text"
                  inputmode="numeric"
                  pattern="[0-9]*"
                  maxlength="6"
                  class="w-full py-4 px-4 bg-bg-deep border border-border-subtle rounded-xs font-mono text-2xl text-center tracking-[0.5em] text-text-primary transition-all duration-200 placeholder:text-text-muted focus:outline-none focus:border-accent-primary focus:shadow-[0_0_0_3px_rgba(var(--accent-primary-rgb),0.12)]"
                  placeholder="000000"
                  autocomplete="one-time-code"
                  :disabled="loading"
                />
                <p class="text-xs text-text-muted mt-2">Enter the code shown in your authenticator to complete setup</p>
              </div>
            </div>
          </Transition>

          <!-- Submit button -->
          <button
            type="submit"
            class="w-full py-[0.9375rem] px-6 border-none rounded-xs font-body text-[0.9375rem] font-medium cursor-pointer transition-all duration-200 relative disabled:opacity-70 disabled:cursor-not-allowed disabled:transform-none"
            :class="accentColor === 'primary' 
              ? 'bg-accent-primary text-bg-deep hover:bg-accent-primary-dim hover:-translate-y-px' 
              : 'bg-accent-secondary text-bg-deep hover:bg-accent-secondary-dim hover:-translate-y-px'"
            :disabled="loading"
          >
            <span class="flex items-center justify-center gap-2">
              <template v-if="loading">
                <Loader2 class="w-4 h-4 animate-spin" />
                <span>Please wait...</span>
              </template>
              <template v-else-if="currentStep === 'username'">
                <span>Continue</span>
                <ArrowRight class="w-4 h-4" />
              </template>
              <template v-else-if="currentStep === 'password'">
                <span>{{ requiresTOTP ? 'Continue' : 'Sign In' }}</span>
                <ArrowRight class="w-4 h-4" />
              </template>
              <template v-else-if="currentStep === 'totp'">
                <span>Verify</span>
                <Check class="w-4 h-4" />
              </template>
              <template v-else>
                <span>Complete Setup</span>
                <Check class="w-4 h-4" />
              </template>
            </span>
          </button>
        </form>
      </div>

      <!-- Footer -->
      <div class="text-center mt-6 text-xs text-text-muted flex flex-col items-center gap-4">
        <ThemeSwitcher />
        <p>
          Secure infrastructure by 
          <a href="https://digit.zone" target="_blank" rel="noopener" class="text-accent-primary no-underline transition-colors duration-200 hover:text-text-primary hover:underline">digit.zone</a>
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Background pattern - needs pseudo-element */
.bg-pattern {
	@reference "../style.css";
  @apply fixed inset-0 pointer-events-none z-0 opacity-[0.12];
  background-image: 
    linear-gradient(var(--border-subtle) 1px, transparent 1px),
    linear-gradient(90deg, var(--border-subtle) 1px, transparent 1px);
  background-size: 60px 60px;
}

/* Background gradients */
.bg-gradient-primary {
  background: radial-gradient(ellipse at 30% 20%, rgba(var(--accent-primary-rgb), 0.08) 0%, transparent 50%),
              radial-gradient(ellipse at 70% 80%, rgba(var(--accent-primary-rgb), 0.05) 0%, transparent 50%);
}

.bg-gradient-secondary {
  background: radial-gradient(ellipse at 30% 20%, rgba(var(--accent-secondary-rgb), 0.08) 0%, transparent 50%),
              radial-gradient(ellipse at 70% 80%, rgba(var(--accent-secondary-rgb), 0.05) 0%, transparent 50%);
}

/* Card accent gradients */
.card-accent-primary {
  background: linear-gradient(90deg, transparent, var(--accent-primary), transparent);
}

.card-accent-secondary {
  background: linear-gradient(90deg, transparent, var(--accent-secondary), transparent);
}
</style>
