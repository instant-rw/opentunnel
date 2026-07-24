import { CheckCircleIcon, CircleIcon, GlobeIcon } from "@phosphor-icons/react"
import { useState } from "react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Spinner } from "@/components/ui/spinner"
import { api, ApiError, type Domain } from "@/lib/api"
import { cn } from "@/lib/utils"

export function NewDomainDialog({
  open,
  domains,
  preview,
  onClose,
  onCreated,
}: {
  open: boolean
  domains: Domain[]
  preview: boolean
  onClose: () => void
  onCreated: (domain: Domain) => void
}) {
  const [slug, setSlug] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const normalized = slug.trim().toLowerCase()
  const formatValid = /^[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])?$/.test(normalized)
  const taken = domains.some((domain) => domain.slug === normalized)
  const ready = formatValid && !taken

  async function create() {
    setLoading(true)
    setError("")
    try {
      const domain = preview
        ? {
            id: `dom_${Date.now()}`,
            slug: normalized,
            hostname: `${normalized}.opts.ink`,
            status: "offline" as const,
            createdAt: new Date().toISOString(),
          }
        : await api.createDomain(normalized)
      onCreated(domain)
      setSlug("")
      onClose()
    } catch (caught) {
      setError(
        caught instanceof ApiError
          ? caught.message
          : "Could not reserve this domain.",
      )
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onClose()
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <div className="mb-2 flex size-10 items-center justify-center bg-muted">
            <GlobeIcon className="size-5" />
          </div>
          <DialogTitle>Reserve a domain</DialogTitle>
          <DialogDescription>
            Choose a memorable URL you can reuse across tunnel sessions.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-3">
          <div className="grid gap-2">
            <Label htmlFor="domain-slug">Domain</Label>
            <div className="flex items-center gap-2">
              <Input
                autoFocus
                id="domain-slug"
                onChange={(event) =>
                  setSlug(
                    event.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""),
                  )
                }
                placeholder="my-project"
                value={slug}
              />
              <span className="shrink-0 text-muted-foreground">.opts.ink</span>
            </div>
          </div>
          {slug ? (
            <div
              className={cn(
                "flex items-center gap-2 text-xs",
                ready ? "text-emerald-700" : "text-destructive",
              )}
            >
              {ready ? (
                <CheckCircleIcon className="size-4" />
              ) : (
                <CircleIcon className="size-4" />
              )}
              {taken
                ? "That domain is already in your workspace."
                : ready
                  ? "Available to reserve"
                  : "Use 3–63 letters, numbers, or hyphens."}
            </div>
          ) : null}
          {error ? <p className="text-xs text-destructive">{error}</p> : null}
        </div>
        <DialogFooter>
          <Button onClick={onClose} variant="secondary">
            Cancel
          </Button>
          <Button disabled={!ready || loading} onClick={create}>
            {loading ? <Spinner /> : null}
            Reserve domain
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
