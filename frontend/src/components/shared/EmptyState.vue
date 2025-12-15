<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  icon?: Component
  title: string
  description?: string
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center py-12 text-[var(--text-muted)]">
    <component 
      :is="icon" 
      v-if="icon" 
      class="w-12 h-12 mb-4 opacity-30" 
    />
    <p class="text-sm font-medium">{{ title }}</p>
    <p v-if="description" class="text-xs mt-1 opacity-75">{{ description }}</p>
  </div>
</template>
