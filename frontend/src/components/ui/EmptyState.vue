<script setup lang="ts">
import type { Component } from 'vue'
import { Inbox } from 'lucide-vue-next'

defineProps<{
  title: string
  description?: string
  icon?: Component
}>()
</script>

<template>
  <div class="flex flex-col items-center justify-center text-center py-12 px-8 min-h-60">
    <div class="w-20 h-20 rounded-full bg-bg-elevated flex items-center justify-center text-text-muted mb-5">
      <component :is="icon || Inbox" class="w-10 h-10" />
    </div>
    <h3 class="text-lg font-semibold text-text-primary m-0 mb-2">{{ title }}</h3>
    <p v-if="description" class="text-sm text-text-secondary m-0 max-w-xs leading-relaxed">{{ description }}</p>
    <div v-if="$slots.action" class="mt-6">
      <slot name="action" />
    </div>
  </div>
</template>
