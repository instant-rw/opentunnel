import { createFileRoute } from "@tanstack/react-router"

import { AuthScreen } from "@/components/auth-screen"

type LoginSearch = {
  user_code?: string
}

export const Route = createFileRoute("/login")({
  validateSearch: (search: Record<string, unknown>): LoginSearch => ({
    user_code:
      typeof search.user_code === "string" ? search.user_code : undefined,
  }),
  component: LoginPage,
})

function LoginPage() {
  const { user_code: userCode } = Route.useSearch()
  return <AuthScreen userCode={userCode} />
}
