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

const iconBgClass = {
  danger: 'bg-[rgba(var(--accent-red-rgb),0.15)] text-accent-red',
  warning: 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber',
  default: ''
}
</script>

<template>
  <Modal v-model="open" :title="title" size="sm" :close-on-backdrop="!loading">
    <div class="text-center py-2">
      <div 
        v-if="variant === 'danger' || variant === 'warning'" 
        class="w-14 h-14 rounded-full flex items-center justify-center mx-auto mb-4"
        :class="iconBgClass[variant]"
      >
        <AlertTriangle class="w-6 h-6" />
      </div>
      <p class="text-[0.9375rem] text-text-secondary leading-relaxed m-0">{{ message }}</p>
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
        :class="variant === 'danger' ? 'bg-accent-red text-white border-none hover:brightness-90' : 'btn-primary'"
        @click="handleConfirm"
        :disabled="loading"
      >
        <Loader2 v-if="loading" class="w-4 h-4 animate-spin" />
        <span>{{ confirmText }}</span>
      </button>
    </template>
  </Modal>
</template>
