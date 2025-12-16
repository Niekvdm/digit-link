<script setup lang="ts">
import { computed } from 'vue'
import { Check, Infinity, Package, Sparkles, Crown, Zap } from 'lucide-vue-next'
import type { Plan } from '@/types/api'

const props = defineProps<{
  plan: Plan
  current?: boolean
  selectable?: boolean
  selected?: boolean
  compact?: boolean
}>()

const emit = defineEmits<{
  select: [plan: Plan]
}>()

const getPlanStyle = (name: string) => {
  const normalizedName = name.toLowerCase()
  
  if (normalizedName.includes('enterprise') || normalizedName.includes('business')) {
    return { 
      icon: Crown,
      gradient: 'from-purple-500/10 to-purple-600/5',
      border: 'border-purple-500/30',
      accent: 'text-purple-500',
      badge: 'bg-purple-500/15 text-purple-400'
    }
  }
  if (normalizedName.includes('pro') || normalizedName.includes('professional')) {
    return { 
      icon: Sparkles,
      gradient: 'from-blue-500/10 to-blue-600/5',
      border: 'border-blue-500/30',
      accent: 'text-accent-blue',
      badge: 'bg-blue-500/15 text-blue-400'
    }
  }
  if (normalizedName.includes('starter') || normalizedName.includes('basic')) {
    return { 
      icon: Zap,
      gradient: 'from-emerald-500/10 to-emerald-600/5',
      border: 'border-emerald-500/30',
      accent: 'text-accent-secondary',
      badge: 'bg-emerald-500/15 text-emerald-400'
    }
  }
  // Default / Free
  return { 
    icon: Package,
    gradient: 'from-gray-500/10 to-gray-600/5',
    border: 'border-border-subtle',
    accent: 'text-text-muted',
    badge: 'bg-bg-elevated text-text-muted'
  }
}

const style = computed(() => getPlanStyle(props.plan.name))

const formatBytes = (bytes?: number): string => {
  if (!bytes) return 'Unlimited'
  if (bytes >= 1073741824) return `${(bytes / 1073741824).toFixed(0)} GB`
  if (bytes >= 1048576) return `${(bytes / 1048576).toFixed(0)} MB`
  return `${(bytes / 1024).toFixed(0)} KB`
}

const formatHours = (hours?: number): string => {
  if (!hours) return 'Unlimited'
  if (hours >= 720) return `${Math.round(hours / 720)} months`
  if (hours >= 24) return `${Math.round(hours / 24)} days`
  return `${hours} hours`
}

const formatNumber = (num?: number): string => {
  if (!num) return 'Unlimited'
  if (num >= 1000000) return `${(num / 1000000).toFixed(0)}M`
  if (num >= 1000) return `${(num / 1000).toFixed(0)}K`
  return num.toString()
}

const features = computed(() => [
  {
    label: 'Bandwidth',
    value: formatBytes(props.plan.bandwidthBytesMonthly),
    unlimited: !props.plan.bandwidthBytesMonthly
  },
  {
    label: 'Tunnel Hours',
    value: formatHours(props.plan.tunnelHoursMonthly),
    unlimited: !props.plan.tunnelHoursMonthly
  },
  {
    label: 'Concurrent Tunnels',
    value: props.plan.concurrentTunnelsMax?.toString() || 'Unlimited',
    unlimited: !props.plan.concurrentTunnelsMax
  },
  {
    label: 'Requests',
    value: formatNumber(props.plan.requestsMonthly),
    unlimited: !props.plan.requestsMonthly
  }
])

const handleSelect = () => {
  if (props.selectable && !props.current) {
    emit('select', props.plan)
  }
}
</script>

<template>
  <div 
    class="relative bg-bg-surface border rounded-xs overflow-hidden transition-all duration-200"
    :class="[
      style.border,
      current ? 'ring-2 ring-accent-primary ring-offset-2' : '',
      selectable && !current ? 'cursor-pointer hover:border-accent-primary/50 hover:shadow-lg' : '',
      selected && !current ? 'ring-2 ring-accent-secondary' : ''
    ]"
    :style="current ? { '--tw-ring-offset-color': 'var(--bg-deep)' } : {}"
    @click="handleSelect"
  >
    <!-- Gradient header -->
    <div 
      class="bg-gradient-to-br p-5"
      :class="style.gradient"
    >
      <div class="flex items-start justify-between">
        <div class="flex items-center gap-3">
          <div 
            class="w-10 h-10 rounded-xs flex items-center justify-center"
            :class="style.badge"
          >
            <component :is="style.icon" class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-lg font-semibold text-text-primary">{{ plan.name }}</h3>
            <p v-if="!compact" class="text-sm text-text-secondary mt-0.5">
              {{ plan.overageAllowedPercent > 0 ? `${plan.overageAllowedPercent}% overage allowed` : 'Hard limits' }}
            </p>
          </div>
        </div>
        <span 
          v-if="current" 
          class="text-[0.6875rem] font-semibold uppercase tracking-wider px-2 py-1 rounded-sm bg-accent-primary/15 text-accent-primary"
        >
          Current
        </span>
      </div>
    </div>

    <!-- Features list -->
    <div class="p-5" :class="compact ? 'pt-3' : ''">
      <ul class="space-y-3">
        <li 
          v-for="feature in features" 
          :key="feature.label"
          class="flex items-center justify-between text-sm"
        >
          <span class="text-text-secondary">{{ feature.label }}</span>
          <span class="font-mono font-medium flex items-center gap-1.5" :class="style.accent">
            <Infinity v-if="feature.unlimited" class="w-4 h-4 opacity-60" />
            <template v-else>{{ feature.value }}</template>
          </span>
        </li>
      </ul>

      <!-- Grace period info -->
      <div 
        v-if="!compact && plan.gracePeriodHours > 0" 
        class="mt-4 pt-4 border-t border-border-subtle"
      >
        <p class="text-xs text-text-muted">
          {{ plan.gracePeriodHours }}h grace period after limit reached
        </p>
      </div>

      <!-- Slot for custom CTA or actions -->
      <div v-if="$slots.actions" class="mt-5">
        <slot name="actions" />
      </div>
    </div>
  </div>
</template>
