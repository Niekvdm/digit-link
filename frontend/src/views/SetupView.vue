<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import type { SetupStatusResponse, SetupInitResponse } from '@/types/api'

const router = useRouter()
const authStore = useAuthStore()

const currentStep = ref(1)
const username = ref('admin')
const autoWhitelist = ref(true)
const generatedToken = ref('')
const tokenSaved = ref(false)
const error = ref('')
const loading = ref(false)

const stepDots = computed(() => [1, 2, 3].map(step => ({
  step,
  isActive: step === currentStep.value,
  isCompleted: step < currentStep.value
})))

onMounted(async () => {
  await checkSetupStatus()
})

async function checkSetupStatus() {
  try {
    const response = await fetch('/setup/status')
    const data: SetupStatusResponse = await response.json()
    
    if (!data.needsSetup) {
      router.push({ name: 'login' })
    }
  } catch (err) {
    console.error('Failed to check setup status:', err)
  }
}

async function createAdmin() {
  loading.value = true
  error.value = ''

  try {
    const response = await fetch('/setup/init', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        username: username.value.trim() || 'admin',
        autoWhitelist: autoWhitelist.value
      })
    })

    const data: SetupInitResponse = await response.json()

    if (!response.ok || !data.token) {
      throw new Error(data.error || 'Setup failed')
    }

    generatedToken.value = data.token
    authStore.setToken(data.token)
    currentStep.value = 2
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Setup failed'
  } finally {
    loading.value = false
  }
}

function copyToken() {
  navigator.clipboard.writeText(generatedToken.value)
  showToast()
}

