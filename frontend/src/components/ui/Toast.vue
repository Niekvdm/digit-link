<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircle, XCircle, AlertTriangle, Info, X } from 'lucide-vue-next'

export interface ToastItem {
  id: string
  type: 'success' | 'error' | 'warning' | 'info'
  message: string
}

const props = defineProps<{
  toasts: ToastItem[]
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
</script>

<template>
  <Teleport to="body">
    <div class="toast-container">
      <TransitionGroup name="toast">
        <div 
          v-for="toast in toasts" 
          :key="toast.id"
          class="toast"
          :class="`toast--${toast.type}`"
        >
          <component :is="getIcon(toast.type)" class="toast-icon" />
          <span class="toast-message">{{ toast.message }}</span>
          <button class="toast-close" @click="emit('dismiss', toast.id)">
            <X class="w-4 h-4" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
.toast-container {
  position: fixed;
  bottom: 1.5rem;
  right: 1.5rem;
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  max-width: 400px;
}

.toast {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1rem 1.25rem;
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 10px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
}

.toast--success {
  border-color: var(--accent-secondary);
  background: linear-gradient(135deg, 
    rgba(var(--accent-secondary-rgb), 0.1), 
    var(--bg-surface)
  );
}

.toast--error {
  border-color: var(--accent-red);
  background: linear-gradient(135deg, 
    rgba(var(--accent-red-rgb), 0.1), 
    var(--bg-surface)
  );
}

.toast--warning {
  border-color: var(--accent-amber);
  background: linear-gradient(135deg, 
    rgba(var(--accent-amber-rgb), 0.1), 
    var(--bg-surface)
  );
}

.toast--info {
  border-color: var(--accent-blue);
  background: linear-gradient(135deg, 
    rgba(var(--accent-blue-rgb), 0.1), 
    var(--bg-surface)
  );
}

.toast-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.toast--success .toast-icon {
  color: var(--accent-secondary);
}

.toast--error .toast-icon {
  color: var(--accent-red);
}

.toast--warning .toast-icon {
  color: var(--accent-amber);
}

.toast--info .toast-icon {
  color: var(--accent-blue);
}

.toast-message {
  flex: 1;
  font-size: 0.875rem;
  color: var(--text-primary);
  line-height: 1.4;
}

.toast-close {
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: 4px;
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.15s ease;
  margin: -0.25rem -0.25rem -0.25rem 0;
  flex-shrink: 0;
}

.toast-close:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
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

@media (max-width: 640px) {
  .toast-container {
    left: 1rem;
    right: 1rem;
    bottom: 1rem;
    max-width: none;
  }
}
</style>
