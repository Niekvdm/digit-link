<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
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
  return accountType.value === 'admin' ? 'copper' : 'emerald'
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
      completeLogin(data.token, data.accountType, data.orgId)
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

    completeLogin(data.token, data.accountType, data.orgId)
  } catch {
    error.value = 'Connection error. Please try again.'
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

    completeLogin(data.token, data.accountType, data.orgId)
  } catch {
    error.value = 'Connection error. Please try again.'
    loading.value = false
  }
}

function completeLogin(token: string, type: string, orgId?: string) {
  const userType = type === 'admin' ? 'admin' : 'org'
  authStore.setToken(token, userType, orgId)
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
  <div class="login-container">
    <!-- Animated background -->
    <div class="bg-pattern" />
    <div class="bg-gradient" :class="`bg-gradient--${accentColor}`" />
    
    <!-- Decorative elements -->
    <div class="corner corner-tl" />
    <div class="corner corner-br" />

    <!-- Main login card -->
    <div class="login-wrapper">
      <!-- Logo -->
      <div class="logo-container">
        <div class="logo" :class="`logo--${accentColor}`">
          <div class="logo-inner" />
          <div class="logo-ring" />
        </div>
        <h1 class="brand-title">digit-link</h1>
        <p class="brand-subtitle">Secure Tunnel Infrastructure</p>
      </div>

      <!-- Login card -->
      <div class="login-card">
        <!-- Top accent line -->
        <div class="card-accent" :class="`card-accent--${accentColor}`" />
        
        <!-- Header with back button -->
        <div class="card-header">
          <button 
            v-if="currentStep !== 'username'" 
            class="back-btn"
            @click="goBack"
            :disabled="loading"
          >
            <ArrowLeft class="w-4 h-4" />
          </button>
          <div class="header-text">
            <h2 class="step-title">{{ stepTitle }}</h2>
            <p class="step-desc">{{ stepDescription }}</p>
          </div>
          <!-- Account type badge -->
          <div 
            v-if="currentStep !== 'username' && accountType" 
            class="account-badge"
            :class="`account-badge--${accentColor}`"
          >
            <Shield v-if="accountType === 'admin'" class="w-3.5 h-3.5" />
            <Building2 v-else class="w-3.5 h-3.5" />
            <span>{{ accountType === 'admin' ? 'Admin' : 'Org' }}</span>
          </div>
        </div>

        <!-- Form content -->
        <form @submit.prevent="handleSubmit" class="card-body">
          <!-- Error message -->
          <Transition name="shake">
            <div v-if="error" class="error-box">
              <AlertCircle class="w-4 h-4 flex-shrink-0" />
              <span>{{ error }}</span>
            </div>
          </Transition>

          <!-- Step: Username -->
          <Transition name="slide" mode="out-in">
            <div v-if="currentStep === 'username'" key="username" class="step-content">
              <div class="input-group">
                <label class="input-label" for="username">Username</label>
                <div class="input-wrapper">
                  <User class="input-icon" />
                  <input
                    id="username"
                    v-model="username"
                    type="text"
                    class="form-input form-input--icon"
                    placeholder="Enter your username"
                    autocomplete="username"
                    autofocus
                    :disabled="loading"
                  />
                </div>
              </div>
            </div>

            <!-- Step: Password -->
            <div v-else-if="currentStep === 'password'" key="password" class="step-content">
              <!-- Username display -->
              <div class="user-info">
                <div class="user-avatar" :class="`user-avatar--${accentColor}`">
                  {{ username.charAt(0).toUpperCase() }}
                </div>
                <span class="user-name">{{ username }}</span>
              </div>

              <div class="input-group">
                <label class="input-label" for="password">Password</label>
                <div class="input-wrapper">
                  <Lock class="input-icon" />
                  <input
                    id="password"
                    v-model="password"
                    type="password"
                    class="form-input form-input--icon"
                    placeholder="Enter your password"
                    autocomplete="current-password"
                    autofocus
                    :disabled="loading"
                  />
                </div>
              </div>
            </div>

            <!-- Step: TOTP Verify -->
            <div v-else-if="currentStep === 'totp'" key="totp" class="step-content">
              <div class="totp-icon-wrapper">
                <div class="totp-icon" :class="`totp-icon--${accentColor}`">
                  <Key class="w-6 h-6" />
                </div>
              </div>

              <div class="input-group">
                <label class="input-label" for="totp-code">Authentication Code</label>
                <div class="input-wrapper">
                  <input
                    id="totp-code"
                    v-model="totpCode"
                    type="text"
                    inputmode="numeric"
                    pattern="[0-9]*"
                    maxlength="6"
                    class="form-input form-input--totp"
                    placeholder="000000"
                    autocomplete="one-time-code"
                    autofocus
                    :disabled="loading"
                  />
                </div>
                <p class="input-hint">Enter the 6-digit code from your authenticator app</p>
              </div>
            </div>

            <!-- Step: TOTP Setup -->
            <div v-else-if="currentStep === 'totp-setup'" key="totp-setup" class="step-content">
              <div class="setup-instructions">
                <p class="setup-text">
                  Scan the QR code below with your authenticator app (Google Authenticator, Authy, etc.)
                </p>
                
                <!-- QR Code placeholder - in production, use a QR library -->
                <div class="qr-container">
                  <div class="qr-placeholder" :class="`qr-placeholder--${accentColor}`">
                    <img 
                      v-if="totpUrl"
                      :src="`https://api.qrserver.com/v1/create-qr-code/?size=150x150&data=${encodeURIComponent(totpUrl)}`"
                      alt="TOTP QR Code"
                      class="qr-image"
                    />
                    <Loader2 v-else class="w-8 h-8 animate-spin" />
                  </div>
                </div>

                <!-- Manual entry secret -->
                <div v-if="totpSecret" class="secret-box">
                  <p class="secret-label">Or enter this code manually:</p>
                  <code class="secret-code">{{ totpSecret }}</code>
                </div>
              </div>

              <div class="input-group">
                <label class="input-label" for="setup-code">Verification Code</label>
                <div class="input-wrapper">
                  <input
                    id="setup-code"
                    v-model="totpCode"
                    type="text"
                    inputmode="numeric"
                    pattern="[0-9]*"
                    maxlength="6"
                    class="form-input form-input--totp"
                    placeholder="000000"
                    autocomplete="one-time-code"
                    :disabled="loading"
                  />
                </div>
                <p class="input-hint">Enter the code shown in your authenticator to complete setup</p>
              </div>
            </div>
          </Transition>

          <!-- Submit button -->
          <button
            type="submit"
            class="submit-btn"
            :class="[`submit-btn--${accentColor}`, { 'submit-btn--loading': loading }]"
            :disabled="loading"
          >
            <span class="btn-content">
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
      <div class="login-footer">
        <p>
          Secure infrastructure by 
          <a href="https://digit.zone" target="_blank" rel="noopener">digit.zone</a>
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Container */
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
  padding: 2rem;
}

