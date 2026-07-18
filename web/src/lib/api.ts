import type { components } from "@/lib/api.generated";

export type User = components["schemas"]["User"];
export type Domain = components["schemas"]["Domain"];
export type CapturedRequest = components["schemas"]["CapturedRequest"];
export type RequestPage = components["schemas"]["RequestPage"];
export type Replay = components["schemas"]["Replay"];
export type TokenSummary = components["schemas"]["TokenSummary"];
export type Problem = components["schemas"]["Problem"];

export class ApiError extends Error {
  readonly status: number;
  readonly problem?: Problem;

  constructor(status: number, message: string, problem?: Problem) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.problem = problem;
  }
}

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: object;
};

const apiBase = "/api/v1";

function readCsrfToken(): string | undefined {
  if (typeof document === "undefined") {
    return undefined;
  }

  return document.cookie
    .split("; ")
    .find((cookie) => cookie.startsWith("opentunnel_csrf="))
    ?.split("=")[1];
}

async function request<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const headers = new Headers(options.headers);
  const csrfToken = readCsrfToken();

  headers.set("Accept", "application/json");
  if (options.body) {
    headers.set("Content-Type", "application/json");
  }
  if (csrfToken) {
    headers.set("X-CSRF-Token", decodeURIComponent(csrfToken));
  }

  let response: Response;
  try {
    response = await fetch(`${apiBase}${path}`, {
      ...options,
      body: options.body ? JSON.stringify(options.body) : undefined,
      credentials: "include",
      headers,
    });
  } catch {
    throw new ApiError(0, "Unable to reach the OpenTunnel API.");
  }

  if (!response.ok) {
    let problem: Problem | undefined;
    try {
      problem = (await response.json()) as Problem;
    } catch {
      problem = undefined;
    }
    throw new ApiError(
      response.status,
      problem?.detail ??
        problem?.title ??
        `Request failed (${response.status}).`,
      problem,
    );
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
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
      { method: "POST" },
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
  listRequests: (domainId: string, cursor?: string, limit = 50) => {
    const query = new URLSearchParams({ limit: String(limit) });
    if (cursor) {
      query.set("cursor", cursor);
    }
    return request<RequestPage>(
      `/domains/${encodeURIComponent(domainId)}/requests?${query}`,
    );
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
};
