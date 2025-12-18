<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Shield, Info, RotateCcw } from 'lucide-vue-next'
import type { AppRateLimitConfig, SetRateLimitRequest } from '@/types/api'

const props = defineProps<{
  config: AppRateLimitConfig | null
  defaults: {
    maxAttempts: number
    windowDurationSeconds: number
    blockDurationSeconds: number
  }
  loading?: boolean
}>()

const emit = defineEmits<{
  submit: [config: SetRateLimitRequest]
  reset: []
  cancel: []
}>()

// Form state
const enabled = ref(true)
const maxAttempts = ref(10)
const windowDuration = ref(900)
const blockDuration = ref(1800)

// Window duration options (in seconds)
const windowDurationOptions = [
  { value: 60, label: '1 minute' },
  { value: 300, label: '5 minutes' },
  { value: 900, label: '15 minutes' },
  { value: 1800, label: '30 minutes' },
  { value: 3600, label: '1 hour' },
]

// Block duration options (in seconds)
const blockDurationOptions = [
  { value: 300, label: '5 minutes' },
  { value: 900, label: '15 minutes' },
  { value: 1800, label: '30 minutes' },
  { value: 3600, label: '1 hour' },
  { value: 86400, label: '24 hours' },
]

// Initialize form from props
watch(() => props.config, (config) => {
  if (config) {
    enabled.value = config.enabled
    maxAttempts.value = config.maxAttempts
    windowDuration.value = config.windowDurationSeconds
    blockDuration.value = config.blockDurationSeconds
  } else {
    // Use defaults
    enabled.value = true
    maxAttempts.value = props.defaults.maxAttempts
    windowDuration.value = props.defaults.windowDurationSeconds
    blockDuration.value = props.defaults.blockDurationSeconds
  }
}, { immediate: true })

const isCustom = computed(() => props.config !== null)

const hasChanges = computed(() => {
  if (!props.config) {
    // No custom config exists, check if form differs from defaults
    return enabled.value !== true ||
      maxAttempts.value !== props.defaults.maxAttempts ||
      windowDuration.value !== props.defaults.windowDurationSeconds ||
      blockDuration.value !== props.defaults.blockDurationSeconds
  }
  // Custom config exists, check if form differs from saved
  return enabled.value !== props.config.enabled ||
    maxAttempts.value !== props.config.maxAttempts ||
    windowDuration.value !== props.config.windowDurationSeconds ||
    blockDuration.value !== props.config.blockDurationSeconds
})

const isValid = computed(() => {
  return maxAttempts.value >= 1 && maxAttempts.value <= 100
})

function handleSubmit() {
  emit('submit', {
    enabled: enabled.value,
    maxAttempts: maxAttempts.value,
    windowDurationSeconds: windowDuration.value,
    blockDurationSeconds: blockDuration.value,
  })
}

function handleReset() {
  emit('reset')
}

function handleCancel() {
  emit('cancel')
}
</script>

<template>
  <form @submit.prevent="handleSubmit" class="flex flex-col gap-6">
    <!-- Info box -->
    <div class="info-box leading-relaxed">
      <Info class="w-4 h-4 shrink-0" />
      <span>
        Rate limiting protects your application from brute force attacks by temporarily blocking IPs after too many failed authentication attempts.
      </span>
    </div>

    <!-- Enable/Disable toggle -->
    <div class="flex items-start gap-3 py-3 px-4 bg-bg-deep border border-border-subtle rounded-xs">
      <input
        id="rate-limit-enabled"
        v-model="enabled"
        type="checkbox"
        class="mt-0.5 w-4 h-4 rounded border-border-subtle text-accent-primary focus:ring-accent-primary cursor-pointer"
      />
      <label for="rate-limit-enabled" class="flex flex-col gap-0.5 cursor-pointer">
        <span class="font-medium text-text-primary">Enable rate limiting</span>
        <span class="text-[0.8125rem] text-text-muted">
          When disabled, no rate limiting will be applied to authentication attempts for this application.
        </span>
      </label>
    </div>

    <!-- Configuration fields (only shown when enabled) -->
    <template v-if="enabled">
      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium uppercase tracking-wider text-text-secondary" for="max-attempts">
          Max Attempts
        </label>
        <input
          id="max-attempts"
          v-model.number="maxAttempts"
          type="number"
          min="1"
          max="100"
          class="form-input"
          placeholder="10"
        />
        <p class="text-xs text-text-muted m-0">
          Maximum failed authentication attempts before blocking (1-100)
        </p>
      </div>

      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium uppercase tracking-wider text-text-secondary" for="window-duration">
          Time Window
        </label>
        <select
          id="window-duration"
          v-model="windowDuration"
          class="form-input"
        >
          <option v-for="opt in windowDurationOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
        <p class="text-xs text-text-muted m-0">
          Time period in which failed attempts are counted
        </p>
      </div>

      <div class="flex flex-col gap-2">
        <label class="text-xs font-medium uppercase tracking-wider text-text-secondary" for="block-duration">
          Block Duration
        </label>
        <select
          id="block-duration"
          v-model="blockDuration"
          class="form-input"
        >
          <option v-for="opt in blockDurationOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </option>
        </select>
        <p class="text-xs text-text-muted m-0">
          How long an IP is blocked after exceeding max attempts
        </p>
      </div>
    </template>

    <!-- Actions -->
    <div class="flex justify-between pt-4 border-t border-border-subtle mt-2">
      <div>
        <button
          v-if="isCustom"
          type="button"
          class="btn btn-secondary"
          :disabled="loading"
          @click="handleReset"
        >
          <RotateCcw class="w-4 h-4" />
          Reset to Defaults
        </button>
      </div>
      <div class="flex gap-3">
        <button type="button" class="btn btn-secondary" :disabled="loading" @click="handleCancel">
          Cancel
        </button>
        <button
          type="submit"
          class="btn btn-primary"
          :disabled="!isValid || !hasChanges || loading"
        >
          <Shield class="w-4 h-4" />
          {{ loading ? 'Saving...' : 'Save Configuration' }}
        </button>
      </div>
    </div>
  </form>
</template>
