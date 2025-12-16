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
  <div class="max-w-[800px]">
    <PageHeader 
      title="Organization Settings" 
      description="Configure authentication and security policies"
    />

    <!-- Loading -->
    <div v-if="loading" class="p-12 text-center text-text-secondary">
      Loading settings...
    </div>

    <template v-else>
      <!-- Success message -->
      <div 
        v-if="success" 
        class="flex items-center gap-3 py-4 px-5 bg-[rgba(var(--accent-secondary-rgb),0.1)] border border-[rgba(var(--accent-secondary-rgb),0.3)] rounded-[10px] text-[0.9375rem] text-accent-secondary mb-6"
      >
        <CheckCircle class="w-5 h-5" />
        {{ success }}
      </div>

      <!-- Error -->
      <div v-if="error" class="error-message mb-4">
        {{ error }}
      </div>

      <!-- Policy Section -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
        <div class="flex gap-4 p-6 border-b border-border-subtle bg-bg-elevated">
          <div class="w-10 h-10 rounded-[10px] bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary flex items-center justify-center shrink-0">
            <Settings class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-lg font-semibold text-text-primary m-0 mb-1">Authentication Policy</h2>
            <p class="text-sm text-text-secondary m-0 leading-relaxed">
              Configure how users authenticate to access your organization's tunnels. 
              Applications set to "inherit" will use this policy.
            </p>
          </div>
        </div>

        <div class="p-6">
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
