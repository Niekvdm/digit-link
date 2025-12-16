<script setup lang="ts">
import { computed } from 'vue'
import type { Component } from 'vue'
import { ArrowUpRight, ArrowDownRight } from 'lucide-vue-next'

const props = defineProps<{
  label: string
  value: string | number
  icon?: Component
  trend?: number
  color?: 'primary' | 'secondary' | 'amber' | 'red' | 'blue'
  loading?: boolean
}>()

const trendDirection = computed(() => {
  if (!props.trend) return null
  return props.trend > 0 ? 'up' : 'down'
})

const trendText = computed(() => {
  if (!props.trend) return ''
  const prefix = props.trend > 0 ? '+' : ''
  return `${prefix}${props.trend}%`
})

const iconColorClass = computed(() => {
  const colors = {
    primary: 'text-accent-primary',
    secondary: 'text-accent-secondary',
    amber: 'text-accent-amber',
    red: 'text-accent-red',
    blue: 'text-accent-blue'
  }
  return colors[props.color || 'primary']
})

const accentClass = computed(() => `stat-card-accent--${props.color || 'primary'}`)
</script>

<template>
  <div class="bg-bg-surface border border-border-subtle rounded-xs py-5 px-6 relative overflow-hidden">
    <div class="stat-card-accent" :class="accentClass" />
    <div class="flex items-center gap-2.5 mb-3">
      <div v-if="icon" :class="iconColorClass">
        <component :is="icon" class="w-5 h-5" />
      </div>
      <span class="text-[0.8125rem] font-medium text-text-secondary uppercase tracking-wide">{{ label }}</span>
    </div>
    <div class="flex items-baseline gap-3">
      <template v-if="loading">
        <div class="h-8 w-20 bg-bg-elevated rounded animate-pulse" />
      </template>
      <template v-else>
        <span class="font-display text-[2rem] font-semibold text-text-primary leading-none">{{ value }}</span>
        <div 
          v-if="trend !== undefined" 
          class="flex items-center gap-1 text-xs font-semibold"
          :class="trendDirection === 'up' ? 'text-accent-secondary' : 'text-accent-red'"
        >
          <ArrowUpRight v-if="trendDirection === 'up'" class="w-3.5 h-3.5" />
          <ArrowDownRight v-else class="w-3.5 h-3.5" />
          <span>{{ trendText }}</span>
        </div>
      </template>
    </div>
    <slot />
  </div>
</template>
