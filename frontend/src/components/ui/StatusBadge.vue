<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircle, XCircle, Clock, AlertCircle } from 'lucide-vue-next'

const props = defineProps<{
  status: 'active' | 'inactive' | 'pending' | 'error' | 'success' | 'warning'
  label?: string
  size?: 'sm' | 'md'
}>()

const statusConfig = computed(() => {
  const configs = {
    active: { color: 'secondary', icon: CheckCircle, defaultLabel: 'Active' },
    inactive: { color: 'muted', icon: XCircle, defaultLabel: 'Inactive' },
    pending: { color: 'amber', icon: Clock, defaultLabel: 'Pending' },
    error: { color: 'red', icon: XCircle, defaultLabel: 'Error' },
    success: { color: 'secondary', icon: CheckCircle, defaultLabel: 'Success' },
    warning: { color: 'amber', icon: AlertCircle, defaultLabel: 'Warning' },
  }
  return configs[props.status]
})

const displayLabel = computed(() => props.label || statusConfig.value.defaultLabel)

const colorClasses = computed(() => {
  const colors = {
    secondary: 'bg-[rgba(var(--accent-secondary-rgb),0.15)] text-accent-secondary',
    muted: 'bg-bg-elevated text-text-muted',
    amber: 'bg-[rgba(var(--accent-amber-rgb),0.15)] text-accent-amber',
    red: 'bg-[rgba(var(--accent-red-rgb),0.15)] text-accent-red'
  }
  return colors[statusConfig.value.color as keyof typeof colors]
})

const sizeClasses = computed(() => {
  return props.size === 'sm' 
    ? 'py-1 px-2 text-[0.6875rem]' 
    : 'py-1.5 px-3 text-xs'
})

const iconSizeClass = computed(() => {
  return props.size === 'sm' ? 'w-3 h-3' : 'w-3.5 h-3.5'
})
</script>

<template>
  <span 
    class="inline-flex items-center gap-1.5 rounded-full font-medium whitespace-nowrap"
    :class="[colorClasses, sizeClasses]"
  >
    <component :is="statusConfig.icon" :class="iconSizeClass" />
    <span>{{ displayLabel }}</span>
  </span>
</template>
