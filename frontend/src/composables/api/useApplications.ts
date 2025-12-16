import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import { usePortalContext } from '@/composables/usePortalContext'
import type { 
  Application, 
  ApplicationsResponse,
  CreateApplicationRequest,
  CreateApplicationResponse,
  UpdateApplicationRequest,
  PolicyResponse,
  SetPolicyRequest,
  DeleteResponse
} from '@/types/api'

export function useApplications() {
  const api = useApi()
  const { isAdmin } = usePortalContext()
  
  const applications = ref<Application[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  function getEndpoint(path: string = '') {
    const base = isAdmin.value ? '/admin' : '/org'
    return `${base}/applications${path}`
  }

  async function fetchAll(orgId?: string) {
    loading.value = true
    error.value = null
    
    try {
      let endpoint = getEndpoint()
      if (orgId && isAdmin.value) {
        endpoint += `?org=${orgId}`
      }
      const res = await api.get<ApplicationsResponse>(endpoint)
      applications.value = res.applications
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load applications'
    } finally {
      loading.value = false
    }
  }

  async function fetchOne(appId: string) {
    const res = await api.get<{ application: Application }>(getEndpoint(`/${appId}`))
    return res.application
  }

  async function create(data: CreateApplicationRequest) {
    const res = await api.post<CreateApplicationResponse>(getEndpoint(), data)
    if (res.success && res.application) {
      applications.value = [...applications.value, res.application]
      return res.application
    }
    throw new Error(res.error || 'Failed to create application')
  }

  async function update(appId: string, data: UpdateApplicationRequest) {
    await api.put(getEndpoint(`/${appId}`), data)
    applications.value = applications.value.map(app => 
      app.id === appId ? { ...app, ...data } : app
    )
  }

  async function remove(appId: string) {
    await api.del<DeleteResponse>(getEndpoint(`/${appId}`))
    applications.value = applications.value.filter(app => app.id !== appId)
  }

  async function getPolicy(appId: string) {
    const res = await api.get<PolicyResponse>(getEndpoint(`/${appId}/policy`))
    return res.policy
  }

  async function setPolicy(appId: string, policy: SetPolicyRequest) {
    await api.put(getEndpoint(`/${appId}/policy`), policy)
  }

  async function getStats(appId: string) {
    return api.get(getEndpoint(`/${appId}/stats`))
  }

  async function getTunnels(appId: string) {
    return api.get(getEndpoint(`/${appId}/tunnels`))
  }

  return {
    applications: readonly(applications),
    loading: readonly(loading),
    error: readonly(error),
    fetchAll,
    fetchOne,
    create,
    update,
    remove,
    getPolicy,
    setPolicy,
    getStats,
    getTunnels
  }
}
