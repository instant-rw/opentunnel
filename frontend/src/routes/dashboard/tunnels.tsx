import {
  ArrowSquareOutIcon,
  GlobeIcon,
  PlusIcon,
  HardDrivesIcon,
} from "@phosphor-icons/react"
import { createFileRoute } from "@tanstack/react-router"

import { CopyButton } from "@/components/copy-button"
import { PageHeader } from "@/components/page-header"
import { RequestList } from "@/components/request-list"
import { TunnelCard } from "@/components/tunnel-card"
import { Badge } from "@/components/ui/badge"
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
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { useDashboard } from "@/lib/dashboard-context"
import { cn } from "@/lib/utils"

export const Route = createFileRoute("/dashboard/tunnels")({
  component: TunnelsPage,
})

function TunnelsPage() {
  const {
    domains,
    loading,
    requestsLoading,
    loadingMore,
    hasMoreRequests,
    requestFilters,
    selectedDomain,
    setSelectedDomainId,
    streamState,
    visibleRequests,
    setSelectedRequest,
    setRequestFilters,
    loadMoreRequests,
    selectedRequest,
    setNewDomainOpen,
  } = useDashboard()

  return (
    <>
      <PageHeader
        action={
          <Button onClick={() => setNewDomainOpen(true)}>
            <PlusIcon /> New domain
          </Button>
        }
        description="Manage reserved domains and inspect their active tunnel sessions."
        eyebrow="Traffic"
        title="Tunnels"
      />

      <div className="mb-6 grid gap-4 lg:grid-cols-[320px_1fr]">
        <Card>
          <CardHeader>
            <CardTitle>Domains</CardTitle>
            <CardDescription>{domains.length} reserved</CardDescription>
          </CardHeader>
          <CardContent className="space-y-2">
            {domains.map((domain) => (
              <TunnelCard
                domain={domain}
                key={domain.id}
                onSelect={() => setSelectedDomainId(domain.id)}
                selected={domain.id === selectedDomain?.id}
              />
            ))}
          </CardContent>
        </Card>

        {selectedDomain ? (
          <Card>
            <CardHeader className="flex-row items-start justify-between space-y-0">
              <div className="flex items-start gap-3">
                <span className="flex size-11 items-center justify-center bg-muted">
                  <GlobeIcon className="size-5" />
                </span>
                <div>
                  <CardTitle className="text-lg">
                    {selectedDomain.hostname}
                  </CardTitle>
                  <Badge
                    className="mt-2"
                    variant={
                      selectedDomain.status === "online" ? "default" : "outline"
                    }
                  >
                    <span
                      className={cn(
                        "size-1.5 rounded-full",
                        selectedDomain.status === "online"
                          ? "bg-emerald-500"
                          : "bg-muted-foreground"
                      )}
                    />
                    {selectedDomain.status}
                  </Badge>
                </div>
              </div>
              <Button
                onClick={() =>
                  window.open(`https://${selectedDomain.hostname}`)
                }
                size="sm"
                variant="secondary"
              >
                Open URL <ArrowSquareOutIcon />
              </Button>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid gap-3 sm:grid-cols-3">
                <div>
                  <span className="text-xs text-muted-foreground">
                    Local target
                  </span>
                  <strong className="mt-1 block text-sm">
                    {selectedDomain.status === "online"
                      ? "localhost:3000"
                      : "Not connected"}
                  </strong>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">
                    Live stream
                  </span>
                  <strong className="mt-1 block text-sm">
                    {streamState === "open" ? "Connected" : "Reconnecting"}
                  </strong>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">Created</span>
                  <strong className="mt-1 block text-sm">
                    {new Date(selectedDomain.createdAt).toLocaleDateString()}
                  </strong>
                </div>
              </div>

              {selectedDomain.status === "offline" ? (
                <div className="flex flex-col gap-3 border bg-muted/30 p-3 sm:flex-row sm:items-center">
                  <HardDrivesIcon className="size-5 shrink-0" />
                  <div className="flex-1 text-sm">
                    <strong className="block">This tunnel is offline</strong>
                    <p className="text-muted-foreground">
                      Start it with{" "}
                      <code className="font-mono text-xs">
                        opentunnel up 3000 --domain {selectedDomain.slug}
                      </code>
                    </p>
                  </div>
                  <CopyButton
                    value={`opentunnel up 3000 --domain ${selectedDomain.slug}`}
                  />
                </div>
              ) : (
                <div className="flex items-center gap-2 border border-emerald-500/20 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-800">
                  <span className="size-1.5 animate-pulse rounded-full bg-emerald-500" />
                  Connected and forwarding requests
                </div>
              )}
            </CardContent>
          </Card>
        ) : (
          <Empty>
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <GlobeIcon />
              </EmptyMedia>
              <EmptyTitle>No tunnel selected</EmptyTitle>
              <EmptyDescription>
                Reserve a domain to view its tunnel status.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}
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
