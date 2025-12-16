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
const sizeClass = computed(() => `status-badge--${props.size || 'md'}`)
</script>

<template>
  <span 
    class="status-badge" 
    :class="[`status-badge--${statusConfig.color}`, sizeClass]"
  >
    <component :is="statusConfig.icon" class="status-icon" />
    <span>{{ displayLabel }}</span>
  </span>
</template>

<style scoped>
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 500;
  white-space: nowrap;
}

.status-badge--sm {
  padding: 0.25rem 0.5rem;
  font-size: 0.6875rem;
}

.status-badge--sm .status-icon {
  width: 12px;
  height: 12px;
}

.status-icon {
  width: 14px;
  height: 14px;
}

.status-badge--secondary {
  background: rgba(var(--accent-secondary-rgb), 0.15);
  color: var(--accent-secondary);
}

.status-badge--muted {
  background: var(--bg-elevated);
  color: var(--text-muted);
}

.status-badge--amber {
  background: rgba(var(--accent-amber-rgb), 0.15);
  color: var(--accent-amber);
}

.status-badge--red {
  background: rgba(var(--accent-red-rgb), 0.15);
  color: var(--accent-red);
}
</style>
