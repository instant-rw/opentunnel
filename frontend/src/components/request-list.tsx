import { ActivityIcon, MagnifyingGlassIcon } from "@phosphor-icons/react"
import { useState } from "react"

import { Badge } from "@/components/ui/badge"
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
import type { CapturedRequest } from "@/lib/api"
import { cn, formatRelativeTime, methodTone, statusVariant } from "@/lib/utils"

export function RequestList({
  requests,
  loading,
  selectedId,
  onSelect,
}: {
  requests: CapturedRequest[]
  loading: boolean
  selectedId?: string
  onSelect: (request: CapturedRequest) => void
}) {
  const [query, setQuery] = useState("")
  const filtered = requests.filter((request) =>
    `${request.method} ${request.path}`
      .toLowerCase()
      .includes(query.toLowerCase()),
  )

  return (
    <Card>
      <CardHeader className="flex-row items-start justify-between gap-4 space-y-0">
        <div>
          <CardTitle>Live requests</CardTitle>
          <CardDescription>
            Requests appear here as they hit your public URL.
          </CardDescription>
        </div>
        <div className="relative w-full max-w-xs">
          <MagnifyingGlassIcon className="pointer-events-none absolute top-1/2 left-2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="pl-7"
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Filter requests"
            value={query}
          />
        </div>
      </CardHeader>
      <CardContent className="px-0">
        {loading ? (
          <div className="space-y-2 px-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : filtered.length ? (
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
              {filtered.map((request) => (
                <TableRow
                  className={cn(
                    "cursor-pointer",
                    selectedId === request.id && "bg-muted/60",
                  )}
                  key={request.id}
                  onClick={() => onSelect(request)}
                >
                  <TableCell>
                    <span
                      className={cn(
                        "px-1.5 py-0.5 font-mono text-[10px] font-semibold",
                        methodTone(request.method),
                      )}
                    >
                      {request.method}
                    </span>
                  </TableCell>
                  <TableCell>
                    <div className="font-mono text-xs font-medium">
                      {request.path}
                    </div>
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
                {query ? "No matching requests" : "Waiting for traffic"}
              </EmptyTitle>
              <EmptyDescription>
                {query
                  ? "Try a different method or path."
                  : "Send a request to your public URL to see it here."}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        )}
      </CardContent>
    </Card>
  )
}
