<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'

const router = useRouter()
const authStore = useAuthStore()

const token = ref('')
const error = ref('')
const loading = ref(false)

onMounted(async () => {
  // Check if already authenticated
  if (authStore.isAuthenticated) {
    const isValid = await authStore.validateToken()
    if (isValid) {
      router.push({ name: 'dashboard' })
    }
  }
})

async function handleSubmit() {
  const trimmedToken = token.value.trim()
  
  if (!trimmedToken) {
    error.value = 'Please enter your admin token'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await fetch('/admin/stats', {
      headers: { 'X-Admin-Token': trimmedToken }
    })

    if (!response.ok) {
      throw new Error('Invalid or expired token')
    }

    authStore.setToken(trimmedToken)
    router.push({ name: 'dashboard' })
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Authentication failed'
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center relative overflow-hidden grid-background">
    <!-- Decorative corners -->
    <div class="corner corner-tl" />
    <div class="corner corner-br" />

    <div class="relative w-full max-w-[420px] px-8 animate-fade-in">
      <!-- Brand -->
      <div class="text-center mb-12">
        <!-- Brand Icon -->
        <div class="relative w-12 h-12 mx-auto mb-6 border-2 border-[var(--accent-copper)] rounded-xl flex items-center justify-center">
          <div class="w-5 h-5 bg-[var(--accent-copper)] rounded rotate-45" />
          <div class="absolute -inset-1 border border-[var(--accent-copper)] rounded-[14px] opacity-30" />
        </div>
        
        <h1 class="font-[var(--font-display)] text-[2rem] font-semibold tracking-tight mb-2">
          digit-link
        </h1>
        <p class="text-sm text-[var(--text-secondary)]">Tunnel Administration</p>
      </div>

      <!-- Login Card -->
      <div class="relative bg-[var(--bg-surface)] border border-[var(--border-subtle)] rounded-xl p-8">
        <!-- Gradient line -->
        <div class="absolute top-0 left-8 right-8 h-px bg-gradient-to-r from-transparent via-[var(--accent-copper)] to-transparent" />

        <!-- Error message -->
        <div 
          v-if="error"
          class="error-message mb-6 animate-shake"
        >
          {{ error }}
        </div>

        <form @submit.prevent="handleSubmit">
          <div class="mb-6">
            <label class="form-label" for="token">Admin Token</label>
            <input
              id="token"
              v-model="token"
              type="password"
              class="form-input form-input-mono"
              placeholder="Enter your admin token"
              autocomplete="off"
              autofocus
            />
            <p class="form-hint">Contact your administrator if you need access.</p>
          </div>

          <button
            type="submit"
            class="btn btn-primary w-full"
            :class="{ 'btn-loading': loading }"
            :disabled="loading"
          >
            <span class="btn-text">Access Dashboard</span>
          </button>
        </form>
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
  </div>
</template>
