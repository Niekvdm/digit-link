<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { Copy, Check, Eye, EyeOff, AlertTriangle } from 'lucide-vue-next'

const props = defineProps<{
  value: string
  label?: string
  showWarning?: boolean
  warningText?: string
}>()

const revealed = ref(false)
const copied = ref(false)

const maskedValue = computed(() => {
  if (revealed.value) return props.value
  return '•'.repeat(Math.min(props.value.length, 40))
})

async function copyToClipboard() {
  try {
    await navigator.clipboard.writeText(props.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

function toggleReveal() {
  revealed.value = !revealed.value
}
</script>

<template>
  <div class="token-reveal">
    <div v-if="label" class="token-label">{{ label }}</div>
    
    <div v-if="showWarning" class="token-warning">
      <AlertTriangle class="w-4 h-4" />
      <span>{{ warningText || 'This token will only be shown once. Save it securely.' }}</span>
    </div>
    
    <div class="token-box">
      <code class="token-value" :class="{ 'token-value--masked': !revealed }">
        {{ maskedValue }}
      </code>
      <div class="token-actions">
        <button class="token-btn" @click="toggleReveal" :title="revealed ? 'Hide' : 'Reveal'">
          <Eye v-if="!revealed" class="w-4 h-4" />
          <EyeOff v-else class="w-4 h-4" />
        </button>
        <button class="token-btn" @click="copyToClipboard" :title="copied ? 'Copied!' : 'Copy'">
          <Check v-if="copied" class="w-4 h-4 text-[var(--accent-secondary)]" />
          <Copy v-else class="w-4 h-4" />
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.token-reveal {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.token-label {
  font-size: 0.75rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-secondary);
}

.token-warning {
  display: flex;
  align-items: flex-start;
  gap: 0.625rem;
  padding: 0.875rem 1rem;
  background: rgba(var(--accent-amber-rgb), 0.1);
  border: 1px solid rgba(var(--accent-amber-rgb), 0.3);
  border-radius: 8px;
  font-size: 0.8125rem;
  color: var(--accent-amber);
  line-height: 1.5;
}

.token-warning svg {
  flex-shrink: 0;
  margin-top: 0.125rem;
}

.token-box {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  background: var(--bg-deep);
  border: 2px solid var(--accent-amber);
  border-radius: 10px;
  position: relative;
}

.token-box::before {
  content: '';
  position: absolute;
  inset: 0;
  background: linear-gradient(135deg, rgba(var(--accent-amber-rgb), 0.05), transparent);
  pointer-events: none;
  border-radius: 8px;
}

.token-value {
  flex: 1;
  font-family: var(--font-mono);
  font-size: 0.8125rem;
  word-break: break-all;
  line-height: 1.6;
  color: var(--accent-amber);
}

.token-value--masked {
  letter-spacing: 0.1em;
}

.token-actions {
  display: flex;
  gap: 0.25rem;
  flex-shrink: 0;
}

.token-btn {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.2s ease;
}

.token-btn:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
}
</style>
