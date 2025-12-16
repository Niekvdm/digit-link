<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { X } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  title?: string
  size?: 'sm' | 'md' | 'lg'
  closeOnBackdrop?: boolean
  closeOnEscape?: boolean
}>(), {
  size: 'md',
  closeOnBackdrop: true,
  closeOnEscape: true
})

const emit = defineEmits<{
  close: []
}>()

const open = defineModel<boolean>({ required: true })

const modalRef = ref<HTMLElement | null>(null)

function handleBackdropClick(e: MouseEvent) {
  if (props.closeOnBackdrop && e.target === e.currentTarget) {
    open.value = false
    emit('close')
  }
}

function handleEscape(e: KeyboardEvent) {
  if (props.closeOnEscape && e.key === 'Escape' && open.value) {
    open.value = false
    emit('close')
  }
}

function close() {
  open.value = false
  emit('close')
}

onMounted(() => {
  document.addEventListener('keydown', handleEscape)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleEscape)
})

// Prevent body scroll when modal is open
watch(open, (isOpen) => {
  if (isOpen) {
    document.body.style.overflow = 'hidden'
  } else {
    document.body.style.overflow = ''
  }
})

onUnmounted(() => {
  document.body.style.overflow = ''
})

const sizeClasses = {
  sm: 'max-w-[400px]',
  md: 'max-w-[560px]',
  lg: 'max-w-[800px]'
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div 
        v-if="open" 
        class="fixed inset-0 bg-black/70 flex items-center justify-center p-4 z-[1000]" 
        @click="handleBackdropClick"
      >
        <div 
          ref="modalRef"
          class="bg-bg-surface border border-border-subtle rounded-2xl w-full max-h-[calc(100vh-2rem)] flex flex-col overflow-hidden animate-modal-slide-in"
          :class="sizeClasses[size]"
          role="dialog" 
          aria-modal="true"
        >
          <header 
            v-if="title || $slots.header" 
            class="flex items-center justify-between py-5 px-6 border-b border-border-subtle shrink-0"
          >
            <slot name="header">
              <h2 class="font-display text-xl font-semibold text-text-primary m-0">{{ title }}</h2>
            </slot>
            <button 
              class="w-9 h-9 flex items-center justify-center border-none rounded-xs bg-transparent text-text-muted cursor-pointer transition-all duration-200 -my-2 -mr-2 hover:bg-bg-elevated hover:text-text-primary"
              @click="close" 
              aria-label="Close modal"
            >
              <X class="w-5 h-5" />
            </button>
          </header>
          
          <div class="p-6 overflow-y-auto flex-1">
            <slot />
          </div>
          
          <footer 
            v-if="$slots.footer" 
            class="flex items-center justify-end gap-3 py-4 px-6 border-t border-border-subtle shrink-0"
          >
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
