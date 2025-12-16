<script setup lang="ts">
import { computed } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import { useRouter } from 'vue-router'

defineProps<{
  title: string
  description?: string
  backTo?: string
}>()

const router = useRouter()

function goBack() {
  router.back()
}
</script>

<template>
  <div class="flex items-start justify-between gap-6 mb-8 flex-wrap max-sm:flex-col max-sm:items-stretch">
    <div class="flex items-start gap-4">
      <button 
        v-if="backTo" 
        class="w-9 h-9 flex items-center justify-center border border-border-subtle rounded-xs bg-transparent text-text-secondary cursor-pointer transition-all duration-200 shrink-0 mt-1 hover:bg-bg-elevated hover:text-text-primary hover:border-border-accent"
        @click="goBack"
      >
        <ArrowLeft class="w-4 h-4" />
      </button>
      <div class="flex-1 min-w-0">
        <h1 class="font-display text-[1.75rem] font-semibold text-text-primary m-0 leading-tight">{{ title }}</h1>
        <p v-if="description" class="text-[0.9375rem] text-text-secondary mt-2 m-0 leading-relaxed">{{ description }}</p>
      </div>
    </div>
    <div class="flex items-center gap-3 shrink-0 max-sm:justify-start">
      <slot name="actions" />
    </div>
  </div>
</template>
