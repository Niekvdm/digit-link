<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { 
  PageHeader, 
  StatCard, 
  Modal,
  PlanBadge,
  PlanCard,
  QuotaMeter,
  LoadingSpinner
} from '@/components/ui'
import { useOrganizations, usePlans, useUsage } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import type { Organization, OrgUsageResponse } from '@/types/api'
import { 
  Building2, 
  AppWindow, 
  Users, 
  Cable, 
  ArrowLeft,
  Package,
  BarChart3,
  Settings,
  Calendar
} from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const { fetchOne, loading: orgLoading, error: orgError } = useOrganizations()
const { plans, fetchAll: fetchPlans, setOrganizationPlan } = usePlans()
const { getOrgUsage, loading: usageLoading } = useUsage()
const { formatDate } = useFormatters()

const org = ref<Organization | null>(null)
const usage = ref<OrgUsageResponse | null>(null)
const showPlanModal = ref(false)
const formPlanId = ref<string | null>(null)
const formLoading = ref(false)
const formError = ref('')

const orgId = computed(() => route.params.orgId as string)

onMounted(async () => {
  await Promise.all([
    loadOrg(),
    loadUsage(),
    fetchPlans()
  ])
})

async function loadOrg() {
  org.value = await fetchOne(orgId.value)
}

async function loadUsage() {
  usage.value = await getOrgUsage(orgId.value)
}

function goBack() {
  router.push({ name: 'admin-organizations' })
}

function openPlanModal() {
  formPlanId.value = org.value?.planId || null
  formError.value = ''
  showPlanModal.value = true
}

async function handleSetPlan() {
  if (!org.value) return
  
  formLoading.value = true
  formError.value = ''
  
  try {
    await setOrganizationPlan(org.value.id, formPlanId.value)
    await loadOrg()
    await loadUsage()
    showPlanModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to update plan'
  } finally {
    formLoading.value = false
  }
}

function navigateToUsage() {
  router.push({ name: 'admin-usage' })
}

function navigateToSettings() {
  router.push({ name: 'admin-organizations' })
}
</script>

