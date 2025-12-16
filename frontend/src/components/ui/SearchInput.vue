<script setup lang="ts">
import { ref, watch } from 'vue'
import { Search, X } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  placeholder?: string
  debounce?: number
}>(), {
  placeholder: 'Search...',
  debounce: 300
})

const modelValue = defineModel<string>({ default: '' })

const inputRef = ref<HTMLInputElement | null>(null)
let debounceTimer: ReturnType<typeof setTimeout>

function handleInput(e: Event) {
  const value = (e.target as HTMLInputElement).value
  
  if (debounceTimer) {
    clearTimeout(debounceTimer)
  }
  
  debounceTimer = setTimeout(() => {
    modelValue.value = value
  }, props.debounce)
}

function clear() {
  modelValue.value = ''
  if (inputRef.value) {
    inputRef.value.value = ''
    inputRef.value.focus()
  }
}
</script>

<template>
  <div class="relative w-full max-w-xs">
    <Search class="absolute left-3.5 top-1/2 -translate-y-1/2 w-[18px] h-[18px] text-text-muted pointer-events-none" />
    <input
      ref="inputRef"
      type="text"
      class="w-full py-2.5 pl-11 pr-10 bg-bg-deep border border-border-subtle rounded-xs font-body text-sm text-text-primary transition-all duration-200 placeholder:text-text-muted focus:outline-none focus:border-accent-primary focus:shadow-[0_0_0_3px_rgba(var(--accent-primary-rgb),0.12)]"
      :placeholder="placeholder"
      :value="modelValue"
      @input="handleInput"
    />
    <button 
      v-if="modelValue" 
      class="absolute right-2 top-1/2 -translate-y-1/2 w-7 h-7 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary"
      @click="clear"
      type="button"
    >
      <X class="w-4 h-4" />
    </button>
  </div>
</template>
