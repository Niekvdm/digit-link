<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useAuthStore } from '@/stores/authStore'
import OrgLayout from '@/components/layout/OrgLayout.vue'
import { BaseModal, LoadingState } from '@/components/shared'
import type { OrgAuthPolicy, AuthType, SetPolicyRequest } from '@/types/api'
import {
  Settings,
  Shield,
  Key,
  Globe,
  Check,
  AlertTriangle
} from 'lucide-vue-next'

const authStore = useAuthStore()

const loading = ref(false)
const hasPolicy = ref(false)
const currentPolicy = ref<OrgAuthPolicy | null>(null)

// Policy configuration
const showPolicyModal = ref(false)
const policyAuthType = ref<AuthType>('basic')
const basicUsername = ref('')
const basicPassword = ref('')
const oidcIssuerUrl = ref('')
const oidcClientId = ref('')
const oidcClientSecret = ref('')
const oidcScopes = ref('openid,email,profile')
const oidcAllowedDomains = ref('')
const policyLoading = ref(false)

async function loadPolicy() {
  loading.value = true
  try {
    const response = await fetch('/org/policy', {
      headers: {
        'Authorization': `Bearer ${authStore.token}`
      }
    })
    if (response.ok) {
      const data = await response.json()
      currentPolicy.value = data.policy
      hasPolicy.value = !!data.policy
    }
  } catch (err) {
    console.error('Failed to load policy:', err)
  } finally {
    loading.value = false
  }
}

function getAuthTypeIcon(authType: AuthType) {
  switch (authType) {
    case 'basic': return Key
    case 'api_key': return Key
    case 'oidc': return Globe
    default: return Key
  }
}

function getAuthTypeLabel(authType: string): string {
  switch (authType) {
    case 'basic': return 'Basic Auth'
    case 'api_key': return 'API Key'
    case 'oidc': return 'OIDC / SSO'
    default: return authType
  }
}

function openPolicyModal() {
  policyAuthType.value = currentPolicy.value?.authType || 'basic'
  basicUsername.value = ''
  basicPassword.value = ''
  oidcIssuerUrl.value = currentPolicy.value?.oidcIssuerUrl || ''
  oidcClientId.value = currentPolicy.value?.oidcClientId || ''
  oidcClientSecret.value = ''
  oidcScopes.value = currentPolicy.value?.oidcScopes?.join(',') || 'openid,email,profile'
  oidcAllowedDomains.value = currentPolicy.value?.oidcAllowedDomains?.join(',') || ''
  showPolicyModal.value = true
}

