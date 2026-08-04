import {
  ArrowClockwiseIcon,
  ArrowRightIcon,
  ArrowUpRightIcon,
  GithubLogoIcon,
  GlobeIcon,
  LightningIcon,
  LockKeyIcon,
  MagnifyingGlassIcon,
  PlugsConnectedIcon,
  TerminalWindowIcon,
  WebhooksLogoIcon,
} from "@phosphor-icons/react"
import type { Icon } from "@phosphor-icons/react"
import { Link, createFileRoute } from "@tanstack/react-router"

import { CopyButton } from "@/components/copy-button"
import { Logo } from "@/components/logo"
import { SectionFrame, SectionSideBorders } from "@/components/section-frame"
import { ThemeToggle } from "@/components/theme-toggle"
import { Button } from "@/components/ui/button"
import {
  CONTRIBUTE_URL,
  DOCS_URL,
  GITHUB_URL,
  ISSUES_URL,
} from "@/lib/nav-links"
import {
  SITE_NAME,
  seoHead,
  softwareApplicationJsonLd,
  websiteJsonLd,
} from "@/lib/seo"

export const Route = createFileRoute("/")({
  head: () => {
    const head = seoHead({
      title: SITE_NAME,
      path: "/",
    })
    return {
      ...head,
      scripts: [websiteJsonLd(), softwareApplicationJsonLd()],
    }
  },
  component: LandingPage,
})

type Feature = {
  title: string
  description: string
  icon: Icon
}

const features: Feature[] = [
  {
    title: "Persistent domains",
    description:
      "Reserve memorable URLs like my-app.opts.ink that stay yours across restarts and redeploys.",
    icon: GlobeIcon,
  },
  {
    title: "Request inspection",
    description:
      "See every request and response in real time—headers, body, status, and latency—straight from the dashboard.",
    icon: MagnifyingGlassIcon,
  },
  {
    title: "Webhook debugging",
    description:
      "Point Stripe, GitHub, or any provider at your tunnel and replay deliveries until the integration is right.",
    icon: WebhooksLogoIcon,
  },
  {
    title: "One-command setup",
    description:
      "A single static binary. No accounts to wire up, no router config, no ngrok-style session limits.",
    icon: TerminalWindowIcon,
  },
  {
    title: "Secure by default",
    description:
      "Automatic TLS on every domain, with token-based access controls so only the right people connect.",
    icon: LockKeyIcon,
  },
  {
    title: "Instant replay",
    description:
      "Resend any captured request with one click to reproduce bugs without triggering the source again.",
    icon: ArrowClockwiseIcon,
  },
]

const INSTALL_COMMAND = "curl -fsSL https://opts.ink/install.sh | sh"

const steps = [
  {
    title: "Install OpenTunnel",
    description: "Run the installer on macOS, Windows or Linux.",
    command: INSTALL_COMMAND,
  },
  {
    title: "Authenticate the CLI",
    description: "Sign in once and your machine is linked to your workspace.",
    command: "opentunnel login",
  },
  {
    title: "Go live",
    description: "Point a persistent public domain at your local server.",
    command: "opentunnel up 3000 --domain <your-domain>",
  },
]

const stats = [
  { label: "Single binary", value: "1" },
  { label: "Config files", value: "0" },
  { label: "TLS domains", value: "∞" },
  { label: "Open source", value: "MIT" },
]

type SampleRequest = {
  method: string
  path: string
  status: number
  latency: string
  tone: string
}

const sampleRequests: SampleRequest[] = [
  {
    method: "POST",
    path: "/webhooks/stripe",
    status: 200,
    latency: "42ms",
    tone: "text-sky-500",
  },
  {
    method: "GET",
    path: "/api/session",
    status: 200,
    latency: "11ms",
    tone: "text-emerald-500",
  },
  {
    method: "POST",
    path: "/graphql",
    status: 400,
    latency: "63ms",
    tone: "text-sky-500",
  },
  {
    method: "DELETE",
    path: "/api/cache",
    status: 500,
    latency: "128ms",
    tone: "text-red-500",
  },
]

