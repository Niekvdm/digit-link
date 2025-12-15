<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/authStore'
import { Cable, ArrowRight, AlertCircle, User, Lock } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  if (!username.value || !password.value) {
    error.value = 'Username and password are required'
    return
  }

  loading.value = true
  error.value = ''

  try {
    const response = await fetch('/auth/org/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        username: username.value,
        password: password.value
      })
    })

    const data = await response.json()

    if (!response.ok || !data.success) {
      error.value = data.error || 'Login failed'
      return
    }

    // Store token and redirect to org dashboard
    authStore.setToken(data.token, 'org', data.orgId)
    router.push('/portal')
  } catch (err) {
    error.value = 'Network error. Please try again.'
    console.error('Login error:', err)
  } finally {
    loading.value = false
  }
}

function goToAdminLogin() {
  router.push('/')
}
</script>

<template>
  <div class="min-h-screen bg-[var(--bg-base)] flex items-center justify-center p-4">
    <div class="w-full max-w-md">
      <!-- Logo -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)] mb-4">
          <Cable class="w-8 h-8" />
        </div>
        <h1 class="font-[var(--font-display)] text-2xl font-semibold text-[var(--text-primary)]">
          Organization Portal
        </h1>
        <p class="text-sm text-[var(--text-secondary)] mt-1">
          Sign in to manage your organization's tunnels
        </p>
      </div>

      <!-- Login Form -->
      <div class="card">
        <div class="card-body">
          <form @submit.prevent="handleLogin" class="space-y-5">
            <!-- Error Message -->
            <div v-if="error" class="flex items-start gap-3 p-3 rounded-lg bg-[rgba(207,89,89,0.1)] text-[var(--accent-red)]">
              <AlertCircle class="w-5 h-5 flex-shrink-0 mt-0.5" />
              <span class="text-sm">{{ error }}</span>
            </div>

            <!-- Username -->
            <div>
              <label class="form-label" for="username">Username</label>
              <div class="relative">
                <User class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-muted)]" />
                <input
                  id="username"
                  v-model="username"
                  type="text"
                  class="form-input pl-10"
                  placeholder="Enter your username"
                  autocomplete="username"
                  required
                />
              </div>
            </div>

            <!-- Password -->
            <div>
              <label class="form-label" for="password">Password</label>
              <div class="relative">
                <Lock class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-[var(--text-muted)]" />
                <input
                  id="password"
                  v-model="password"
                  type="password"
                  class="form-input pl-10"
                  placeholder="Enter your password"
                  autocomplete="current-password"
                  required
                />
              </div>
            </div>

            <!-- Submit -->
            <button
              type="submit"
              class="btn btn-primary w-full"
              :class="{ 'btn-loading': loading }"
              :disabled="loading"
            >
              <span class="btn-text flex items-center justify-center gap-2">
                Sign In
                <ArrowRight class="w-4 h-4" />
              </span>
            </button>
          </form>
        </div>
      </div>

      <!-- Admin Login Link -->
      <div class="text-center mt-6">
        <button
          class="text-sm text-[var(--text-muted)] hover:text-[var(--text-secondary)] transition-colors"
          @click="goToAdminLogin"
        >
          Administrator? Sign in here
        </button>
      </div>
    </div>
  </div>
</template>
