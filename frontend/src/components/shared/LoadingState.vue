<script setup lang="ts">
import { Loader2 } from 'lucide-vue-next'

defineProps<{
  message?: string
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center py-12 text-[var(--text-muted)]">
    <Loader2 class="w-12 h-12 mb-4 opacity-30 animate-spin" />
    <p class="text-sm">{{ message || 'Loading...' }}</p>
  </div>
</template>
