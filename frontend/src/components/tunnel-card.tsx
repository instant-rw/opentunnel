import { ArrowUpRightIcon, GlobeIcon } from "@phosphor-icons/react"

import { Badge } from "@/components/ui/badge"
import type { Domain } from "@/lib/api"
import { cn } from "@/lib/utils"

export function TunnelCard({
  domain,
  selected,
  onSelect,
}: {
  domain: Domain
  selected: boolean
  onSelect: () => void
}) {
  return (
    <button
      className={cn(
        "flex w-full items-center gap-3 border p-3 text-left transition-colors hover:bg-muted/50",
        selected && "border-foreground/30 bg-muted/40",
      )}
      onClick={onSelect}
      type="button"
    >
      <span className="flex size-9 items-center justify-center bg-muted">
        <GlobeIcon className="size-4" />
      </span>
      <span className="min-w-0 flex-1">
        <span className="flex flex-wrap items-center gap-2">
          <strong className="truncate text-sm">{domain.hostname}</strong>
          <Badge
            variant={domain.status === "online" ? "default" : "outline"}
          >
            <span
              className={cn(
                "size-1.5 rounded-full",
                domain.status === "online" ? "bg-emerald-500" : "bg-muted-foreground",
              )}
            />
            {domain.status}
          </Badge>
        </span>
        <small className="text-xs text-muted-foreground">
          {domain.status === "online"
            ? "Forwarding to localhost:3000"
            : "No active tunnel session"}
        </small>
      </span>
      <ArrowUpRightIcon className="size-4 shrink-0 text-muted-foreground" />
    </button>
  )
}
