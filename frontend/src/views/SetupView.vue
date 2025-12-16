<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import { ThemeSwitcher } from '@/components/shared'
import { User, Lock, Key, Eye, EyeOff, Check, AlertCircle, Loader2, ArrowRight, Shield } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()

// Current step in wizard
const currentStep = ref(1) // 1 = credentials, 2 = TOTP setup, 3 = complete

// Form state
const username = ref('admin')
const password = ref('')
const confirmPassword = ref('')
const showPassword = ref(false)
const autoWhitelist = ref(true)
const totpCode = ref('')

// TOTP setup state
const pendingToken = ref('')
const totpSecret = ref('')
const totpUrl = ref('')

// UI state
const loading = ref(false)
const error = ref('')

// Computed
const stepDots = computed(() => [1, 2, 3].map(step => ({
  step,
  isActive: step === currentStep.value,
  isCompleted: step < currentStep.value
})))

const passwordsMatch = computed(() => {
  return password.value === confirmPassword.value
})

const passwordStrength = computed(() => {
  const p = password.value
  if (p.length < 8) return { level: 0, text: 'Too short', color: 'red' }
  
  let score = 0
  if (p.length >= 12) score++
  if (/[a-z]/.test(p) && /[A-Z]/.test(p)) score++
  if (/\d/.test(p)) score++
  if (/[^a-zA-Z0-9]/.test(p)) score++
  
  if (score <= 1) return { level: 1, text: 'Weak', color: 'red' }
  if (score === 2) return { level: 2, text: 'Fair', color: 'amber' }
  if (score === 3) return { level: 3, text: 'Good', color: 'secondary' }
  return { level: 4, text: 'Strong', color: 'secondary' }
})

const canProceedStep1 = computed(() => {
  return username.value.trim().length > 0 && 
         password.value.length >= 8 && 
         passwordsMatch.value
})

// Check setup status on mount
onMounted(async () => {
  await checkSetupStatus()
})

async function checkSetupStatus() {
  try {
    const response = await fetch('/setup/status')
    const data = await response.json()
    
    if (!data.needsSetup) {
      router.push({ name: 'login' })
    }
  } catch (err) {
    console.error('Failed to check setup status:', err)
  }
}

// Step 1: Create account with credentials
async function handleCreateAccount() {
  if (!canProceedStep1.value) return

  loading.value = true
  error.value = ''

  try {
    const response = await fetch('/setup/init', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: username.value.trim(),
        password: password.value,
        autoWhitelist: autoWhitelist.value
      })
    })

    const data = await response.json()

    if (!response.ok || !data.success) {
      throw new Error(data.error || 'Setup failed')
    }

    pendingToken.value = data.pendingToken

    // Fetch TOTP setup info
    await fetchTOTPSetup()
    currentStep.value = 2
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Setup failed'
  } finally {
    loading.value = false
  }
}

// Fetch TOTP secret for setup
async function fetchTOTPSetup() {
  const response = await fetch('/setup/totp', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pendingToken: pendingToken.value })
  })

  const data = await response.json()
  
  if (data.success) {
    totpSecret.value = data.secret
    totpUrl.value = data.url
  } else {
    error.value = data.error || 'Failed to initialize TOTP setup'
  }
}

// Step 2: Complete TOTP setup
async function handleCompleteTOTP() {
  if (!totpCode.value || totpCode.value.length !== 6) {
    error.value = 'Please enter a valid 6-digit code'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await fetch('/setup/complete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        pendingToken: pendingToken.value,
        code: totpCode.value
      })
    })

    const data = await response.json()

    if (!response.ok || !data.success) {
      throw new Error(data.error || 'Failed to verify code')
    }

    // Store the JWT token
    authStore.setToken(data.token, 'admin')
    currentStep.value = 3
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Verification failed'
    totpCode.value = ''
  } finally {
    loading.value = false
  }
}

function goToDashboard() {
  router.push({ name: 'dashboard' })
}

function copySecret() {
  navigator.clipboard.writeText(totpSecret.value)
  showToast()
}

