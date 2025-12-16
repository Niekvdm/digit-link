import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import type { Plan, PlansResponse, PlanResponse, CreatePlanRequest } from '@/types/api'

export function usePlans() {
  const api = useApi()
  
  const plans = ref<Plan[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    
    try {
      const response = await api.get<PlansResponse>('/admin/plans')
      plans.value = response.plans || []
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load plans'
    } finally {
      loading.value = false
    }
  }

  async function fetchOne(id: string): Promise<PlanResponse | null> {
    loading.value = true
    error.value = null
    
    try {
      return await api.get<PlanResponse>(`/admin/plans/${id}`)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load plan'
      return null
    } finally {
      loading.value = false
    }
  }

  async function create(data: CreatePlanRequest): Promise<Plan | null> {
    loading.value = true
    error.value = null
    
    try {
      const plan = await api.post<Plan>('/admin/plans', data)
      await fetchAll()
      return plan
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to create plan'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function update(id: string, data: CreatePlanRequest): Promise<Plan | null> {
    loading.value = true
    error.value = null
    
    try {
      const plan = await api.put<Plan>(`/admin/plans/${id}`, data)
      await fetchAll()
      return plan
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update plan'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function remove(id: string): Promise<boolean> {
    loading.value = true
    error.value = null
    
    try {
      await api.del(`/admin/plans/${id}`)
      await fetchAll()
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to delete plan'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function setOrganizationPlan(orgId: string, planId: string | null): Promise<boolean> {
    loading.value = true
    error.value = null
    
    try {
      await api.put(`/admin/organizations/${orgId}/plan`, { planId })
      return true
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to set organization plan'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    plans: readonly(plans),
    loading: readonly(loading),
    error: readonly(error),
    fetchAll,
    fetchOne,
    create,
    update,
    remove,
    setOrganizationPlan
  }
}
