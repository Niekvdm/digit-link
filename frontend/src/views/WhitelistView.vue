<script setup lang="ts">
import { ref } from 'vue'
import { useWhitelist } from '@/composables'
import AppLayout from '@/components/layout/AppLayout.vue'
import { LoadingState, EmptyState } from '@/components/shared'
import { 
  ShieldCheck, 
  ShieldOff,
  PlusCircle, 
  RefreshCw, 
  Trash2,
  Info
} from 'lucide-vue-next'

const { 
  entries, 
  loading, 
  refresh,
  addEntry,
  removeEntry
} = useWhitelist()

const ipRange = ref('')
const description = ref('')
const addLoading = ref(false)

function formatDate(timestamp?: string) {
  if (!timestamp) return ''
  return new Date(timestamp).toLocaleDateString()
}

async function handleAdd(e: Event) {
  e.preventDefault()
  
  const trimmedIp = ipRange.value.trim()
  if (!trimmedIp) return
  
  addLoading.value = true
  const result = await addEntry({
    ipRange: trimmedIp,
    description: description.value.trim() || undefined
  })
  addLoading.value = false

  if (result.success) {
    ipRange.value = ''
    description.value = ''
  } else {
    alert(result.error || 'Failed to add IP')
  }
}

async function handleRemove(entryId: string) {
  if (!confirm('Remove this IP from the whitelist? Clients from this IP will no longer be able to connect.')) return
  
  const result = await removeEntry(entryId)
  
  if (!result.success) {
    alert(result.error || 'Failed to remove IP')
  }
}
</script>

<template>
  <AppLayout>
    <!-- Page Header -->
    <div class="mb-8">
      <h1 class="font-[var(--font-display)] text-3xl font-semibold mb-2">IP Whitelist</h1>
      <p class="text-sm text-[var(--text-secondary)]">Manage allowed IP addresses for tunnel connections</p>
    </div>

    <!-- Info Box -->
    <div class="info-box mb-6">
      <Info class="w-[18px] h-[18px] flex-shrink-0 mt-0.5" />
      <div>
        <strong>Strict Mode Active:</strong> Only clients connecting from whitelisted IP addresses can establish tunnels. 
        Add IP addresses or CIDR ranges (e.g., <code class="bg-black/20 px-1.5 py-0.5 rounded text-[0.8125rem]">192.168.1.0/24</code>) to allow connections.
      </div>
    </div>

    <!-- Add IP Card -->
    <div class="card mb-6">
      <div class="card-header">
        <h2 class="card-title">
          <PlusCircle class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          Add IP Address
        </h2>
      </div>
      <div class="card-body">
        <form 
          class="flex gap-4 p-6 bg-[var(--bg-deep)] rounded-lg"
          @submit="handleAdd"
        >
          <div class="flex-1">
            <label class="form-label" for="ipRange">IP Address or CIDR Range</label>
            <input
              id="ipRange"
              v-model="ipRange"
              type="text"
              class="form-input form-input-mono"
              placeholder="e.g., 192.168.1.100 or 10.0.0.0/8"
              required
            />
            <p class="form-hint">Single IP or CIDR notation for ranges</p>
          </div>
          <div class="flex-1">
            <label class="form-label" for="description">Description (Optional)</label>
            <input
              id="description"
              v-model="description"
              type="text"
              class="form-input"
              placeholder="e.g., Office network"
            />
          </div>
          <button 
            type="submit"
            class="btn btn-primary self-end"
            :class="{ 'btn-loading': addLoading }"
            :disabled="addLoading || !ipRange.trim()"
          >
            <span class="btn-text flex items-center gap-2">
              <PlusCircle class="w-4 h-4" />
              Add
            </span>
          </button>
        </form>
      </div>
    </div>

    <!-- Whitelist Table Card -->
    <div class="card">
      <div class="card-header">
        <h2 class="card-title">
          <ShieldCheck class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
          Whitelisted IPs
        </h2>
        <button 
          class="btn btn-secondary btn-sm"
          :disabled="loading"
          @click="refresh"
        >
          <RefreshCw 
            class="w-3.5 h-3.5" 
            :class="{ 'animate-spin': loading }"
          />
          Refresh
        </button>
      </div>
      <div class="p-0">
        <LoadingState v-if="loading && !entries.length" message="Loading whitelist..." />
        
        <EmptyState 
          v-else-if="!entries.length"
          :icon="ShieldOff"
          title="No IPs whitelisted yet"
          description="Add an IP address above to allow connections"
        />

        <table v-else class="w-full">
          <thead>
            <tr class="bg-[var(--bg-deep)]">
              <th class="text-left py-3.5 px-4 text-xs font-medium uppercase tracking-wider text-[var(--text-secondary)]">
                IP / Range
              </th>
              <th class="text-left py-3.5 px-4 text-xs font-medium uppercase tracking-wider text-[var(--text-secondary)]">
                Description
              </th>
              <th class="text-left py-3.5 px-4 text-xs font-medium uppercase tracking-wider text-[var(--text-secondary)]">
                Added
              </th>
              <th class="w-[60px]"></th>
            </tr>
          </thead>
          <tbody>
            <tr 
              v-for="entry in entries"
              :key="entry.id"
              class="border-t border-[var(--border-subtle)] hover:bg-[var(--bg-elevated)] transition-colors"
            >
              <td class="py-3.5 px-4 font-mono text-sm text-[var(--accent-emerald)]">
                {{ entry.ipRange }}
              </td>
              <td class="py-3.5 px-4 text-sm text-[var(--text-secondary)]">
                {{ entry.description || '—' }}
              </td>
              <td class="py-3.5 px-4 text-xs text-[var(--text-muted)]">
                {{ formatDate(entry.createdAt) }}
              </td>
              <td class="py-3.5 px-4">
                <button 
                  class="btn btn-danger btn-icon btn-sm"
                  title="Remove"
                  @click="handleRemove(entry.id)"
                >
                  <Trash2 class="w-4 h-4" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </AppLayout>
</template>
