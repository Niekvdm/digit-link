import { ref, onMounted } from 'vue'
import { useApi } from './useApi'
import type {
  Organization,
  OrganizationsResponse,
  CreateOrganizationRequest,
  CreateOrganizationResponse,
  DeleteResponse,
  OrgAuthPolicy,
  PolicyResponse,
  SetPolicyRequest
} from '@/types/api'

export function useOrganizations() {
  const { get, post, put, del } = useApi()

  const organizations = ref<Organization[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function loadOrganizations() {
    loading.value = true
    error.value = null

    try {
      const data = await get<OrganizationsResponse>('/admin/organizations')
      organizations.value = data.organizations || []
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to load organizations'
    } finally {
      loading.value = false
    }
  }

  async function createOrganization(request: CreateOrganizationRequest): Promise<CreateOrganizationResponse> {
    try {
      const response = await post<CreateOrganizationResponse>('/admin/organizations', request)
      if (response.success) {
        await loadOrganizations()
      }
      return response
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : 'Failed to create organization'
      }
    }
  }

  async function updateOrganization(orgId: string, name: string): Promise<DeleteResponse> {
    try {
      const response = await put<DeleteResponse>(`/admin/organizations/${orgId}`, { name })
      if (response.success) {
        await loadOrganizations()
      }
      return response
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : 'Failed to update organization'
      }
    }
  }

  async function deleteOrganization(orgId: string): Promise<DeleteResponse> {
    try {
      const response = await del<DeleteResponse>(`/admin/organizations/${orgId}`)
      if (response.success) {
        await loadOrganizations()
      }
      return response
    } catch (err) {
      return {
        success: false,
        error: err instanceof Error ? err.message : 'Failed to delete organization'
      }
    }
  }

  async function getPolicy(orgId: string): Promise<OrgAuthPolicy | null> {
    try {
      const data = await get<PolicyResponse>(`/admin/organizations/${orgId}/policy`)
      return data.policy as OrgAuthPolicy | null
    } catch {
      return null
    }
  }

  async function setPolicy(orgId: string, policy: SetPolicyRequest): Promise<DeleteResponse> {
    try {
      const response = await put<DeleteResponse>(`/admin/organizations/${orgId}/policy`, policy)
      if (response.success) {
        await loadOrganizations()
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
    loadOrganizations()
  })

  return {
    organizations,
    loading,
    error,
    refresh: loadOrganizations,
    createOrganization,
    updateOrganization,
    deleteOrganization,
    getPolicy,
    setPolicy
  }
}
