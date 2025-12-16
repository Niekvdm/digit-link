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
  <div class="flex flex-col gap-3">
    <div v-if="label" class="text-xs font-medium uppercase tracking-wider text-text-secondary">{{ label }}</div>
    
    <div 
      v-if="showWarning" 
      class="flex items-start gap-2.5 py-3.5 px-4 rounded-xs text-[0.8125rem] text-accent-amber leading-relaxed bg-[rgba(var(--accent-amber-rgb),0.1)] border border-[rgba(var(--accent-amber-rgb),0.3)]"
    >
      <AlertTriangle class="w-4 h-4 shrink-0 mt-0.5" />
      <span>{{ warningText || 'This token will only be shown once. Save it securely.' }}</span>
    </div>
    
    <div class="token-box flex items-center gap-3">
      <code 
        class="flex-1 font-mono text-[0.8125rem] break-all leading-relaxed text-accent-amber"
        :class="{ 'tracking-wider': !revealed }"
      >
        {{ maskedValue }}
      </code>
      <div class="flex gap-1 shrink-0">
        <button 
          class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary"
          @click="toggleReveal" 
          :title="revealed ? 'Hide' : 'Reveal'"
        >
          <Eye v-if="!revealed" class="w-4 h-4" />
          <EyeOff v-else class="w-4 h-4" />
        </button>
        <button 
          class="w-8 h-8 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary"
          @click="copyToClipboard" 
          :title="copied ? 'Copied!' : 'Copy'"
        >
          <Check v-if="copied" class="w-4 h-4 text-accent-secondary" />
          <Copy v-else class="w-4 h-4" />
        </button>
      </div>
    </div>
  </div>
</template>
