import type { ReactNode } from "react"

export function PageHeader({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow: string
  title: string
  description: string
  action?: ReactNode
}) {
  return (
    <header className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between">
      <div className="space-y-1">
        <p className="text-xs tracking-wide text-muted-foreground uppercase">
          {eyebrow}
        </p>
        <h1 className="font-heading text-2xl font-medium tracking-tight">
          {title}
        </h1>
        <p className="max-w-2xl text-sm text-muted-foreground">{description}</p>
      </div>
      {action}
    </header>
  )
}
