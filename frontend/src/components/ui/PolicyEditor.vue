<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Key, Lock, Globe, Plus, X, Info } from 'lucide-vue-next'
import type { AuthType, SetPolicyRequest } from '@/types/api'

const props = defineProps<{
  initialPolicy?: {
    authType?: AuthType
    oidcIssuerUrl?: string
    oidcClientId?: string
    oidcScopes?: string[]
    oidcAllowedDomains?: string[]
  } | null
}>()

const emit = defineEmits<{
  submit: [policy: SetPolicyRequest]
  cancel: []
}>()

// Form state
const authType = ref<AuthType>(props.initialPolicy?.authType || 'basic')

// Basic auth fields
const basicUsername = ref('')
const basicPassword = ref('')

// OIDC fields
const oidcIssuerUrl = ref(props.initialPolicy?.oidcIssuerUrl || '')
const oidcClientId = ref(props.initialPolicy?.oidcClientId || '')
const oidcClientSecret = ref('')
const oidcScopes = ref<string[]>(props.initialPolicy?.oidcScopes || ['openid', 'email', 'profile'])
const oidcAllowedDomains = ref<string[]>(props.initialPolicy?.oidcAllowedDomains || [])
const newScope = ref('')
const newDomain = ref('')

const authTypes: { value: AuthType; label: string; icon: typeof Key; description: string }[] = [
  { 
    value: 'basic', 
    label: 'Basic Auth', 
    icon: Lock,
    description: 'Username and password authentication'
  },
  { 
    value: 'api_key', 
    label: 'API Key', 
    icon: Key,
    description: 'Authenticate using API keys'
  },
  { 
    value: 'oidc', 
    label: 'OIDC', 
    icon: Globe,
    description: 'OpenID Connect single sign-on'
  },
]

const isValid = computed(() => {
  switch (authType.value) {
    case 'basic':
      return basicUsername.value.length >= 8 && basicPassword.value.length >= 8
    case 'api_key':
      return true // API keys don't need configuration
    case 'oidc':
      return oidcIssuerUrl.value.length > 0 && oidcClientId.value.length > 0
    default:
      return false
  }
})

function addScope() {
  const scope = newScope.value.trim().toLowerCase()
  if (scope && !oidcScopes.value.includes(scope)) {
    oidcScopes.value.push(scope)
    newScope.value = ''
  }
}

function removeScope(scope: string) {
  oidcScopes.value = oidcScopes.value.filter(s => s !== scope)
}

function addDomain() {
  const domain = newDomain.value.trim().toLowerCase()
  if (domain && !oidcAllowedDomains.value.includes(domain)) {
    oidcAllowedDomains.value.push(domain)
    newDomain.value = ''
  }
}

function removeDomain(domain: string) {
  oidcAllowedDomains.value = oidcAllowedDomains.value.filter(d => d !== domain)
}

function handleSubmit() {
  const policy: SetPolicyRequest = {
    authType: authType.value
  }

  if (authType.value === 'basic') {
    policy.basicUsername = basicUsername.value
    policy.basicPassword = basicPassword.value
  } else if (authType.value === 'oidc') {
    policy.oidcIssuerUrl = oidcIssuerUrl.value
    policy.oidcClientId = oidcClientId.value
    if (oidcClientSecret.value) {
      policy.oidcClientSecret = oidcClientSecret.value
    }
    if (oidcScopes.value.length > 0) {
      policy.oidcScopes = oidcScopes.value
    }
    if (oidcAllowedDomains.value.length > 0) {
      policy.oidcAllowedDomains = oidcAllowedDomains.value
    }
  }

  emit('submit', policy)
}

function handleCancel() {
  emit('cancel')
}
</script>

