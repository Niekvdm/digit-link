<script setup lang="ts" generic="T extends Record<string, unknown>">
import { computed, ref } from 'vue'
import { ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-vue-next'
import EmptyState from './EmptyState.vue'
import LoadingSpinner from './LoadingSpinner.vue'

export interface Column<T> {
  key: keyof T | string
  label: string
  sortable?: boolean
  width?: string
  align?: 'left' | 'center' | 'right'
}

const props = withDefaults(defineProps<{
  columns: Column<T>[]
  data: T[]
  loading?: boolean
  emptyTitle?: string
  emptyDescription?: string
  rowKey?: keyof T
}>(), {
  emptyTitle: 'No data found',
  emptyDescription: 'There are no items to display.',
  rowKey: 'id' as keyof T
})

const emit = defineEmits<{
  rowClick: [row: T]
}>()

const sortKey = ref<string | null>(null)
const sortDirection = ref<'asc' | 'desc'>('asc')

const sortedData = computed(() => {
  if (!sortKey.value) return props.data
  
  const key = sortKey.value
  const direction = sortDirection.value === 'asc' ? 1 : -1
  
  return [...props.data].sort((a, b) => {
    const aVal = getNestedValue(a, key)
    const bVal = getNestedValue(b, key)
    
    if (aVal === bVal) return 0
    if (aVal === null || aVal === undefined) return 1
    if (bVal === null || bVal === undefined) return -1
    
    if (typeof aVal === 'string' && typeof bVal === 'string') {
      return aVal.localeCompare(bVal) * direction
    }
    
    return (aVal < bVal ? -1 : 1) * direction
  })
})

function getNestedValue(obj: T, key: string): unknown {
  return key.split('.').reduce<unknown>((o, k) => (o as Record<string, unknown>)?.[k], obj as unknown)
}

function toggleSort(column: Column<T>) {
  if (!column.sortable) return
  
  const key = String(column.key)
  if (sortKey.value === key) {
    if (sortDirection.value === 'asc') {
      sortDirection.value = 'desc'
    } else {
      sortKey.value = null
      sortDirection.value = 'asc'
    }
  } else {
    sortKey.value = key
    sortDirection.value = 'asc'
  }
}

function handleRowClick(row: T) {
  emit('rowClick', row)
}

function getColumnStyle(column: Column<T>) {
  const styles: Record<string, string> = {}
  if (column.width) styles.width = column.width
  if (column.align) styles.textAlign = column.align
  return styles
}

function getRowKey(row: T, index: number): string | number {
  const key = row[props.rowKey as keyof T]
  if (typeof key === 'string' || typeof key === 'number') {
    return key
  }
  return index
}
</script>

<template>
  <div class="data-table-wrapper">
    <div class="data-table-container">
      <table class="data-table">
        <thead>
          <tr>
            <th
              v-for="column in columns"
              :key="String(column.key)"
              :style="getColumnStyle(column)"
              :class="{ 'sortable': column.sortable }"
              @click="toggleSort(column)"
            >
              <div class="th-content">
                <span>{{ column.label }}</span>
                <span v-if="column.sortable" class="sort-icon">
                  <ArrowUp 
                    v-if="sortKey === String(column.key) && sortDirection === 'asc'" 
                    class="w-3.5 h-3.5" 
                  />
                  <ArrowDown 
                    v-else-if="sortKey === String(column.key) && sortDirection === 'desc'" 
                    class="w-3.5 h-3.5" 
                  />
                  <ArrowUpDown v-else class="w-3.5 h-3.5 opacity-30" />
                </span>
              </div>
            </th>
            <th v-if="$slots.actions" class="actions-header">Actions</th>
          </tr>
        </thead>
        <tbody v-if="!loading && sortedData.length > 0">
          <tr 
            v-for="(row, index) in sortedData" 
            :key="getRowKey(row, index)"
            @click="handleRowClick(row)"
          >
            <td
              v-for="column in columns"
              :key="String(column.key)"
              :style="getColumnStyle(column)"
            >
              <slot :name="`cell-${String(column.key)}`" :row="row" :value="getNestedValue(row, String(column.key))">
                {{ getNestedValue(row, String(column.key)) ?? '—' }}
              </slot>
            </td>
            <td v-if="$slots.actions" class="actions-cell">
              <slot name="actions" :row="row" />
            </td>
          </tr>
        </tbody>
      </table>
      
      <!-- Loading state -->
      <div v-if="loading" class="table-loading">
        <LoadingSpinner label="Loading..." />
      </div>
      
      <!-- Empty state -->
      <div v-else-if="sortedData.length === 0" class="table-empty">
        <EmptyState :title="emptyTitle" :description="emptyDescription">
          <template #action>
            <slot name="emptyAction" />
          </template>
        </EmptyState>
      </div>
    </div>
  </div>
</template>

<style scoped>
.data-table-wrapper {
  background: var(--bg-surface);
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
  overflow: hidden;
}

.data-table-container {
  overflow-x: auto;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  min-width: 600px;
}

.data-table th {
  padding: 0.875rem 1rem;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-secondary);
  background: var(--bg-elevated);
  border-bottom: 1px solid var(--border-subtle);
  text-align: left;
  white-space: nowrap;
}

.data-table th.sortable {
  cursor: pointer;
  user-select: none;
}

.data-table th.sortable:hover {
  color: var(--text-primary);
}

.th-content {
  display: flex;
  align-items: center;
  gap: 0.375rem;
}

.sort-icon {
  display: flex;
  align-items: center;
  color: var(--accent-primary);
}

.data-table td {
  padding: 1rem;
  font-size: 0.875rem;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle);
  vertical-align: middle;
}

.data-table tbody tr {
  transition: background 0.15s ease;
}

.data-table tbody tr:hover {
  background: var(--bg-elevated);
}

.data-table tbody tr:last-child td {
  border-bottom: none;
}

.actions-header,
.actions-cell {
  width: 1%;
  white-space: nowrap;
  text-align: right;
}

.actions-cell {
  padding-right: 1.25rem;
}

.table-loading,
.table-empty {
  padding: 2rem;
}

.table-empty {
  min-height: 280px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
