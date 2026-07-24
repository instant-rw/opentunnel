import type { components } from "@/lib/api.generated"

export type User = components["schemas"]["User"]
export type Domain = components["schemas"]["Domain"]
export type CapturedRequest = components["schemas"]["CapturedRequest"]
export type RequestPage = components["schemas"]["RequestPage"]
export type Replay = components["schemas"]["Replay"]
export type TokenSummary = components["schemas"]["TokenSummary"]
export type Problem = components["schemas"]["Problem"]

export type StatusClass = "" | "2xx" | "3xx" | "4xx" | "5xx"
export type PathMode = "include" | "exclude"

export type RequestFilters = {
  method: string
  path: string
  pathMode: PathMode
  statusClass: StatusClass
}

export type ListRequestsQuery = {
  cursor?: string
  limit?: number
  method?: string
  path?: string
  pathMode?: PathMode
  statusMin?: number
  statusMax?: number
}

export class ApiError extends Error {
  readonly status: number
  readonly problem?: Problem

  constructor(status: number, message: string, problem?: Problem) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.problem = problem
  }
}

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: object
}

const apiBase =
  import.meta.env.VITE_API_URL?.replace(/\/$/, "") ||
  "http://localhost:8080/api/v1"

const emptyRequestFilters: RequestFilters = {
  method: "",
  path: "",
  pathMode: "include",
  statusClass: "",
}

export function emptyFilters(): RequestFilters {
  return { ...emptyRequestFilters }
}

export function statusClassRange(
  statusClass: StatusClass
): Pick<ListRequestsQuery, "statusMin" | "statusMax"> {
  switch (statusClass) {
    case "2xx":
      return { statusMin: 200, statusMax: 299 }
    case "3xx":
      return { statusMin: 300, statusMax: 399 }
    case "4xx":
      return { statusMin: 400, statusMax: 499 }
    case "5xx":
      return { statusMin: 500, statusMax: 599 }
    case "":
      return {}
    default: {
      const _exhaustive: never = statusClass
      void _exhaustive
      return {}
    }
  }
}

export function toListRequestsQuery(
  filters: RequestFilters,
  options: { cursor?: string; limit?: number } = {}
): ListRequestsQuery {
  const query: ListRequestsQuery = {
    limit: options.limit ?? 20,
  }
  if (options.cursor) {
    query.cursor = options.cursor
  }
  if (filters.method) {
    query.method = filters.method
  }
  if (filters.path) {
    query.path = filters.path
    query.pathMode = filters.pathMode
  }
  Object.assign(query, statusClassRange(filters.statusClass))
  return query
}

export function requestMatchesFilters(
  request: CapturedRequest,
  filters: RequestFilters
): boolean {
  if (
    filters.method &&
    request.method.toUpperCase() !== filters.method.toUpperCase()
  ) {
    return false
  }
  if (filters.path) {
    const matches = request.path
      .toLowerCase()
      .includes(filters.path.toLowerCase())
    if (filters.pathMode === "exclude" ? matches : !matches) {
      return false
    }
  }
  if (filters.statusClass) {
    const status = request.response?.status
    if (status == null) {
      return false
    }
    const { statusMin, statusMax } = statusClassRange(filters.statusClass)
    if (statusMin != null && status < statusMin) {
      return false
    }
    if (statusMax != null && status > statusMax) {
      return false
    }
  }
  return true
}

export function filtersActive(filters: RequestFilters): boolean {
  return Boolean(filters.method || filters.path || filters.statusClass)
}

function readCsrfToken(): string | undefined {
  if (typeof document === "undefined") {
    return undefined
  }

  return document.cookie
    .split("; ")
    .find((cookie) => cookie.startsWith("opentunnel_csrf="))
    ?.split("=")[1]
}

async function request<T>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const headers = new Headers(options.headers)
  const csrfToken = readCsrfToken()

  headers.set("Accept", "application/json")
  if (options.body) {
    headers.set("Content-Type", "application/json")
  }
  if (csrfToken) {
    headers.set("X-CSRF-Token", decodeURIComponent(csrfToken))
  }

  let response: Response
  try {
    response = await fetch(`${apiBase}${path}`, {
      ...options,
      body: options.body ? JSON.stringify(options.body) : undefined,
      credentials: "include",
      headers,
    })
  } catch {
    throw new ApiError(0, "Unable to reach the OpenTunnel API.")
  }

  if (!response.ok) {
    let problem: Problem | undefined
    try {
      problem = (await response.json()) as Problem
    } catch {
      problem = undefined
    }
    throw new ApiError(
      response.status,
      problem?.detail ??
        problem?.title ??
        `Request failed (${response.status}).`,
      problem
    )
  }

  if (response.status === 204) {
    return undefined as T
  }

  return (await response.json()) as T
}

export function getApiBaseUrl(): string {
  return apiBase
}

export const api = {
  register: (email: string, password: string) =>
    request<User>("/auth/register", {
      method: "POST",
      body: { email, password },
    }),
  login: (email: string, password: string) =>
    request<void>("/auth/login", {
      method: "POST",
      body: { email, password },
    }),
  logout: () => request<void>("/auth/logout", { method: "POST" }),
  me: () => request<User>("/auth/me"),
  approveDevice: (userCode: string) =>
    request<void>(
      `/device/authorizations/${encodeURIComponent(userCode)}/approve`,
      { method: "POST" }
    ),
  listDomains: () => request<Domain[]>("/domains"),
  getDomain: (domainId: string) =>
    request<Domain>(`/domains/${encodeURIComponent(domainId)}`),
  createDomain: (slug: string) =>
    request<Domain>("/domains", { method: "POST", body: { slug } }),
  deleteDomain: (domainId: string) =>
    request<void>(`/domains/${encodeURIComponent(domainId)}`, {
      method: "DELETE",
    }),
  listRequests: (domainId: string, query: ListRequestsQuery = {}) => {
    const params = new URLSearchParams()
    if (query.limit != null) {
      params.set("limit", String(query.limit))
    }
    if (query.cursor) {
      params.set("cursor", query.cursor)
    }
    if (query.method) {
      params.set("method", query.method)
    }
    if (query.path) {
      params.set("path", query.path)
    }
    if (query.pathMode) {
      params.set("pathMode", query.pathMode)
    }
    if (query.statusMin != null) {
      params.set("statusMin", String(query.statusMin))
    }
    if (query.statusMax != null) {
      params.set("statusMax", String(query.statusMax))
    }
    const search = params.toString()
    return request<RequestPage>(
      `/domains/${encodeURIComponent(domainId)}/requests${search ? `?${search}` : ""}`
    )
  },
  getRequest: (requestId: string) =>
    request<CapturedRequest>(`/requests/${encodeURIComponent(requestId)}`),
  replayRequest: (requestId: string) =>
    request<Replay>(`/requests/${encodeURIComponent(requestId)}/replays`, {
      method: "POST",
    }),
  listTokens: () => request<TokenSummary[]>("/tokens"),
  revokeToken: (tokenId: string) =>
    request<void>(`/tokens/${encodeURIComponent(tokenId)}`, {
      method: "DELETE",
    }),
}
