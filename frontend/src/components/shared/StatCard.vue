<script setup lang="ts">
import type { Component } from 'vue'

defineProps<{
  label: string
  value: string | number
  icon?: Component
  loading?: boolean
}>()
</script>

<template>
  <div class="stat-card">
    <div class="flex items-center gap-2 text-xs font-medium uppercase tracking-wider text-[var(--text-secondary)] mb-3">
      <component :is="icon" v-if="icon" class="w-3.5 h-3.5" />
      {{ label }}
    </div>
    <div 
      class="font-mono text-3xl font-medium"
      :class="loading ? 'text-[var(--text-muted)]' : 'text-[var(--text-primary)]'"
    >
      {{ loading ? '—' : value }}
    </div>
  </div>
</template>
