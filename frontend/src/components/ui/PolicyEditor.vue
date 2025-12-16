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
  <form @submit.prevent="handleSubmit" class="policy-editor">
    <!-- Auth type selector -->
    <div class="form-group">
      <label class="form-label">Authentication Type</label>
      <div class="auth-type-grid">
        <button
          v-for="type in authTypes"
          :key="type.value"
          type="button"
          class="auth-type-option"
          :class="{ 'auth-type-option--active': authType === type.value }"
          @click="authType = type.value"
        >
          <component :is="type.icon" class="w-5 h-5" />
          <div class="auth-type-text">
            <span class="auth-type-label">{{ type.label }}</span>
            <span class="auth-type-desc">{{ type.description }}</span>
          </div>
        </button>
      </div>
    </div>

    <!-- Basic Auth fields -->
    <template v-if="authType === 'basic'">
      <div class="form-group">
        <label class="form-label" for="basic-username">Username</label>
        <input
          id="basic-username"
          v-model="basicUsername"
          type="text"
          class="form-input"
          placeholder="Enter username (min 8 characters)"
          autocomplete="username"
        />
        <p class="form-hint">Must be at least 8 characters</p>
      </div>
      
      <div class="form-group">
        <label class="form-label" for="basic-password">Password</label>
        <input
          id="basic-password"
          v-model="basicPassword"
          type="password"
          class="form-input"
          placeholder="Enter password (min 8 characters)"
          autocomplete="new-password"
        />
        <p class="form-hint">Must be at least 8 characters</p>
      </div>
    </template>

    <!-- API Key info -->
    <template v-if="authType === 'api_key'">
      <div class="info-box">
        <Info class="w-4 h-4 flex-shrink-0" />
        <span>API key authentication is automatically enabled. Create and manage API keys in the API Keys section.</span>
      </div>
    </template>

    <!-- OIDC fields -->
    <template v-if="authType === 'oidc'">
      <div class="form-group">
        <label class="form-label" for="oidc-issuer">Issuer URL</label>
        <input
          id="oidc-issuer"
          v-model="oidcIssuerUrl"
          type="url"
          class="form-input"
          placeholder="https://accounts.google.com"
        />
        <p class="form-hint">The OIDC provider's issuer URL</p>
      </div>
      
      <div class="form-group">
        <label class="form-label" for="oidc-client-id">Client ID</label>
        <input
          id="oidc-client-id"
          v-model="oidcClientId"
          type="text"
          class="form-input"
          placeholder="your-client-id"
        />
      </div>
      
      <div class="form-group">
        <label class="form-label" for="oidc-client-secret">
          Client Secret
          <span class="form-label-optional">(optional)</span>
        </label>
        <input
          id="oidc-client-secret"
          v-model="oidcClientSecret"
          type="password"
          class="form-input"
          placeholder="Leave blank to keep existing"
        />
      </div>
      
      <div class="form-group">
        <label class="form-label">Scopes</label>
        <div class="tag-input">
          <div v-for="scope in oidcScopes" :key="scope" class="tag">
            <span>{{ scope }}</span>
            <button type="button" class="tag-remove" @click="removeScope(scope)">
              <X class="w-3 h-3" />
            </button>
          </div>
          <input
            v-model="newScope"
            type="text"
            class="tag-field"
            placeholder="Add scope..."
            @keydown.enter.prevent="addScope"
          />
          <button type="button" class="tag-add" @click="addScope">
            <Plus class="w-4 h-4" />
          </button>
        </div>
      </div>
      
      <div class="form-group">
        <label class="form-label">
          Allowed Email Domains
          <span class="form-label-optional">(optional)</span>
        </label>
        <div class="tag-input">
          <div v-for="domain in oidcAllowedDomains" :key="domain" class="tag">
            <span>{{ domain }}</span>
            <button type="button" class="tag-remove" @click="removeDomain(domain)">
              <X class="w-3 h-3" />
            </button>
          </div>
          <input
            v-model="newDomain"
            type="text"
            class="tag-field"
            placeholder="example.com"
            @keydown.enter.prevent="addDomain"
          />
          <button type="button" class="tag-add" @click="addDomain">
            <Plus class="w-4 h-4" />
          </button>
        </div>
        <p class="form-hint">Leave empty to allow all domains</p>
      </div>
    </template>

    <!-- Actions -->
    <div class="form-actions">
      <button type="button" class="btn btn-secondary" @click="handleCancel">
        Cancel
      </button>
      <button type="submit" class="btn btn-primary" :disabled="!isValid">
        Save Policy
      </button>
    </div>
  </form>
</template>

<style scoped>
.policy-editor {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.form-label {
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.form-label-optional {
  font-weight: 400;
  text-transform: none;
  letter-spacing: normal;
  color: var(--text-muted);
}

.form-hint {
  font-size: 0.75rem;
  color: var(--text-muted);
  margin: 0;
}

/* Auth type selector */
.auth-type-grid {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.auth-type-option {
  display: flex;
  align-items: center;
  gap: 0.875rem;
  padding: 1rem 1.25rem;
  background: var(--bg-deep);
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: left;
  color: var(--text-secondary);
}

.auth-type-option:hover {
  border-color: var(--border-accent);
  background: var(--bg-elevated);
}

.auth-type-option--active {
  border-color: var(--accent-primary);
  background: rgba(var(--accent-primary-rgb), 0.05);
  color: var(--accent-primary);
}

.auth-type-text {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
}

.auth-type-label {
  font-weight: 500;
  color: var(--text-primary);
}

.auth-type-option--active .auth-type-label {
  color: var(--accent-primary);
}

.auth-type-desc {
  font-size: 0.8125rem;
  color: var(--text-muted);
}

/* Tag input */
.tag-input {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  padding: 0.5rem;
  background: var(--bg-deep);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  min-height: 48px;
  align-items: center;
}

.tag {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem 0.25rem 0.75rem;
  background: var(--bg-elevated);
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  font-size: 0.8125rem;
  color: var(--text-primary);
}

.tag-remove {
  width: 18px;
  height: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s ease;
}

.tag-remove:hover {
  background: var(--accent-red);
  color: white;
}

.tag-field {
  flex: 1;
  min-width: 100px;
  padding: 0.25rem 0.5rem;
  background: transparent;
  border: none;
  font-family: var(--font-body);
  font-size: 0.875rem;
  color: var(--text-primary);
  outline: none;
}

.tag-field::placeholder {
  color: var(--text-muted);
}

.tag-add {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.tag-add:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
  border-color: var(--border-accent);
}

/* Info box */
.info-box {
  display: flex;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  background: rgba(var(--accent-blue-rgb), 0.1);
  border: 1px solid rgba(var(--accent-blue-rgb), 0.3);
  border-radius: 8px;
  font-size: 0.875rem;
  color: var(--accent-blue);
  line-height: 1.5;
}

/* Form actions */
.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-subtle);
  margin-top: 0.5rem;
}
</style>
