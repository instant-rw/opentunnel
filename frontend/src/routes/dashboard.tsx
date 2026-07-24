import {
  Outlet,
  createFileRoute,
  redirect,
  useNavigate,
} from "@tanstack/react-router"

import { AppSidebar } from "@/components/app-sidebar"
import { NewDomainDialog } from "@/components/new-domain-dialog"
import { RequestInspector } from "@/components/request-inspector"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { Spinner } from "@/components/ui/spinner"
import { api } from "@/lib/api"
import { DashboardProvider, useDashboard } from "@/lib/dashboard-context"

export const Route = createFileRoute("/dashboard")({
  beforeLoad: async () => {
    try {
      const user = await api.me()
      return { user }
    } catch {
      throw redirect({ to: "/login" })
    }
  },
  component: DashboardLayout,
  pendingComponent: DashboardPending,
})

function DashboardPending() {
  return (
    <main className="flex min-h-svh items-center justify-center">
      <Spinner className="size-6" />
    </main>
  )
}

function DashboardLayout() {
  const { user } = Route.useRouteContext()
  const navigate = useNavigate()

  return (
    <DashboardProvider
      onSignOut={() => void navigate({ to: "/login" })}
      user={user}
    >
      <DashboardShell />
    </DashboardProvider>
  )
}

function DashboardShell() {
  const {
    error,
    loadDashboard,
    newDomainOpen,
    setNewDomainOpen,
    domains,
    onDomainCreated,
    selectedRequest,
    setSelectedRequest,
    selectedDomain,
  } = useDashboard()

  return (
    <SidebarProvider>
      <AppSidebar />
      <SidebarInset>
        <header className="flex h-14 shrink-0 items-center gap-2 border-b px-4">
          <SidebarTrigger className="-ml-1" />
          <Separator
            orientation="vertical"
            className="mr-2 data-[orientation=vertical]:h-4"
          />
          <div className="flex flex-1 items-center justify-between gap-3">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <span className="size-1.5 animate-pulse rounded-full bg-emerald-500" />
              All systems operational
            </div>
          </div>
        </header>
        <div className="flex flex-1 flex-col gap-4 p-4 md:p-6">
          {error ? (
            <div className="flex items-center justify-between gap-3 border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm">
              <span>{error}</span>
              <Button onClick={() => void loadDashboard()} size="sm" variant="secondary">
                Retry
              </Button>
            </div>
          ) : null}
          <Outlet />
        </div>
      </SidebarInset>

      <NewDomainDialog
        domains={domains}
        onClose={() => setNewDomainOpen(false)}
        onCreated={onDomainCreated}
        open={newDomainOpen}
      />

      {selectedRequest ? (
        <RequestInspector
          online={selectedDomain?.status === "online"}
          onClose={() => setSelectedRequest(undefined)}
          request={selectedRequest}
        />
      ) : null}
    </SidebarProvider>
  )
}
