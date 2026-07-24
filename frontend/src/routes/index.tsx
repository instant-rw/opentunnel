import { BroadcastIcon } from "@phosphor-icons/react"
import { Link, createFileRoute } from "@tanstack/react-router"

export const Route = createFileRoute("/")({
  component: LandingPage,
})

function LandingPage() {
  return (
    <main className="relative min-h-svh overflow-hidden bg-[linear-gradient(160deg,oklch(0.97_0.01_240),oklch(0.94_0.02_220)_45%,oklch(0.98_0.005_90))]">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-[radial-gradient(ellipse_at_20%_10%,oklch(0.85_0.04_230/0.55),transparent_50%),radial-gradient(ellipse_at_90%_80%,oklch(0.9_0.03_200/0.4),transparent_45%)]"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-x-0 top-0 h-px bg-linear-to-r from-transparent via-foreground/20 to-transparent"
      />

      <div className="relative mx-auto flex min-h-svh w-full max-w-5xl flex-col px-6 py-8 sm:px-10">
        <header className="animate-in fade-in slide-in-from-top-2 flex items-center justify-between duration-700">
          <a className="flex items-center gap-2 text-sm font-medium" href="/">
            <span className="flex size-8 items-center justify-center bg-foreground text-background">
              <BroadcastIcon className="size-4" />
            </span>
            OpenTunnel
          </a>
          <nav className="flex items-center gap-3 text-sm">
            <Link
              className="text-muted-foreground transition-colors hover:text-foreground"
              to="/login"
            >
              Sign in
            </Link>
            <Link
              className="bg-foreground px-3 py-1.5 text-background transition-opacity hover:opacity-90"
              to="/login"
            >
              Get started
            </Link>
          </nav>
        </header>

        <section className="flex flex-1 flex-col justify-center gap-10 py-16 lg:flex-row lg:items-end lg:justify-between lg:gap-16">
          <div className="animate-in fade-in slide-in-from-bottom-3 max-w-xl space-y-6 duration-700">
            <p className="font-mono text-xs tracking-[0.22em] text-muted-foreground uppercase">
              OpenTunnel
            </p>
            <h1 className="font-heading text-4xl leading-[1.1] font-medium tracking-tight sm:text-5xl">
              Localhost, publicly reachable.
            </h1>
            <p className="max-w-md text-base leading-relaxed text-muted-foreground">
              Persistent domains for local apps, webhooks, and demos—without
              opening ports or fighting your router.
            </p>
            <div className="flex flex-wrap items-center gap-3 pt-1">
              <Link
                className="bg-foreground px-4 py-2.5 text-sm text-background transition-opacity hover:opacity-90"
                to="/login"
              >
                Open dashboard
              </Link>
              <a
                className="border border-foreground/15 bg-background/60 px-4 py-2.5 text-sm backdrop-blur transition-colors hover:border-foreground/30"
                href="https://github.com/instant-rw/opentunnel"
                rel="noreferrer"
                target="_blank"
              >
                View on GitHub
              </a>
            </div>
          </div>

          <div className="animate-in fade-in slide-in-from-bottom-4 w-full max-w-md delay-150 duration-700">
            <pre className="overflow-hidden border border-foreground/10 bg-[oklch(0.22_0.03_240)] p-5 font-mono text-[13px] leading-relaxed text-white/90 shadow-[0_24px_80px_-32px_oklch(0.3_0.05_240)]">
              <code>
                <span className="text-white/45">$</span> opentunnel connect
                {"\n"}
                <span className="text-emerald-300/90">✓</span> online{"  "}
                <span className="text-sky-200/90">demo.opts.ink</span>
                {"\n"}
                <span className="text-white/45">→</span> localhost:3000
              </code>
            </pre>
          </div>
        </section>
      </div>
    </main>
  )
}
