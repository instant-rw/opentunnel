import type { CapturedRequest, Domain, TokenSummary, User } from "@/lib/api"

const now = Date.now()
const at = (millisecondsAgo: number) =>
  new Date(now - millisecondsAgo).toISOString()
const body = (value: string) => ({
  base64: btoa(value),
  size: value.length,
  truncated: false,
})

export const previewUser: User = {
  id: "usr_preview",
  email: "troy@example.com",
  createdAt: at(86_400_000 * 42),
}

export const previewDomains: Domain[] = [
  {
    id: "dom_checkout",
    slug: "checkout",
    hostname: "checkout.opts.ink",
    status: "online",
    createdAt: at(86_400_000 * 18),
  },
  {
    id: "dom_webhooks",
    slug: "webhooks-dev",
    hostname: "webhooks-dev.opts.ink",
    status: "offline",
    createdAt: at(86_400_000 * 6),
  },
]

export const previewRequests: CapturedRequest[] = [
  {
    id: "req_01",
    domainId: "dom_checkout",
    method: "POST",
    path: "/api/checkout",
    query: "",
    headers: [
      { name: "content-type", values: ["application/json"] },
      { name: "user-agent", values: ["Stripe/1.0 (+https://stripe.com)"] },
      { name: "authorization", values: ["[REDACTED]"] },
    ],
    body: body(
      JSON.stringify(
        { event: "checkout.completed", orderId: "ord_8h2Kp9" },
        null,
        2,
      ),
    ),
    response: {
      status: 201,
      headers: [{ name: "content-type", values: ["application/json"] }],
      body: body(JSON.stringify({ received: true }, null, 2)),
      durationMs: 84,
    },
    receivedAt: at(12_000),
  },
  {
    id: "req_02",
    domainId: "dom_checkout",
    method: "GET",
    path: "/api/products",
    query: "limit=20&active=true",
    headers: [{ name: "accept", values: ["application/json"] }],
    response: {
      status: 200,
      headers: [{ name: "content-type", values: ["application/json"] }],
      body: {
        ...body(JSON.stringify({ items: [{ id: "prod_01" }] })),
        truncated: true,
        size: 18432,
      },
      durationMs: 42,
    },
    receivedAt: at(49_000),
  },
  {
    id: "req_03",
    domainId: "dom_checkout",
    method: "OPTIONS",
    path: "/api/checkout",
    query: "",
    headers: [{ name: "origin", values: ["https://example.com"] }],
    response: {
      status: 204,
      headers: [],
      durationMs: 9,
    },
    receivedAt: at(93_000),
  },
  {
    id: "req_04",
    domainId: "dom_checkout",
    method: "POST",
    path: "/webhooks/github",
    query: "",
    headers: [{ name: "content-type", values: ["application/json"] }],
    body: body(JSON.stringify({ action: "opened", number: 42 })),
    response: {
      status: 500,
      headers: [{ name: "content-type", values: ["text/plain"] }],
      body: body("Internal Server Error"),
      durationMs: 216,
    },
    receivedAt: at(127_000),
  },
]

export const previewTokens: TokenSummary[] = [
  {
    id: "tok_macbook",
    name: "Troy's MacBook Pro",
    createdAt: at(86_400_000 * 24),
    lastUsedAt: at(43_000),
  },
  {
    id: "tok_ci",
    name: "staging-ci",
    createdAt: at(86_400_000 * 8),
    lastUsedAt: at(86_400_000 * 2),
  },
]