async function handleSavePolicy() {
  const policy: SetPolicyRequest = {
    authType: policyAuthType.value
  }

  if (policyAuthType.value === 'basic') {
    if (!basicUsername.value || !basicPassword.value) {
      alert('Username and password are required for basic auth')
      return
    }
    policy.basicUsername = basicUsername.value
    policy.basicPassword = basicPassword.value
  } else if (policyAuthType.value === 'oidc') {
    if (!oidcIssuerUrl.value || !oidcClientId.value) {
      alert('Issuer URL and Client ID are required for OIDC')
      return
    }
    policy.oidcIssuerUrl = oidcIssuerUrl.value
    policy.oidcClientId = oidcClientId.value
    policy.oidcClientSecret = oidcClientSecret.value || undefined
    policy.oidcScopes = oidcScopes.value.split(',').map(s => s.trim()).filter(Boolean)
    policy.oidcAllowedDomains = oidcAllowedDomains.value
      ? oidcAllowedDomains.value.split(',').map(s => s.trim()).filter(Boolean)
      : undefined
  }

  policyLoading.value = true
  try {
    const response = await fetch('/org/policy', {
      method: 'PUT',
      headers: {
        'Authorization': `Bearer ${authStore.token}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(policy)
    })

    const data = await response.json()
    if (!response.ok || !data.success) {
      alert(data.error || 'Failed to save policy')
      return
    }

    showPolicyModal.value = false
    await loadPolicy()
  } catch (err) {
    alert('Failed to save policy')
  } finally {
    policyLoading.value = false
  }
}

onMounted(() => {
  loadPolicy()
})
</script>

<template>
  <OrgLayout>
    <!-- Page Header -->
    <div class="mb-8">
      <h1 class="font-[var(--font-display)] text-3xl font-semibold mb-1">Settings</h1>
      <p class="text-sm text-[var(--text-secondary)]">Configure your organization's authentication policy</p>
    </div>

    <LoadingState v-if="loading" message="Loading settings..." />

    <template v-else>
      <!-- Auth Policy Card -->
      <div class="card">
        <div class="card-header">
          <h2 class="card-title">
            <Shield class="w-[18px] h-[18px] text-[var(--text-secondary)]" />
            Organization Auth Policy
          </h2>
          <button class="btn btn-primary btn-sm" @click="openPolicyModal">
            <Settings class="w-3.5 h-3.5" />
            {{ hasPolicy ? 'Edit Policy' : 'Configure Policy' }}
          </button>
        </div>
        <div class="card-body">
          <div v-if="hasPolicy && currentPolicy" class="space-y-4">
            <div class="flex items-center gap-4 p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg">
              <div class="w-12 h-12 rounded-xl bg-[rgba(74,159,126,0.15)] text-[var(--accent-emerald)] flex items-center justify-center">
                <component :is="getAuthTypeIcon(currentPolicy.authType)" class="w-6 h-6" />
              </div>
              <div>
                <div class="font-medium text-[var(--text-primary)]">{{ getAuthTypeLabel(currentPolicy.authType) }}</div>
                <div class="text-sm text-[var(--text-muted)]">
                  <template v-if="currentPolicy.authType === 'oidc'">
                    Issuer: {{ currentPolicy.oidcIssuerUrl }}
                  </template>
                  <template v-else-if="currentPolicy.authType === 'basic'">
                    Basic authentication configured
                  </template>
                  <template v-else>
                    API Key authentication
                  </template>
                </div>
              </div>
            </div>

            <div v-if="currentPolicy.authType === 'oidc'" class="grid grid-cols-2 gap-4 text-sm">
              <div>
                <div class="text-[var(--text-muted)] mb-1">Client ID</div>
                <div class="font-mono text-[var(--text-secondary)]">{{ currentPolicy.oidcClientId }}</div>
              </div>
              <div>
                <div class="text-[var(--text-muted)] mb-1">Scopes</div>
                <div class="font-mono text-[var(--text-secondary)]">{{ currentPolicy.oidcScopes?.join(', ') || 'openid, email, profile' }}</div>
              </div>
              <div v-if="currentPolicy.oidcAllowedDomains?.length">
                <div class="text-[var(--text-muted)] mb-1">Allowed Domains</div>
                <div class="font-mono text-[var(--text-secondary)]">{{ currentPolicy.oidcAllowedDomains.join(', ') }}</div>
              </div>
            </div>
          </div>

          <div v-else class="text-center py-8">
            <Shield class="w-12 h-12 mx-auto mb-4 text-[var(--text-muted)]" />
            <div class="text-[var(--text-secondary)] mb-2">No authentication policy configured</div>
            <div class="text-sm text-[var(--text-muted)] mb-4">
              Configure an org-level auth policy. Apps set to "Inherit" will use this policy.
            </div>
            <button class="btn btn-primary" @click="openPolicyModal">
              <Settings class="w-4 h-4" />
              Configure Policy
            </button>
          </div>
        </div>
      </div>

      <!-- Info Card -->
      <div class="card mt-6">
        <div class="card-body">
          <div class="flex items-start gap-3">
            <AlertTriangle class="w-5 h-5 text-[var(--accent-copper)] flex-shrink-0 mt-0.5" />
            <div class="text-sm text-[var(--text-secondary)]">
              <p class="mb-2">
                <strong>Organization Auth Policy</strong> serves as the default authentication for all applications in your organization.
              </p>
              <ul class="list-disc list-inside space-y-1 text-[var(--text-muted)]">
                <li>Apps with <strong>Inherit</strong> mode will use this policy</li>
                <li>Apps with <strong>Custom</strong> mode can override with their own policy</li>
                <li>Apps with <strong>Disabled</strong> mode have no authentication</li>
              </ul>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Policy Configuration Modal -->
    <BaseModal
      :show="showPolicyModal"
      title="Configure Organization Auth Policy"
      @close="showPolicyModal = false"
    >
      <div class="space-y-4">
        <!-- Auth Type Selection -->
        <div>
          <label class="form-label">Authentication Type</label>
          <div class="grid grid-cols-3 gap-2">
            <button
              v-for="type in (['basic', 'api_key', 'oidc'] as AuthType[])"
              :key="type"
              type="button"
              class="p-3 rounded-lg border text-center transition-all"
              :class="policyAuthType === type
                ? 'border-[var(--accent-emerald)] bg-[rgba(74,159,126,0.1)] text-[var(--accent-emerald)]'
                : 'border-[var(--border-subtle)] text-[var(--text-secondary)] hover:border-[var(--border-accent)]'"
              @click="policyAuthType = type"
            >
              <component :is="getAuthTypeIcon(type)" class="w-5 h-5 mx-auto mb-1" />
              <div class="text-xs font-medium">{{ type === 'api_key' ? 'API Key' : type === 'oidc' ? 'OIDC' : 'Basic' }}</div>
            </button>
          </div>
        </div>

        <!-- Basic Auth Fields -->
        <div v-if="policyAuthType === 'basic'" class="space-y-4">
          <div>
            <label class="form-label" for="basicUsername">Username</label>
            <input
              id="basicUsername"
              v-model="basicUsername"
              type="text"
              class="form-input"
              placeholder="Enter username (min 8 chars)"
            />
          </div>
          <div>
            <label class="form-label" for="basicPassword">Password</label>
            <input
              id="basicPassword"
              v-model="basicPassword"
              type="password"
              class="form-input"
              placeholder="Enter password (min 8 chars)"
            />
          </div>
        </div>

        <!-- API Key Info -->
        <div v-else-if="policyAuthType === 'api_key'" class="p-4 bg-[var(--bg-deep)] border border-[var(--border-subtle)] rounded-lg">
          <p class="text-sm text-[var(--text-secondary)]">
            API Key authentication uses the X-API-Key header. Create API keys in the API Keys section.
          </p>
        </div>

        <!-- OIDC Fields -->
        <div v-else-if="policyAuthType === 'oidc'" class="space-y-4">
          <div>
            <label class="form-label" for="oidcIssuerUrl">Issuer URL</label>
            <input
              id="oidcIssuerUrl"
              v-model="oidcIssuerUrl"
              type="url"
              class="form-input"
              placeholder="https://accounts.google.com"
            />
          </div>
          <div>
            <label class="form-label" for="oidcClientId">Client ID</label>
            <input
              id="oidcClientId"
              v-model="oidcClientId"
              type="text"
              class="form-input"
              placeholder="your-client-id"
            />
          </div>
          <div>
            <label class="form-label" for="oidcClientSecret">Client Secret</label>
            <input
              id="oidcClientSecret"
              v-model="oidcClientSecret"
              type="password"
              class="form-input"
              placeholder="your-client-secret"
            />
          </div>
          <div>
            <label class="form-label" for="oidcScopes">Scopes</label>
            <input
              id="oidcScopes"
              v-model="oidcScopes"
              type="text"
              class="form-input"
              placeholder="openid,email,profile"
            />
          </div>
          <div>
            <label class="form-label" for="oidcAllowedDomains">Allowed Email Domains (optional)</label>
            <input
              id="oidcAllowedDomains"
              v-model="oidcAllowedDomains"
              type="text"
              class="form-input"
              placeholder="example.com,company.org"
            />
            <p class="form-hint">Comma-separated list of allowed email domains</p>
          </div>
        </div>
      </div>

      <template #footer>
        <button class="btn btn-secondary" @click="showPolicyModal = false">
          Cancel
        </button>
        <button
          class="btn btn-primary"
          :class="{ 'btn-loading': policyLoading }"
          :disabled="policyLoading"
          @click="handleSavePolicy"
        >
          <span class="btn-text flex items-center gap-2">
            <Check class="w-4 h-4" />
            Save Policy
          </span>
        </button>
      </template>
    </BaseModal>
  </OrgLayout>
</template>
