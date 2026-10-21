const BASE_URL = "/api"

let accessToken: string | null = localStorage.getItem("access_token")
let refreshToken: string | null = localStorage.getItem("refresh_token")

// Ensures only one refresh request runs at a time — concurrent 401s queue behind it
let refreshPromise: Promise<string | null> | null = null

export function setTokens(access: string, refresh: string) {
  accessToken = access
  refreshToken = refresh
  localStorage.setItem("access_token", access)
  localStorage.setItem("refresh_token", refresh)
}

export function clearTokens() {
  accessToken = null
  refreshToken = null
  refreshPromise = null
  localStorage.removeItem("access_token")
  localStorage.removeItem("refresh_token")
}

export function getAccessToken() {
  return accessToken
}

async function refreshAccessToken(): Promise<string | null> {
  // If a refresh is already in-flight, wait for it instead of starting another
  if (refreshPromise) {
    return refreshPromise
  }

  if (!refreshToken) return null

  refreshPromise = (async (): Promise<string | null> => {
    try {
      const res = await fetch(`${BASE_URL}/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ refresh_token: refreshToken }),
      })
      if (!res.ok) {
        clearTokens()
        return null
      }
      const data = await res.json()
      accessToken = data.access_token
      localStorage.setItem("access_token", data.access_token)
      return data.access_token
    } catch {
      clearTokens()
      return null
    } finally {
      refreshPromise = null
    }
  })()

  return refreshPromise
}

export async function apiRequest<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  }

  if (accessToken) {
    headers["Authorization"] = `Bearer ${accessToken}`
  }

  let res = await fetch(`${BASE_URL}${endpoint}`, { ...options, headers })

  // If 401, try refreshing token
  if (res.status === 401 && refreshToken) {
    const newToken = await refreshAccessToken()
    if (newToken) {
      headers["Authorization"] = `Bearer ${newToken}`
      res = await fetch(`${BASE_URL}${endpoint}`, { ...options, headers })
    }
  }

  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: "Request failed" }))
    throw new Error(error.error || `HTTP ${res.status}`)
  }

  // Handle empty JSON response
  let json: T
  try {
    json = await res.json()
  } catch (e) {
    // If parsing fails (e.g., empty body), treat as null
    json = null as unknown as T
  }
  return json
}

// Convenience methods
export const api = {
  get: <T>(endpoint: string) => apiRequest<T>(endpoint),
  post: <T>(endpoint: string, data?: unknown) =>
    apiRequest<T>(endpoint, { method: "POST", body: data ? JSON.stringify(data) : undefined }),
  put: <T>(endpoint: string, data?: unknown) =>
    apiRequest<T>(endpoint, { method: "PUT", body: data ? JSON.stringify(data) : undefined }),
  patch: <T>(endpoint: string, data?: unknown) =>
    apiRequest<T>(endpoint, { method: "PATCH", body: data ? JSON.stringify(data) : undefined }),
  delete: <T>(endpoint: string) => apiRequest<T>(endpoint, { method: "DELETE" }),
}