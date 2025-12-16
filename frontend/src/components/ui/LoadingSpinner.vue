<script setup lang="ts">
import { computed } from 'vue'
import { Loader2 } from 'lucide-vue-next'

const props = defineProps<{
  size?: 'sm' | 'md' | 'lg'
  label?: string
}>()

const iconSizeClass = computed(() => {
  const sizes = {
    sm: 'w-5 h-5',
    md: 'w-8 h-8',
    lg: 'w-12 h-12'
  }
  return sizes[props.size || 'md']
})
</script>

<template>
  <div class="flex flex-col items-center justify-center gap-3 p-8 text-text-muted">
    <Loader2 class="animate-spin" :class="iconSizeClass" />
    <span v-if="label" class="text-sm text-text-secondary">{{ label }}</span>
  </div>
</template>
