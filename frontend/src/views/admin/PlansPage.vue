<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { 
  PageHeader, 
  DataTable, 
  Modal, 
  ConfirmDialog,
  EmptyState
} from '@/components/ui'
import { usePlans } from '@/composables/api'
import type { Plan, CreatePlanRequest } from '@/types/api'
import { Plus, Edit, Trash2, Package, Infinity } from 'lucide-vue-next'

const { plans, loading, error, fetchAll, create, update, remove } = usePlans()

// Modals
const showEditModal = ref(false)
const showDeleteConfirm = ref(false)
const isEditing = ref(false)

// Form state
const formData = ref<CreatePlanRequest>({
  name: '',
  bandwidthBytesMonthly: undefined,
  tunnelHoursMonthly: undefined,
  concurrentTunnelsMax: undefined,
  requestsMonthly: undefined,
  overageAllowedPercent: 0,
  gracePeriodHours: 0
})
const formLoading = ref(false)
const formError = ref('')
const selectedPlan = ref<Plan | null>(null)

// Table columns
const columns = [
  { key: 'name', label: 'Plan Name', sortable: true },
  { key: 'bandwidthBytesMonthly', label: 'Bandwidth', width: '140px' },
  { key: 'tunnelHoursMonthly', label: 'Tunnel Hours', width: '120px' },
  { key: 'concurrentTunnelsMax', label: 'Concurrent', width: '100px' },
  { key: 'requestsMonthly', label: 'Requests', width: '120px' },
  { key: 'overageAllowedPercent', label: 'Overage', width: '100px' },
  { key: 'gracePeriodHours', label: 'Grace Period', width: '110px' },
]

onMounted(() => {
  fetchAll()
})