<template>
  <form @submit.prevent="handleSubmit" class="flex flex-col gap-6">
    <!-- Auth type selector -->
    <div class="flex flex-col gap-2">
      <label class="text-xs font-medium uppercase tracking-wider text-text-secondary flex items-center gap-2">Authentication Type</label>
      <div class="flex flex-col gap-2">
        <button
          v-for="type in authTypes"
          :key="type.value"
          type="button"
          class="flex items-center gap-3.5 py-4 px-5 bg-bg-deep border rounded-[10px] cursor-pointer transition-all duration-200 text-left"
          :class="authType === type.value 
            ? 'border-accent-primary bg-[rgba(var(--accent-primary-rgb),0.05)] text-accent-primary' 
            : 'border-border-subtle text-text-secondary hover:border-border-accent hover:bg-bg-elevated'"
          @click="authType = type.value"
        >
          <component :is="type.icon" class="w-5 h-5" />
          <div class="flex flex-col gap-0.5">
            <span 
              class="font-medium"
              :class="authType === type.value ? 'text-accent-primary' : 'text-text-primary'"
            >{{ type.label }}</span>
            <span class="text-[0.8125rem] text-text-muted">{{ type.description }}</span>
          </div>
        </button>
      </div>
    </div>

    <!-- Basic Auth fields -->
    <template v-if="authType === 'basic'">
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium uppercase tracking-wider text-text-secondary" for="basic-username">Username</label>
        <input
          id="basic-username"
          v-model="basicUsername"
          type="text"
          class="form-input"
          placeholder="Enter username (min 8 characters)"
          autocomplete="username"
        />
        <p class="text-xs text-text-muted m-0">Must be at least 8 characters</p>
      </div>
      
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium uppercase tracking-wider text-text-secondary" for="basic-password">Password</label>
        <input
          id="basic-password"
          v-model="basicPassword"
          type="password"
          class="form-input"
          placeholder="Enter password (min 8 characters)"
          autocomplete="new-password"
        />
        <p class="text-xs text-text-muted m-0">Must be at least 8 characters</p>
      </div>
    </template>

    <!-- API Key info -->
    <template v-if="authType === 'api_key'">
      <div class="info-box leading-relaxed">
        <Info class="w-4 h-4 shrink-0" />
        <span>API key authentication is automatically enabled. Create and manage API keys in the API Keys section.</span>
      </div>
    </template>

    <!-- OIDC fields -->
    <template v-if="authType === 'oidc'">
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium uppercase tracking-wider text-text-secondary" for="oidc-issuer">Issuer URL</label>
        <input
          id="oidc-issuer"
          v-model="oidcIssuerUrl"
          type="url"
          class="form-input"
          placeholder="https://accounts.google.com"
        />
        <p class="text-xs text-text-muted m-0">The OIDC provider's issuer URL</p>
      </div>
      
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium uppercase tracking-wider text-text-secondary" for="oidc-client-id">Client ID</label>
        <input
          id="oidc-client-id"
          v-model="oidcClientId"
          type="text"
          class="form-input"
          placeholder="your-client-id"
        />
      </div>
      
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium uppercase tracking-wider text-text-secondary flex items-center gap-2" for="oidc-client-secret">
          Client Secret
          <span class="font-normal normal-case tracking-normal text-text-muted">(optional)</span>
        </label>
        <input
          id="oidc-client-secret"
          v-model="oidcClientSecret"
          type="password"
          class="form-input"
          placeholder="Leave blank to keep existing"
        />
      </div>
      
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium uppercase tracking-wider text-text-secondary">Scopes</label>
        <div class="flex flex-wrap gap-2 p-2 bg-bg-deep border border-border-subtle rounded-xs min-h-12 items-center">
          <div 
            v-for="scope in oidcScopes" 
            :key="scope" 
            class="flex items-center gap-1 py-1 pl-3 pr-2 bg-bg-elevated border border-border-subtle rounded-xs text-[0.8125rem] text-text-primary"
          >
            <span>{{ scope }}</span>
            <button 
              type="button" 
              class="w-[18px] h-[18px] flex items-center justify-center border-none rounded bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-accent-red hover:text-white"
              @click="removeScope(scope)"
            >
              <X class="w-3 h-3" />
            </button>
          </div>
          <input
            v-model="newScope"
            type="text"
            class="flex-1 min-w-[100px] py-1 px-2 bg-transparent border-none font-body text-sm text-text-primary outline-none placeholder:text-text-muted"
            placeholder="Add scope..."
            @keydown.enter.prevent="addScope"
          />
          <button 
            type="button" 
            class="w-7 h-7 flex items-center justify-center border border-border-subtle rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary hover:border-border-accent"
            @click="addScope"
          >
            <Plus class="w-4 h-4" />
          </button>
        </div>
      </div>
      
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium uppercase tracking-wider text-text-secondary flex items-center gap-2">
          Allowed Email Domains
          <span class="font-normal normal-case tracking-normal text-text-muted">(optional)</span>
        </label>
        <div class="flex flex-wrap gap-2 p-2 bg-bg-deep border border-border-subtle rounded-xs min-h-12 items-center">
          <div 
            v-for="domain in oidcAllowedDomains" 
            :key="domain" 
            class="flex items-center gap-1 py-1 pl-3 pr-2 bg-bg-elevated border border-border-subtle rounded-xs text-[0.8125rem] text-text-primary"
          >
            <span>{{ domain }}</span>
            <button 
              type="button" 
              class="w-[18px] h-[18px] flex items-center justify-center border-none rounded bg-transparent text-text-muted cursor-pointer transition-all duration-150 hover:bg-accent-red hover:text-white"
              @click="removeDomain(domain)"
            >
              <X class="w-3 h-3" />
            </button>
          </div>
          <input
            v-model="newDomain"
            type="text"
            class="flex-1 min-w-[100px] py-1 px-2 bg-transparent border-none font-body text-sm text-text-primary outline-none placeholder:text-text-muted"
            placeholder="example.com"
            @keydown.enter.prevent="addDomain"
          />
          <button 
            type="button" 
            class="w-7 h-7 flex items-center justify-center border border-border-subtle rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary hover:border-border-accent"
            @click="addDomain"
          >
            <Plus class="w-4 h-4" />
          </button>
        </div>
        <p class="text-xs text-text-muted m-0">Leave empty to allow all domains</p>
      </div>
    </template>

    <!-- Actions -->
    <div class="flex justify-end gap-3 pt-4 border-t border-border-subtle mt-2">
      <button type="button" class="btn btn-secondary" @click="handleCancel">
        Cancel
      </button>
      <button type="submit" class="btn btn-primary" :disabled="!isValid">
        Save Policy
      </button>
    </div>
  </form>
</template>