const footerColumns: {
  title: string
  links: { label: string; href: string; to?: string }[]
}[] = [
  {
    title: "Product",
    links: [
      { label: "Dashboard", href: "", to: "/login" },
      { label: "Features", href: "#features" },
      { label: "How it works", href: "#how-it-works" },
    ],
  },
  {
    title: "Resources",
    links: [
      { label: "Documentation", href: DOCS_URL },
      { label: "CLI reference", href: DOCS_URL },
      { label: "Troubleshooting", href: DOCS_URL },
    ],
  },
  {
    title: "Community",
    links: [
      { label: "GitHub", href: GITHUB_URL },
      { label: "Contribute", href: CONTRIBUTE_URL },
      { label: "Report an issue", href: ISSUES_URL },
    ],
  },
]

function statusColor(status: number): string {
  if (status >= 500) return "text-red-500"
  if (status >= 400) return "text-amber-500"
  return "text-emerald-500"
}

function LandingPage() {
  return (
    <div className="relative min-h-svh bg-background text-foreground">
      <GridBackground />

      <div className="relative">
        <SiteHeader />

        <main className="relative">
          <SectionSideBorders />
          <Hero />
          <StatsBar />
          <Features />
          <HowItWorks />
          <Inspection />
          <CtaBand />
        </main>

        <SiteFooter />
      </div>
    </div>
  )
}

function GridBackground() {
  return (
    <div
      aria-hidden
      className="pointer-events-none fixed inset-0 z-0 bg-[linear-gradient(to_right,var(--border)_1px,transparent_1px),linear-gradient(to_bottom,var(--border)_1px,transparent_1px)] mask-[radial-gradient(ellipse_at_50%_0%,black,transparent_70%)] bg-size-[56px_56px] opacity-60"
    />
  )
}

function SiteHeader() {
  return (
    <header className="sticky top-0 z-30 border-b border-border bg-background/80 backdrop-blur">
      <div className="mx-auto flex h-16 w-full max-w-6xl items-center justify-between gap-4 px-4 sm:px-6">
        <a className="flex items-center gap-2 font-medium" href="/">
          <Logo className="size-7" />
          <span>OpenTunnel</span>
        </a>

        <nav className="hidden items-center gap-6 text-sm text-muted-foreground md:flex">
          <a
            className="transition-colors hover:text-foreground"
            href="#features"
          >
            Features
          </a>
          <a
            className="transition-colors hover:text-foreground"
            href="#how-it-works"
          >
            How it works
          </a>
          <a
            className="transition-colors hover:text-foreground"
            href={DOCS_URL}
            rel="noreferrer"
            target="_blank"
          >
            Docs
          </a>
        </nav>

        <div className="flex items-center gap-2">
          <Button
            aria-label="OpenTunnel on GitHub"
            render={<a href={GITHUB_URL} rel="noreferrer" target="_blank" />}
            size="icon-sm"
            variant="ghost"
          >
            <GithubLogoIcon className="size-4" />
          </Button>
          <ThemeToggle />
          <Button render={<Link to="/login" />} size="sm">
            Get started
            <ArrowRightIcon />
          </Button>
        </div>
      </div>
    </header>
  )
}

function Hero() {
  return (
    <section className="relative">
      <div className="relative mx-auto w-full max-w-6xl">
        <SectionFrame>
          <div className="grid gap-12 px-4 py-16 sm:px-6 lg:grid-cols-[1.05fr_0.95fr] lg:items-center lg:py-24">
            <div className="animate-in space-y-7 duration-700 fade-in slide-in-from-bottom-3">
              <h1 className="font-heading text-4xl leading-[1.08] font-medium sm:text-5xl">
                Localhost,
                <br />
                publicly reachable.
              </h1>

              <p className="max-w-lg text-base leading-relaxed text-muted-foreground">
                OpenTunnel gives your local apps persistent public domains—perfect
                for webhooks, live demos, and mobile testing. No ports to open, no
                router to fight, no traffic left unseen.
              </p>

              <div className="flex flex-wrap items-center gap-3">
                <Button render={<Link to="/login" />} size="lg">
                  Open dashboard
                  <ArrowRightIcon />
                </Button>
                <Button
                  render={
                    <a href={GITHUB_URL} rel="noreferrer" target="_blank" />
                  }
                  size="lg"
                  variant="outline"
                >
                  <GithubLogoIcon />
                  View on GitHub
                </Button>
              </div>

              <div className="flex max-w-md items-center justify-between gap-2 border border-border bg-muted/30 px-3 py-2 text-xs">
                <span className="truncate">
                  <span className="text-muted-foreground">$ </span>
                  {INSTALL_COMMAND}
                </span>
                <CopyButton label="" value={INSTALL_COMMAND} />
              </div>
            </div>

            <div className="animate-in delay-150 duration-700 fade-in slide-in-from-bottom-4">
              <TerminalCard />
            </div>
          </div>
        </SectionFrame>
      </div>
    </section>
  )
}

