import {
  Outlet,
  createFileRoute,
  redirect,
  useNavigate,
} from "@tanstack/react-router"

import { AppSidebar } from "@/components/app-sidebar"
import { NewDomainDialog } from "@/components/new-domain-dialog"
import { RequestInspector } from "@/components/request-inspector"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { Spinner } from "@/components/ui/spinner"
import { api, ApiError } from "@/lib/api"
import { DashboardProvider, useDashboard } from "@/lib/dashboard-context"
import { previewUser } from "@/lib/fixtures"

type DashboardSearch = {
  preview?: boolean
}

export const Route = createFileRoute("/dashboard")({
  validateSearch: (search: Record<string, unknown>): DashboardSearch => ({
    preview: search.preview === true || search.preview === "true",
  }),
  beforeLoad: async ({ search }) => {
    if (search.preview) {
      return { user: previewUser, preview: true }
    }
    try {
      const user = await api.me()
      return { user, preview: false }
    } catch (error) {
      if (error instanceof ApiError && error.status === 0) {
        // Offline / API unreachable during local UI work — allow preview.
        return { user: previewUser, preview: true }
      }
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
  const { user, preview } = Route.useRouteContext()
  const navigate = useNavigate()

  return (
    <DashboardProvider
      onSignOut={() => void navigate({ to: "/login" })}
      preview={preview}
      user={user}
    >
      <DashboardShell />
    </DashboardProvider>
  )
}

function DashboardShell() {
  const {
    preview,
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
            <div className="flex items-center gap-2">
              {preview ? <Badge variant="outline">Preview data</Badge> : null}
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
        preview={preview}
      />

      {selectedRequest ? (
        <RequestInspector
          online={selectedDomain?.status === "online"}
          onClose={() => setSelectedRequest(undefined)}
          preview={preview}
          request={selectedRequest}
        />
      ) : null}
    </SidebarProvider>
  )
}
