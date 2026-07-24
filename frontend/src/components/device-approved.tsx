import { CheckIcon } from "@phosphor-icons/react"
import { Link } from "@tanstack/react-router"

import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

export function DeviceApproved({ onContinue }: { onContinue?: () => void }) {
  return (
    <main className="flex min-h-svh items-center justify-center bg-background p-6">
      <Card className="w-full max-w-md text-center">
        <CardHeader className="items-center">
          <span className="mb-2 flex size-14 items-center justify-center rounded-full bg-emerald-500/15 text-emerald-700">
            <CheckIcon className="size-7" weight="bold" />
          </span>
          <p className="text-xs tracking-wide text-muted-foreground uppercase">
            Device approved
          </p>
          <CardTitle className="text-2xl">You're all set</CardTitle>
          <CardDescription>
            Return to your terminal. OpenTunnel will finish signing in
            automatically.
          </CardDescription>
        </CardHeader>
        <CardContent />
        <CardFooter className="justify-center">
          {onContinue ? (
            <Button onClick={onContinue} variant="secondary">
              Open dashboard
            </Button>
          ) : (
            <Button render={<Link to="/dashboard" />} variant="secondary">
              Open dashboard
            </Button>
          )}
        </CardFooter>
      </Card>
    </main>
  )
}
