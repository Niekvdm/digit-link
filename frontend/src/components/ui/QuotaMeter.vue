<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  used: number
  limit?: number
  label: string
  unit?: 'bytes' | 'hours' | 'count' | 'concurrent'
  size?: 'sm' | 'md' | 'lg'
  showPercentage?: boolean
}>()

const percentage = computed(() => {
  if (!props.limit || props.limit === 0) return 0
  return Math.min(100, Math.round((props.used / props.limit) * 100))
})

const colorClass = computed(() => {
  if (!props.limit) return 'bg-accent-blue'
  if (percentage.value >= 90) return 'bg-accent-red'
  if (percentage.value >= 70) return 'bg-accent-amber'
  return 'bg-accent-secondary'
})

const textColorClass = computed(() => {
  if (!props.limit) return 'text-accent-blue'
  if (percentage.value >= 90) return 'text-accent-red'
  if (percentage.value >= 70) return 'text-accent-amber'
  return 'text-accent-secondary'
})

const formatValue = (value: number): string => {
  switch (props.unit) {
    case 'bytes':
      return formatBytes(value)
    case 'hours':
      return formatHours(value)
    case 'concurrent':
    case 'count':
    default:
      return formatNumber(value)
  }
}

const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

const formatHours = (seconds: number): string => {
  const hours = seconds / 3600
  if (hours < 1) return `${Math.round(seconds / 60)}m`
  if (hours < 24) return `${hours.toFixed(1)}h`
  return `${(hours / 24).toFixed(1)}d`
}

const formatNumber = (num: number): string => {
  if (num >= 1000000) return `${(num / 1000000).toFixed(1)}M`
  if (num >= 1000) return `${(num / 1000).toFixed(1)}K`
  return num.toString()
}

const sizeClasses = computed(() => ({
  sm: { bar: 'h-1.5', text: 'text-[0.6875rem]', gap: 'gap-1.5' },
  md: { bar: 'h-2', text: 'text-xs', gap: 'gap-2' },
  lg: { bar: 'h-3', text: 'text-sm', gap: 'gap-2.5' }
}))

const currentSize = computed(() => sizeClasses.value[props.size || 'md'])
</script>

<template>
  <div class="flex flex-col" :class="currentSize.gap">
    <div class="flex items-center justify-between" :class="currentSize.text">
      <span class="text-text-secondary font-medium">{{ label }}</span>
      <span class="font-mono" :class="textColorClass">
        <template v-if="limit">
          {{ formatValue(used) }} / {{ formatValue(limit) }}
          <span v-if="showPercentage" class="text-text-muted ml-1">({{ percentage }}%)</span>
        </template>
        <template v-else>
          {{ formatValue(used) }}
          <span class="text-text-muted">/ Unlimited</span>
        </template>
      </span>
    </div>
    <div class="w-full bg-bg-elevated rounded-full overflow-hidden" :class="currentSize.bar">
      <div 
        class="h-full rounded-full transition-all duration-500 ease-out"
        :class="colorClass"
        :style="{ width: limit ? `${percentage}%` : '100%' }"
      />
    </div>
  </div>
</template>
