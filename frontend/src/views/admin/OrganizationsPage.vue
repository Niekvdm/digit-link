<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { 
  PageHeader, 
  DataTable, 
  Modal, 
  ConfirmDialog,
  SearchInput,
  PolicyEditor,
  StatusBadge,
  PlanBadge
} from '@/components/ui'
import { useOrganizations, usePlans } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import type { Organization, SetPolicyRequest, Plan } from '@/types/api'
import { Plus, Settings, Trash2, Pencil, Building2, Eye, Package } from 'lucide-vue-next'

const router = useRouter()
const { organizations, loading, error, fetchAll, create, update, remove, getPolicy, setPolicy } = useOrganizations()
const { plans, fetchAll: fetchPlans, setOrganizationPlan } = usePlans()
const { formatDate } = useFormatters()

// Search
const searchQuery = ref('')

const filteredOrganizations = computed(() => {
  if (!searchQuery.value) return organizations.value
  const query = searchQuery.value.toLowerCase()
  return organizations.value.filter(org => 
    org.name.toLowerCase().includes(query) ||
    (org.planName?.toLowerCase().includes(query))
  )
})

// Modals
const showCreateModal = ref(false)
const showEditModal = ref(false)
const showPolicyModal = ref(false)
const showDeleteConfirm = ref(false)
const showPlanModal = ref(false)

// Form state
const formName = ref('')
const formPlanId = ref<string | null>(null)
const formLoading = ref(false)
const formError = ref('')
const selectedOrg = ref<Organization | null>(null)
const currentPolicy = ref<SetPolicyRequest | null>(null)

// Table columns
const columns = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'planName', label: 'Plan', sortable: true, width: '140px' },
  { key: 'appCount', label: 'Applications', sortable: true, width: '120px' },
  { key: 'hasPolicy', label: 'Auth Policy', width: '120px' },
  { key: 'createdAt', label: 'Created', sortable: true, width: '160px' },
]

onMounted(() => {
  fetchAll()
  fetchPlans()
})

// Navigate to org detail
function viewOrgDetail(org: Organization) {
  router.push({ name: 'admin-organization-detail', params: { orgId: org.id } })
}

// Create
function openCreateModal() {
  formName.value = ''
  formPlanId.value = null
  formError.value = ''
  showCreateModal.value = true
}

async function handleCreate() {
  if (!formName.value.trim()) {
    formError.value = 'Name is required'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    const org = await create(formName.value.trim())
    // If a plan was selected, assign it
    if (formPlanId.value && org) {
      await setOrganizationPlan(org.id, formPlanId.value)
      await fetchAll() // Refresh to get plan info
    }
    showCreateModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to create organization'
  } finally {
    formLoading.value = false
  }
}

// Edit
function openEditModal(org: Organization) {
  selectedOrg.value = org
  formName.value = org.name
  formPlanId.value = org.planId || null
  formError.value = ''
  showEditModal.value = true
}

async function handleUpdate() {
  if (!selectedOrg.value || !formName.value.trim()) {
    formError.value = 'Name is required'
    return
  }
  
  formLoading.value = true
  formError.value = ''
  
  try {
    await update(selectedOrg.value.id, formName.value.trim())
    // Update plan if changed
    if (formPlanId.value !== (selectedOrg.value.planId || null)) {
      await setOrganizationPlan(selectedOrg.value.id, formPlanId.value)
      await fetchAll() // Refresh to get updated plan info
    }
    showEditModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to update organization'
  } finally {
    formLoading.value = false
  }
}

// Plan assignment modal
function openPlanModal(org: Organization) {
  selectedOrg.value = org
  formPlanId.value = org.planId || null
  formError.value = ''
  showPlanModal.value = true
}

async function handleSetPlan() {
  if (!selectedOrg.value) return
  
  formLoading.value = true
  formError.value = ''
  
  try {
    await setOrganizationPlan(selectedOrg.value.id, formPlanId.value)
    await fetchAll() // Refresh to get updated plan info
    showPlanModal.value = false
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to update plan'
  } finally {
    formLoading.value = false
  }
}

// Policy
async function openPolicyModal(org: Organization) {
  selectedOrg.value = org
  formError.value = ''
  
  try {
    const policy = await getPolicy(org.id)
    currentPolicy.value = policy as SetPolicyRequest | null
  } catch (e) {
    currentPolicy.value = null
  }
  
  showPolicyModal.value = true
}

async function handleSetPolicy(policy: SetPolicyRequest) {
  if (!selectedOrg.value) return
  
  formLoading.value = true
  formError.value = ''
  
  try {
    await setPolicy(selectedOrg.value.id, policy)
    showPolicyModal.value = false
    await fetchAll() // Refresh to update hasPolicy
  } catch (e) {
    formError.value = e instanceof Error ? e.message : 'Failed to save policy'
  } finally {
    formLoading.value = false
  }
}

// Delete
function openDeleteConfirm(org: Organization) {
  selectedOrg.value = org
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!selectedOrg.value) return
  
  try {
    await remove(selectedOrg.value.id)
    showDeleteConfirm.value = false
  } catch (e) {
    console.error('Delete failed:', e)
  }
}
</script>

