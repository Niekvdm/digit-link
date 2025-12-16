<script setup lang="ts">
import { computed } from 'vue'
import { ArrowLeft } from 'lucide-vue-next'
import { useRouter } from 'vue-router'

defineProps<{
  title: string
  description?: string
  backTo?: string
}>()

const router = useRouter()

function goBack() {
  router.back()
}
</script>

<template>
  <div class="page-header">
    <div class="page-header-main">
      <button v-if="backTo" class="back-btn" @click="goBack">
        <ArrowLeft class="w-4 h-4" />
      </button>
      <div class="page-header-text">
        <h1 class="page-title">{{ title }}</h1>
        <p v-if="description" class="page-description">{{ description }}</p>
      </div>
    </div>
    <div class="page-header-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1.5rem;
  margin-bottom: 2rem;
  flex-wrap: wrap;
}

.page-header-main {
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}

.back-btn {
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-subtle);
  border-radius: 8px;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  flex-shrink: 0;
  margin-top: 0.25rem;
}

.back-btn:hover {
  background: var(--bg-elevated);
  color: var(--text-primary);
  border-color: var(--border-accent);
}

.page-header-text {
  flex: 1;
  min-width: 0;
}

.page-title {
  font-family: var(--font-display);
  font-size: 1.75rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
  line-height: 1.2;
}

.page-description {
  font-size: 0.9375rem;
  color: var(--text-secondary);
  margin: 0.5rem 0 0;
  line-height: 1.5;
}

.page-header-actions {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-shrink: 0;
}

@media (max-width: 640px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
  }
  
  .page-header-actions {
    justify-content: flex-start;
  }
}
</style>
