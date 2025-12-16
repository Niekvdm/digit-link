<script setup lang="ts">
import { ref } from 'vue'
import { AlertTriangle, Loader2 } from 'lucide-vue-next'
import Modal from './Modal.vue'

const props = withDefaults(defineProps<{
  title: string
  message: string
  confirmText?: string
  cancelText?: string
  variant?: 'danger' | 'warning' | 'default'
}>(), {
  confirmText: 'Confirm',
  cancelText: 'Cancel',
  variant: 'default'
})

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const open = defineModel<boolean>({ required: true })
const loading = ref(false)

async function handleConfirm() {
  loading.value = true
  emit('confirm')
}

function handleCancel() {
  if (loading.value) return
  open.value = false
  emit('cancel')
}
</script>

<template>
  <Modal v-model="open" :title="title" size="sm" :close-on-backdrop="!loading">
    <div class="confirm-content">
      <div v-if="variant === 'danger' || variant === 'warning'" class="confirm-icon" :class="`confirm-icon--${variant}`">
        <AlertTriangle class="w-6 h-6" />
      </div>
      <p class="confirm-message">{{ message }}</p>
    </div>
    
    <template #footer>
      <button 
        class="btn btn-secondary" 
        @click="handleCancel"
        :disabled="loading"
      >
        {{ cancelText }}
      </button>
      <button 
        class="btn"
        :class="variant === 'danger' ? 'btn-danger-solid' : 'btn-primary'"
        @click="handleConfirm"
        :disabled="loading"
      >
        <Loader2 v-if="loading" class="w-4 h-4 animate-spin" />
        <span>{{ confirmText }}</span>
      </button>
    </template>
  </Modal>
</template>

<style scoped>
.confirm-content {
  text-align: center;
  padding: 0.5rem 0;
}

.confirm-icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 1rem;
}

.confirm-icon--danger {
  background: rgba(var(--accent-red-rgb), 0.15);
  color: var(--accent-red);
}

.confirm-icon--warning {
  background: rgba(var(--accent-amber-rgb), 0.15);
  color: var(--accent-amber);
}

.confirm-message {
  font-size: 0.9375rem;
  color: var(--text-secondary);
  line-height: 1.6;
  margin: 0;
}

.btn-danger-solid {
  background: var(--accent-red);
  color: white;
  border: none;
}

.btn-danger-solid:hover:not(:disabled) {
  filter: brightness(0.9);
}

.animate-spin {
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
