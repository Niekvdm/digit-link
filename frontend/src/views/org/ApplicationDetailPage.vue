<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { 
  PageHeader, 
  StatCard, 
  Modal,
  ConfirmDialog,
  PolicyEditor,
  StatusBadge
} from '@/components/ui'
import { useApplications } from '@/composables/api'
import { useFormatters } from '@/composables/useFormatters'
import type { Application, UpdateApplicationRequest, SetPolicyRequest } from '@/types/api'
import { 
  Cable, 
  Activity, 
  ArrowDownUp,
  Settings,
  Save,
  Trash2,
  ExternalLink
} from 'lucide-vue-next'

const props = defineProps<{
  appId: string
}>()

const router = useRouter()
const { fetchOne, update, remove, getPolicy, setPolicy } = useApplications()
const { formatDate, formatBytes } = useFormatters()

const application = ref<Application | null>(null)
const loading = ref(true)
const error = ref('')

// Edit form
const editMode = ref(false)
const editName = ref('')
const editAuthMode = ref<'inherit' | 'disabled' | 'custom'>('inherit')
const editLoading = ref(false)

// Policy modal
const showPolicyModal = ref(false)
const currentPolicy = ref<SetPolicyRequest | null>(null)

// Delete confirm
const showDeleteConfirm = ref(false)

onMounted(async () => {
  await loadApplication()
})

async function loadApplication() {
  loading.value = true
  error.value = ''
  
  try {
    application.value = await fetchOne(props.appId)
    editName.value = application.value.name
    editAuthMode.value = application.value.authMode
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load application'
  } finally {
    loading.value = false
  }
}

async function handleSaveChanges() {
  if (!application.value) return
  
  editLoading.value = true
  
  try {
    const data: UpdateApplicationRequest = {
      name: editName.value,
      authMode: editAuthMode.value
    }
    await update(application.value.id, data)
    application.value = { ...application.value, ...data }
    editMode.value = false
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to update application'
  } finally {
    editLoading.value = false
  }
}

async function openPolicyModal() {
  if (!application.value) return
  
  try {
    const policy = await getPolicy(application.value.id)
    currentPolicy.value = policy as SetPolicyRequest | null
  } catch {
    currentPolicy.value = null
  }
  
  showPolicyModal.value = true
}

async function handleSetPolicy(policy: SetPolicyRequest) {
  if (!application.value) return
  
  try {
    await setPolicy(application.value.id, policy)
    showPolicyModal.value = false
    await loadApplication()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to save policy'
  }
}

async function handleDelete() {
  if (!application.value) return
  
  try {
    await remove(application.value.id)
    router.push({ name: 'org-applications' })
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to delete application'
  }
}

function cancelEdit() {
  if (application.value) {
    editName.value = application.value.name
    editAuthMode.value = application.value.authMode
  }
  editMode.value = false
}

const authModeClasses: Record<string, string> = {
  inherit: 'bg-bg-elevated text-text-secondary',
  disabled: 'bg-[rgba(var(--accent-amber-rgb),0.1)] text-accent-amber',
  custom: 'bg-[rgba(var(--accent-secondary-rgb),0.1)] text-accent-secondary'
}
</script>

