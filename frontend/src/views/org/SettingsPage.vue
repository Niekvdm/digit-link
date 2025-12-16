<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { 
  PageHeader, 
  PolicyEditor,
  PlanBadge,
  QuotaMeter
} from '@/components/ui'
import { useApi } from '@/composables/useApi'
import { useUsage } from '@/composables/api'
import type { PolicyResponse, SetPolicyRequest, OrgUsageResponse } from '@/types/api'
import { Settings, CheckCircle, Package, ArrowRight, CreditCard } from 'lucide-vue-next'

const router = useRouter()
const api = useApi()
const { getOrgUsage, loading: usageLoading } = useUsage()

const loading = ref(true)
const saving = ref(false)
const error = ref('')
const success = ref('')
const currentPolicy = ref<SetPolicyRequest | null>(null)
const usage = ref<OrgUsageResponse | null>(null)

onMounted(async () => {
  await Promise.all([
    loadPolicy(),
    loadUsage()
  ])
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

async function loadUsage() {
  usage.value = await getOrgUsage('', true)
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

function navigateToBilling() {
  router.push({ name: 'org-billing' })
}

function formatBytes(bytes?: number): string {
  if (!bytes) return 'Unlimited'
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(0)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(0)} MB`
  return `${(bytes / 1024).toFixed(0)} KB`
}

function formatNumber(num?: number): string {
  if (!num) return 'Unlimited'
  return num.toLocaleString()
}
</script>

<template>
  <div class="max-w-[800px]">
    <PageHeader 
      title="Organization Settings" 
      description="Manage your subscription and configure authentication policies"
    />

    <!-- Loading -->
    <div v-if="loading && usageLoading" class="p-12 text-center text-text-secondary">
      Loading settings...
    </div>

    <template v-else>
      <!-- Success message -->
      <div 
        v-if="success" 
        class="flex items-center gap-3 py-4 px-5 bg-[rgba(var(--accent-secondary-rgb),0.1)] border border-[rgba(var(--accent-secondary-rgb),0.3)] rounded-xs text-[0.9375rem] text-accent-secondary mb-6"
      >
        <CheckCircle class="w-5 h-5" />
        {{ success }}
      </div>

      <!-- Error -->
      <div v-if="error" class="error-message mb-4">
        {{ error }}
      </div>

      <!-- Subscription Section -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden mb-6">
        <div class="flex gap-4 p-6 border-b border-border-subtle bg-bg-elevated">
          <div class="w-10 h-10 rounded-xs bg-[rgba(168,85,247,0.15)] text-[rgb(168,85,247)] flex items-center justify-center shrink-0">
            <Package class="w-5 h-5" />
          </div>
          <div class="flex-1">
            <h2 class="text-lg font-semibold text-text-primary m-0 mb-1">Subscription</h2>
            <p class="text-sm text-text-secondary m-0 leading-relaxed">
              Your current plan and usage limits
            </p>
          </div>
          <button 
            class="flex items-center gap-2 text-sm text-accent-primary hover:underline self-start"
            @click="navigateToBilling"
          >
            View Billing
            <ArrowRight class="w-4 h-4" />
          </button>
        </div>

        <div class="p-6">
          <div v-if="usageLoading" class="h-24 bg-bg-elevated rounded animate-pulse" />
          
          <template v-else-if="usage">
            <!-- Plan Info -->
            <div class="flex items-start justify-between mb-6 pb-6 border-b border-border-subtle">
              <div>
                <p class="text-sm text-text-secondary mb-2">Current Plan</p>
                <PlanBadge :plan-name="usage.plan?.name" size="lg" />
              </div>
              <div v-if="usage.plan" class="text-right">
                <p class="text-sm text-text-secondary mb-1">Grace Period</p>
                <p class="text-text-primary font-mono">{{ usage.plan.gracePeriodHours }}h</p>
              </div>
            </div>

            <!-- Quota Limits -->
            <div v-if="usage.plan" class="grid grid-cols-2 gap-4">
              <div class="p-4 bg-bg-elevated rounded-xs">
                <p class="text-xs text-text-muted uppercase tracking-wide mb-1">Bandwidth</p>
                <p class="text-lg font-semibold text-text-primary font-mono">
                  {{ formatBytes(usage.plan.bandwidthBytesMonthly) }}
                </p>
                <p class="text-xs text-text-secondary mt-1">per month</p>
              </div>
              <div class="p-4 bg-bg-elevated rounded-xs">
                <p class="text-xs text-text-muted uppercase tracking-wide mb-1">Requests</p>
                <p class="text-lg font-semibold text-text-primary font-mono">
                  {{ formatNumber(usage.plan.requestsMonthly) }}
                </p>
                <p class="text-xs text-text-secondary mt-1">per month</p>
              </div>
              <div class="p-4 bg-bg-elevated rounded-xs">
                <p class="text-xs text-text-muted uppercase tracking-wide mb-1">Tunnel Hours</p>
                <p class="text-lg font-semibold text-text-primary font-mono">
                  {{ usage.plan.tunnelHoursMonthly || 'Unlimited' }}
                </p>
                <p class="text-xs text-text-secondary mt-1">per month</p>
              </div>
              <div class="p-4 bg-bg-elevated rounded-xs">
                <p class="text-xs text-text-muted uppercase tracking-wide mb-1">Concurrent</p>
                <p class="text-lg font-semibold text-text-primary font-mono">
                  {{ usage.plan.concurrentTunnelsMax || 'Unlimited' }}
                </p>
                <p class="text-xs text-text-secondary mt-1">tunnels</p>
              </div>
            </div>

            <!-- No Plan State -->
            <div v-else class="text-center py-6">
              <Package class="w-12 h-12 mx-auto mb-3 text-text-muted opacity-40" />
              <p class="text-text-secondary mb-4">No subscription plan assigned</p>
              <p class="text-sm text-text-muted">Contact your administrator to set up a plan.</p>
            </div>
          </template>
        </div>
      </div>

      <!-- Policy Section -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
        <div class="flex gap-4 p-6 border-b border-border-subtle bg-bg-elevated">
          <div class="w-10 h-10 rounded-xs bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary flex items-center justify-center shrink-0">
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
