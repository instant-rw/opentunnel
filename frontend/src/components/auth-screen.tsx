import {
  ArrowRightIcon,
  CheckIcon,
  EyeIcon,
  EyeSlashIcon,
  BroadcastIcon,
  ShieldCheckIcon,
} from "@phosphor-icons/react"
import { useNavigate } from "@tanstack/react-router"
import { useState, type FormEvent } from "react"

import { DeviceApproved } from "@/components/device-approved"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { api, ApiError, type User } from "@/lib/api"
import { clearUserCodeFromUrl } from "@/lib/device-auth"

type AuthMode = "login" | "register"

export function AuthScreen({
  userCode,
}: {
  userCode?: string
}) {
  const navigate = useNavigate()
  const [mode, setMode] = useState<AuthMode>("login")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [showPassword, setShowPassword] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [approvedUser, setApprovedUser] = useState<User>()

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError("")
    setLoading(true)

    try {
      if (mode === "register") {
        await api.register(email, password)
        await api.login(email, password)
      } else {
        await api.login(email, password)
      }
      const user = await api.me()
      if (userCode) {
        await api.approveDevice(userCode)
        clearUserCodeFromUrl()
        setApprovedUser(user)
      } else {
        void navigate({ to: "/dashboard" })
      }
    } catch (caught) {
      setError(
        caught instanceof ApiError
          ? caught.message
          : "Something went wrong. Please try again.",
      )
    } finally {
      setLoading(false)
    }
  }

  if (approvedUser) {
    return (
      <DeviceApproved
        onContinue={() => void navigate({ to: "/dashboard" })}
      />
    )
  }

  return (
    <main className="grid min-h-svh lg:grid-cols-2">
      <section className="relative hidden flex-col justify-between bg-[linear-gradient(160deg,_oklch(0.28_0.04_250),_oklch(0.18_0.03_230))] p-10 text-white lg:flex">
        <a className="flex items-center gap-2 text-sm font-medium" href="/">
          <span className="flex size-8 items-center justify-center bg-white/10">
            <BroadcastIcon className="size-4" />
          </span>
          OpenTunnel
        </a>
        <div className="max-w-md space-y-4">
          <p className="text-xs tracking-[0.2em] text-white/60 uppercase">
            Local, but live
          </p>
          <h1 className="font-heading text-4xl leading-tight font-medium">
            Bring localhost into the open.
          </h1>
          <p className="text-sm leading-relaxed text-white/75">
            Secure public URLs, instant request inspection, and painless webhook
            debugging—without touching your router.
          </p>
          <ul className="space-y-2 text-sm text-white/80">
            <li className="flex items-center gap-2">
              <CheckIcon className="size-4" /> Persistent, memorable domains
            </li>
            <li className="flex items-center gap-2">
              <CheckIcon className="size-4" /> End-to-end request visibility
            </li>
            <li className="flex items-center gap-2">
              <CheckIcon className="size-4" /> One-command tunnel setup
            </li>
          </ul>
        </div>
        <p className="text-sm text-white/55 italic">
          "The missing devtool between a webhook and localhost."
        </p>
      </section>

      <section className="flex flex-col justify-between bg-[radial-gradient(circle_at_top,_oklch(0.98_0.01_240),_oklch(0.96_0.01_220))] p-6 sm:p-10">
        <div className="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center gap-6">
          {userCode ? (
            <div className="flex items-start gap-3 border border-emerald-500/30 bg-emerald-500/10 p-3">
              <ShieldCheckIcon className="mt-0.5 size-5 text-emerald-700" />
              <div className="text-sm">
                <strong className="block">Approve CLI sign-in</strong>
                <span className="text-muted-foreground">
                  Code{" "}
                  <kbd className="rounded bg-background px-1.5 py-0.5 font-mono text-xs">
                    {userCode.toUpperCase()}
                  </kbd>
                </span>
              </div>
            </div>
          ) : null}

          <div className="space-y-2">
            <span className="flex items-center gap-2 text-sm font-medium lg:hidden">
              <BroadcastIcon className="size-4" /> OpenTunnel
            </span>
            <h2 className="font-heading text-2xl font-medium">
              {mode === "login" ? "Welcome back" : "Create your account"}
            </h2>
            <p className="text-sm text-muted-foreground">
              {mode === "login"
                ? "Sign in to manage your tunnels."
                : "Start exposing localhost in under a minute."}
            </p>
          </div>

          <form className="grid gap-4" onSubmit={submit}>
            <div className="grid gap-2">
              <Label htmlFor="email">Email</Label>
              <Input
                autoComplete="email"
                id="email"
                onChange={(event) => setEmail(event.target.value)}
                placeholder="you@example.com"
                required
                type="email"
                value={email}
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="password">Password</Label>
              <div className="relative">
                <Input
                  autoComplete={
                    mode === "login" ? "current-password" : "new-password"
                  }
                  id="password"
                  minLength={8}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="At least 8 characters"
                  required
                  type={showPassword ? "text" : "password"}
                  value={password}
                />
                <button
                  aria-label={showPassword ? "Hide password" : "Show password"}
                  className="absolute top-1/2 right-2 -translate-y-1/2 text-muted-foreground"
                  onClick={() => setShowPassword((visible) => !visible)}
                  type="button"
                >
                  {showPassword ? (
                    <EyeSlashIcon className="size-4" />
                  ) : (
                    <EyeIcon className="size-4" />
                  )}
                </button>
              </div>
            </div>
            {error ? <p className="text-xs text-destructive">{error}</p> : null}
            <Button className="w-full" disabled={loading} type="submit">
              {loading ? <Spinner /> : null}
              {userCode
                ? "Sign in & approve"
                : mode === "login"
                  ? "Sign in"
                  : "Create account"}
              <ArrowRightIcon />
            </Button>
          </form>

          <p className="text-sm text-muted-foreground">
            {mode === "login"
              ? "New to OpenTunnel?"
              : "Already have an account?"}{" "}
            <button
              className="font-medium text-foreground underline-offset-4 hover:underline"
              onClick={() =>
                setMode((current) =>
                  current === "login" ? "register" : "login",
                )
              }
              type="button"
            >
              {mode === "login" ? "Create an account" : "Sign in"}
            </button>
          </p>
        </div>
        <p className="mt-8 text-center text-xs text-muted-foreground">
          By continuing, you agree to the Terms and Privacy Policy.
        </p>
      </section>
    </main>
  )
}
