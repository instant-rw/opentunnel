import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react"

import {
  api,
  ApiError,
  type CapturedRequest,
  type Domain,
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
  error: string
  newDomainOpen: boolean
  setNewDomainOpen: (open: boolean) => void
  setSelectedDomainId: (id: string) => void
  setSelectedRequest: (request?: CapturedRequest) => void
  loadDashboard: () => Promise<void>
  onDomainCreated: (domain: Domain) => void
  revokeToken: (id: string) => Promise<void>
  signOut: () => Promise<void>
  visibleRequests: CapturedRequest[]
  onlineCount: number
}

const DashboardContext = createContext<DashboardContextValue | null>(null)

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
  const [newDomainOpen, setNewDomainOpen] = useState(false)
  const [error, setError] = useState("")

  const selectedDomain = useMemo(
    () =>
      domains.find((domain) => domain.id === selectedDomainId) ?? domains[0],
    [domains, selectedDomainId],
  )

  const visibleRequests = useMemo(
    () =>
      selectedDomain
        ? requests.filter((request) => request.domainId === selectedDomain.id)
        : requests,
    [requests, selectedDomain],
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
        const page = await api.listRequests(domainId)
        setRequests(page.items)
      }
    } catch (caught) {
      setError(
        caught instanceof ApiError
          ? caught.message
          : "Could not load dashboard.",
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
    if (!selectedDomainId) return
    return subscribeToDomain(selectedDomainId, {
      onStateChange: setStreamState,
      onEvent: (event) => {
        switch (event.type) {
          case "domain.status":
            setDomains((current) =>
              current.map((domain) =>
                domain.id === event.domain.id ? event.domain : domain,
              ),
            )
            break
          case "request.created":
          case "request.updated":
            setRequests((current) => [
              event.request,
              ...current.filter((item) => item.id !== event.request.id),
            ])
            setSelectedRequest((current) =>
              current?.id === event.request.id ? event.request : current,
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
    (domain) => domain.status === "online",
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
        error,
        newDomainOpen,
        setNewDomainOpen,
        setSelectedDomainId,
        setSelectedRequest,
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