<template>
  <div class="max-w-[1200px]">
    <!-- Loading State -->
    <div v-if="orgLoading && !org" class="flex items-center justify-center py-20">
      <LoadingSpinner />
    </div>

    <!-- Error State -->
    <div v-else-if="orgError" class="text-center py-20">
      <p class="text-accent-red mb-4">{{ orgError }}</p>
      <button class="btn btn-secondary" @click="goBack">
        <ArrowLeft class="w-4 h-4" />
        Back to Organizations
      </button>
    </div>

    <!-- Content -->
    <template v-else-if="org">
      <!-- Header -->
      <div class="mb-8">
        <button 
          class="flex items-center gap-2 text-sm text-text-muted hover:text-text-primary mb-4 transition-colors"
          @click="goBack"
        >
          <ArrowLeft class="w-4 h-4" />
          Back to Organizations
        </button>
        
        <div class="flex items-start justify-between">
          <div class="flex items-center gap-4">
            <div class="w-14 h-14 rounded-xs bg-[rgba(var(--accent-primary-rgb),0.1)] text-accent-primary flex items-center justify-center">
              <Building2 class="w-7 h-7" />
            </div>
            <div>
              <div class="flex items-center gap-3 mb-1">
                <h1 class="text-2xl font-display font-semibold text-text-primary">{{ org.name }}</h1>
                <PlanBadge :plan-name="org.planName" size="md" />
              </div>
              <div class="flex items-center gap-4 text-sm text-text-secondary">
                <span class="flex items-center gap-1.5">
                  <Calendar class="w-4 h-4" />
                  Created {{ formatDate(org.createdAt) }}
                </span>
              </div>
            </div>
          </div>
          
          <div class="flex items-center gap-2">
            <button class="btn btn-secondary" @click="openPlanModal">
              <Package class="w-4 h-4" />
              Change Plan
            </button>
          </div>
        </div>
      </div>

      <!-- Stats Grid -->
      <div class="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-5 mb-8">
        <StatCard
          label="Applications"
          :value="org.appCount ?? 0"
          :icon="AppWindow"
          color="primary"
        />
        <StatCard
          label="Accounts"
          :value="org.accountCount ?? 0"
          :icon="Users"
          color="amber"
        />
        <StatCard
          label="Active Tunnels"
          :value="org.activeTunnels ?? 0"
          :icon="Cable"
          color="secondary"
        />
      </div>

      <!-- Main Content Grid -->
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <!-- Current Plan -->
        <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
          <div class="flex items-center justify-between p-5 border-b border-border-subtle bg-bg-elevated">
            <h2 class="text-lg font-semibold text-text-primary flex items-center gap-2">
              <Package class="w-5 h-5 text-accent-primary" />
              Current Plan
            </h2>
            <button 
              class="text-sm text-accent-primary hover:underline"
              @click="openPlanModal"
            >
              Change
            </button>
          </div>
          <div class="p-5">
            <template v-if="org.plan">
              <PlanCard :plan="org.plan" current compact />
            </template>
            <template v-else>
              <div class="text-center py-8 text-text-muted">
                <Package class="w-10 h-10 mx-auto mb-3 opacity-40" />
                <p class="mb-4">No plan assigned</p>
                <button class="btn btn-primary btn-sm" @click="openPlanModal">
                  Assign Plan
                </button>
              </div>
            </template>
          </div>
        </div>

        <!-- Usage Overview -->
        <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
          <div class="flex items-center justify-between p-5 border-b border-border-subtle bg-bg-elevated">
            <h2 class="text-lg font-semibold text-text-primary flex items-center gap-2">
              <BarChart3 class="w-5 h-5 text-accent-secondary" />
              Usage This Period
            </h2>
            <span v-if="usage" class="text-xs text-text-muted">
              {{ formatDate(usage.periodStart) }} - {{ formatDate(usage.periodEnd) }}
            </span>
          </div>
          <div class="p-5 space-y-5">
            <template v-if="usageLoading">
              <div class="flex items-center justify-center py-8">
                <LoadingSpinner />
              </div>
            </template>
            <template v-else-if="usage">
              <QuotaMeter
                label="Bandwidth"
                :used="usage.usage.bandwidthBytes"
                :limit="usage.plan?.bandwidthBytesMonthly"
                unit="bytes"
                show-percentage
              />
              <QuotaMeter
                label="Tunnel Hours"
                :used="usage.usage.tunnelSeconds"
                :limit="usage.plan?.tunnelHoursMonthly ? usage.plan.tunnelHoursMonthly * 3600 : undefined"
                unit="hours"
                show-percentage
              />
              <QuotaMeter
                label="Requests"
                :used="usage.usage.requestCount"
                :limit="usage.plan?.requestsMonthly"
                unit="count"
                show-percentage
              />
              <QuotaMeter
                label="Concurrent Tunnels"
                :used="usage.usage.currentConcurrent"
                :limit="usage.plan?.concurrentTunnelsMax"
                unit="concurrent"
              />
            </template>
            <template v-else>
              <div class="text-center py-8 text-text-muted">
                <BarChart3 class="w-10 h-10 mx-auto mb-3 opacity-40" />
                <p>No usage data available</p>
              </div>
            </template>
          </div>
        </div>
      </div>

      <!-- Quick Actions -->
      <div class="mt-8">
        <h2 class="text-lg font-semibold text-text-primary mb-4">Quick Actions</h2>
        <div class="grid grid-cols-[repeat(auto-fit,minmax(250px,1fr))] gap-4">
          <button 
            class="flex items-center gap-4 p-5 bg-bg-surface border border-border-subtle rounded-xs text-left transition-all hover:border-border-accent hover:bg-bg-elevated"
            @click="navigateToUsage"
          >
            <div class="w-10 h-10 rounded-xs bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary flex items-center justify-center">
              <BarChart3 class="w-5 h-5" />
            </div>
            <div>
              <h3 class="font-semibold text-text-primary">View Usage Details</h3>
              <p class="text-sm text-text-secondary">See detailed usage history and analytics</p>
            </div>
          </button>
          <button 
            class="flex items-center gap-4 p-5 bg-bg-surface border border-border-subtle rounded-xs text-left transition-all hover:border-border-accent hover:bg-bg-elevated"
            @click="navigateToSettings"
          >
            <div class="w-10 h-10 rounded-xs bg-[rgba(var(--accent-primary-rgb),0.15)] text-accent-primary flex items-center justify-center">
              <Settings class="w-5 h-5" />
            </div>
            <div>
              <h3 class="font-semibold text-text-primary">Manage Settings</h3>
              <p class="text-sm text-text-secondary">Configure policies and authentication</p>
            </div>
          </button>
        </div>
      </div>
    </template>

    <!-- Plan Assignment Modal -->
    <Modal v-model="showPlanModal" :title="`Change Plan: ${org?.name}`">
      <div class="flex flex-col gap-4">
        <div v-if="formError" class="error-message">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label">Current Plan</label>
          <PlanBadge :plan-name="org?.planName" size="md" />
        </div>

        <div class="flex flex-col gap-2">
          <label class="form-label" for="plan-select">New Plan</label>
          <select
            id="plan-select"
            v-model="formPlanId"
            class="form-input"
          >
            <option :value="null">No Plan (Remove limits)</option>
            <option v-for="plan in plans" :key="plan.id" :value="plan.id">
              {{ plan.name }}
            </option>
          </select>
        </div>

        <div v-if="formPlanId" class="p-4 bg-bg-elevated rounded-xs border border-border-subtle">
          <h4 class="text-sm font-semibold text-text-primary mb-2">Plan Details</h4>
          <div class="space-y-1.5 text-sm">
            <template v-for="plan in plans" :key="plan.id">
              <template v-if="plan.id === formPlanId">
                <div class="flex justify-between">
                  <span class="text-text-secondary">Bandwidth</span>
                  <span class="text-text-primary font-mono">{{ plan.bandwidthBytesMonthly ? `${Math.round(plan.bandwidthBytesMonthly / 1048576)} MB` : 'Unlimited' }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-text-secondary">Tunnel Hours</span>
                  <span class="text-text-primary font-mono">{{ plan.tunnelHoursMonthly || 'Unlimited' }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-text-secondary">Concurrent Tunnels</span>
                  <span class="text-text-primary font-mono">{{ plan.concurrentTunnelsMax || 'Unlimited' }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="text-text-secondary">Requests</span>
                  <span class="text-text-primary font-mono">{{ plan.requestsMonthly ? plan.requestsMonthly.toLocaleString() : 'Unlimited' }}</span>
                </div>
              </template>
            </template>
          </div>
        </div>
      </div>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showPlanModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button class="btn btn-primary" @click="handleSetPlan" :disabled="formLoading">
          {{ formLoading ? 'Updating...' : 'Update Plan' }}
        </button>
      </template>
    </Modal>
  </div>
</template>
