<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

const props = defineProps<{
  total: number
  pageSize: number
  maxVisiblePages?: number
}>()

const currentPage = defineModel<number>({ default: 1 })

const totalPages = computed(() => Math.ceil(props.total / props.pageSize))
const maxVisible = computed(() => props.maxVisiblePages || 5)

const visiblePages = computed(() => {
  const pages: (number | 'ellipsis')[] = []
  const total = totalPages.value
  const current = currentPage.value
  const max = maxVisible.value

  if (total <= max) {
    for (let i = 1; i <= total; i++) {
      pages.push(i)
    }
  } else {
    const half = Math.floor(max / 2)
    let start = Math.max(1, current - half)
    let end = Math.min(total, current + half)

    if (current <= half) {
      end = max - 1
    } else if (current >= total - half) {
      start = total - max + 2
    }

    if (start > 1) {
      pages.push(1)
      if (start > 2) pages.push('ellipsis')
    }

    for (let i = start; i <= end; i++) {
      pages.push(i)
    }

    if (end < total) {
      if (end < total - 1) pages.push('ellipsis')
      pages.push(total)
    }
  }

  return pages
})

const startItem = computed(() => (currentPage.value - 1) * props.pageSize + 1)
const endItem = computed(() => Math.min(currentPage.value * props.pageSize, props.total))

function goToPage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
  }
}

function prevPage() {
  goToPage(currentPage.value - 1)
}

function nextPage() {
  goToPage(currentPage.value + 1)
}
</script>

<template>
  <div v-if="total > 0" class="pagination">
    <div class="pagination-info">
      Showing {{ startItem }}-{{ endItem }} of {{ total }}
    </div>
    
    <div class="pagination-controls">
      <button 
        class="pagination-btn pagination-btn--nav"
        :disabled="currentPage === 1"
        @click="prevPage"
      >
        <ChevronLeft class="w-4 h-4" />
      </button>
      
      <template v-for="page in visiblePages" :key="page">
        <span v-if="page === 'ellipsis'" class="pagination-ellipsis">...</span>
        <button
          v-else
          class="pagination-btn"
          :class="{ 'pagination-btn--active': page === currentPage }"
          @click="goToPage(page)"
        >
          {{ page }}
        </button>
      </template>
      
      <button 
        class="pagination-btn pagination-btn--nav"
        :disabled="currentPage === totalPages"
        @click="nextPage"
      >
        <ChevronRight class="w-4 h-4" />
      </button>
    </div>
  </div>
</template>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 0;
  flex-wrap: wrap;
}

.pagination-info {
  font-size: 0.8125rem;
  color: var(--text-secondary);
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.pagination-btn {
  min-width: 32px;
  height: 32px;
  padding: 0 0.5rem;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-subtle);
  border-radius: 6px;
  background: transparent;
  color: var(--text-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.pagination-btn:hover:not(:disabled) {
  background: var(--bg-elevated);
  color: var(--text-primary);
  border-color: var(--border-accent);
}

.pagination-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.pagination-btn--active {
  background: rgba(var(--accent-primary-rgb), 0.15);
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}

.pagination-btn--active:hover:not(:disabled) {
  background: rgba(var(--accent-primary-rgb), 0.2);
  color: var(--accent-primary);
  border-color: var(--accent-primary);
}

.pagination-ellipsis {
  padding: 0 0.25rem;
  color: var(--text-muted);
}
</style>
