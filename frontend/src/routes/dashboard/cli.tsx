import { createFileRoute } from "@tanstack/react-router"
import { useState } from "react"

import { CopyButton } from "@/components/copy-button"
import { PageHeader } from "@/components/page-header"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { cn } from "@/lib/utils"

export const Route = createFileRoute("/dashboard/cli")({
  component: CliSetupPage,
})

function CliSetupPage() {
  const [platform, setPlatform] = useState<"macos" | "linux" | "windows">(
    "macos"
  )
  const installCommand =
    platform === "windows"
      ? "irm https://opts.ink/install.ps1 | iex"
      : "curl -fsSL https://opts.ink/install.sh | sh"

  return (
    <>
      <PageHeader
        description="Install the CLI, authenticate this machine, and open your first tunnel."
        eyebrow="Get connected"
        title="CLI setup"
      />

      <div className="grid gap-4 lg:grid-cols-[1.2fr_1fr]">
        <Card>
          <CardContent className="space-y-8 pt-(--card-spacing)">
            <SetupStep
              description="Choose your platform and run the installer."
              step={1}
              title="Install OpenTunnel"
            >
              <div className="mb-3 flex gap-1">
                {(["macos", "linux", "windows"] as const).map((item) => (
                  <Button
                    key={item}
                    onClick={() => setPlatform(item)}
                    size="sm"
                    variant={platform === item ? "default" : "outline"}
                  >
                    {item === "macos"
                      ? "macOS"
                      : item === "linux"
                        ? "Linux"
                        : "Windows"}
                  </Button>
                ))}
              </div>
              <CommandBlock value={installCommand} />
            </SetupStep>

            <SetupStep
              description="A browser window opens so you can securely approve the CLI."
              step={2}
              title="Sign in from the terminal"
            >
              <CommandBlock value="opentunnel login" />
            </SetupStep>

            <SetupStep
              description="Point a persistent public domain at your local server."
              step={3}
              title="Start a tunnel"
            >
              <CommandBlock value="opentunnel up 3000 --domain <your-domain>" />
            </SetupStep>
          </CardContent>
        </Card>

        <Card className="overflow-hidden bg-[#111] text-[#d6d6d6]">
          <CardHeader className="border-b border-white/10">
            <div className="flex items-center gap-2">
              <i className="size-2.5 rounded-full bg-[#ff5f56]" />
              <i className="size-2.5 rounded-full bg-[#ffbd2e]" />
              <i className="size-2.5 rounded-full bg-[#27c93f]" />
              <span className="ml-2 text-xs text-white/50">Terminal</span>
            </div>
          </CardHeader>
          <CardContent className="text-xs leading-relaxed tracking-wide whitespace-pre-wrap">
            <span className="text-white/50">
              $ opentunnel up 3000 --domain &lt;your-domain&gt;
            </span>
            {"\n\n"}
            <span className="text-emerald-400">
              ✓ Authenticated as johndoe@example.com
            </span>
            {"\n"}
            <span className="text-emerald-400">
              ✓ Connected to OpenTunnel edge
            </span>
            {"\n\n"}
            <span className="text-white/50">Public URL</span>
            {"\n"}
            <span className="text-sky-300">
              https://&lt;your-domain&gt;.opts.ink
            </span>
            {"\n\n"}
            <span className="text-white/50">Forwarding</span>
            {"\n"}
            http://127.0.0.1:3000
            {"\n\n"}
            <span className="text-white/40">
              Watching for requests… press Ctrl+C to stop
            </span>
          </CardContent>
        </Card>
      </div>
    </>
  )
}

function SetupStep({
  step,
  title,
  description,
  children,
}: {
  step: number
  title: string
  description: string
  children: React.ReactNode
}) {
  return (
    <div className="flex gap-4">
      <span className="flex size-7 shrink-0 items-center justify-center bg-muted text-xs font-medium">
        {step}
      </span>
      <div className="min-w-0 flex-1">
        <CardTitle className="mb-1">{title}</CardTitle>
        <CardDescription className="mb-3">{description}</CardDescription>
        {children}
      </div>
    </div>
  )
}

function CommandBlock({ value }: { value: string }) {
  return (
    <div
      className={cn(
        "flex items-center justify-between gap-2 bg-muted/50 px-3 py-2 text-xs"
      )}
    >
      <code className="break-all">{value}</code>
      <CopyButton value={value} />
    </div>
  )
}
