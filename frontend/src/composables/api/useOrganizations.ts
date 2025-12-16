import { ref, readonly } from 'vue'
import { useApi } from '@/composables/useApi'
import type { 
  Organization, 
  OrganizationsResponse, 
  CreateOrganizationResponse,
  PolicyResponse,
  SetPolicyRequest,
  DeleteResponse
} from '@/types/api'

export function useOrganizations() {
  const api = useApi()
  
  const organizations = ref<Organization[]>([])
  const currentOrg = ref<Organization | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  async function fetchAll() {
    loading.value = true
    error.value = null
    
    try {
      const res = await api.get<OrganizationsResponse>('/admin/organizations')
      organizations.value = res.organizations
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load organizations'
    } finally {
      loading.value = false
    }
  }

  async function fetchOne(id: string): Promise<Organization | null> {
    loading.value = true
    error.value = null
    
    try {
      const org = await api.get<Organization>(`/admin/organizations/${id}`)
      currentOrg.value = org
      return org
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load organization'
      return null
    } finally {
      loading.value = false
    }
  }

  async function create(name: string) {
    const res = await api.post<CreateOrganizationResponse>('/admin/organizations', { name })
    if (res.success && res.organization) {
      organizations.value = [...organizations.value, res.organization]
      return res.organization
    }
    throw new Error(res.error || 'Failed to create organization')
  }

  async function update(id: string, name: string) {
    await api.put(`/admin/organizations/${id}`, { name })
    organizations.value = organizations.value.map(org => 
      org.id === id ? { ...org, name } : org
    )
  }

  async function remove(id: string) {
    await api.del<DeleteResponse>(`/admin/organizations/${id}`)
    organizations.value = organizations.value.filter(org => org.id !== id)
  }

  async function getPolicy(orgId: string) {
    const res = await api.get<PolicyResponse>(`/admin/organizations/${orgId}/policy`)
    return res.policy
  }

  async function setPolicy(orgId: string, policy: SetPolicyRequest) {
    await api.put(`/admin/organizations/${orgId}/policy`, policy)
  }

  return {
    organizations: readonly(organizations),
    currentOrg: readonly(currentOrg),
    loading: readonly(loading),
    error: readonly(error),
    fetchAll,
    fetchOne,
    create,
    update,
    remove,
    getPolicy,
    setPolicy
  }
}
