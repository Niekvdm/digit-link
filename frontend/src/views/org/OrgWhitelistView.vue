<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useAuthStore } from '@/stores/authStore'
import OrgLayout from '@/components/layout/OrgLayout.vue'
import { BaseModal, LoadingState, EmptyState } from '@/components/shared'
import {
  ShieldCheck,
  Plus,
  RefreshCw,
  Trash2,
  AlertTriangle,
  AppWindow
} from 'lucide-vue-next'

interface WhitelistEntry {
  id: string
  orgId?: string
  appId?: string
  ipRange: string
  description?: string
  createdAt: string
}

interface Application {
  id: string
  subdomain: string
  name: string
}

const authStore = useAuthStore()

const orgWhitelist = ref<WhitelistEntry[]>([])
const appWhitelists = ref<Record<string, WhitelistEntry[]>>({})
const applications = ref<Application[]>([])
const loading = ref(false)

// Add modal
const showAddModal = ref(false)
const addScope = ref<'org' | 'app'>('org')
const addAppId = ref('')
const addIPRange = ref('')
const addDescription = ref('')
const addLoading = ref(false)

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingEntry = ref<WhitelistEntry | null>(null)
const deletingScope = ref<'org' | 'app'>('org')

async function loadWhitelist() {
  loading.value = true
  try {
    const response = await fetch('/org/whitelist', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const data = await response.json()
      orgWhitelist.value = data.orgWhitelist || []
      appWhitelists.value = data.appWhitelists || {}
    }
  } catch (err) {
    console.error('Failed to load whitelist:', err)
  } finally {
    loading.value = false
  }
}

async function loadApplications() {
  try {
    const response = await fetch('/org/applications', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const data = await response.json()
      applications.value = data.applications || []
    }
  } catch (err) {
    console.error('Failed to load applications:', err)
  }
}

function formatDate(timestamp: string) {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleDateString()
}

function getAppName(appId: string): string {
  const app = applications.value.find(a => a.id === appId)
  return app ? (app.name || app.subdomain) : appId
}

function openAddModal(scope: 'org' | 'app' = 'org', appId?: string) {
  addScope.value = scope
  addAppId.value = appId || (applications.value[0]?.id || '')
  addIPRange.value = ''
  addDescription.value = ''
  showAddModal.value = true
}

async function handleAdd() {
  if (!addIPRange.value.trim()) return

  addLoading.value = true
  try {
    const endpoint = addScope.value === 'org' ? '/org/whitelist' : '/org/app-whitelist'
    const body: Record<string, string> = {
      ipRange: addIPRange.value.trim(),
      description: addDescription.value.trim()
    }
    if (addScope.value === 'app') {
      body.appId = addAppId.value
    }

    const response = await fetch(endpoint, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(body)
    })

    const data = await response.json()
    if (!response.ok || !data.success) {
      alert(data.error || 'Failed to add whitelist entry')
      return
    }

    showAddModal.value = false
    await loadWhitelist()
  } catch (err) {
    alert('Failed to add whitelist entry')
  } finally {
    addLoading.value = false
  }
}

function openDeleteConfirm(entry: WhitelistEntry, scope: 'org' | 'app') {
  deletingEntry.value = entry
  deletingScope.value = scope
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deletingEntry.value) return

  try {
    const endpoint = deletingScope.value === 'org' 
      ? `/org/whitelist/${deletingEntry.value.id}`
      : `/org/app-whitelist/${deletingEntry.value.id}`

    const response = await fetch(endpoint, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })

    const data = await response.json()
    if (!response.ok || !data.success) {
      alert(data.error || 'Failed to delete whitelist entry')
      return
    }

    showDeleteConfirm.value = false
    deletingEntry.value = null
    await loadWhitelist()
  } catch (err) {
    alert('Failed to delete whitelist entry')
  }
}

const hasAppWhitelists = computed(() => Object.keys(appWhitelists.value).length > 0)

onMounted(async () => {
  await Promise.all([loadWhitelist(), loadApplications()])
})
</script>

