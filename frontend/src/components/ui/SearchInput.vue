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
  <div class="search-input">
    <Search class="search-icon" />
    <input
      ref="inputRef"
      type="text"
      class="search-field"
      :placeholder="placeholder"
      :value="modelValue"
      @input="handleInput"
    />
    <button 
      v-if="modelValue" 
      class="search-clear" 
      @click="clear"
      type="button"
    >
      <X class="w-4 h-4" />
    </button>
  </div>
</template>

<style scoped>
.search-input {
  position: relative;
  width: 100%;
  max-width: 320px;
}

.search-icon {
  position: absolute;
  left: 0.875rem;
  top: 50%;
  transform: translateY(-50%);
  width: 18px;
  height: 18px;
  color: var(--text-muted);
  pointer-events: none;
}

.search-field {
  width: 100%;
  padding: 0.625rem 2.5rem 0.625rem 2.75rem;
  background: var(--bg-deep);
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  font-family: var(--font-body);
  font-size: 0.875rem;
  color: var(--text-primary);
  transition: all 0.2s ease;
}

.search-field::placeholder {
  color: var(--text-muted);
}

.search-field:focus {
  outline: none;
  border-color: var(--accent-primary);
  box-shadow: 0 0 0 3px rgba(var(--accent-primary-rgb), 0.12);
}

.search-clear {
  position: absolute;
  right: 0.5rem;
  top: 50%;
  transform: translateY(-50%);
  width: 28px;
  height: 28px;
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

.search-clear:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
}
</style>
