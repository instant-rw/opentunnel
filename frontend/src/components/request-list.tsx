import {
  ActivityIcon,
  CaretLeftIcon,
  CaretRightIcon,
  MagnifyingGlassIcon,
} from "@phosphor-icons/react"
import { useEffect, useRef, useState } from "react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Input } from "@/components/ui/input"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import {
  emptyFilters,
  filtersActive,
  type CapturedRequest,
  type PathMode,
  type RequestFilters,
  type StatusClass,
} from "@/lib/api"
import { cn, formatRelativeTime, methodTone, statusVariant } from "@/lib/utils"

const METHODS = [
  "GET",
  "POST",
  "PUT",
  "PATCH",
  "DELETE",
  "HEAD",
  "OPTIONS",
] as const

const STATUS_OPTIONS: { value: StatusClass; label: string }[] = [
  { value: "", label: "All statuses" },
  { value: "2xx", label: "2xx" },
  { value: "3xx", label: "3xx" },
  { value: "4xx", label: "4xx" },
  { value: "5xx", label: "5xx" },
]

const selectClassName = cn(
  "h-8 rounded-none border border-input bg-transparent px-2.5 text-xs outline-none",
  "focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-ring/50",
  "disabled:cursor-not-allowed disabled:opacity-50 dark:bg-input/30"
)

export function RequestList({
  requests,
  loading,
  page,
  hasPrevious,
  hasNext,
  filters,
  selectedId,
  onSelect,
  onFiltersChange,
  onPrevious,
  onNext,
}: {
  requests: CapturedRequest[]
  loading: boolean
  page: number
  hasPrevious: boolean
  hasNext: boolean
  filters: RequestFilters
  selectedId?: string
  onSelect: (request: CapturedRequest) => void
  onFiltersChange: (filters: RequestFilters) => void
  onPrevious: () => void
  onNext: () => void
}) {
  const [pathInput, setPathInput] = useState(filters.path)
  const filtersRef = useRef(filters)
  const onFiltersChangeRef = useRef(onFiltersChange)
  const active = filtersActive(filters)
  const showPagination = hasPrevious || hasNext

  useEffect(() => {
    filtersRef.current = filters
    onFiltersChangeRef.current = onFiltersChange
  }, [filters, onFiltersChange])

  useEffect(() => {
    setPathInput(filters.path)
  }, [filters.path])

  useEffect(() => {
    const timeout = window.setTimeout(() => {
      const nextPath = pathInput.trim()
      if (nextPath === filtersRef.current.path) return
      onFiltersChangeRef.current({ ...filtersRef.current, path: nextPath })
    }, 300)
    return () => window.clearTimeout(timeout)
  }, [pathInput])

  function updateFilters(patch: Partial<RequestFilters>) {
    onFiltersChange({ ...filtersRef.current, ...patch })
  }

  return (
    <Card>
      <CardHeader className="gap-4 space-y-0">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <CardTitle>Live requests</CardTitle>
            <CardDescription>
              Requests appear here as they hit your public URL.
            </CardDescription>
          </div>
          {active ? (
            <Button
              onClick={() => {
                setPathInput("")
                onFiltersChange(emptyFilters())
              }}
              size="sm"
              variant="ghost"
            >
              Clear filters
            </Button>
          ) : null}
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:flex-wrap sm:items-center">
          <select
            aria-label="Filter by method"
            className={cn(selectClassName, "w-full sm:w-32")}
            onChange={(event) => updateFilters({ method: event.target.value })}
            value={filters.method}
          >
            <option value="">All methods</option>
            {METHODS.map((method) => (
              <option key={method} value={method}>
                {method}
              </option>
            ))}
          </select>
          <select
            aria-label="Filter by status"
            className={cn(selectClassName, "w-full sm:w-36")}
            onChange={(event) =>
              updateFilters({ statusClass: event.target.value as StatusClass })
            }
            value={filters.statusClass}
          >
            {STATUS_OPTIONS.map((option) => (
              <option key={option.label} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
          <div className="flex w-full min-w-0 flex-1 flex-col gap-2 sm:max-w-md sm:flex-row sm:items-center">
            <ToggleGroup
              aria-label="Path filter mode"
              className="shrink-0"
              onValueChange={(value) => {
                const next = value[0] as PathMode | undefined
                if (next === "include" || next === "exclude") {
                  updateFilters({ pathMode: next })
                }
              }}
              size="sm"
              spacing={0}
              value={[filters.pathMode]}
              variant="outline"
            >
              <ToggleGroupItem value="include">Include</ToggleGroupItem>
              <ToggleGroupItem value="exclude">Exclude</ToggleGroupItem>
            </ToggleGroup>
            <div className="relative min-w-0 flex-1">
              <MagnifyingGlassIcon className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="pl-7"
                onChange={(event) => setPathInput(event.target.value)}
                placeholder={
                  filters.pathMode === "exclude"
                    ? "Exclude path…"
                    : "Include path…"
                }
                value={pathInput}
              />
            </div>
          </div>
        </div>
      </CardHeader>
      <CardContent className="px-0">
        {loading ? (
          <div className="space-y-2 px-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : requests.length ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Method</TableHead>
                <TableHead>Path</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>Time</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {requests.map((request) => (
                <TableRow
                  className={cn(
                    "cursor-pointer",
                    selectedId === request.id && "bg-muted/60"
                  )}
                  key={request.id}
                  onClick={() => onSelect(request)}
                >
                  <TableCell>
                    <span
                      className={cn(
                        "px-1.5 py-0.5 text-[10px] font-semibold",
                        methodTone(request.method)
                      )}
                    >
                      {request.method}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="text-xs font-medium">{request.path}</div>
                    {request.query ? (
                      <div className="truncate text-[11px] text-muted-foreground">
                        ?{request.query}
                      </div>
                    ) : null}
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant(request.response?.status)}>
                      {request.response?.status ?? "Pending"}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {request.response?.durationMs ?? "—"} ms
                  </TableCell>
                  <TableCell>
                    {formatRelativeTime(request.receivedAt)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        ) : (
          <Empty className="border-0 py-10">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <ActivityIcon />
              </EmptyMedia>
              <EmptyTitle>
                {active ? "No matching requests" : "Waiting for traffic"}
              </EmptyTitle>
              <EmptyDescription>
                {active
                  ? "Try a different method, status, or path."
                  : "Send a request to your public URL to see it here."}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}
        {showPagination ? (
          <div className="flex items-center justify-between gap-3 border-t px-4 py-3">
            <Button
              disabled={!hasPrevious || loading}
              onClick={onPrevious}
              size="sm"
              variant="outline"
            >
              <CaretLeftIcon />
              Back
            </Button>
            <span className="text-xs text-muted-foreground">Page {page}</span>
            <Button
              disabled={!hasNext || loading}
              onClick={onNext}
              size="sm"
              variant="outline"
            >
              Next
              <CaretRightIcon />
            </Button>
          </div>
        ) : null}
      </CardContent>
    </Card>
  )
}
