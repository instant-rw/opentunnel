import { ArrowCounterClockwiseIcon, XIcon } from "@phosphor-icons/react"
import { useState } from "react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Spinner } from "@/components/ui/spinner"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { api, ApiError, type CapturedRequest } from "@/lib/api"
import { cn, decodeBody, formatRelativeTime, methodTone, statusVariant } from "@/lib/utils"

export function RequestInspector({
  request,
  online,
  preview,
  onClose,
}: {
  request: CapturedRequest
  online: boolean
  preview: boolean
  onClose: () => void
}) {
  const [tab, setTab] = useState("request")
  const [replaying, setReplaying] = useState(false)
  const [replayStatus, setReplayStatus] = useState("")

  async function replay() {
    setReplaying(true)
    setReplayStatus("")
    try {
      if (!preview) {
        await api.replayRequest(request.id)
      } else {
        await new Promise((resolve) => window.setTimeout(resolve, 650))
      }
      setReplayStatus("Replay queued")
    } catch (caught) {
      setReplayStatus(
        caught instanceof ApiError ? caught.message : "Replay failed",
      )
    } finally {
      setReplaying(false)
    }
  }

  return (
    <>
      <button
        aria-label="Close request inspector"
        className="fixed inset-0 z-40 bg-black/20"
        onClick={onClose}
        type="button"
      />
      <aside
        aria-label="Request details"
        className="fixed inset-y-0 right-0 z-50 flex w-full max-w-lg flex-col border-l bg-background shadow-xl"
      >
        <header className="flex items-start justify-between gap-3 border-b p-4">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <span
                className={cn(
                  "px-1.5 py-0.5 font-mono text-[10px] font-semibold",
                  methodTone(request.method),
                )}
              >
                {request.method}
              </span>
              <strong className="truncate font-mono text-sm">
                {request.path}
              </strong>
            </div>
            <p className="mt-1 text-xs text-muted-foreground">
              {formatRelativeTime(request.receivedAt)}
            </p>
          </div>
          <Button
            aria-label="Close inspector"
            onClick={onClose}
            size="icon-sm"
            variant="ghost"
          >
            <XIcon />
          </Button>
        </header>

        <div className="grid grid-cols-3 gap-3 border-b p-4 text-xs">
          <div>
            <span className="text-muted-foreground">Status</span>
            <div className="mt-1">
              <Badge variant={statusVariant(request.response?.status)}>
                {request.response?.status ?? "Pending"}
              </Badge>
            </div>
          </div>
          <div>
            <span className="text-muted-foreground">Duration</span>
            <strong className="mt-1 block">
              {request.response?.durationMs ?? "—"} ms
            </strong>
          </div>
          <div className="min-w-0">
            <span className="text-muted-foreground">Request ID</span>
            <code className="mt-1 block truncate text-[11px]">{request.id}</code>
          </div>
        </div>

        <Tabs
          className="flex min-h-0 flex-1 flex-col"
          onValueChange={setTab}
          value={tab}
        >
          <TabsList className="mx-4 mt-3 w-auto self-start">
            <TabsTrigger value="request">Request</TabsTrigger>
            <TabsTrigger value="response">Response</TabsTrigger>
          </TabsList>
          {(["request", "response"] as const).map((panel) => {
            const panelContent =
              panel === "request" ? request.body : request.response?.body
            const panelHeaders =
              panel === "request"
                ? request.headers
                : (request.response?.headers ?? [])
            return (
              <TabsContent
                className="min-h-0 flex-1 overflow-auto p-4"
                key={panel}
                value={panel}
              >
                <section className="mb-4">
                  <div className="mb-2 flex items-center justify-between">
                    <h3 className="text-sm font-medium">Headers</h3>
                    <span className="text-xs text-muted-foreground">
                      {panelHeaders.length}
                    </span>
                  </div>
                  <div className="space-y-2">
                    {panelHeaders.length ? (
                      panelHeaders.map((header) => (
                        <div
                          className="grid gap-1 border-b pb-2 last:border-0"
                          key={header.name}
                        >
                          <code className="text-[11px] font-medium">
                            {header.name}
                          </code>
                          <span className="break-all text-xs text-muted-foreground">
                            {header.values.join(", ")}
                          </span>
                        </div>
                      ))
                    ) : (
                      <p className="text-xs text-muted-foreground">
                        No headers captured.
                      </p>
                    )}
                  </div>
                </section>
                <section>
                  <div className="mb-2 flex items-center justify-between">
                    <h3 className="text-sm font-medium">Body</h3>
                    {panelContent?.truncated ? (
                      <Badge variant="secondary">Truncated</Badge>
                    ) : null}
                  </div>
                  {panelContent ? (
                    <>
                      <pre className="overflow-auto bg-muted/50 p-3 font-mono text-[11px] leading-relaxed">
                        {decodeBody(panelContent.base64)}
                      </pre>
                      <p className="mt-2 text-xs text-muted-foreground">
                        {panelContent.size.toLocaleString()} bytes captured
                        {panelContent.truncated
                          ? " · capture limit reached"
                          : ""}
                      </p>
                    </>
                  ) : (
                    <div className="border border-dashed p-6 text-center text-xs text-muted-foreground">
                      No body captured
                    </div>
                  )}
                </section>
              </TabsContent>
            )
          })}
        </Tabs>

        <footer className="flex items-center justify-between gap-3 border-t p-4">
          <div className="space-y-1">
            <Button
              disabled={!online || replaying}
              onClick={replay}
              variant="secondary"
            >
              {replaying ? <Spinner /> : <ArrowCounterClockwiseIcon />}
              Replay request
            </Button>
            {!online ? (
              <p className="text-[11px] text-muted-foreground">
                Tunnel must be online to replay.
              </p>
            ) : null}
          </div>
          {replayStatus ? <Badge variant="outline">{replayStatus}</Badge> : null}
        </footer>
      </aside>
    </>
  )
}
