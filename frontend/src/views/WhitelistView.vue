<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useApi } from '@/composables/useApi'
import { useOrganizations } from '@/composables'
import AppLayout from '@/components/layout/AppLayout.vue'
import { LoadingState, EmptyState } from '@/components/shared'
import { 
  ShieldCheck, 
  ShieldOff,
  RefreshCw, 
  Info,
  Building2,
  AppWindow,
  ChevronDown,
  ChevronRight
} from 'lucide-vue-next'

interface OrgWhitelistEntry {
  id: string
  orgId: string
  ipRange: string
  description?: string
  createdAt: string
}

interface AppWhitelistEntry {
  id: string
  appId: string
  ipRange: string
  description?: string
  createdAt: string
}

interface Organization {
  id: string
  name: string
}

interface Application {
  id: string
  orgId: string
  subdomain: string
  name: string
}

const { get } = useApi()
const { organizations } = useOrganizations()

const orgWhitelists = ref<Record<string, OrgWhitelistEntry[]>>({})
const appWhitelists = ref<Record<string, AppWhitelistEntry[]>>({})
const applications = ref<Application[]>([])
const loading = ref(false)
const expandedOrgs = ref<Set<string>>(new Set())

async function loadWhitelists() {
  loading.value = true
  try {
    // Load all org whitelists
    const orgData = await get<{ entries: OrgWhitelistEntry[] }>('/admin/org-whitelists')
    
    // Group by org
    const byOrg: Record<string, OrgWhitelistEntry[]> = {}
    for (const entry of (orgData.entries || [])) {
      if (!byOrg[entry.orgId]) {
        byOrg[entry.orgId] = []
      }
      byOrg[entry.orgId]!.push(entry)
    }
    orgWhitelists.value = byOrg

    // Load all app whitelists
    const appData = await get<{ entries: AppWhitelistEntry[] }>('/admin/app-whitelists')
    
    // Group by app
    const byApp: Record<string, AppWhitelistEntry[]> = {}
    for (const entry of (appData.entries || [])) {
      if (!byApp[entry.appId]) {
        byApp[entry.appId] = []
      }
      byApp[entry.appId]!.push(entry)
    }
    appWhitelists.value = byApp
  } catch (err) {
    console.error('Failed to load whitelists:', err)
  } finally {
    loading.value = false
  }
}

async function loadApplications() {
  try {
    const data = await get<{ applications: Application[] }>('/admin/applications')
    applications.value = data.applications || []
  } catch (err) {
    console.error('Failed to load applications:', err)
  }
}

function formatDate(timestamp?: string) {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleDateString()
}

function getOrgName(orgId: string): string {
  const org = organizations.value.find(o => o.id === orgId)
  return org?.name || orgId
}

function getAppName(appId: string): string {
  const app = applications.value.find(a => a.id === appId)
  return app ? (app.name || app.subdomain) : appId
}

function getAppOrgId(appId: string): string {
  const app = applications.value.find(a => a.id === appId)
  return app?.orgId || ''
}

function toggleOrg(orgId: string) {
  if (expandedOrgs.value.has(orgId)) {
    expandedOrgs.value.delete(orgId)
  } else {
    expandedOrgs.value.add(orgId)
  }
  // Force reactivity
  expandedOrgs.value = new Set(expandedOrgs.value)
}

const totalOrgEntries = computed(() => {
  return Object.values(orgWhitelists.value).reduce((sum, entries) => sum + entries.length, 0)
})

const totalAppEntries = computed(() => {
  return Object.values(appWhitelists.value).reduce((sum, entries) => sum + entries.length, 0)
})

const orgIds = computed(() => {
  const ids = new Set<string>()
  // Add orgs with whitelist entries
  Object.keys(orgWhitelists.value).forEach(id => ids.add(id))
  // Add orgs with apps that have whitelist entries
  Object.keys(appWhitelists.value).forEach(appId => {
    const orgId = getAppOrgId(appId)
    if (orgId) ids.add(orgId)
  })
  return Array.from(ids)
})

function getOrgAppWhitelists(orgId: string): Record<string, AppWhitelistEntry[]> {
  const result: Record<string, AppWhitelistEntry[]> = {}
  for (const [appId, entries] of Object.entries(appWhitelists.value)) {
    if (getAppOrgId(appId) === orgId) {
      result[appId] = entries
    }
  }
  return result
}

onMounted(async () => {
  await loadApplications()
  await loadWhitelists()
  // Expand first org by default if any
  const firstOrgId = orgIds.value[0]
  if (firstOrgId) {
    expandedOrgs.value.add(firstOrgId)
  }
})
</script>

