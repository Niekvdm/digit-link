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

const colorClass = computed(() => `stat-card--${props.color || 'primary'}`)

const trendDirection = computed(() => {
  if (!props.trend) return null
  return props.trend > 0 ? 'up' : 'down'
})

const trendText = computed(() => {
  if (!props.trend) return ''
  const prefix = props.trend > 0 ? '+' : ''
  return `${prefix}${props.trend}%`
})
</script>

<template>
  <div class="stat-card" :class="[colorClass, { 'stat-card--loading': loading }]">
    <div class="stat-card-accent" />
    <div class="stat-header">
      <div v-if="icon" class="stat-icon">
        <component :is="icon" class="w-5 h-5" />
      </div>
      <span class="stat-label">{{ label }}</span>
    </div>
    <div class="stat-body">
      <template v-if="loading">
        <div class="stat-skeleton" />
      </template>
      <template v-else>
        <span class="stat-value">{{ value }}</span>
        <div v-if="trend !== undefined" class="stat-trend" :class="`stat-trend--${trendDirection}`">
          <ArrowUpRight v-if="trendDirection === 'up'" class="w-3.5 h-3.5" />
          <ArrowDownRight v-else class="w-3.5 h-3.5" />
          <span>{{ trendText }}</span>
        </div>
      </template>
    </div>
    <slot />
  </div>
</template>

<style scoped>
.stat-card {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  padding: 1.25rem 1.5rem;
  position: relative;
  overflow: hidden;
}

.stat-card-accent {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 2px;
}

.stat-card--primary .stat-card-accent {
  background: linear-gradient(90deg, var(--accent-primary), transparent);
}

.stat-card--secondary .stat-card-accent {
  background: linear-gradient(90deg, var(--accent-secondary), transparent);
}

.stat-card--amber .stat-card-accent {
  background: linear-gradient(90deg, var(--accent-amber), transparent);
}

.stat-card--red .stat-card-accent {
  background: linear-gradient(90deg, var(--accent-red), transparent);
}

.stat-card--blue .stat-card-accent {
  background: linear-gradient(90deg, var(--accent-blue), transparent);
}

.stat-header {
  display: flex;
  align-items: center;
  gap: 0.625rem;
  margin-bottom: 0.75rem;
}

.stat-icon {
  color: var(--text-muted);
}

.stat-card--primary .stat-icon {
  color: var(--accent-primary);
}

.stat-card--secondary .stat-icon {
  color: var(--accent-secondary);
}

.stat-card--amber .stat-icon {
  color: var(--accent-amber);
}

.stat-card--red .stat-icon {
  color: var(--accent-red);
}

.stat-card--blue .stat-icon {
  color: var(--accent-blue);
}

.stat-label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.stat-body {
  display: flex;
  align-items: baseline;
  gap: 0.75rem;
}

.stat-value {
  font-family: var(--font-display);
  font-size: 2rem;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1;
}

.stat-trend {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.75rem;
  font-weight: 600;
}

.stat-trend--up {
  color: var(--accent-secondary);
}

.stat-trend--down {
  color: var(--accent-red);
}

.stat-skeleton {
  height: 2rem;
  width: 80px;
  background: var(--bg-elevated);
  border-radius: 4px;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
</style>
