<script setup lang="ts">
import { X } from 'lucide-vue-next'

defineProps<{
  show: boolean
  title: string
  maxWidth?: string
}>()

const emit = defineEmits<{
  close: []
}>()
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="modal-overlay" @click.self="emit('close')">
        <div 
          class="modal animate-fade-in" 
          :style="{ maxWidth: maxWidth || '480px' }"
        >
          <div class="modal-header">
            <h3 class="modal-title">{{ title }}</h3>
            <button 
              class="p-1 text-[var(--text-muted)] hover:text-[var(--text-primary)] transition-colors"
              @click="emit('close')"
            >
              <X class="w-5 h-5" />
            </button>
          </div>
          <div class="modal-body">
            <slot />
          </div>
          <div v-if="$slots.footer" class="modal-footer">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