/* Background effects */
.bg-pattern {
  position: fixed;
  inset: 0;
  background-image: 
    linear-gradient(var(--border-subtle) 1px, transparent 1px),
    linear-gradient(90deg, var(--border-subtle) 1px, transparent 1px);
  background-size: 60px 60px;
  opacity: 0.12;
  pointer-events: none;
  z-index: 0;
}

.bg-gradient {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
  transition: background 0.6s ease;
}

.bg-gradient--copper {
  background: radial-gradient(ellipse at 30% 20%, rgba(201, 149, 108, 0.08) 0%, transparent 50%),
              radial-gradient(ellipse at 70% 80%, rgba(201, 149, 108, 0.05) 0%, transparent 50%);
}

.bg-gradient--emerald {
  background: radial-gradient(ellipse at 30% 20%, rgba(74, 159, 126, 0.08) 0%, transparent 50%),
              radial-gradient(ellipse at 70% 80%, rgba(74, 159, 126, 0.05) 0%, transparent 50%);
}

/* Decorative corners */
.corner {
  position: fixed;
  width: 100px;
  height: 100px;
  pointer-events: none;
  z-index: 1;
}

.corner::before,
.corner::after {
  content: '';
  position: absolute;
  background: var(--accent-copper);
  opacity: 0.25;
  transition: background 0.3s ease;
}

