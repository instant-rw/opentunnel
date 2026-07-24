import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react"

import {
  api,
  ApiError,
  emptyFilters,
  requestMatchesFilters,
  toListRequestsQuery,
  type CapturedRequest,
  type Domain,
  type RequestFilters,
  type TokenSummary,
  type User,
} from "@/lib/api"
import { subscribeToDomain, type StreamState } from "@/lib/sse"

type DashboardContextValue = {
  user: User
  domains: Domain[]
  tokens: TokenSummary[]
  requests: CapturedRequest[]
  selectedDomain?: Domain
  selectedDomainId: string
  selectedRequest?: CapturedRequest
  streamState: StreamState
  loading: boolean
  requestsLoading: boolean
  loadingMore: boolean
  hasMoreRequests: boolean
  requestFilters: RequestFilters
  error: string
  newDomainOpen: boolean
  setNewDomainOpen: (open: boolean) => void
  setSelectedDomainId: (id: string) => void
  setSelectedRequest: (request?: CapturedRequest) => void
  setRequestFilters: (filters: RequestFilters) => void
  loadMoreRequests: () => Promise<void>
  loadDashboard: () => Promise<void>
  onDomainCreated: (domain: Domain) => void
  revokeToken: (id: string) => Promise<void>
  signOut: () => Promise<void>
  visibleRequests: CapturedRequest[]
  onlineCount: number
}

const DashboardContext = createContext<DashboardContextValue | null>(null)

const PAGE_SIZE = 50

