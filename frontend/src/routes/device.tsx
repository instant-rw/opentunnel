import { BroadcastIcon } from "@phosphor-icons/react"
import {
  createFileRoute,
  redirect,
  useNavigate,
} from "@tanstack/react-router"
import { useEffect, useState } from "react"

import { DeviceApproved } from "@/components/device-approved"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Spinner } from "@/components/ui/spinner"
import { api, ApiError } from "@/lib/api"
import { clearUserCodeFromUrl } from "@/lib/device-auth"

type DeviceSearch = {
  user_code?: string
}

type DeviceState =
  | { status: "loading" }
  | { status: "approving" }
  | { status: "approved" }
  | { status: "failed"; message: string }

export const Route = createFileRoute("/device")({
  validateSearch: (search: Record<string, unknown>): DeviceSearch => ({
    user_code:
      typeof search.user_code === "string" ? search.user_code : undefined,
  }),
  beforeLoad: async ({ search }) => {
    if (!search.user_code) {
      throw redirect({ to: "/dashboard" })
    }
    try {
      await api.me()
    } catch {
      throw redirect({
        to: "/login",
        search: { user_code: search.user_code },
      })
    }
  },
  component: DevicePage,
})

function DevicePage() {
  const { user_code: userCode } = Route.useSearch()
  const navigate = useNavigate()
  const [state, setState] = useState<DeviceState>({ status: "loading" })

  useEffect(() => {
    if (!userCode) return
    let cancelled = false

    ;(async () => {
      setState({ status: "approving" })
      try {
        await api.approveDevice(userCode)
        if (cancelled) return
        clearUserCodeFromUrl()
        setState({ status: "approved" })
      } catch (caught) {
        if (cancelled) return
        setState({
          status: "failed",
          message:
            caught instanceof ApiError
              ? caught.message
              : "Could not approve this device.",
        })
      }
    })()

    return () => {
      cancelled = true
    }
  }, [userCode])

  if (state.status === "loading" || state.status === "approving") {
    return (
      <main className="flex min-h-svh flex-col items-center justify-center gap-3 bg-[radial-gradient(circle_at_top,_oklch(0.97_0.02_250),_oklch(0.94_0.01_220))] p-6">
        <span className="flex size-10 items-center justify-center bg-foreground text-background">
          <BroadcastIcon className="size-5" />
        </span>
        <strong>OpenTunnel</strong>
        <Spinner className="size-5" />
        <p className="text-xs tracking-wide text-muted-foreground uppercase">
          Approving device…
        </p>
      </main>
    )
  }

  if (state.status === "approved") {
    return (
      <DeviceApproved
        onContinue={() => void navigate({ to: "/dashboard" })}
      />
    )
  }

  return (
    <main className="flex min-h-svh items-center justify-center bg-[radial-gradient(circle_at_top,_oklch(0.97_0.02_250),_oklch(0.94_0.01_220))] p-6">
      <Card className="w-full max-w-md text-center">
        <CardHeader>
          <p className="text-xs tracking-wide text-muted-foreground uppercase">
            Approval failed
          </p>
          <CardTitle className="text-2xl">
            Couldn't approve this device
          </CardTitle>
          <CardDescription>{state.message}</CardDescription>
        </CardHeader>
        <CardContent />
        <CardFooter className="justify-center">
          <Button
            onClick={() => void navigate({ to: "/dashboard" })}
            variant="secondary"
          >
            Open dashboard
          </Button>
        </CardFooter>
      </Card>
    </main>
  )
}