.corner-tl { top: 2rem; left: 2rem; }
.corner-tl::before { top: 0; left: 0; width: 50px; height: 2px; }
.corner-tl::after { top: 0; left: 0; width: 2px; height: 50px; }

.corner-br { bottom: 2rem; right: 2rem; }
.corner-br::before { bottom: 0; right: 0; width: 50px; height: 2px; }
.corner-br::after { bottom: 0; right: 0; width: 2px; height: 50px; }

/* Main wrapper */
.login-wrapper {
  position: relative;
  z-index: 10;
  width: 100%;
  max-width: 400px;
  animation: fadeIn 0.6s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}

/* Logo */
.logo-container {
  text-align: center;
  margin-bottom: 2.5rem;
}

.logo {
  position: relative;
  width: 56px;
  height: 56px;
  margin: 0 auto 1.25rem;
  border: 2px solid var(--accent-copper);
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: border-color 0.3s ease;
}

.logo--emerald {
  border-color: var(--accent-emerald);
}

.logo-inner {
  width: 20px;
  height: 20px;
  background: var(--accent-copper);
  border-radius: 4px;
  transform: rotate(45deg);
  transition: background 0.3s ease;
}

.logo--emerald .logo-inner {
  background: var(--accent-emerald);
}

.logo-ring {
  position: absolute;
  inset: -4px;
  border: 1px solid var(--accent-copper);
  border-radius: 16px;
  opacity: 0.3;
  transition: border-color 0.3s ease;
}

.logo--emerald .logo-ring {
  border-color: var(--accent-emerald);
}

.brand-title {
  font-family: var(--font-display);
  font-size: 2rem;
  font-weight: 600;
  letter-spacing: -0.02em;
  margin-bottom: 0.25rem;
}

.brand-subtitle {
  font-size: 0.875rem;
  color: var(--text-secondary);
}

/* Login card */
.login-card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 16px;
  overflow: hidden;
  position: relative;
}

.card-accent {
  position: absolute;
  top: 0;
  left: 2rem;
  right: 2rem;
  height: 2px;
  background: linear-gradient(90deg, transparent, var(--accent-copper), transparent);
  transition: background 0.3s ease;
}

.card-accent--emerald {
  background: linear-gradient(90deg, transparent, var(--accent-emerald), transparent);
}

/* Card header */
.card-header {
  padding: 1.5rem 1.5rem 0;
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
}

.back-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  flex-shrink: 0;
  margin-top: 2px;
}

.back-btn:hover:not(:disabled) {
  background: var(--bg-elevated);
  color: var(--text-primary);
  border-color: var(--border-accent);
}

.back-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.header-text {
  flex: 1;
  min-width: 0;
}

.step-title {
  font-size: 1.25rem;
  font-weight: 600;
  margin-bottom: 0.25rem;
}

.step-desc {
  font-size: 0.875rem;
  color: var(--text-secondary);
}

.account-badge {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.625rem;
  border-radius: 6px;
  font-size: 0.75rem;
  font-weight: 500;
  flex-shrink: 0;
}

.account-badge--copper {
  background: rgba(201, 149, 108, 0.15);
  color: var(--accent-copper);
}

.account-badge--emerald {
  background: rgba(74, 159, 126, 0.15);
  color: var(--accent-emerald);
}

/* Card body */
.card-body {
  padding: 1.5rem;
}

/* Error box */
.error-box {
  display: flex;
  align-items: flex-start;
  gap: 0.625rem;
  padding: 0.875rem 1rem;
  background: rgba(201, 108, 108, 0.1);
  border: 1px solid rgba(201, 108, 108, 0.3);
  border-radius: 10px;
  margin-bottom: 1.25rem;
  font-size: 0.875rem;
  color: var(--accent-red);
}

/* Step content */
.step-content {
  margin-bottom: 1.5rem;
}

/* Input groups */
.input-group {
  margin-bottom: 1rem;
}

.input-group:last-child {
  margin-bottom: 0;
}

.input-label {
  display: block;
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-secondary);
  margin-bottom: 0.625rem;
}

.input-wrapper {
  position: relative;
}

.input-icon {
  position: absolute;
  left: 1rem;
  top: 50%;
  transform: translateY(-50%);
  width: 18px;
  height: 18px;
  color: var(--text-muted);
  pointer-events: none;
}

