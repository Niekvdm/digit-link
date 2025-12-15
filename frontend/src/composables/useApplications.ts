import { ref, onMounted } from 'vue'
import { useApi } from './useApi'
import type {
  Application,
  ApplicationsResponse,
  CreateApplicationRequest,
  CreateApplicationResponse,
  UpdateApplicationRequest,
  DeleteResponse,
  AppAuthPolicy,
  PolicyResponse,
  SetPolicyRequest
} from '@/types/api'

export function useApplications(orgIdFilter?: string) {
  const { get, post, put, del } = useApi()

  const applications = ref<Application[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadApplications(orgId?: string) {
    loading.value = true
    error.value = null

    try {
      const params = orgId || orgIdFilter ? `?org=${orgId || orgIdFilter}` : ''
      const data = await get<ApplicationsResponse>(`/admin/applications${params}`)
      applications.value = data.applications || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load applications'
    } finally {
      loading.value = false
    }
  }

  async function createApplication(request: CreateApplicationRequest): Promise<CreateApplicationResponse> {
    try {
      const response = await post<CreateApplicationResponse>('/admin/applications', request)
      if (response.success) {
        await loadApplications()
      }
      return response
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : 'Failed to create application'
      }
    }
  }

  async function updateApplication(appId: string, request: UpdateApplicationRequest): Promise<DeleteResponse> {
    try {
      const response = await put<DeleteResponse>(`/admin/applications/${appId}`, request)
      if (response.success) {
        await loadApplications()
      }
      return response
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : 'Failed to update application'
      }
    }
  }

  async function deleteApplication(appId: string): Promise<DeleteResponse> {
    try {
      const response = await del<DeleteResponse>(`/admin/applications/${appId}`)
      if (response.success) {
        await loadApplications()
      }
      return response
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : 'Failed to delete application'
      }
    }
  }

  async function getPolicy(appId: string): Promise<AppAuthPolicy | null> {
    try {
      const data = await get<PolicyResponse>(`/admin/applications/${appId}/policy`)
      return data.policy as AppAuthPolicy | null
    } catch {
      return null
    }
  }

  async function setPolicy(appId: string, policy: SetPolicyRequest): Promise<DeleteResponse> {
    try {
      const response = await put<DeleteResponse>(`/admin/applications/${appId}/policy`, policy)
      if (response.success) {
        await loadApplications()
      }
      return response
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : 'Failed to set policy'
      }
    }
  }

  onMounted(() => {
    loadApplications()
  })

  return {
    applications,
    loading,
    error,
    refresh: loadApplications,
    createApplication,
    updateApplication,
    deleteApplication,
    getPolicy,
    setPolicy
  }
}