<template>
  <div class="max-w-[1200px]">
    <PageHeader 
      title="Organizations" 
      description="Manage organizations, plans, and authentication policies"
    >
      <template #actions>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          New Organization
        </button>
      </template>
    </PageHeader>

    <!-- Search -->
    <div class="mb-6">
      <SearchInput v-model="searchQuery" placeholder="Search organizations..." />
    </div>

    <!-- Error -->
    <div v-if="error" class="error-message mb-4">
      {{ error }}
    </div>

    <!-- Table -->
    <DataTable
      :columns="columns"
      :data="filteredOrganizations"
      :loading="loading"
      empty-title="No organizations"
      empty-description="Create your first organization to get started."
      row-key="id"
    >
      <template #cell-name="{ row }">
        <button 
          class="flex items-center gap-3 text-left hover:text-accent-primary transition-colors"
          @click="viewOrgDetail(row)"
        >
          <div class="w-8 h-8 rounded-xs bg-[rgba(var(--accent-primary-rgb),0.1)] text-accent-primary flex items-center justify-center">
            <Building2 class="w-4 h-4" />
          </div>
          <span>{{ row.name }}</span>
        </button>
      </template>
      
      <template #cell-planName="{ row }">
        <button 
          class="hover:opacity-80 transition-opacity"
          @click.stop="openPlanModal(row)"
          title="Click to change plan"
        >
          <PlanBadge :plan-name="row.planName" size="sm" />
        </button>
      </template>
      
      <template #cell-hasPolicy="{ value }">
        <StatusBadge 
          :status="value ? 'active' : 'inactive'" 
          :label="value ? 'Configured' : 'Not Set'"
          size="sm"
        />
      </template>
      
      <template #cell-createdAt="{ value }">
        {{ formatDate(value as string) }}
      </template>
      
      <template #actions="{ row }">
        <div class="flex items-center gap-1">
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-bg-elevated hover:text-text-primary"
            title="View Details" 
            @click.stop="viewOrgDetail(row)"
          >
            <Eye class="w-4 h-4" />
          </button>
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-bg-elevated hover:text-text-primary"
            title="Change Plan" 
            @click.stop="openPlanModal(row)"
          >
            <Package class="w-4 h-4" />
          </button>
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-bg-elevated hover:text-text-primary"
            title="Configure Policy" 
            @click.stop="openPolicyModal(row)"
          >
            <Settings class="w-4 h-4" />
          </button>
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-bg-elevated hover:text-text-primary"
            title="Edit" 
            @click.stop="openEditModal(row)"
          >
            <Pencil class="w-4 h-4" />
          </button>
          <button 
            class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-[rgba(var(--accent-red-rgb),0.1)] hover:text-accent-red disabled:opacity-30 disabled:cursor-not-allowed"
            title="Delete" 
            @click.stop="openDeleteConfirm(row)"
            :disabled="(row.appCount ?? 0) > 0"
          >
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </template>
      
      <template #emptyAction>
        <button class="btn btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          Create Organization
        </button>
      </template>
    </DataTable>

    <!-- Create Modal -->
    <Modal v-model="showCreateModal" title="New Organization">
      <form @submit.prevent="handleCreate" class="flex flex-col gap-4">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="org-name">Organization Name</label>
          <input
            id="org-name"
            v-model="formName"
            type="text"
            class="form-input"
            placeholder="Enter organization name"
            autofocus
          />
        </div>

        <div class="flex flex-col gap-2">
          <label class="form-label" for="org-plan">Subscription Plan</label>
          <select
            id="org-plan"
            v-model="formPlanId"
            class="form-input"
          >
            <option :value="null">No Plan</option>
            <option v-for="plan in plans" :key="plan.id" :value="plan.id">
              {{ plan.name }}
            </option>
          </select>
          <p class="text-xs text-text-muted">Select a plan to set quota limits for this organization.</p>
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showCreateModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button class="btn btn-primary" @click="handleCreate" :disabled="formLoading || !formName.trim()">
          {{ formLoading ? 'Creating...' : 'Create' }}
        </button>
      </template>
    </Modal>

    <!-- Edit Modal -->
    <Modal v-model="showEditModal" title="Edit Organization">
      <form @submit.prevent="handleUpdate" class="flex flex-col gap-4">
        <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label" for="edit-org-name">Organization Name</label>
          <input
            id="edit-org-name"
            v-model="formName"
            type="text"
            class="form-input"
            placeholder="Enter organization name"
            autofocus
          />
        </div>

        <div class="flex flex-col gap-2">
          <label class="form-label" for="edit-org-plan">Subscription Plan</label>
          <select
            id="edit-org-plan"
            v-model="formPlanId"
            class="form-input"
          >
            <option :value="null">No Plan</option>
            <option v-for="plan in plans" :key="plan.id" :value="plan.id">
              {{ plan.name }}
            </option>
          </select>
        </div>
      </form>
      
      <template #footer>
        <button class="btn btn-secondary" @click="showEditModal = false" :disabled="formLoading">
          Cancel
        </button>
        <button class="btn btn-primary" @click="handleUpdate" :disabled="formLoading || !formName.trim()">
          {{ formLoading ? 'Saving...' : 'Save Changes' }}
        </button>
      </template>
    </Modal>

    <!-- Plan Assignment Modal -->
    <Modal v-model="showPlanModal" :title="`Change Plan: ${selectedOrg?.name}`">
      <div class="flex flex-col gap-4">
        <div v-if="formError" class="error-message">{{ formError }}</div>
        
        <div class="flex flex-col gap-2">
          <label class="form-label">Current Plan</label>
          <PlanBadge :plan-name="selectedOrg?.planName" size="md" />
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

    <!-- Policy Modal -->
    <Modal v-model="showPolicyModal" :title="`Auth Policy: ${selectedOrg?.name}`" size="lg">
      <div v-if="formError" class="error-message mb-4">{{ formError }}</div>
      <PolicyEditor 
        :initial-policy="currentPolicy"
        @submit="handleSetPolicy"
        @cancel="showPolicyModal = false"
      />
    </Modal>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="Delete Organization"
      :message="`Are you sure you want to delete '${selectedOrg?.name}'? This action cannot be undone.`"
      confirm-text="Delete"
      variant="danger"
      @confirm="handleDelete"
    />
  </div>
</template>