<template>
  <OrgLayout>
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="font-[var(--font-display)] text-3xl font-semibold mb-1">IP Whitelist</h1>
        <p class="text-sm text-[var(--text-secondary)]">Manage allowed IP addresses for your organization</p>
      </div>
      <div class="flex gap-2">
        <button class="btn btn-secondary" @click="openAddModal('app')" :disabled="!applications.length">
          <Plus class="w-4 h-4" />
          Add App Whitelist
        </button>
        <button class="btn btn-primary" @click="openAddModal('org')">
          <Plus class="w-4 h-4" />
          Add Org Whitelist
        </button>
      </div>
    </div>

    <!-- Organization Whitelist -->
    <div class="card mb-6">
      <div class="card-header">
        <h2 class="card-title">
          <ShieldCheck class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          Organization Whitelist
        </h2>
        <button
          class="btn btn-secondary btn-sm"
          :disabled="loading"
          @click="loadWhitelist"
        >
          <RefreshCw
            class="w-3.5 h-3.5"
            :class="{ 'animate-spin': loading }"
          />
          Refresh
        </button>
      </div>
      <div class="card-body">
        <LoadingState v-if="loading && !orgWhitelist.length" message="Loading whitelist..." />

        <EmptyState
          v-else-if="!orgWhitelist.length"
          :icon="ShieldCheck"
          title="No organization whitelist entries"
          description="Add IP addresses that can access all your applications"
        />

        <div v-else class="flex flex-col gap-2">
          <div
            v-for="entry in orgWhitelist"
            :key="entry.id"
            class="flex items-center justify-between p-3 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg"
          >
            <div class="flex items-center gap-4">
              <code class="text-sm font-mono text-[var(--accent-copper)]">{{ entry.ipRange }}</code>
              <span v-if="entry.description" class="text-sm text-[var(--text-muted)]">{{ entry.description }}</span>
            </div>
            <div class="flex items-center gap-4">
              <span class="text-xs text-[var(--text-muted)]">{{ formatDate(entry.createdAt) }}</span>
              <button
                class="btn btn-danger btn-sm"
                @click="openDeleteConfirm(entry, 'org')"
              >
                <Trash2 class="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Application Whitelists -->
    <div v-if="hasAppWhitelists || applications.length" class="card">
      <div class="card-header">
        <h2 class="card-title">
          <AppWindow class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          Application Whitelists
        </h2>
      </div>
      <div class="card-body">
        <EmptyState
          v-if="!hasAppWhitelists"
          :icon="AppWindow"
          title="No application-specific whitelists"
          description="Add IP addresses for specific applications"
        />

        <div v-else class="space-y-6">
          <div v-for="(entries, appId) in appWhitelists" :key="appId">
            <h3 class="text-sm font-medium text-[var(--text-secondary)] mb-2 flex items-center gap-2">
              <AppWindow class="w-4 h-4" />
              {{ getAppName(appId) }}
            </h3>
            <div class="flex flex-col gap-2">
              <div
                v-for="entry in entries"
                :key="entry.id"
                class="flex items-center justify-between p-3 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg"
              >
                <div class="flex items-center gap-4">
                  <code class="text-sm font-mono text-[var(--accent-copper)]">{{ entry.ipRange }}</code>
                  <span v-if="entry.description" class="text-sm text-[var(--text-muted)]">{{ entry.description }}</span>
                </div>
                <div class="flex items-center gap-4">
                  <span class="text-xs text-[var(--text-muted)]">{{ formatDate(entry.createdAt) }}</span>
                  <button
                    class="btn btn-danger btn-sm"
                    @click="openDeleteConfirm(entry, 'app')"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Add Modal -->
    <BaseModal
      :show="showAddModal"
      :title="addScope === 'org' ? 'Add Organization Whitelist' : 'Add Application Whitelist'"
      @close="showAddModal = false"
    >
      <form @submit.prevent="handleAdd" class="space-y-4">
        <div v-if="addScope === 'app'">
          <label class="form-label" for="addAppId">Application</label>
          <select
            id="addAppId"
            v-model="addAppId"
            class="form-input"
            required
          >
            <option v-for="app in applications" :key="app.id" :value="app.id">
              {{ app.name || app.subdomain }}
            </option>
          </select>
        </div>

        <div>
          <label class="form-label" for="addIPRange">IP Address / CIDR Range</label>
          <input
            id="addIPRange"
            v-model="addIPRange"
            type="text"
            class="form-input form-input-mono"
            placeholder="192.168.1.0/24 or 10.0.0.1"
            required
          />
          <p class="form-hint">Single IP or CIDR notation (e.g., 192.168.1.0/24)</p>
        </div>

        <div>
          <label class="form-label" for="addDescription">Description (optional)</label>
          <input
            id="addDescription"
            v-model="addDescription"
            type="text"
            class="form-input"
            placeholder="Office network"
          />
        </div>
      </form>

      <template #footer>
        <button class="btn btn-secondary" @click="showAddModal = false">
          Cancel
        </button>
        <button
          class="btn btn-primary"
          :class="{ 'btn-loading': addLoading }"
          :disabled="addLoading || !addIPRange.trim()"
          @click="handleAdd"
        >
          <span class="btn-text flex items-center gap-2">
            <Plus class="w-4 h-4" />
            Add Entry
          </span>
        </button>
      </template>
    </BaseModal>

    <!-- Delete Confirmation Modal -->
    <BaseModal
      :show="showDeleteConfirm"
      title="Delete Whitelist Entry"
      @close="showDeleteConfirm = false"
    >
      <div class="warning-box mb-4">
        <AlertTriangle class="w-5 h-5 text-[var(--accent-red)] flex-shrink-0" />
        <div class="text-sm text-[var(--text-secondary)]">
          Are you sure you want to delete the whitelist entry for 
          <strong class="text-[var(--accent-copper)] font-mono">{{ deletingEntry?.ipRange }}</strong>?
        </div>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showDeleteConfirm = false">
          Cancel
        </button>
        <button class="btn btn-danger" @click="handleDelete">
          <Trash2 class="w-4 h-4" />
          Delete
        </button>
      </template>
    </BaseModal>
  </OrgLayout>
</template>