export function DashboardProvider({
  user,
  children,
  onSignOut,
}: {
  user: User
  children: ReactNode
  onSignOut: () => void
}) {
  const [domains, setDomains] = useState<Domain[]>([])
  const [requests, setRequests] = useState<CapturedRequest[]>([])
  const [tokens, setTokens] = useState<TokenSummary[]>([])
  const [selectedDomainId, setSelectedDomainId] = useState("")
  const [selectedRequest, setSelectedRequest] = useState<CapturedRequest>()
  const [streamState, setStreamState] = useState<StreamState>("closed")
  const [loading, setLoading] = useState(true)
  const [requestsLoading, setRequestsLoading] = useState(false)
  const [loadingMore, setLoadingMore] = useState(false)
  const [nextCursor, setNextCursor] = useState<string>()
  const [requestFilters, setRequestFiltersState] =
    useState<RequestFilters>(emptyFilters)
  const [newDomainOpen, setNewDomainOpen] = useState(false)
  const [error, setError] = useState("")
  const requestFiltersRef = useRef(requestFilters)

  useEffect(() => {
    requestFiltersRef.current = requestFilters
  }, [requestFilters])

  const selectedDomain = useMemo(
    () =>
      domains.find((domain) => domain.id === selectedDomainId) ?? domains[0],
    [domains, selectedDomainId]
  )

  const visibleRequests = useMemo(
    () =>
      selectedDomain
        ? requests.filter((request) => request.domainId === selectedDomain.id)
        : requests,
    [requests, selectedDomain]
  )

  const loadDashboard = useCallback(async () => {
    setLoading(true)
    setError("")
    try {
      const [loadedDomains, loadedTokens] = await Promise.all([
        api.listDomains(),
        api.listTokens(),
      ])
      setDomains(loadedDomains)
      setTokens(loadedTokens)
      const domainId = selectedDomainId || loadedDomains[0]?.id
      if (domainId) {
        setSelectedDomainId(domainId)
      } else {
        setRequests([])
        setNextCursor(undefined)
      }
    } catch (caught) {
      setError(
        caught instanceof ApiError
          ? caught.message
          : "Could not load dashboard."
      )
    } finally {
      setLoading(false)
    }
  }, [selectedDomainId])

  useEffect(() => {
    const timeout = window.setTimeout(() => void loadDashboard(), 0)
    return () => window.clearTimeout(timeout)
  }, [loadDashboard])

  useEffect(() => {
    if (!selectedDomainId) {
      setRequests([])
      setNextCursor(undefined)
      return
    }

    let cancelled = false
    setRequestsLoading(true)
    setError("")

    void api
      .listRequests(
        selectedDomainId,
        toListRequestsQuery(requestFilters, { limit: PAGE_SIZE })
      )
      .then((page) => {
        if (cancelled) return
        setRequests(page.items)
        setNextCursor(page.nextCursor)
      })
      .catch((caught) => {
        if (cancelled) return
        setRequests([])
        setNextCursor(undefined)
        setError(
          caught instanceof ApiError
            ? caught.message
            : "Could not load requests."
        )
      })
      .finally(() => {
        if (!cancelled) {
          setRequestsLoading(false)
        }
      })

    return () => {
      cancelled = true
    }
  }, [selectedDomainId, requestFilters])

  useEffect(() => {
    if (!selectedDomainId) return
    return subscribeToDomain(selectedDomainId, {
      onStateChange: setStreamState,
      onEvent: (event) => {
        switch (event.type) {
          case "domain.status":
            setDomains((current) =>
              current.map((domain) =>
                domain.id === event.domain.id ? event.domain : domain
              )
            )
            break
          case "request.created":
          case "request.updated":
            setRequests((current) => {
              if (
                !requestMatchesFilters(event.request, requestFiltersRef.current)
              ) {
                return current.filter((item) => item.id !== event.request.id)
              }
              return [
                event.request,
                ...current.filter((item) => item.id !== event.request.id),
              ]
            })
            setSelectedRequest((current) =>
              current?.id === event.request.id ? event.request : current
            )
            break
          default: {
            const _exhaustive: never = event
            void _exhaustive
          }
        }
      },
    })
  }, [selectedDomainId])

  const loadMoreRequests = useCallback(async () => {
    if (!selectedDomainId || !nextCursor || loadingMore) return
    setLoadingMore(true)
    setError("")
    try {
      const page = await api.listRequests(
        selectedDomainId,
        toListRequestsQuery(requestFilters, {
          cursor: nextCursor,
          limit: PAGE_SIZE,
        })
      )
      setRequests((current) => {
        const seen = new Set(current.map((item) => item.id))
        return [...current, ...page.items.filter((item) => !seen.has(item.id))]
      })
      setNextCursor(page.nextCursor)
    } catch (caught) {
      setError(
        caught instanceof ApiError
          ? caught.message
          : "Could not load more requests."
      )
    } finally {
      setLoadingMore(false)
    }
  }, [loadingMore, nextCursor, requestFilters, selectedDomainId])

  const setRequestFilters = useCallback((filters: RequestFilters) => {
    setRequestFiltersState(filters)
  }, [])

  async function signOut() {
    try {
      await api.logout()
    } catch {
      // Clear local UI state even if the API is unavailable.
    }
    onSignOut()
  }

  async function revokeToken(id: string) {
    await api.revokeToken(id)
    setTokens((current) => current.filter((token) => token.id !== id))
  }

  function onDomainCreated(domain: Domain) {
    setDomains((current) => [domain, ...current])
    setSelectedDomainId(domain.id)
  }

  const onlineCount = domains.filter(
    (domain) => domain.status === "online"
  ).length

  return (
    <DashboardContext.Provider
      value={{
        user,
        domains,
        tokens,
        requests,
        selectedDomain,
        selectedDomainId,
        selectedRequest,
        streamState,
        loading,
        requestsLoading,
        loadingMore,
        hasMoreRequests: Boolean(nextCursor),
        requestFilters,
        error,
        newDomainOpen,
        setNewDomainOpen,
        setSelectedDomainId,
        setSelectedRequest,
        setRequestFilters,
        loadMoreRequests,
        loadDashboard,
        onDomainCreated,
        revokeToken,
        signOut,
        visibleRequests,
        onlineCount,
      }}
    >
      {children}
    </DashboardContext.Provider>
  )
}

export function useDashboard() {
  const context = useContext(DashboardContext)
  if (!context) {
    throw new Error("useDashboard must be used within DashboardProvider")
  }
  return context
}
