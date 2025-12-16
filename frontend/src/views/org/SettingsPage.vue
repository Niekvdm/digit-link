<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { 
  PageHeader, 
  PolicyEditor
} from '@/components/ui'
import { useApi } from '@/composables/useApi'
import type { PolicyResponse, SetPolicyRequest } from '@/types/api'
import { Settings, CheckCircle } from 'lucide-vue-next'

const api = useApi()

const loading = ref(true)
const saving = ref(false)
const error = ref('')
const success = ref('')
const currentPolicy = ref<SetPolicyRequest | null>(null)

onMounted(async () => {
  await loadPolicy()
})

async function loadPolicy() {
  loading.value = true
  error.value = ''
  
  try {
    const res = await api.get<PolicyResponse>('/org/policy')
    currentPolicy.value = res.policy as SetPolicyRequest | null
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load policy'
    currentPolicy.value = null
  } finally {
    loading.value = false
  }
}

async function handleSubmit(policy: SetPolicyRequest) {
  saving.value = true
  error.value = ''
  success.value = ''
  
  try {
    await api.put('/org/policy', policy)
    currentPolicy.value = policy
    success.value = 'Authentication policy updated successfully'
    setTimeout(() => { success.value = '' }, 5000)
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to save policy'
  } finally {
    saving.value = false
  }
}

function handleCancel() {
  loadPolicy()
}
</script>

<template>
  <div class="settings-page">
    <PageHeader 
      title="Organization Settings" 
      description="Configure authentication and security policies"
    />

    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      Loading settings...
    </div>

    <template v-else>
      <!-- Success message -->
      <div v-if="success" class="success-message">
        <CheckCircle class="w-5 h-5" />
        {{ success }}
      </div>

      <!-- Error -->
      <div v-if="error" class="error-message mb-4">
        {{ error }}
      </div>

      <!-- Policy Section -->
      <div class="settings-card">
        <div class="settings-header">
          <div class="settings-icon">
            <Settings class="w-5 h-5" />
          </div>
          <div>
            <h2 class="settings-title">Authentication Policy</h2>
            <p class="settings-desc">
              Configure how users authenticate to access your organization's tunnels. 
              Applications set to "inherit" will use this policy.
            </p>
          </div>
        </div>

        <div class="settings-body">
          <PolicyEditor 
            :initial-policy="currentPolicy"
            @submit="handleSubmit"
            @cancel="handleCancel"
          />
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.settings-page {
  max-width: 800px;
}

.loading-state {
  padding: 3rem;
  text-align: center;
  color: var(--text-secondary);
}

.success-message {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  background: rgba(var(--accent-secondary-rgb), 0.1);
  border: 1px solid rgba(var(--accent-secondary-rgb), 0.3);
  border-radius: 10px;
  font-size: 0.9375rem;
  color: var(--accent-secondary);
  margin-bottom: 1.5rem;
}

.settings-card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  overflow: hidden;
}

.settings-header {
  display: flex;
  gap: 1rem;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
}

.settings-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  background: rgba(var(--accent-secondary-rgb), 0.15);
  color: var(--accent-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.settings-title {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0 0 0.25rem;
}

.settings-desc {
  font-size: 0.875rem;
  color: var(--text-secondary);
  margin: 0;
  line-height: 1.5;
}

.settings-body {
  padding: 1.5rem;
}

.mb-4 {
  margin-bottom: 1rem;
}
</style>