function TerminalCard() {
  return (
    <div className="border border-border bg-zinc-950 shadow-[0_32px_80px_-40px_rgba(0,0,0,0.6)]">
      <div className="flex items-center gap-2 border-b border-white/10 px-4 py-3">
        <span className="size-2.5 rounded-full bg-[#ff5f56]" />
        <span className="size-2.5 rounded-full bg-[#ffbd2e]" />
        <span className="size-2.5 rounded-full bg-[#27c93f]" />
        <span className="ml-2 text-xs text-zinc-500">Terminal</span>
      </div>
      <div className="space-y-1 p-5 text-[13px] leading-relaxed">
        <p className="text-zinc-400">
          $ opentunnel up 3000 --domain &lt;your-domain&gt;
        </p>
        <p className="pt-2 text-emerald-400">
          ✓ Authenticated as johndoe@example.com
        </p>
        <p className="text-emerald-400">✓ Connected to OpenTunnel edge</p>
        <p className="pt-2 text-zinc-500">Public URL</p>
        <p className="text-sky-300">https://&lt;your-domain&gt;.opts.ink</p>
        <p className="pt-2 text-zinc-500">Forwarding</p>
        <p className="text-zinc-300">http://127.0.0.1:3000</p>
        <p className="pt-2 text-zinc-500">
          Watching for requests… press Ctrl+C to stop
        </p>
      </div>
    </div>
  )
}

function StatsBar() {
  return (
    <section className="bg-muted/20">
      <div className="relative mx-auto w-full max-w-6xl">
        <SectionFrame>
          <div className="grid grid-cols-2 divide-x divide-border px-4 sm:px-6 lg:grid-cols-4">
            {stats.map((stat) => (
              <div
                className="flex flex-col items-center gap-1 px-4 py-8 text-center"
                key={stat.label}
              >
                <span className="font-heading text-3xl font-medium">
                  {stat.value}
                </span>
                <span className="text-xs tracking-wide text-muted-foreground uppercase">
                  {stat.label}
                </span>
              </div>
            ))}
          </div>
        </SectionFrame>
      </div>
    </section>
  )
}

function SectionHeading({
  eyebrow,
  title,
  description,
}: {
  eyebrow: string
  title: string
  description: string
}) {
  return (
    <div className="mx-auto max-w-2xl text-center">
      <p className="text-xs tracking-[0.22em] text-muted-foreground uppercase">
        {eyebrow}
      </p>
      <h2 className="mt-3 font-heading text-3xl font-medium  sm:text-4xl">
        {title}
      </h2>
      <p className="mt-4 text-base leading-relaxed text-muted-foreground">
        {description}
      </p>
    </div>
  )
}

