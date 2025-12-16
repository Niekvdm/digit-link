<script setup lang="ts">
import { ref, watch } from 'vue'
import type { Plan } from '@/types/api'
import PlanCard from './PlanCard.vue'

const props = defineProps<{
  plans: Plan[]
  currentPlanId?: string
  disabled?: boolean
  showContactMessage?: boolean
}>()

const selectedPlanId = defineModel<string | null>('modelValue')

const emit = defineEmits<{
  select: [plan: Plan]
}>()

const handleSelect = (plan: Plan) => {
  if (props.disabled) return
  if (plan.id === props.currentPlanId) return
  
  selectedPlanId.value = plan.id
  emit('select', plan)
}

// Reset selection when plans change
watch(() => props.plans, () => {
  if (selectedPlanId.value && !props.plans.find(p => p.id === selectedPlanId.value)) {
    selectedPlanId.value = null
  }
})
</script>

<template>
  <div class="space-y-6">
    <!-- Plans grid -->
    <div 
      class="grid gap-5"
      :class="plans.length <= 2 ? 'grid-cols-1 md:grid-cols-2' : 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3'"
    >
      <PlanCard
        v-for="plan in plans"
        :key="plan.id"
        :plan="plan"
        :current="plan.id === currentPlanId"
        :selectable="!disabled && plan.id !== currentPlanId"
        :selected="plan.id === selectedPlanId"
        @select="handleSelect"
      />
    </div>

    <!-- Contact message for org users -->
    <div 
      v-if="showContactMessage && currentPlanId" 
      class="flex items-center justify-center py-4 px-5 bg-bg-elevated rounded-xs border border-border-subtle"
    >
      <p class="text-sm text-text-secondary text-center">
        To change your plan, please contact your administrator or 
        <a href="mailto:sales@example.com" class="text-accent-primary hover:underline">reach out to our sales team</a>.
      </p>
    </div>

    <!-- Empty state -->
    <div 
      v-if="plans.length === 0" 
      class="text-center py-12 text-text-muted"
    >
      <p>No plans available</p>
    </div>
  </div>
</template>
