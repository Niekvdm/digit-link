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
  data: readonly T[]
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
  <div class="bg-bg-surface border border-border-subtle rounded-xs overflow-hidden">
    <div class="overflow-x-auto">
      <table class="w-full border-collapse min-w-[600px]">
        <thead>
          <tr>
            <th
              v-for="column in columns"
              :key="String(column.key)"
              :style="getColumnStyle(column)"
              class="py-3.5 px-4 text-xs font-semibold uppercase tracking-wider text-text-secondary bg-bg-elevated border-b border-border-subtle text-left whitespace-nowrap"
              :class="{ 'cursor-pointer select-none hover:text-text-primary': column.sortable }"
              @click="toggleSort(column)"
            >
              <div class="flex items-center gap-1.5">
                <span>{{ column.label }}</span>
                <span v-if="column.sortable" class="flex items-center text-accent-primary">
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
            <th 
              v-if="$slots.actions" 
              class="py-3.5 px-4 text-xs font-semibold uppercase tracking-wider text-text-secondary bg-bg-elevated border-b border-border-subtle w-[1%] whitespace-nowrap text-right"
            >
              Actions
            </th>
          </tr>
        </thead>
        <tbody v-if="!loading && sortedData.length > 0">
          <tr 
            v-for="(row, index) in sortedData" 
            :key="getRowKey(row, index)"
            class="transition-colors duration-150 hover:bg-bg-elevated last:[&>td]:border-b-0"
            @click="handleRowClick(row)"
          >
            <td
              v-for="column in columns"
              :key="String(column.key)"
              :style="getColumnStyle(column)"
              class="p-4 text-sm text-text-primary border-b border-border-subtle align-middle"
            >
              <slot :name="`cell-${String(column.key)}`" :row="row" :value="getNestedValue(row, String(column.key))">
                {{ getNestedValue(row, String(column.key)) ?? '—' }}
              </slot>
            </td>
            <td 
              v-if="$slots.actions" 
              class="p-4 pr-5 text-sm border-b border-border-subtle w-[1%] whitespace-nowrap text-right align-middle"
            >
              <slot name="actions" :row="row" />
            </td>
          </tr>
        </tbody>
      </table>
      
      <!-- Loading state -->
      <div v-if="loading" class="p-8">
        <LoadingSpinner label="Loading..." />
      </div>
      
      <!-- Empty state -->
      <div v-else-if="sortedData.length === 0" class="p-8 min-h-[280px] flex items-center justify-center">
        <EmptyState :title="emptyTitle" :description="emptyDescription">
          <template #action>
            <slot name="emptyAction" />
          </template>
        </EmptyState>
      </div>
    </div>
  </div>
</template>
