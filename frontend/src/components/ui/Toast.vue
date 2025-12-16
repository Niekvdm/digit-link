<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircle, XCircle, AlertTriangle, Info, X } from 'lucide-vue-next'

export interface ToastItem {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  message: string
}

const props = defineProps<{
  toasts: readonly ToastItem[]
}>()

const emit = defineEmits<{
  dismiss: [id: string]
}>()

function getIcon(type: ToastItem['type']) {
  switch (type) {
    case 'success': return CheckCircle
    case 'error': return XCircle
    case 'warning': return AlertTriangle
    case 'info': return Info
  }
}

const typeStyles = {
  success: 'border-accent-secondary toast-gradient-success',
  error: 'border-accent-red toast-gradient-error',
  warning: 'border-accent-amber toast-gradient-warning',
  info: 'border-accent-blue toast-gradient-info'
}

const iconStyles = {
  success: 'text-accent-secondary',
  error: 'text-accent-red',
  warning: 'text-accent-amber',
  info: 'text-accent-blue'
}
</script>

<template>
  <Teleport to="body">
    <div class="fixed bottom-6 right-6 z-[9999] flex flex-col gap-3 max-w-[400px] max-sm:left-4 max-sm:right-4 max-sm:bottom-4 max-sm:max-w-none">
      <TransitionGroup name="toast">
        <div 
          v-for="toast in toasts" 
          :key="toast.id"
          class="flex items-start gap-3 py-4 px-5 bg-bg-surface border rounded-xs shadow-[0_8px_32px_rgba(0,0,0,0.3)]"
          :class="typeStyles[toast.type]"
        >
          <component 
            :is="getIcon(toast.type)" 
            class="w-5 h-5 shrink-0"
            :class="iconStyles[toast.type]"
          />
          <span class="flex-1 text-sm text-text-primary leading-relaxed">{{ toast.message }}</span>
          <button 
            class="w-6 h-6 flex items-center justify-center border-none rounded bg-transparent text-text-muted cursor-pointer transition-all duration-150 -my-1 -mr-1 shrink-0 hover:bg-bg-elevated hover:text-text-primary"
            @click="emit('dismiss', toast.id)"
          >
            <X class="w-4 h-4" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-gradient-success {
  background: linear-gradient(135deg, rgba(var(--accent-secondary-rgb), 0.1), var(--bg-surface));
}

.toast-gradient-error {
  background: linear-gradient(135deg, rgba(var(--accent-red-rgb), 0.1), var(--bg-surface));
}

.toast-gradient-warning {
  background: linear-gradient(135deg, rgba(var(--accent-amber-rgb), 0.1), var(--bg-surface));
}

.toast-gradient-info {
  background: linear-gradient(135deg, rgba(var(--accent-blue-rgb), 0.1), var(--bg-surface));
}

/* Transitions */
.toast-enter-active {
  animation: toast-in 0.3s ease-out;
}

.toast-leave-active {
  animation: toast-out 0.2s ease-in forwards;
}

.toast-move {
  transition: transform 0.3s ease;
}

@keyframes toast-in {
  from {
    opacity: 0;
    transform: translateX(100%);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@keyframes toast-out {
  from {
    opacity: 1;
    transform: translateX(0);
  }
  to {
    opacity: 0;
    transform: translateX(100%);
  }
}
</style>
