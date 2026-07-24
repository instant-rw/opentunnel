import { getApiBaseUrl, type CapturedRequest, type Domain } from "@/lib/api"

export type DomainEvent =
  | { type: "domain.status"; domain: Domain }
  | { type: "request.created"; request: CapturedRequest }
  | { type: "request.updated"; request: CapturedRequest }

export type StreamState = "connecting" | "open" | "closed"

type StreamHandlers = {
  onEvent: (event: DomainEvent) => void
  onStateChange: (state: StreamState) => void
}

function isDomainEvent(value: unknown): value is DomainEvent {
  if (!value || typeof value !== "object" || !("type" in value)) {
    return false
  }
  const type = (value as { type: unknown }).type
  return (
    type === "domain.status" ||
    type === "request.created" ||
    type === "request.updated"
  )
}

export function subscribeToDomain(
  domainId: string,
  handlers: StreamHandlers,
): () => void {
  handlers.onStateChange("connecting")
  const source = new EventSource(
    `${getApiBaseUrl()}/domains/${encodeURIComponent(domainId)}/events`,
    { withCredentials: true },
  )

  source.onopen = () => handlers.onStateChange("open")
  const receive = (
    message: MessageEvent<string>,
    type?: DomainEvent["type"],
  ) => {
    try {
      const payload: unknown = JSON.parse(message.data)
      const event: unknown =
        type === "domain.status"
          ? { type, domain: payload }
          : type === "request.created" || type === "request.updated"
            ? { type, request: payload }
            : payload
      if (isDomainEvent(event)) {
        handlers.onEvent(event)
      }
    } catch {
      // A malformed event is ignored; the stream remains usable.
    }
  }
  source.onmessage = (message) => receive(message)
  source.addEventListener("domain.status", (message) =>
    receive(message as MessageEvent<string>, "domain.status"),
  )
  source.addEventListener("request.created", (message) =>
    receive(message as MessageEvent<string>, "request.created"),
  )
  source.addEventListener("request.updated", (message) =>
    receive(message as MessageEvent<string>, "request.updated"),
  )
  source.onerror = () => handlers.onStateChange("connecting")

  return () => {
    source.close()
    handlers.onStateChange("closed")
  }
}