.form-input {
  width: 100%;
  padding: 0.875rem 1rem;
  background: var(--bg-deep);
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  font-family: var(--font-body);
  font-size: 0.9375rem;
  color: var(--text-primary);
  transition: all 0.2s ease;
}

.form-input::placeholder {
  color: var(--text-muted);
}

.form-input:focus {
  outline: none;
  border-color: var(--accent-copper);
  box-shadow: 0 0 0 3px rgba(201, 149, 108, 0.12);
}

.form-input--icon {
  padding-left: 2.75rem;
}

.form-input--totp {
  text-align: center;
  font-family: var(--font-mono);
  font-size: 1.5rem;
  letter-spacing: 0.5em;
  padding: 1rem;
}

.input-hint {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-top: 0.5rem;
}

/* User info display */
.user-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.875rem 1rem;
  background: var(--bg-deep);
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  margin-bottom: 1.25rem;
}

.user-avatar {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 0.875rem;
}

.user-avatar--copper {
  background: rgba(201, 149, 108, 0.2);
  color: var(--accent-copper);
}

.user-avatar--emerald {
  background: rgba(74, 159, 126, 0.2);
  color: var(--accent-emerald);
}

.user-name {
  font-weight: 500;
  color: var(--text-primary);
}

/* TOTP icon */
.totp-icon-wrapper {
  display: flex;
  justify-content: center;
  margin-bottom: 1.5rem;
}

.totp-icon {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.totp-icon--copper {
  background: rgba(201, 149, 108, 0.15);
  color: var(--accent-copper);
}

.totp-icon--emerald {
  background: rgba(74, 159, 126, 0.15);
  color: var(--accent-emerald);
}

/* TOTP Setup */
.setup-instructions {
  margin-bottom: 1.5rem;
}

.setup-text {
  font-size: 0.875rem;
  color: var(--text-secondary);
  text-align: center;
  margin-bottom: 1.25rem;
  line-height: 1.5;
}

.qr-container {
  display: flex;
  justify-content: center;
  margin-bottom: 1.25rem;
}

.qr-placeholder {
  width: 180px;
  height: 180px;
  background: var(--bg-deep);
  border: 2px solid var(--border-subtle);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.qr-placeholder--copper {
  border-color: rgba(201, 149, 108, 0.3);
}

.qr-placeholder--emerald {
  border-color: rgba(74, 159, 126, 0.3);
}

.qr-image {
  width: 150px;
  height: 150px;
  border-radius: 4px;
}

.secret-box {
  text-align: center;
  padding: 1rem;
  background: var(--bg-deep);
  border: 1px dashed var(--border-accent);
  border-radius: 8px;
}

.secret-label {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin-bottom: 0.5rem;
}

.secret-code {
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  color: var(--accent-amber);
  letter-spacing: 0.05em;
  word-break: break-all;
}

/* Submit button */
.submit-btn {
  width: 100%;
  padding: 0.9375rem 1.5rem;
  border: none;
  border-radius: 10px;
  font-family: var(--font-body);
  font-size: 0.9375rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
}

.submit-btn--copper {
  background: var(--accent-copper);
  color: var(--bg-deep);
}

.submit-btn--copper:hover:not(:disabled) {
  background: var(--accent-copper-dim);
  transform: translateY(-1px);
}

.submit-btn--emerald {
  background: var(--accent-emerald);
  color: var(--bg-deep);
}

.submit-btn--emerald:hover:not(:disabled) {
  background: #3d8a6b;
  transform: translateY(-1px);
}

.submit-btn:disabled {
  opacity: 0.7;
  cursor: not-allowed;
  transform: none !important;
}

.btn-content {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
}

/* Footer */
.login-footer {
  text-align: center;
  margin-top: 1.5rem;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.login-footer a {
  color: var(--accent-copper);
  text-decoration: none;
  transition: color 0.2s ease;
}

.login-footer a:hover {
  color: var(--text-primary);
  text-decoration: underline;
}

/* Transitions */
.slide-enter-active,
.slide-leave-active {
  transition: all 0.25s ease;
}

.slide-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.slide-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

.shake-enter-active {
  animation: shake 0.4s ease;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-6px); }
  75% { transform: translateX(6px); }
}

/* Utility */
.animate-spin {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>

