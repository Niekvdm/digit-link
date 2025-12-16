<script setup lang="ts">
import { computed } from 'vue'
import { Package, Sparkles, Crown, Zap } from 'lucide-vue-next'

defineProps<{
  planName?: string
  size?: 'sm' | 'md' | 'lg'
}>()

const getPlanConfig = (name?: string) => {
  const normalizedName = (name || '').toLowerCase()
  
  if (normalizedName.includes('enterprise') || normalizedName.includes('business')) {
    return { 
      color: 'purple', 
      icon: Crown,
      bgClass: 'bg-[rgba(168,85,247,0.15)]',
      textClass: 'text-[rgb(168,85,247)]'
    }
  }
  if (normalizedName.includes('pro') || normalizedName.includes('professional')) {
    return { 
      color: 'blue', 
      icon: Sparkles,
      bgClass: 'bg-[rgba(var(--accent-blue-rgb),0.15)]',
      textClass: 'text-accent-blue'
    }
  }
  if (normalizedName.includes('starter') || normalizedName.includes('basic')) {
    return { 
      color: 'secondary', 
      icon: Zap,
      bgClass: 'bg-[rgba(var(--accent-secondary-rgb),0.15)]',
      textClass: 'text-accent-secondary'
    }
  }
  // Default / Free
  return { 
    color: 'muted', 
    icon: Package,
    bgClass: 'bg-bg-elevated',
    textClass: 'text-text-muted'
  }
}

const sizeClasses = computed(() => ({
  sm: 'py-1 px-2 text-[0.6875rem] gap-1',
  md: 'py-1.5 px-2.5 text-xs gap-1.5',
  lg: 'py-2 px-3 text-sm gap-2'
}))

const iconSizeClasses = computed(() => ({
  sm: 'w-3 h-3',
  md: 'w-3.5 h-3.5',
  lg: 'w-4 h-4'
}))
</script>

<template>
  <span 
    v-if="planName"
    class="inline-flex items-center rounded-sm font-semibold whitespace-nowrap"
    :class="[
      getPlanConfig(planName).bgClass, 
      getPlanConfig(planName).textClass,
      sizeClasses[size || 'md']
    ]"
  >
    <component :is="getPlanConfig(planName).icon" :class="iconSizeClasses[size || 'md']" />
    <span>{{ planName }}</span>
  </span>
  <span 
    v-else 
    class="inline-flex items-center rounded-sm font-medium whitespace-nowrap bg-bg-elevated text-text-muted"
    :class="sizeClasses[size || 'md']"
  >
    <Package :class="iconSizeClasses[size || 'md']" />
    <span>No Plan</span>
  </span>
</template>
