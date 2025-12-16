<script setup lang="ts">
import { computed } from 'vue'
import { Loader2 } from 'lucide-vue-next'

const props = defineProps<{
  size?: 'sm' | 'md' | 'lg'
  label?: string
}>()

const sizeClass = computed(() => `spinner--${props.size || 'md'}`)
</script>

<template>
  <div class="loading-spinner" :class="sizeClass">
    <Loader2 class="spinner-icon" />
    <span v-if="label" class="spinner-label">{{ label }}</span>
  </div>
</template>

<style scoped>
.loading-spinner {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  padding: 2rem;
  color: var(--text-muted);
}

.spinner-icon {
  animation: spin 0.8s linear infinite;
}

.spinner--sm .spinner-icon {
  width: 20px;
  height: 20px;
}

.spinner--md .spinner-icon {
  width: 32px;
  height: 32px;
}

.spinner--lg .spinner-icon {
  width: 48px;
  height: 48px;
}

.spinner-label {
  font-size: 0.875rem;
  color: var(--text-secondary);
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
