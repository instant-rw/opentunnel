import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useEffect } from "react"

import { Spinner } from "@/components/ui/spinner"

export const Route = createFileRoute("/")({
  component: HomeRedirect,
})

function HomeRedirect() {
  const navigate = useNavigate()

  useEffect(() => {
    void navigate({ to: "/dashboard" })
  }, [navigate])

  return (
    <main className="flex min-h-svh items-center justify-center">
      <Spinner className="size-6" />
    </main>
  )
}
