import {
  GlobeIcon,
  KeyIcon,
  LaptopIcon,
  ShieldCheckIcon,
  TrashIcon,
} from "@phosphor-icons/react"
import { createFileRoute } from "@tanstack/react-router"
import { useState } from "react"

import { PageHeader } from "@/components/page-header"
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
import { Spinner } from "@/components/ui/spinner"
import { useDashboard } from "@/lib/dashboard-context"
import { formatRelativeTime } from "@/lib/utils"

export const Route = createFileRoute("/dashboard/access")({
  component: AccessPage,
})

function AccessPage() {
  const { tokens, revokeToken } = useDashboard()
  const [revoking, setRevoking] = useState("")

  async function revoke(id: string) {
    setRevoking(id)
    try {
      await revokeToken(id)
    } finally {
      setRevoking("")
    }
  }

  return (
    <>
      <PageHeader
        description="Review machines with access to your workspace and revoke credentials."
        eyebrow="Security"
        title="Sessions & tokens"
      />

      <Card className="mb-4">
        <CardContent className="flex items-start gap-3 pt-(--card-spacing)">
          <ShieldCheckIcon className="mt-0.5 size-5 shrink-0" />
          <div>
            <strong className="text-sm">
              Your CLI tokens are stored securely
            </strong>
            <p className="text-xs text-muted-foreground">
              Token values are never shown here. Revocation takes effect
              immediately and disconnects active tunnels.
            </p>
          </div>
        </CardContent>
      </Card>

      <Card className="mb-4">
        <CardContent className="flex items-center gap-3 pt-(--card-spacing)">
          <span className="flex size-9 items-center justify-center bg-muted">
            <GlobeIcon className="size-4" />
          </span>
          <div className="flex-1">
            <strong className="text-sm">Current web session</strong>
            <p className="text-xs text-muted-foreground">
              This browser · Active now · Protected with a secure, HttpOnly
              session cookie
            </p>
          </div>
          <Badge>Current</Badge>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Authorized devices</CardTitle>
          <CardDescription>
            {tokens.length} active CLI credentials
          </CardDescription>
        </CardHeader>
        <CardContent>
          {tokens.length ? (
            <div className="space-y-3">
              {tokens.map((token) => (
                <div
                  className="flex items-center gap-3 border p-3"
                  key={token.id}
                >
                  <span className="flex size-9 items-center justify-center bg-muted">
                    <LaptopIcon className="size-4" />
                  </span>
                  <div className="min-w-0 flex-1">
                    <strong className="text-sm">{token.name}</strong>
                    <p className="text-xs text-muted-foreground">
                      Last used{" "}
                      {token.lastUsedAt
                        ? formatRelativeTime(token.lastUsedAt)
                        : "never"}{" "}
                      · Created {formatRelativeTime(token.createdAt)}
                    </p>
                  </div>
                  <Button
                    aria-label={`Revoke ${token.name}`}
                    disabled={revoking === token.id}
                    onClick={() => void revoke(token.id)}
                    size="sm"
                    variant="destructive"
                  >
                    {revoking === token.id ? <Spinner /> : <TrashIcon />}
                    Revoke
                  </Button>
                </div>
              ))}
            </div>
          ) : (
            <Empty className="border-0 py-8">
              <EmptyHeader>
                <EmptyMedia variant="icon">
                  <KeyIcon />
                </EmptyMedia>
                <EmptyTitle>No authorized devices</EmptyTitle>
                <EmptyDescription>
                  Run opentunnel login to authorize a machine.
                </EmptyDescription>
              </EmptyHeader>
            </Empty>
          )}
        </CardContent>
      </Card>
    </>
  )
}
