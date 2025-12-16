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
</script>

<template>
  <div class="application-detail">
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
    <div v-if="loading" class="loading-state">
      Loading application...
    </div>

    <!-- Error -->
    <div v-else-if="error" class="error-message">
      {{ error }}
    </div>

    <template v-else-if="application">
      <!-- Stats -->
      <div class="stats-grid">
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
      <div class="detail-card">
        <div class="detail-header">
          <h3 class="detail-title">Application Details</h3>
          <StatusBadge 
            :status="application.isActive ? 'active' : 'inactive'"
            :label="application.isActive ? 'Active' : 'Inactive'"
          />
        </div>

        <div class="detail-grid">
          <div class="detail-item">
            <span class="detail-label">Subdomain</span>
            <div class="detail-value">
              <code class="subdomain-code">{{ application.subdomain }}</code>
              <a 
                :href="`https://${application.subdomain}.tunnel.digit.zone`" 
                target="_blank"
                class="external-link"
              >
                <ExternalLink class="w-4 h-4" />
              </a>
            </div>
          </div>

          <div class="detail-item">
            <span class="detail-label">Display Name</span>
            <template v-if="editMode">
              <input 
                v-model="editName" 
                type="text" 
                class="form-input"
                placeholder="Application name"
              />
            </template>
            <span v-else class="detail-value">{{ application.name }}</span>
          </div>

          <div class="detail-item">
            <span class="detail-label">Auth Mode</span>
            <template v-if="editMode">
              <select v-model="editAuthMode" class="form-input">
                <option value="inherit">Inherit from Organization</option>
                <option value="disabled">Disabled</option>
                <option value="custom">Custom Policy</option>
              </select>
            </template>
            <span v-else class="detail-value auth-mode" :class="`auth-mode--${application.authMode}`">
              {{ application.authMode === 'inherit' ? 'Inherit' : application.authMode === 'disabled' ? 'Disabled' : 'Custom' }}
            </span>
          </div>

          <div class="detail-item">
            <span class="detail-label">Created</span>
            <span class="detail-value">{{ formatDate(application.createdAt) }}</span>
          </div>

          <div class="detail-item">
            <span class="detail-label">Auth Policy</span>
            <div class="detail-value">
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

<style scoped>
.application-detail {
  max-width: 1000px;
}

.loading-state {
  padding: 3rem;
  text-align: center;
  color: var(--text-secondary);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1.25rem;
  margin-bottom: 2rem;
}

.detail-card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  overflow: hidden;
}

.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 1.25rem 1.5rem;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--bg-elevated);
}

.detail-title {
  font-size: 1rem;
  font-weight: 600;
  margin: 0;
}

.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1px;
  background: var(--border-subtle);
}

.detail-item {
  padding: 1.25rem 1.5rem;
  background: var(--bg-surface);
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.detail-label {
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-secondary);
}

.detail-value {
  font-size: 0.9375rem;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.subdomain-code {
  font-family: var(--font-mono);
  color: var(--accent-secondary);
}

.external-link {
  color: var(--text-muted);
  transition: color 0.15s ease;
}

.external-link:hover {
  color: var(--accent-secondary);
}

.auth-mode {
  font-size: 0.8125rem;
  font-weight: 500;
  padding: 0.25rem 0.625rem;
  border-radius: 4px;
  display: inline-block;
}

.auth-mode--inherit {
  background: var(--bg-elevated);
  color: var(--text-secondary);
}

.auth-mode--disabled {
  background: rgba(var(--accent-amber-rgb), 0.1);
  color: var(--accent-amber);
}

.auth-mode--custom {
  background: rgba(var(--accent-secondary-rgb), 0.1);
  color: var(--accent-secondary);
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.75rem;
}

.ml-2 {
  margin-left: 0.5rem;
}
</style>