function Features() {
  return (
    <section id="features">
      <div className="relative mx-auto w-full max-w-6xl">
        <SectionFrame>
          <div className="px-4 py-20 sm:px-6">
            <SectionHeading
              description="A focused toolkit for getting local code onto the public internet—and understanding exactly what happens next."
              eyebrow="Features"
              title="Everything you need to ship from localhost"
            />

            <div className="mt-14 grid gap-px border border-border bg-border sm:grid-cols-2 lg:grid-cols-3">
              {features.map((feature) => (
                <div
                  className="group bg-background p-7 transition-colors hover:bg-muted/40"
                  key={feature.title}
                >
                  <div className="flex size-10 items-center justify-center border border-border bg-muted text-foreground transition-colors group-hover:border-foreground/30">
                    <feature.icon className="size-5" />
                  </div>
                  <h3 className="mt-5 font-medium">{feature.title}</h3>
                  <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                    {feature.description}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </SectionFrame>
      </div>
    </section>
  )
}

function HowItWorks() {
  return (
    <section className="bg-muted/20" id="how-it-works">
      <div className="relative mx-auto w-full max-w-6xl">
        <SectionFrame>
          <div className="px-4 py-20 sm:px-6">
            <SectionHeading
              description="Three commands from a fresh terminal to a live, inspectable public URL."
              eyebrow="How it works"
              title="Live in under a minute"
            />

            <div className="mt-14 grid gap-4 lg:grid-cols-3">
              {steps.map((step, index) => (
                <div
                  className="flex flex-col border border-border bg-background p-6"
                  key={step.title}
                >
                  <div className="flex items-center gap-3">
                    <span className="flex size-8 items-center justify-center border border-border bg-muted text-sm font-medium">
                      {index + 1}
                    </span>
                    <h3 className="font-medium">{step.title}</h3>
                  </div>
                  <p className="mt-3 text-sm leading-relaxed text-muted-foreground">
                    {step.description}
                  </p>
                  <div className="mt-4 overflow-x-auto border border-border bg-zinc-950 px-3 py-2 text-xs text-zinc-100">
                    <span className="text-zinc-500">$ </span>
                    {step.command}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </SectionFrame>
      </div>
    </section>
  )
}

function Inspection() {
  return (
    <section>
      <div className="relative mx-auto w-full max-w-6xl">
        <SectionFrame>
          <div className="grid items-center gap-12 px-4 py-20 sm:px-6 lg:grid-cols-2">
            <div className="space-y-6">
              <p className="text-xs tracking-[0.22em] text-muted-foreground uppercase">
                Traffic inspector
              </p>
              <h2 className="font-heading text-3xl font-medium  sm:text-4xl">
                Inspect every request as it happens
              </h2>
              <p className="text-base leading-relaxed text-muted-foreground">
                Every request through your tunnel is captured with full headers,
                payloads, status codes, and timing. Filter by method or status,
                dig into a single call, and replay it instantly to reproduce the
                exact conditions that broke your code.
              </p>
              <ul className="space-y-3 text-sm">
                {[
                  {
                    icon: LightningIcon,
                    text: "Real-time streaming of requests and responses",
                  },
                  {
                    icon: ArrowClockwiseIcon,
                    text: "One-click replay without hitting the source again",
                  },
                  {
                    icon: PlugsConnectedIcon,
                    text: "Works with any framework, port, or webhook provider",
                  },
                ].map((item) => (
                  <li className="flex items-center gap-3" key={item.text}>
                    <span className="flex size-6 items-center justify-center border border-border bg-muted">
                      <item.icon className="size-3.5" />
                    </span>
                    {item.text}
                  </li>
                ))}
              </ul>
              <Button render={<Link to="/login" />} variant="outline">
                Try the dashboard
                <ArrowUpRightIcon />
              </Button>
            </div>

            <div className="border border-border bg-card">
              <div className="flex items-center justify-between border-b border-border px-4 py-3">
                <span className="text-xs text-muted-foreground">
                  Live requests
                </span>
                <span className="flex items-center gap-1.5 text-xs text-emerald-500">
                  <span className="size-1.5 animate-pulse bg-emerald-500 rounded-full" />
                  streaming
                </span>
              </div>
              <div className="divide-y divide-border">
                {sampleRequests.map((request, index) => (
                  <div
                    className="grid grid-cols-[auto_1fr_auto_auto] items-center gap-3 px-4 py-3 text-xs"
                    key={`${request.path}-${index}`}
                  >
                    <span className={`w-14 font-medium ${request.tone}`}>
                      {request.method}
                    </span>
                    <span className="truncate text-foreground">
                      {request.path}
                    </span>
                    <span className={statusColor(request.status)}>
                      {request.status}
                    </span>
                    <span className="w-14 text-right text-muted-foreground">
                      {request.latency}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </SectionFrame>
      </div>
    </section>
  )
}

function CtaBand() {
  return (
    <section>
      <div className="relative mx-auto w-full max-w-6xl">
        <SectionFrame>
          <div className="px-4 py-20 sm:px-6">
            <div className="relative overflow-hidden border border-border bg-zinc-950 px-6 py-16 text-center">
              <div
                aria-hidden
                className="pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,rgba(255,255,255,0.06)_1px,transparent_1px),linear-gradient(to_bottom,rgba(255,255,255,0.06)_1px,transparent_1px)] mask-[radial-gradient(ellipse_at_center,black,transparent_75%)] bg-size-[40px_40px]"
              />
              <div className="relative mx-auto max-w-2xl space-y-6">
                <h2 className="font-heading text-3xl font-medium text-white sm:text-4xl">
                  Bring localhost into the open
                </h2>
                <p className="text-base leading-relaxed text-zinc-400">
                  Spin up a persistent, inspectable public URL for your local
                  app in seconds. Free, open source, and yours to self-host.
                </p>
                <div className="flex flex-wrap items-center justify-center gap-3">
                  <Button
                    render={<Link to="/login" />}
                    size="lg"
                    variant="secondary"
                  >
                    Get started
                    <ArrowRightIcon />
                  </Button>
                  <Button
                    className="border-white/20 bg-transparent text-white hover:bg-white/10 hover:text-white"
                    render={
                      <a href={GITHUB_URL} rel="noreferrer" target="_blank" />
                    }
                    size="lg"
                    variant="outline"
                  >
                    <GithubLogoIcon />
                    Star on GitHub
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </SectionFrame>
      </div>
    </section>
  )
}

function SiteFooter() {
  return (
    <footer className="border-t border-border bg-background">
      <div className="mx-auto w-full max-w-6xl px-4 py-14 sm:px-6">
        <div className="grid gap-10 md:grid-cols-[1.5fr_1fr_1fr_1fr]">
          <div className="space-y-4">
            <a className="flex items-center gap-2 font-medium" href="/">
              <Logo className="size-7" />
              <span>OpenTunnel</span>
            </a>
            <p className="max-w-xs text-sm leading-relaxed text-muted-foreground">
              Persistent public domains for local apps, webhooks, and demos.
              Open source and self-hostable.
            </p>
          </div>

          {footerColumns.map((column) => (
            <div key={column.title}>
              <h3 className="text-xs tracking-[0.2em] text-muted-foreground uppercase">
                {column.title}
              </h3>
              <ul className="mt-4 space-y-3 text-sm">
                {column.links.map((link) => (
                  <li key={link.label}>
                    {link.to ? (
                      <Link
                        className="text-muted-foreground transition-colors hover:text-foreground"
                        to={link.to}
                      >
                        {link.label}
                      </Link>
                    ) : link.href.startsWith("#") ? (
                      <a
                        className="text-muted-foreground transition-colors hover:text-foreground"
                        href={link.href}
                      >
                        {link.label}
                      </a>
                    ) : (
                      <a
                        className="text-muted-foreground transition-colors hover:text-foreground"
                        href={link.href}
                        rel="noreferrer"
                        target="_blank"
                      >
                        {link.label}
                      </a>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-12 flex flex-col items-center justify-between gap-4 border-t border-border pt-6 sm:flex-row">
          <p className="text-xs text-muted-foreground">
            © {new Date().getFullYear()} OpenTunnel · MIT Licensed
          </p>
          <div className="flex items-center gap-1.5">
            <Button
              aria-label="OpenTunnel on GitHub"
              render={<a href={GITHUB_URL} rel="noreferrer" target="_blank" />}
              size="icon-sm"
              variant="ghost"
            >
              <GithubLogoIcon className="size-4" />
            </Button>
            <ThemeToggle />
          </div>
        </div>
      </div>
    </footer>
  )
}