<template>
  <AppLayout>
    <!-- Page Header -->
    <div class="mb-8">
      <h1 class="font-[var(--font-display)] text-3xl font-semibold mb-2">IP Whitelist</h1>
      <p class="text-sm text-[var(--text-secondary)]">View IP whitelist entries across all organizations and applications</p>
    </div>

    <!-- Info Box -->
    <div class="info-box mb-6">
      <Info class="w-[18px] h-[18px] flex-shrink-0 mt-0.5" />
      <div>
        <strong>Hierarchical Whitelist:</strong> IP addresses can be whitelisted at the organization level (applies to all apps) 
        or at the application level (specific to that app). Manage whitelists through organization accounts.
      </div>
    </div>

    <!-- Stats -->
    <div class="grid grid-cols-3 gap-4 mb-6">
      <div class="p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg text-center">
        <div class="font-mono text-2xl font-semibold text-[var(--text-primary)]">{{ orgIds.length }}</div>
        <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">Organizations</div>
      </div>
      <div class="p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg text-center">
        <div class="font-mono text-2xl font-semibold text-[var(--text-primary)]">{{ totalOrgEntries }}</div>
        <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">Org Entries</div>
      </div>
      <div class="p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg text-center">
        <div class="font-mono text-2xl font-semibold text-[var(--text-primary)]">{{ totalAppEntries }}</div>
        <div class="text-xs text-[var(--text-muted)] uppercase tracking-wide">App Entries</div>
      </div>
    </div>

    <!-- Whitelist Card -->
    <div class="card">
      <div class="card-header">
        <h2 class="card-title">
          <ShieldCheck class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          All Whitelists
        </h2>
        <button 
          class="btn btn-secondary btn-sm"
          :disabled="loading"
          @click="loadWhitelists"
        >
          <RefreshCw 
            class="w-3.5 h-3.5" 
            :class="{ 'animate-spin': loading }"
          />
          Refresh
        </button>
      </div>
      <div class="card-body">
        <LoadingState v-if="loading && orgIds.length === 0" message="Loading whitelists..." />
        
        <EmptyState 
          v-else-if="orgIds.length === 0"
          :icon="ShieldOff"
          title="No whitelist entries"
          description="Organizations can manage their whitelist entries through the organization portal"
        />

        <div v-else class="space-y-4">
          <div 
            v-for="orgId in orgIds" 
            :key="orgId"
            class="border border-[var(--border-subtle)] rounded-lg overflow-hidden"
          >
            <!-- Org Header -->
            <button
              class="w-full flex items-center gap-3 p-4 bg-[var(--bg-deep)] hover:bg-[var(--bg-elevated)] transition-colors text-left"
              @click="toggleOrg(orgId)"
            >
              <ChevronRight 
                v-if="!expandedOrgs.has(orgId)"
                class="w-4 h-4 text-[var(--text-muted)]" 
              />
              <ChevronDown 
                v-else
                class="w-4 h-4 text-[var(--text-muted)]" 
              />
              <Building2 class="w-5 h-5 text-[var(--accent-copper)]" />
              <span class="font-medium">{{ getOrgName(orgId) }}</span>
              <span class="text-xs text-[var(--text-muted)] ml-auto">
                {{ (orgWhitelists[orgId]?.length || 0) }} org entries, 
                {{ (Object.values(getOrgAppWhitelists(orgId)) as AppWhitelistEntry[][]).reduce((sum, e) => sum + e.length, 0) }} app entries
              </span>
            </button>

            <!-- Org Content -->
            <div v-if="expandedOrgs.has(orgId)" class="border-t border-[var(--border-subtle)]">
              <!-- Org Whitelist -->
              <div v-if="orgWhitelists[orgId]?.length" class="p-4">
                <h4 class="text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide mb-2">
                  Organization Whitelist
                </h4>
                <div class="space-y-2">
                  <div
                    v-for="entry in orgWhitelists[orgId]"
                    :key="entry.id"
                    class="flex items-center justify-between p-2 bg-[var(--bg-base)] rounded"
                  >
                    <div class="flex items-center gap-3">
                      <code class="text-sm font-mono text-[var(--accent-emerald)]">{{ entry.ipRange }}</code>
                      <span v-if="entry.description" class="text-sm text-[var(--text-muted)]">{{ entry.description }}</span>
                    </div>
                    <span class="text-xs text-[var(--text-muted)]">{{ formatDate(entry.createdAt) }}</span>
                  </div>
                </div>
              </div>

              <!-- App Whitelists -->
              <div 
                v-for="(entries, appId) in getOrgAppWhitelists(orgId)" 
                :key="appId"
                class="p-4 border-t border-[var(--border-subtle)]"
              >
                <h4 class="text-xs font-medium text-[var(--text-muted)] uppercase tracking-wide mb-2 flex items-center gap-2">
                  <AppWindow class="w-3 h-3" />
                  {{ getAppName(appId) }}
                </h4>
                <div class="space-y-2">
                  <div
                    v-for="entry in entries"
                    :key="entry.id"
                    class="flex items-center justify-between p-2 bg-[var(--bg-base)] rounded"
                  >
                    <div class="flex items-center gap-3">
                      <code class="text-sm font-mono text-[var(--accent-emerald)]">{{ entry.ipRange }}</code>
                      <span v-if="entry.description" class="text-sm text-[var(--text-muted)]">{{ entry.description }}</span>
                    </div>
                    <span class="text-xs text-[var(--text-muted)]">{{ formatDate(entry.createdAt) }}</span>
                  </div>
                </div>
              </div>

              <!-- Empty state for org with no entries -->
              <div 
                v-if="!orgWhitelists[orgId]?.length && Object.keys(getOrgAppWhitelists(orgId)).length === 0"
                class="p-4 text-center text-sm text-[var(--text-muted)]"
              >
                No whitelist entries for this organization
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </AppLayout>
</template>