// Format helpers
function formatBytes(bytes: number | undefined): string {
  if (bytes === undefined || bytes === null) return '∞'
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

function formatNumber(n: number | undefined): string {
  if (n === undefined || n === null) return '∞'
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}K`
  return n.toString()
}

// Create/Edit
function openCreateModal() {
  isEditing.value = false
  selectedPlan.value = null
  formData.value = {
    name: '',
    bandwidthBytesMonthly: undefined,
    tunnelHoursMonthly: undefined,
    concurrentTunnelsMax: undefined,
    requestsMonthly: undefined,
    overageAllowedPercent: 0,
    gracePeriodHours: 0
  }
  formError.value = ''
  showEditModal.value = true
}

function openEditModal(plan: Plan) {
  isEditing.value = true
  selectedPlan.value = plan
  formData.value = {
    name: plan.name,
    bandwidthBytesMonthly: plan.bandwidthBytesMonthly,
    tunnelHoursMonthly: plan.tunnelHoursMonthly,
    concurrentTunnelsMax: plan.concurrentTunnelsMax,
    requestsMonthly: plan.requestsMonthly,
    overageAllowedPercent: plan.overageAllowedPercent,
    gracePeriodHours: plan.gracePeriodHours
  }
  formError.value = ''
  showEditModal.value = true
}

async function handleSave() {
  if (!formData.value.name.trim()) {
    formError.value = 'Plan name is required'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    if (isEditing.value && selectedPlan.value) {
      await update(selectedPlan.value.id, formData.value)
    } else {
      await create(formData.value)
    }
    showEditModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to save plan'
  } finally {
    formLoading.value = false
  }
}

// Delete
function openDeleteConfirm(plan: Plan) {
  selectedPlan.value = plan
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!selectedPlan.value) return
  
  try {
    await remove(selectedPlan.value.id)
    showDeleteConfirm.value = false
  } catch (e) {
    console.error('Delete failed:', e)
  }
}

// Convert MB to bytes for storage (must be integer for backend)
function mbToBytes(mb: number | undefined): number | undefined {
  if (mb === undefined || mb === null || mb === 0) return undefined
  return Math.round(mb * 1024 * 1024)
}

function bytesToMb(bytes: number | undefined): number | undefined {
  if (bytes === undefined || bytes === null) return undefined
  return bytes / (1024 * 1024)
}

// Form value wrappers for MB input
const bandwidthMb = computed({
  get: () => bytesToMb(formData.value.bandwidthBytesMonthly),
  set: (val: number | undefined) => {
    formData.value.bandwidthBytesMonthly = mbToBytes(val)
  }
})
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      title="Plans" 
      description="Manage subscription plans and quota limits"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          New Plan
        </button>
      </template>
    </PageHeader>

    <!-- Error -->
    <div v-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="plans"
      :loading="loading"
      empty-title="No plans"
      empty-description="Create your first plan to set quota limits for organizations."
      row-key="id"
    >
      <template #cell-name="{ row }">
        <div class="flex items-center gap-3">
          <div class="w-8 h-8 rounded-xs flex items-center justify-center bg-[rgba(var(--accent-primary-rgb),0.1)] text-accent-primary">
            <Package class="w-4 h-4" />
          </div>
          <span class="font-medium">{{ row.name }}</span>
        </div>
      </template>
      
      <template #cell-bandwidthBytesMonthly="{ value }">
        <span :class="value === undefined ? 'text-text-muted' : ''">
          {{ formatBytes(value as number | undefined) }}
        </span>
      </template>
      
      <template #cell-tunnelHoursMonthly="{ value }">
        <span :class="value === undefined ? 'text-text-muted' : ''">
          {{ value ?? '∞' }} hrs
        </span>
      </template>
      
      <template #cell-concurrentTunnelsMax="{ value }">
        <span :class="value === undefined ? 'text-text-muted' : ''">
          {{ value ?? '∞' }}
        </span>
      </template>
      
      <template #cell-requestsMonthly="{ value }">
        <span :class="value === undefined ? 'text-text-muted' : ''">
          {{ formatNumber(value as number | undefined) }}
        </span>
      </template>
      
      <template #cell-overageAllowedPercent="{ value }">
        <span :class="value === 0 ? 'text-text-muted' : 'text-accent-amber'">
          {{ value }}%
        </span>
      </template>
      
      <template #cell-gracePeriodHours="{ value }">
        <span :class="value === 0 ? 'text-text-muted' : ''">
          {{ value }} hrs
        </span>
      </template>
      
      <template #actions="{ row }">
        <div class="flex items-center gap-1">
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-bg-elevated hover:text-text-primary" 
            title="Edit Plan" 
            @click.stop="openEditModal(row)"
          >
            <Edit class="w-4 h-4" />
          </button>
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-[rgba(var(--accent-red-rgb),0.1)] hover:text-accent-red" 
            title="Delete Plan" 
            @click.stop="openDeleteConfirm(row)"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </template>
      
      <template #emptyAction>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          Create Plan
        </button>
      </template>
    </DataTable>

    <!-- Edit Modal -->
    <Modal v-model="showEditModal" :title="isEditing ? 'Edit Plan' : 'New Plan'">
      <form @submit.prevent="handleSave" class="flex flex-col gap-5">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="plan-name">Plan Name</label>
          <input
            id="plan-name"
            v-model="formData.name"
            type="text"
            class="form-input"
            placeholder="e.g., Free, Pro, Enterprise"
            autofocus
          />
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div class="flex flex-col gap-2">
            <label class="form-label" for="plan-bandwidth">
              Bandwidth (MB/month)
              <span class="font-normal normal-case tracking-normal text-text-muted">(blank = unlimited)</span>
            </label>
            <input
              id="plan-bandwidth"
              v-model.number="bandwidthMb"
              type="number"
              min="0"
              step="1"
              class="form-input"
              placeholder="∞"
            />
          </div>
          
          <div class="flex flex-col gap-2">
            <label class="form-label" for="plan-hours">
              Tunnel Hours/month
              <span class="font-normal normal-case tracking-normal text-text-muted">(blank = unlimited)</span>
            </label>
            <input
              id="plan-hours"
              v-model.number="formData.tunnelHoursMonthly"
              type="number"
              min="0"
              class="form-input"
              placeholder="∞"
            />
          </div>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div class="flex flex-col gap-2">
            <label class="form-label" for="plan-concurrent">
              Max Concurrent Tunnels
              <span class="font-normal normal-case tracking-normal text-text-muted">(blank = unlimited)</span>
            </label>
            <input
              id="plan-concurrent"
              v-model.number="formData.concurrentTunnelsMax"
              type="number"
              min="0"
              class="form-input"
              placeholder="∞"
            />
          </div>
          
          <div class="flex flex-col gap-2">
            <label class="form-label" for="plan-requests">
              Requests/month
              <span class="font-normal normal-case tracking-normal text-text-muted">(blank = unlimited)</span>
            </label>
            <input
              id="plan-requests"
              v-model.number="formData.requestsMonthly"
              type="number"
              min="0"
              class="form-input"
              placeholder="∞"
            />
          </div>
        </div>

        <div class="border-t border-border-subtle pt-5 mt-2">
          <h4 class="text-sm font-semibold text-text-primary mb-4">Enforcement</h4>
          
          <div class="grid grid-cols-2 gap-4">
            <div class="flex flex-col gap-2">
              <label class="form-label" for="plan-overage">Overage Allowed (%)</label>
              <input
                id="plan-overage"
                v-model.number="formData.overageAllowedPercent"
                type="number"
                min="0"
                max="100"
                class="form-input"
              />
              <p class="form-hint">0 = hard block at limit</p>
            </div>
            
            <div class="flex flex-col gap-2">
              <label class="form-label" for="plan-grace">Grace Period (hours)</label>
              <input
                id="plan-grace"
                v-model.number="formData.gracePeriodHours"
                type="number"
                min="0"
                class="form-input"
              />
              <p class="form-hint">Time after limit before blocking</p>
            </div>
          </div>
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showEditModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button 
          class="btn btn-primary" 
          @click="handleSave" 
          :disabled="formLoading || !formData.name.trim()"
        >
          {{ formLoading ? 'Saving...' : isEditing ? 'Save Changes' : 'Create Plan' }}
        </button>
      </template>
    </Modal>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="Delete Plan"
      :message="`Are you sure you want to delete '${selectedPlan?.name}'? This action cannot be undone. Plans with organizations assigned cannot be deleted.`"
      confirm-text="Delete"
      variant="danger"
      @confirm="handleDelete"
    />
  </div>
</template>
