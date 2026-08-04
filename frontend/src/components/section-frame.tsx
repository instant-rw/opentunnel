import type { ComponentProps } from "react"

import { DecorIcon } from "@/components/decor-icon"
import { FullWidthDivider } from "@/components/full-width-divider"
import { cn } from "@/lib/utils"

/** Corner crosses + horizontal/vertical rules from @efferd/hero-2. */
export function SectionFrame({
  children,
  className,
  ...props
}: ComponentProps<"div">) {
  return (
    <div className={cn("relative", className)} {...props}>
      <DecorIcon className="size-4" position="top-left" />
      <DecorIcon className="size-4" position="top-right" />
      <DecorIcon className="size-4" position="bottom-left" />
      <DecorIcon className="size-4" position="bottom-right" />
      <FullWidthDivider className="-top-px" />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-y-0 left-0 w-px bg-border"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute inset-y-0 right-0 w-px bg-border"
      />
      {children}
      <FullWidthDivider className="-bottom-px" />
    </div>
  )
}

/** Faded double vertical accents from @efferd/hero-2 (page/hero sides). */
export function SectionSideBorders({ className }: { className?: string }) {
  return (
    <div
      aria-hidden
      className={cn(
        "pointer-events-none absolute inset-0 z-10 overflow-hidden",
        className
      )}
    >
      <div className="absolute inset-y-0 left-4 w-px bg-gradient-to-b from-transparent via-border to-transparent md:left-8" />
      <div className="absolute inset-y-0 right-4 w-px bg-gradient-to-b from-transparent via-border to-transparent md:right-8" />
      <div className="absolute inset-y-0 left-8 w-px bg-gradient-to-b from-transparent via-border/50 to-transparent md:left-12" />
      <div className="absolute inset-y-0 right-8 w-px bg-gradient-to-b from-transparent via-border/50 to-transparent md:right-12" />
    </div>
  )
}