const showingToast = ref(false)
function showToast() {
  showingToast.value = true
  setTimeout(() => { showingToast.value = false }, 2000)
}

const strengthColorClass = computed(() => {
  const colors = {
    red: 'bg-accent-red',
    amber: 'bg-accent-amber',
    secondary: 'bg-accent-secondary'
  }
  return colors[passwordStrength.value.color as keyof typeof colors]
})

const strengthTextClass = computed(() => {
  const colors = {
    red: 'text-accent-red',
    amber: 'text-accent-amber',
    secondary: 'text-accent-secondary'
  }
  return colors[passwordStrength.value.color as keyof typeof colors]
})
</script>

<template>
  <div class="min-h-screen flex items-center justify-center relative overflow-hidden grid-background grid-background-animated p-8">
    <!-- Decorative corners -->
    <div class="corner corner-tl" />
    <div class="corner corner-br" />

    <div class="relative w-full max-w-[520px] animate-fade-in">
      <!-- Step indicator -->
      <div class="flex justify-center gap-2 mb-10">
        <div
          v-for="dot in stepDots"
          :key="dot.step"
          class="w-2 h-2 rounded-full transition-all duration-300"
          :class="{
            'bg-accent-primary shadow-[0_0_12px_rgba(var(--accent-primary-rgb),0.4)]': dot.isActive,
            'bg-accent-secondary': dot.isCompleted,
            'bg-border-accent': !dot.isActive && !dot.isCompleted
          }"
        />
      </div>

      <!-- Brand -->
      <div class="text-center mb-8">
        <div class="relative w-16 h-16 mx-auto mb-6 border-2 border-accent-primary rounded-2xl flex items-center justify-center animate-icon-float">
          <div class="w-6 h-6 bg-accent-primary rounded-[0.375rem] rotate-45" />
          <div class="absolute -inset-1.5 border border-accent-primary rounded-[1.25rem] opacity-30 animate-[icon-ring_2s_ease-out_infinite]" />
        </div>
        
        <h1 class="font-display text-4xl font-semibold tracking-tight mb-2">
          digit-link
        </h1>
        <p class="text-text-secondary">Tunnel Administration</p>
      </div>

      <!-- Setup Card -->
      <div class="relative bg-bg-surface border border-border-subtle rounded-2xl p-10 overflow-hidden">
        <!-- Gradient line -->
        <div class="absolute top-0 left-8 right-8 h-0.5 bg-gradient-to-r from-transparent via-accent-primary to-transparent" />

        <!-- Error message -->
        <Transition name="shake">
          <div 
            v-if="error" 
            class="flex items-start gap-2.5 py-3.5 px-4 bg-[rgba(var(--accent-red-rgb),0.1)] border border-[rgba(var(--accent-red-rgb),0.3)] rounded-[10px] mb-6 text-sm text-accent-red"
          >
            <AlertCircle class="w-4 h-4 shrink-0" />
            <span>{{ error }}</span>
          </div>
        </Transition>

        <!-- Step 1: Credentials -->
        <Transition name="slide" mode="out-in">
          <div v-if="currentStep === 1" key="credentials" class="animate-fade-in-slide">
            <div class="inline-flex items-center gap-2 py-2 px-4 rounded-full text-xs font-medium uppercase tracking-widest mb-6 bg-[rgba(var(--accent-primary-rgb),0.1)] border border-[rgba(var(--accent-primary-rgb),0.3)] text-accent-primary">
              <Shield class="w-3.5 h-3.5" />
              Administrator Setup
            </div>

            <h2 class="font-display text-2xl font-semibold mb-3">
              Create Admin Account
            </h2>
            <p class="text-sm text-text-secondary leading-relaxed mb-8">
              Set up your administrator credentials. You'll use these to sign in to the management dashboard.
            </p>

            <form @submit.prevent="handleCreateAccount" class="space-y-5">
              <!-- Username -->
              <div>
                <label class="form-label" for="username">Username</label>
                <div class="relative">
                  <User class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
                  <input
                    id="username"
                    v-model="username"
                    type="text"
                    class="form-input pl-10"
                    placeholder="Enter admin username"
                    autocomplete="username"
                  />
                </div>
              </div>

              <!-- Password -->
              <div>
                <label class="form-label" for="password">Password</label>
                <div class="relative">
                  <Lock class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
                  <input
                    id="password"
                    v-model="password"
                    :type="showPassword ? 'text' : 'password'"
                    class="form-input pl-10 pr-10"
                    placeholder="Enter a strong password"
                    autocomplete="new-password"
                  />
                  <button
                    type="button"
                    class="absolute right-3 top-1/2 -translate-y-1/2 text-text-muted hover:text-text-secondary transition-colors"
                    @click="showPassword = !showPassword"
                  >
                    <EyeOff v-if="showPassword" class="w-4 h-4" />
                    <Eye v-else class="w-4 h-4" />
                  </button>
                </div>
                <!-- Password strength indicator -->
                <div v-if="password.length > 0" class="mt-2">
                  <div class="flex items-center gap-2">
                    <div class="flex-1 h-1 bg-bg-deep rounded overflow-hidden">
                      <div 
                        class="h-full transition-all duration-300"
                        :class="strengthColorClass"
                        :style="{ width: `${passwordStrength.level * 25}%` }"
                      />
                    </div>
                    <span class="text-xs font-medium" :class="strengthTextClass">
                      {{ passwordStrength.text }}
                    </span>
                  </div>
                </div>
              </div>

              <!-- Confirm Password -->
              <div>
                <label class="form-label" for="confirm-password">Confirm Password</label>
                <div class="relative">
                  <Lock class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
                  <input
                    id="confirm-password"
                    v-model="confirmPassword"
                    :type="showPassword ? 'text' : 'password'"
                    class="form-input pl-10"
                    placeholder="Confirm your password"
                    autocomplete="new-password"
                  />
                </div>
                <p 
                  v-if="confirmPassword.length > 0 && !passwordsMatch"
                  class="text-xs text-accent-red mt-1"
                >
                  Passwords do not match
                </p>
              </div>

              <!-- Auto-whitelist -->
              <label class="flex items-start gap-3 p-4 bg-bg-deep border border-border-subtle rounded-xs cursor-pointer transition-colors hover:border-border-accent">
                <input v-model="autoWhitelist" type="checkbox" class="hidden" />
                <div 
                  class="w-5 h-5 border-2 rounded flex items-center justify-center shrink-0 mt-0.5 transition-all duration-200"
                  :class="autoWhitelist ? 'bg-accent-primary border-accent-primary' : 'border-border-accent'"
                >
                  <Check v-if="autoWhitelist" class="w-3 h-3 text-bg-deep" />
                </div>
                <div class="flex-1">
                  <strong class="block text-sm mb-1">Auto-whitelist my current IP</strong>
                  <span class="text-xs text-text-muted">Allow tunnel connections from your current location</span>
                </div>
              </label>

              <button
                type="submit"
                class="btn btn-primary w-full"
                :disabled="loading || !canProceedStep1"
              >
                <span class="flex items-center justify-center gap-2">
                  <template v-if="loading">
                    <Loader2 class="w-4 h-4 animate-spin" />
                    Creating account...
                  </template>
                  <template v-else>
                    Continue to Security Setup
                    <ArrowRight class="w-4 h-4" />
                  </template>
                </span>
              </button>
            </form>
          </div>

          <!-- Step 2: TOTP Setup -->
          <div v-else-if="currentStep === 2" key="totp" class="animate-fade-in-slide">
            <div class="inline-flex items-center gap-2 py-2 px-4 rounded-full text-xs font-medium uppercase tracking-widest mb-6 bg-[rgba(var(--accent-secondary-rgb),0.1)] border border-[rgba(var(--accent-secondary-rgb),0.3)] text-accent-secondary">
              <Key class="w-3.5 h-3.5" />
              Two-Factor Authentication
            </div>

            <h2 class="font-display text-2xl font-semibold mb-3">
              Setup Authenticator
            </h2>
            <p class="text-sm text-text-secondary leading-relaxed mb-6">
              Scan the QR code with your authenticator app (Google Authenticator, Authy, etc.) to enable two-factor authentication.
            </p>

            <!-- QR Code -->
            <div class="flex justify-center mb-6">
              <div class="w-[180px] h-[180px] bg-white rounded-xs flex items-center justify-center overflow-hidden">
                <img 
                  v-if="totpUrl"
                  :src="`https://api.qrserver.com/v1/create-qr-code/?size=160x160&data=${encodeURIComponent(totpUrl)}`"
                  alt="TOTP QR Code"
                  class="w-[160px] h-[160px]"
                />
                <Loader2 v-else class="w-8 h-8 text-text-muted animate-spin" />
              </div>
            </div>

            <!-- Manual entry secret -->
            <div v-if="totpSecret" class="mb-6">
              <p class="text-xs text-text-muted text-center mb-2">Can't scan? Enter this code manually:</p>
              <div class="flex items-center gap-2 p-3 bg-bg-deep border border-dashed border-border-accent rounded-xs">
                <code class="flex-1 font-mono text-[0.8125rem] text-accent-amber text-center tracking-wider break-all">
                  {{ totpSecret }}
                </code>
                <button 
                  class="p-1.5 text-text-muted hover:text-text-secondary transition-colors"
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
            <form @submit.prevent="handleCompleteTOTP">
              <div class="mb-6">
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
                  autofocus
                />
                <p class="form-hint text-center">Enter the 6-digit code from your authenticator app</p>
              </div>

              <button
                type="submit"
                class="btn btn-success w-full"
                :disabled="loading || totpCode.length !== 6"
              >
                <span class="flex items-center justify-center gap-2">
                  <template v-if="loading">
                    <Loader2 class="w-4 h-4 animate-spin" />
                    Verifying...
                  </template>
                  <template v-else>
                    Complete Setup
                    <Check class="w-4 h-4" />
                  </template>
                </span>
              </button>
            </form>
          </div>

          <!-- Step 3: Complete -->
          <div v-else-if="currentStep === 3" key="complete" class="animate-fade-in-slide text-center">
            <div class="w-20 h-20 mx-auto mb-6 bg-[rgba(var(--accent-secondary-rgb),0.15)] border-2 border-accent-secondary rounded-full flex items-center justify-center animate-success-pop">
              <Check class="w-10 h-10 text-accent-secondary" />
            </div>

            <h2 class="font-display text-2xl font-semibold mb-3">
              Setup Complete!
            </h2>
            <p class="text-sm text-text-secondary leading-relaxed mb-6">
              Your digit-link server is ready. Your admin account is secured with two-factor authentication.
            </p>

            <div class="bg-bg-deep rounded-xs p-5 mb-6 text-left">
              <p class="text-[0.8rem] text-text-secondary mb-4">Quick start guide:</p>
              <ol class="text-[0.8rem] text-text-muted pl-5 leading-loose list-decimal">
                <li>Add IP addresses to the whitelist</li>
                <li>Create organizations and applications</li>
                <li>Generate API keys for tunnel clients</li>
              </ol>
            </div>

            <button class="btn btn-primary w-full" @click="goToDashboard">
              <span class="flex items-center justify-center gap-2">
                Open Dashboard
                <ArrowRight class="w-4 h-4" />
              </span>
            </button>
          </div>
        </Transition>
      </div>

      <!-- Footer -->
      <div class="text-center mt-8 text-xs text-text-muted flex flex-col items-center gap-4">
        <ThemeSwitcher />
        <p>
          Secure tunnel infrastructure by 
          <a 
            href="https://digit.zone" 
            class="text-accent-primary hover:underline"
            target="_blank"
          >
            digit.zone
          </a>
        </p>
      </div>
    </div>

    <!-- Toast -->
    <div 
      class="toast"
      :class="{ visible: showingToast }"
    >
      Secret copied to clipboard
    </div>
  </div>
</template>
