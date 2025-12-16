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
  <div v-if="total > 0" class="flex items-center justify-between gap-4 py-4 flex-wrap">
    <div class="text-[0.8125rem] text-text-secondary">
      Showing {{ startItem }}-{{ endItem }} of {{ total }}
    </div>
    
    <div class="flex items-center gap-1">
      <button 
        class="min-w-8 h-8 px-2 flex items-center justify-center border border-border-subtle rounded-xs bg-transparent text-text-secondary text-[0.8125rem] font-medium cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary hover:border-border-accent disabled:opacity-40 disabled:cursor-not-allowed"
        :disabled="currentPage === 1"
        @click="prevPage"
      >
        <ChevronLeft class="w-4 h-4" />
      </button>
      
      <template v-for="page in visiblePages" :key="page">
        <span v-if="page === 'ellipsis'" class="px-1 text-text-muted">...</span>
        <button
          v-else
          class="min-w-8 h-8 px-2 flex items-center justify-center border rounded-xs text-[0.8125rem] font-medium cursor-pointer transition-all duration-200"
          :class="page === currentPage 
            ? 'bg-[rgba(var(--accent-primary-rgb),0.15)] border-accent-primary text-accent-primary hover:bg-[rgba(var(--accent-primary-rgb),0.2)]' 
            : 'border-border-subtle bg-transparent text-text-secondary hover:bg-bg-elevated hover:text-text-primary hover:border-border-accent'"
          @click="goToPage(page)"
        >
          {{ page }}
        </button>
      </template>
      
      <button 
        class="min-w-8 h-8 px-2 flex items-center justify-center border border-border-subtle rounded-xs bg-transparent text-text-secondary text-[0.8125rem] font-medium cursor-pointer transition-all duration-200 hover:bg-bg-elevated hover:text-text-primary hover:border-border-accent disabled:opacity-40 disabled:cursor-not-allowed"
        :disabled="currentPage === totalPages"
        @click="nextPage"
      >
        <ChevronRight class="w-4 h-4" />
      </button>
    </div>
  </div>
</template>