<template>
  <div class="max-w-[1000px]">
    <PageHeader 
      :title="application?.name || 'Application'"
      :description="application ? `Subdomain: ${application.subdomain}` : ''"
      back-to="org-applications"
    >
      <template #actions>
        <template v-if="!editMode">
          <button class="btn btn-secondary" @click="editMode = true">
            Edit
          </button>
          <button class="btn btn-danger" @click="showDeleteConfirm = true">
            <Trash2 class="w-4 h-4" />
            Delete
          </button>
        </template>
        <template v-else>
          <button class="btn btn-secondary" @click="cancelEdit" :disabled="editLoading">
            Cancel
          </button>
          <button class="btn btn-primary" @click="handleSaveChanges" :disabled="editLoading">
            <Save class="w-4 h-4" />
            {{ editLoading ? 'Saving...' : 'Save Changes' }}
          </button>
        </template>
      </template>
    </PageHeader>

    <!-- Loading -->
    <div v-if="loading" class="p-12 text-center text-text-secondary">
      Loading application...
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-message">
      {{ error }}
    </div>

    <template v-else-if="application">
      <!-- Stats -->
      <div class="grid grid-cols-[repeat(auto-fit,minmax(200px,1fr))] gap-5 mb-8">
        <StatCard
          label="Active Tunnels"
          :value="application.activeTunnelCount ?? 0"
          :icon="Cable"
          color="secondary"
        />
        <StatCard
          label="Total Connections"
          :value="application.stats?.totalConnections ?? 0"
          :icon="Activity"
          color="primary"
        />
        <StatCard
          label="Data Transferred"
          :value="formatBytes((application.stats?.bytesSent ?? 0) + (application.stats?.bytesReceived ?? 0))"
          :icon="ArrowDownUp"
          color="blue"
        />
      </div>

      <!-- Details Card -->
      <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
        <div class="flex items-center justify-between py-5 px-6 border-b border-border-subtle bg-bg-elevated">
          <h3 class="text-base font-semibold m-0">Application Details</h3>
          <StatusBadge 
            :status="application.isActive ? 'active' : 'inactive'"
            :label="application.isActive ? 'Active' : 'Inactive'"
          />
        </div>

        <div class="grid grid-cols-[repeat(auto-fit,minmax(280px,1fr))] gap-px bg-border-subtle">
          <div class="flex flex-col gap-2 py-5 px-6 bg-bg-surface">
            <span class="text-xs font-medium uppercase tracking-wider text-text-secondary">Subdomain</span>
            <div class="text-[0.9375rem] text-text-primary flex items-center gap-2">
              <code class="font-mono text-accent-secondary">{{ application.subdomain }}</code>
              <a 
                :href="`https://${application.subdomain}.tunnel.digit.zone`" 
                target="_blank"
                class="text-text-muted transition-colors hover:text-accent-secondary"
              >
                <ExternalLink class="w-4 h-4" />
              </a>
            </div>
          </div>

          <div class="flex flex-col gap-2 py-5 px-6 bg-bg-surface">
            <span class="text-xs font-medium uppercase tracking-wider text-text-secondary">Display Name</span>
            <template v-if="editMode">
              <input 
                v-model="editName" 
                type="text" 
                class="form-input"
                placeholder="Application name"
              />
            </template>
            <span v-else class="text-[0.9375rem] text-text-primary">{{ application.name }}</span>
          </div>

          <div class="flex flex-col gap-2 py-5 px-6 bg-bg-surface">
            <span class="text-xs font-medium uppercase tracking-wider text-text-secondary">Auth Mode</span>
            <template v-if="editMode">
              <select v-model="editAuthMode" class="form-input">
                <option value="inherit">Inherit from Organization</option>
                <option value="disabled">Disabled</option>
                <option value="custom">Custom Policy</option>
              </select>
            </template>
            <span 
              v-else 
              class="text-[0.8125rem] font-medium py-1 px-2.5 rounded inline-block w-fit"
              :class="authModeClasses[application.authMode]"
            >
              {{ application.authMode === 'inherit' ? 'Inherit' : application.authMode === 'disabled' ? 'Disabled' : 'Custom' }}
            </span>
          </div>

          <div class="flex flex-col gap-2 py-5 px-6 bg-bg-surface">
            <span class="text-xs font-medium uppercase tracking-wider text-text-secondary">Created</span>
            <span class="text-[0.9375rem] text-text-primary">{{ formatDate(application.createdAt) }}</span>
          </div>

          <div class="flex flex-col gap-2 py-5 px-6 bg-bg-surface">
            <span class="text-xs font-medium uppercase tracking-wider text-text-secondary">Auth Policy</span>
            <div class="text-[0.9375rem] text-text-primary flex items-center gap-2">
              <StatusBadge 
                :status="application.hasPolicy ? 'active' : 'inactive'"
                :label="application.hasPolicy ? 'Configured' : 'Not Set'"
                size="sm"
              />
              <button 
                v-if="application.authMode === 'custom'"
                class="btn btn-sm btn-secondary ml-2"
                @click="openPolicyModal"
              >
                <Settings class="w-3.5 h-3.5" />
                Configure
              </button>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Policy Modal -->
    <Modal v-model="showPolicyModal" :title="`Auth Policy: ${application?.name}`" size="lg">
      <PolicyEditor 
        :initial-policy="currentPolicy"
        @submit="handleSetPolicy"
        @cancel="showPolicyModal = false"
      />
    </Modal>

    <!-- Delete Confirmation -->
    <ConfirmDialog
      v-model="showDeleteConfirm"
      title="Delete Application"
      :message="`Are you sure you want to delete '${application?.name}'? This will delete all associated API keys.`"
      confirm-text="Delete"
      variant="danger"
      @confirm="handleDelete"
    />
  </div>
</template>
