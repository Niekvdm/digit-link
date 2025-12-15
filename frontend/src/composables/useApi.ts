import { useAuthStore } from '@/stores/authStore'
import { useRouter } from 'vue-router'
import type { ApiError } from '@/types/api'

interface RequestOptions extends RequestInit {
  skipAuth?: boolean
}

export function useApi() {
  const authStore = useAuthStore()
  const router = useRouter()

  async function request<T>(
    endpoint: string,
    options: RequestOptions = {}
  ): Promise<T> {
    const { skipAuth = false, ...fetchOptions } = options

    const headers = new Headers(fetchOptions.headers)

    // Add auth header if authenticated
    if (!skipAuth && authStore.token) {
      headers.set('X-Admin-Token', authStore.token)
    }

    // Add content type for JSON bodies
    if (fetchOptions.body && typeof fetchOptions.body === 'string') {
      if (!headers.has('Content-Type')) {
        headers.set('Content-Type', 'application/json')
      }
    }

    const response = await fetch(endpoint, {
      ...fetchOptions,
      headers,
    })

    // Handle 401 - unauthorized
    if (response.status === 401 && !skipAuth) {
      authStore.clearToken()
      router.push({ name: 'login' })
      throw new Error('Session expired')
    }

    if (!response.ok) {
      const error: ApiError = await response.json().catch(() => ({
        error: 'unknown_error',
        error_description: 'An unknown error occurred',
      }))
      throw new Error(error.error_description || error.error)
    }

    return response.json()
  }

  function get<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return request<T>(endpoint, { ...options, method: 'GET' })
  }

  function post<T>(
    endpoint: string,
    body?: unknown,
    options?: RequestOptions
  ): Promise<T> {
    return request<T>(endpoint, {
      ...options,
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  function put<T>(
    endpoint: string,
    body?: unknown,
    options?: RequestOptions
  ): Promise<T> {
    return request<T>(endpoint, {
      ...options,
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
    })
  }

  function del<T>(endpoint: string, options?: RequestOptions): Promise<T> {
    return request<T>(endpoint, { ...options, method: 'DELETE' })
  }

  return {
    request,
    get,
    post,
    put,
    del,
  }
}
