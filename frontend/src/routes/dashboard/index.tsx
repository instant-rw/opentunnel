import {
  ActivityIcon,
  ArrowUpRightIcon,
  GaugeIcon,
  GlobeIcon,
  PlusIcon,
  ShieldCheckIcon,
  BroadcastIcon,
  TerminalWindowIcon,
} from "@phosphor-icons/react"
import { Link, createFileRoute } from "@tanstack/react-router"

import { CopyButton } from "@/components/copy-button"
import { PageHeader } from "@/components/page-header"
import { RequestList } from "@/components/request-list"
import { TunnelCard } from "@/components/tunnel-card"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Skeleton } from "@/components/ui/skeleton"
import { useDashboard } from "@/lib/dashboard-context"

export const Route = createFileRoute("/dashboard/")({
  component: OverviewPage,
})

function StatCard({
  label,
  value,
  detail,
  icon,
}: {
  label: string
  value: string
  detail: string
  icon: React.ReactNode
}) {
  return (
    <Card size="sm">
      <CardHeader className="flex-row items-center justify-between space-y-0">
        <CardDescription>{label}</CardDescription>
        <span className="text-muted-foreground">{icon}</span>
      </CardHeader>
      <CardContent>
        <div className="font-heading text-2xl font-medium">{value}</div>
        <p className="mt-1 text-xs text-muted-foreground">{detail}</p>
      </CardContent>
    </Card>
  )
}

function OverviewPage() {
  const {
    user,
    loading,
    requestsLoading,
    loadingMore,
    hasMoreRequests,
    requestFilters,
    domains,
    visibleRequests,
    onlineCount,
    selectedDomain,
    setSelectedDomainId,
    setSelectedRequest,
    setRequestFilters,
    loadMoreRequests,
    selectedRequest,
    setNewDomainOpen,
  } = useDashboard()

  const name = user.email.split("@")[0] || "there"
  const successful = visibleRequests.filter(
    (request) =>
      request.response &&
      request.response.status >= 200 &&
      request.response.status < 400
  ).length
  const successRate = visibleRequests.length
    ? Math.round((successful / visibleRequests.length) * 100)
    : 0
  const avgLatency = visibleRequests.length
    ? `${Math.round(
        visibleRequests.reduce(
          (total, request) => total + (request.response?.durationMs ?? 0),
          0
        ) / visibleRequests.length
      )} ms`
    : "—"

  return (
    <>
      <PageHeader
        action={
          <Button onClick={() => setNewDomainOpen(true)}>
            <PlusIcon /> New domain
          </Button>
        }
        description="Monitor tunnel health, inspect traffic, and keep localhost connected."
        eyebrow="Dashboard"
        title={`Good afternoon, ${name}`}
      />

      <div className="mb-6 grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          detail={`${onlineCount} active now`}
          icon={<BroadcastIcon className="size-4" />}
          label="Active tunnels"
          value={loading ? "—" : String(onlineCount)}
        />
        <StatCard
          detail="Across selected tunnel"
          icon={<ActivityIcon className="size-4" />}
          label="Requests today"
          value={loading ? "—" : String(visibleRequests.length)}
        />
        <StatCard
          detail="p50 response time"
          icon={<GaugeIcon className="size-4" />}
          label="Avg. latency"
          value={avgLatency}
        />
        <StatCard
          detail="2xx and 3xx responses"
          icon={<ShieldCheckIcon className="size-4" />}
          label="Success rate"
          value={`${successRate}%`}
        />
      </div>

      <div className="mb-6 grid gap-4 lg:grid-cols-[1.4fr_1fr]">
        <Card>
          <CardHeader className="flex-row items-start justify-between space-y-0">
            <div>
              <CardTitle>Your tunnels</CardTitle>
              <CardDescription>
                Persistent domains and their current session.
              </CardDescription>
            </div>
            <Button
              render={<Link to="/dashboard/tunnels" />}
              size="sm"
              variant="ghost"
            >
              View all <ArrowUpRightIcon />
            </Button>
          </CardHeader>
          <CardContent className="space-y-2">
            {loading ? (
              <>
                <Skeleton className="h-16 w-full" />
                <Skeleton className="h-16 w-full" />
              </>
            ) : domains.length ? (
              domains
                .slice(0, 3)
                .map((domain) => (
                  <TunnelCard
                    domain={domain}
                    key={domain.id}
                    onSelect={() => setSelectedDomainId(domain.id)}
                    selected={domain.id === selectedDomain?.id}
                  />
                ))
            ) : (
              <Empty className="border-0 py-8">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <GlobeIcon />
                  </EmptyMedia>
                  <EmptyTitle>No domains yet</EmptyTitle>
                  <EmptyDescription>
                    Reserve a persistent URL for your first tunnel.
                  </EmptyDescription>
                </EmptyHeader>
                <EmptyContent>
                  <Button onClick={() => setNewDomainOpen(true)} size="sm">
                    <PlusIcon /> Reserve domain
                  </Button>
                </EmptyContent>
              </Empty>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="mb-2 flex size-9 items-center justify-center bg-muted">
              <TerminalWindowIcon className="size-4" />
            </div>
            <CardTitle>Start a tunnel</CardTitle>
            <CardDescription>
              Run this command from your project directory.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex items-center justify-between gap-2 bg-muted/50 px-3 py-2 font-mono text-xs">
              <code>opentunnel up 3000</code>
              <CopyButton value="opentunnel up 3000" />
            </div>
            <Button
              render={<Link to="/dashboard/cli" />}
              size="sm"
              variant="ghost"
            >
              CLI setup guide <ArrowUpRightIcon />
            </Button>
          </CardContent>
        </Card>
      </div>

      <RequestList
        filters={requestFilters}
        hasMore={hasMoreRequests}
        loading={loading || requestsLoading}
        loadingMore={loadingMore}
        onFiltersChange={setRequestFilters}
        onLoadMore={() => void loadMoreRequests()}
        onSelect={setSelectedRequest}
        requests={visibleRequests}
        selectedId={selectedRequest?.id}
      />
    </>
  )
}
