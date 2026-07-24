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
  requestsPage: number
  hasPreviousRequests: boolean
  hasMoreRequests: boolean
  requestFilters: RequestFilters
  error: string
  newDomainOpen: boolean
  setNewDomainOpen: (open: boolean) => void
  setSelectedDomainId: (id: string) => void
  setSelectedRequest: (request?: CapturedRequest) => void
  setRequestFilters: (filters: RequestFilters) => void
  goToPreviousRequests: () => void
  goToNextRequests: () => void
  loadDashboard: () => Promise<void>
  onDomainCreated: (domain: Domain) => void
  revokeToken: (id: string) => Promise<void>
  signOut: () => Promise<void>
  visibleRequests: CapturedRequest[]
  onlineCount: number
}

const DashboardContext = createContext<DashboardContextValue | null>(null)

const PAGE_SIZE = 20

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
  const [page, setPage] = useState(0)
  const [pageCursors, setPageCursors] = useState<(string | undefined)[]>([
    undefined,
  ])
  const [nextCursor, setNextCursor] = useState<string>()
  const [requestFilters, setRequestFiltersState] =
    useState<RequestFilters>(emptyFilters)
  const [newDomainOpen, setNewDomainOpen] = useState(false)
  const [error, setError] = useState("")
  const requestFiltersRef = useRef(requestFilters)
  const pageRef = useRef(page)

  useEffect(() => {
    requestFiltersRef.current = requestFilters
  }, [requestFilters])

  useEffect(() => {
    pageRef.current = page
  }, [page])

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

  const resetRequestPages = useCallback(() => {
    setPage(0)
    setPageCursors([undefined])
    setNextCursor(undefined)
  }, [])

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
        resetRequestPages()
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
  }, [resetRequestPages, selectedDomainId])

  useEffect(() => {
    const timeout = window.setTimeout(() => void loadDashboard(), 0)
    return () => window.clearTimeout(timeout)
  }, [loadDashboard])

  const pageCursor = pageCursors[page]

  useEffect(() => {
    if (!selectedDomainId) {
      setRequests([])
      resetRequestPages()
      return
    }

    let cancelled = false
    setRequestsLoading(true)
    setError("")

    void api
      .listRequests(
        selectedDomainId,
        toListRequestsQuery(requestFilters, {
          cursor: pageCursor,
          limit: PAGE_SIZE,
        })
      )
      .then((result) => {
        if (cancelled) return
        setRequests(result.items)
        setNextCursor(result.nextCursor)
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
  }, [page, pageCursor, requestFilters, resetRequestPages, selectedDomainId])

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
              const matches = requestMatchesFilters(
                event.request,
                requestFiltersRef.current
              )
              const exists = current.some(
                (item) => item.id === event.request.id
              )
              if (!matches) {
                return current.filter((item) => item.id !== event.request.id)
              }
              if (exists) {
                return current.map((item) =>
                  item.id === event.request.id ? event.request : item
                )
              }
              if (pageRef.current !== 0 || event.type !== "request.created") {
                return current
              }
              return [event.request, ...current].slice(0, PAGE_SIZE)
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

  const setRequestFilters = useCallback((filters: RequestFilters) => {
    setRequestFiltersState(filters)
    setPage(0)
    setPageCursors([undefined])
    setNextCursor(undefined)
  }, [])

  const setSelectedDomainIdAndReset = useCallback((id: string) => {
    setSelectedDomainId(id)
    setPage(0)
    setPageCursors([undefined])
    setNextCursor(undefined)
  }, [])

  const goToPreviousRequests = useCallback(() => {
    setPage((current) => Math.max(0, current - 1))
  }, [])

  const goToNextRequests = useCallback(() => {
    if (!nextCursor) return
    setPageCursors((current) => {
      const next = current.slice(0, page + 1)
      next[page + 1] = nextCursor
      return next
    })
    setPage((current) => current + 1)
  }, [nextCursor, page])

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
    setSelectedDomainIdAndReset(domain.id)
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
        requestsPage: page + 1,
        hasPreviousRequests: page > 0,
        hasMoreRequests: Boolean(nextCursor),
        requestFilters,
        error,
        newDomainOpen,
        setNewDomainOpen,
        setSelectedDomainId: setSelectedDomainIdAndReset,
        setSelectedRequest,
        setRequestFilters,
        goToPreviousRequests,
        goToNextRequests,
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