function downloadToken() {
  const content = `digit-link Admin Token
=======================

Token: ${generatedToken.value}

Generated: ${new Date().toISOString()}

IMPORTANT: Keep this token secure. It provides full administrative
access to your digit-link server.

Usage:
  - Dashboard: Enter at the login page
  - API: Set X-Admin-Token header
  - Client: Use --token flag or DIGIT_LINK_TOKEN env var
`
  
  const blob = new Blob([content], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'digit-link-admin-token.txt'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

function goToStep(step: number) {
  currentStep.value = step
}

function goToDashboard() {
  router.push({ name: 'dashboard' })
}

const showingToast = ref(false)
function showToast() {
  showingToast.value = true
  setTimeout(() => { showingToast.value = false }, 2000)
}
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
          :class="[
            dot.isActive ? 'bg-[var(--accent-copper)] shadow-[0_0_12px_rgba(201,149,108,0.4)]' :
            dot.isCompleted ? 'bg-[var(--accent-emerald)]' : 'bg-[var(--border-accent)]'
          ]"
        />
      </div>

      <!-- Brand -->
      <div class="text-center mb-8">
        <div 
          class="relative w-16 h-16 mx-auto mb-6 border-2 border-[var(--accent-copper)] rounded-2xl flex items-center justify-center"
          style="animation: iconFloat 4s ease-in-out infinite;"
        >
          <div class="w-6 h-6 bg-[var(--accent-copper)] rounded-md rotate-45" />
          <div 
            class="absolute -inset-1.5 border border-[var(--accent-copper)] rounded-[20px] opacity-30"
            style="animation: iconRing 2s ease-out infinite;"
          />
        </div>
        
        <h1 class="font-[var(--font-display)] text-[2.25rem] font-semibold tracking-tight mb-2">
          digit-link
        </h1>
        <p class="text-[var(--text-secondary)]">Tunnel Administration</p>
      </div>

      <!-- Setup Card -->
      <div class="relative bg-[var(--bg-surface)] border border-[var(--border-subtle)] rounded-2xl p-10 overflow-hidden">
        <!-- Gradient line -->
        <div class="absolute top-0 left-8 right-8 h-0.5 bg-gradient-to-r from-transparent via-[var(--accent-copper)] to-transparent" />

        <!-- Error message -->
        <div v-if="error" class="error-message mb-6 animate-shake">
          {{ error }}
        </div>

        <!-- Step 1: Welcome -->
        <div v-if="currentStep === 1" class="animate-fade-in-slide">
          <div class="inline-flex items-center gap-2 px-4 py-2 bg-[rgba(201,149,108,0.1)] border border-[rgba(201,149,108,0.3)] rounded-full text-xs font-medium text-[var(--accent-copper)] uppercase tracking-widest mb-6">
            <span class="text-[0.625rem]">✦</span>
            First-Time Setup
          </div>

          <h2 class="font-[var(--font-display)] text-2xl font-semibold mb-3">
            Welcome to digit-link
          </h2>
          <p class="text-sm text-[var(--text-secondary)] leading-relaxed mb-8">
            This is the initial configuration wizard. You'll create the first administrator account 
            that will be used to manage tunnels, accounts, and IP whitelisting.
          </p>

          <div class="mb-6">
            <label class="form-label" for="username">Admin Username</label>
            <input
              id="username"
              v-model="username"
              type="text"
              class="form-input"
              placeholder="Enter admin username"
              autocomplete="off"
            />
            <p class="form-hint">This username identifies the admin account.</p>
          </div>

          <label class="flex items-start gap-3 p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg cursor-pointer hover:border-[var(--border-accent)] transition-colors">
            <input v-model="autoWhitelist" type="checkbox" class="hidden" />
            <div 
              class="w-5 h-5 border-2 rounded flex items-center justify-center flex-shrink-0 transition-all"
              :class="autoWhitelist 
                ? 'bg-[var(--accent-copper)] border-[var(--accent-copper)]' 
                : 'border-[var(--border-accent)]'"
            >
              <span 
                v-if="autoWhitelist" 
                class="text-xs text-[var(--bg-deep)]"
              >✓</span>
            </div>
            <div class="flex-1">
              <strong class="block text-sm mb-1">Auto-whitelist my current IP</strong>
              <span class="text-xs text-[var(--text-muted)]">Allow immediate access from your current location</span>
            </div>
          </label>

          <button
            class="btn btn-primary w-full mt-6"
            :class="{ 'btn-loading': loading }"
            :disabled="loading"
            @click="createAdmin"
          >
            <span class="btn-text">Create Admin Account</span>
          </button>
        </div>

        <!-- Step 2: Token Display -->
        <div v-if="currentStep === 2" class="animate-fade-in-slide">
          <h2 class="font-[var(--font-display)] text-2xl font-semibold mb-3">
            Your Admin Token
          </h2>
          <p class="text-sm text-[var(--text-secondary)] leading-relaxed mb-6">
            Your administrator token has been generated. This is the only time it will be displayed.
          </p>

          <!-- Warning -->
          <div class="warning-box mb-6">
            <div class="w-5 h-5 bg-[var(--accent-red)] rounded-full flex items-center justify-center flex-shrink-0 text-xs text-[var(--bg-deep)] font-bold">
              !
            </div>
            <div class="text-[0.8rem] leading-relaxed text-[var(--accent-red)]">
              <strong class="block mb-1">Save this token now!</strong>
              This token cannot be recovered if lost. Store it in a secure password manager.
            </div>
          </div>

          <!-- Token box -->
          <div class="mb-6">
            <div class="token-box">
              <div class="token-value">{{ generatedToken }}</div>
            </div>
            <div class="flex gap-3 mt-4">
              <button class="btn btn-secondary flex-1" @click="copyToken">
                Copy Token
              </button>
              <button class="btn btn-secondary flex-1" @click="downloadToken">
                Download
              </button>
            </div>
          </div>

          <!-- Confirmation -->
          <label class="flex items-center gap-3 p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg cursor-pointer hover:border-[var(--accent-emerald)] transition-colors mb-6">
            <input v-model="tokenSaved" type="checkbox" class="w-4 h-4 accent-[var(--accent-copper)]" />
            <span class="flex-1" :class="tokenSaved ? 'text-[var(--accent-emerald)]' : ''">
              I have saved my token securely
            </span>
          </label>

          <button
            class="btn btn-success w-full"
            :disabled="!tokenSaved"
            @click="goToStep(3)"
          >
            <span class="btn-text">Continue to Dashboard</span>
          </button>
        </div>

        <!-- Step 3: Complete -->
        <div v-if="currentStep === 3" class="animate-fade-in-slide text-center">
          <div 
            class="w-20 h-20 mx-auto mb-6 bg-[rgba(74,159,126,0.15)] border-2 border-[var(--accent-emerald)] rounded-full flex items-center justify-center"
            style="animation: successPop 0.5s ease-out;"
          >
            <span class="text-3xl text-[var(--accent-emerald)]">✓</span>
          </div>

          <h2 class="font-[var(--font-display)] text-2xl font-semibold mb-3">
            Setup Complete!
          </h2>
          <p class="text-sm text-[var(--text-secondary)] leading-relaxed mb-6">
            Your digit-link server is ready. You'll now be redirected to the admin dashboard.
          </p>

          <div class="bg-[var(--bg-deep)] rounded-lg p-5 mb-6 text-left">
            <p class="text-[0.8rem] text-[var(--text-secondary)] mb-4">Quick start guide:</p>
            <ol class="text-[0.8rem] text-[var(--text-muted)] pl-5 leading-loose list-decimal">
              <li>Add IP addresses to the whitelist</li>
              <li>Create user accounts as needed</li>
              <li>Share tokens securely with users</li>
            </ol>
          </div>

          <button class="btn btn-primary w-full" @click="goToDashboard">
            <span class="btn-text">Open Dashboard</span>
          </button>
        </div>
      </div>

      <!-- Footer -->
      <div class="text-center mt-8 text-xs text-[var(--text-muted)]">
        <p>
          Secure tunnel infrastructure by 
          <a 
            href="https://digit.zone" 
            class="text-[var(--accent-copper)] hover:underline"
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
      Token copied to clipboard
    </div>
  </div>
</template>
