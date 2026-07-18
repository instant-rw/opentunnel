"use client";

import {
  Activity,
  ArrowUpRight,
  Check,
  CircleDot,
  Copy,
  ExternalLink,
  Gauge,
  Globe2,
  KeyRound,
  Laptop,
  LayoutDashboard,
  LogOut,
  Menu,
  MoreHorizontal,
  Plus,
  RadioTower,
  RefreshCw,
  RotateCcw,
  Search,
  Server,
  Settings,
  ShieldCheck,
  Terminal,
  Trash2,
  X,
  Zap,
} from "lucide-react";
import {
  useCallback,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

import {
  Badge,
  Button,
  Card,
  EmptyState,
  Input,
  Skeleton,
} from "@/components/ui";
import {
  api,
  ApiError,
  type CapturedRequest,
  type Domain,
  type TokenSummary,
  type User,
} from "@/lib/api";
import { previewDomains, previewRequests, previewTokens } from "@/lib/fixtures";
import { subscribeToDomain, type StreamState } from "@/lib/sse";
import { cn, decodeBody, formatRelativeTime } from "@/lib/utils";

type Section = "overview" | "tunnels" | "setup" | "access";
type InspectorTab = "request" | "response";

const navItems: { id: Section; label: string; icon: ReactNode }[] = [
  { id: "overview", label: "Overview", icon: <LayoutDashboard size={17} /> },
  { id: "tunnels", label: "Tunnels", icon: <RadioTower size={17} /> },
  { id: "setup", label: "CLI setup", icon: <Terminal size={17} /> },
  { id: "access", label: "Access", icon: <KeyRound size={17} /> },
];

function methodTone(method: string) {
  if (method === "GET") return "method-get";
  if (method === "POST") return "method-post";
  if (method === "DELETE") return "method-delete";
  return "method-neutral";
}

function statusTone(status?: number) {
  if (!status) return "neutral" as const;
  if (status >= 500) return "danger" as const;
  if (status >= 400) return "warning" as const;
  if (status >= 200 && status < 300) return "success" as const;
  return "neutral" as const;
}

function CopyButton({ value }: { value: string }) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    await navigator.clipboard.writeText(value);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  }

  return (
    <button
      aria-label="Copy to clipboard"
      className="copy-button"
      onClick={copy}
      type="button"
    >
      {copied ? <Check size={15} /> : <Copy size={15} />}
      {copied ? "Copied" : "Copy"}
    </button>
  );
}

function StatCard({
  label,
  value,
  detail,
  icon,
}: {
  label: string;
  value: string;
  detail: ReactNode;
  icon: ReactNode;
}) {
  return (
    <Card className="stat-card">
      <div className="stat-card-head">
        <span>{label}</span>
        <span className="stat-icon">{icon}</span>
      </div>
      <strong>{value}</strong>
      <div className="stat-detail">{detail}</div>
    </Card>
  );
}

function PageHeader({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow: string;
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <header className="page-header">
      <div>
        <p className="page-eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        <p>{description}</p>
      </div>
      {action}
    </header>
  );
}

function NewDomainDialog({
  domains,
  onClose,
  onCreated,
  preview,
}: {
  domains: Domain[];
  onClose: () => void;
  onCreated: (domain: Domain) => void;
  preview: boolean;
}) {
  const [slug, setSlug] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const normalized = slug.trim().toLowerCase();
  const formatValid = /^[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])?$/.test(normalized);
  const taken = domains.some((domain) => domain.slug === normalized);
  const ready = formatValid && !taken;

  async function create() {
    setLoading(true);
    setError("");
    try {
      const domain = preview
        ? {
            id: `dom_${Date.now()}`,
            slug: normalized,
            hostname: `${normalized}.opts.ink`,
            status: "offline" as const,
            createdAt: new Date().toISOString(),
          }
        : await api.createDomain(normalized);
      onCreated(domain);
      onClose();
    } catch (caught) {
      setError(
        caught instanceof ApiError
          ? caught.message
          : "Could not reserve this domain.",
      );
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="dialog-backdrop" onMouseDown={onClose} role="presentation">
      <Card
        aria-labelledby="new-domain-title"
        aria-modal="true"
        className="dialog"
        onMouseDown={(event) => event.stopPropagation()}
        role="dialog"
      >
        <button
          aria-label="Close"
          className="dialog-close"
          onClick={onClose}
          type="button"
        >
          <X size={18} />
        </button>
        <span className="dialog-icon">
          <Globe2 size={22} />
        </span>
        <h2 id="new-domain-title">Reserve a domain</h2>
        <p>Choose a memorable URL you can reuse across tunnel sessions.</p>
        <label className="domain-field">
          Domain
          <div>
            <Input
              autoFocus
              onChange={(event) =>
                setSlug(
                  event.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ""),
                )
              }
              placeholder="my-project"
              value={slug}
            />
            <span>.opts.ink</span>
          </div>
        </label>
        {slug ? (
          <div
            className={cn(
              "availability",
              ready ? "availability-ready" : "availability-error",
            )}
          >
            {ready ? <Check size={15} /> : <CircleDot size={15} />}
            {taken
              ? "That domain is already in your workspace."
              : ready
                ? "Available to reserve"
                : "Use 3–63 letters, numbers, or hyphens."}
          </div>
        ) : null}
        {error ? <p className="form-error">{error}</p> : null}
        <div className="dialog-actions">
          <Button onClick={onClose} variant="secondary">
            Cancel
          </Button>
          <Button disabled={!ready} loading={loading} onClick={create}>
            Reserve domain
          </Button>
        </div>
      </Card>
    </div>
  );
}

function RequestInspector({
  request,
  online,
  preview,
  onClose,
}: {
  request: CapturedRequest;
  online: boolean;
  preview: boolean;
  onClose: () => void;
}) {
  const [tab, setTab] = useState<InspectorTab>("request");
  const [replaying, setReplaying] = useState(false);
  const [replayStatus, setReplayStatus] = useState("");
  const content = tab === "request" ? request.body : request.response?.body;
  const headers =
    tab === "request" ? request.headers : (request.response?.headers ?? []);

  async function replay() {
    setReplaying(true);
    setReplayStatus("");
    try {
      if (!preview) {
        await api.replayRequest(request.id);
      } else {
        await new Promise((resolve) => window.setTimeout(resolve, 650));
      }
      setReplayStatus("Replay queued");
    } catch (caught) {
      setReplayStatus(
        caught instanceof ApiError ? caught.message : "Replay failed",
      );
    } finally {
      setReplaying(false);
    }
  }

  return (
    <aside className="inspector" aria-label="Request details">
      <header className="inspector-head">
        <div>
          <div className="request-title">
            <span className={cn("method", methodTone(request.method))}>
              {request.method}
            </span>
            <strong>{request.path}</strong>
          </div>
          <p>{formatRelativeTime(request.receivedAt)}</p>
        </div>
        <button aria-label="Close inspector" onClick={onClose} type="button">
          <X size={19} />
        </button>
      </header>
      <div className="inspector-summary">
        <div>
          <span>Status</span>
          <Badge tone={statusTone(request.response?.status)}>
            {request.response?.status ?? "Pending"}
          </Badge>
        </div>
        <div>
          <span>Duration</span>
          <strong>{request.response?.durationMs ?? "—"} ms</strong>
        </div>
        <div>
          <span>Request ID</span>
          <code>{request.id}</code>
        </div>
      </div>
      <div className="tab-list" role="tablist">
        <button
          aria-selected={tab === "request"}
          onClick={() => setTab("request")}
          role="tab"
          type="button"
        >
          Request
        </button>
        <button
          aria-selected={tab === "response"}
          onClick={() => setTab("response")}
          role="tab"
          type="button"
        >
          Response
        </button>
      </div>
      <div className="inspector-body">
        <section>
          <div className="section-title">
            <h3>Headers</h3>
            <span>{headers.length}</span>
          </div>
          <div className="header-table">
            {headers.length ? (
              headers.map((header) => (
                <div key={header.name}>
                  <code>{header.name}</code>
                  <span>{header.values.join(", ")}</span>
                </div>
              ))
            ) : (
              <p className="muted">No headers captured.</p>
            )}
          </div>
        </section>
        <section>
          <div className="section-title">
            <h3>Body</h3>
            {content?.truncated ? (
              <Badge tone="warning">Truncated</Badge>
            ) : null}
          </div>
          {content ? (
            <>
              <pre>{decodeBody(content.base64)}</pre>
              <p className="capture-meta">
                {content.size.toLocaleString()} bytes captured
                {content.truncated ? " · capture limit reached" : ""}
              </p>
            </>
          ) : (
            <div className="no-body">No body captured</div>
          )}
        </section>
      </div>
      <footer className="inspector-footer">
        <div>
          <Button
            disabled={!online}
            loading={replaying}
            onClick={replay}
            variant="secondary"
          >
            <RotateCcw size={15} /> Replay request
          </Button>
          {!online ? <span>Tunnel must be online to replay.</span> : null}
        </div>
        {replayStatus ? <Badge tone="blue">{replayStatus}</Badge> : null}
      </footer>
    </aside>
  );
}

function RequestList({
  requests,
  loading,
  selectedId,
  onSelect,
}: {
  requests: CapturedRequest[];
  loading: boolean;
  selectedId?: string;
  onSelect: (request: CapturedRequest) => void;
}) {
  const [query, setQuery] = useState("");
  const filtered = requests.filter((request) =>
    `${request.method} ${request.path}`
      .toLowerCase()
      .includes(query.toLowerCase()),
  );

  return (
    <Card className="requests-card">
      <div className="card-header">
        <div>
          <h2>Live requests</h2>
          <p>Requests appear here as they hit your public URL.</p>
        </div>
        <label className="search-field">
          <Search size={15} />
          <input
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Filter requests"
            value={query}
          />
        </label>
      </div>
      <div className="request-table" role="table">
        <div className="request-table-head" role="row">
          <span>Method</span>
          <span>Path</span>
          <span>Status</span>
          <span>Duration</span>
          <span>Time</span>
        </div>
        {loading ? (
          <div className="request-loading">
            <Skeleton />
            <Skeleton />
            <Skeleton />
          </div>
        ) : filtered.length ? (
          filtered.map((request) => (
            <button
              className={cn(
                "request-row",
                selectedId === request.id && "request-row-selected",
              )}
              key={request.id}
              onClick={() => onSelect(request)}
              role="row"
              type="button"
            >
              <span className={cn("method", methodTone(request.method))}>
                {request.method}
              </span>
              <span className="path-cell">
                <strong>{request.path}</strong>
                {request.query ? <small>?{request.query}</small> : null}
              </span>
              <span>
                <Badge tone={statusTone(request.response?.status)}>
                  {request.response?.status ?? "Pending"}
                </Badge>
              </span>
              <span>{request.response?.durationMs ?? "—"} ms</span>
              <span>{formatRelativeTime(request.receivedAt)}</span>
            </button>
          ))
        ) : (
          <EmptyState
            description={
              query
                ? "Try a different method or path."
                : "Send a request to your public URL to see it here."
            }
            icon={<Activity size={20} />}
            title={query ? "No matching requests" : "Waiting for traffic"}
          />
        )}
      </div>
    </Card>
  );
}

function TunnelCard({
  domain,
  selected,
  onSelect,
}: {
  domain: Domain;
  selected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      className={cn("tunnel-card", selected && "tunnel-card-selected")}
      onClick={onSelect}
      type="button"
    >
      <span className="tunnel-icon">
        <Globe2 size={19} />
      </span>
      <span className="tunnel-main">
        <span>
          <strong>{domain.hostname}</strong>
          <Badge tone={domain.status === "online" ? "success" : "neutral"}>
            <i className={cn("status-dot", domain.status)} />
            {domain.status}
          </Badge>
        </span>
        <small>
          {domain.status === "online"
            ? "Forwarding to localhost:3000"
            : "No active tunnel session"}
        </small>
      </span>
      <ArrowUpRight size={17} />
    </button>
  );
}

function SetupSection() {
  const [platform, setPlatform] = useState<"macos" | "linux" | "windows">(
    "macos",
  );
  const installCommand =
    platform === "windows"
      ? "irm https://opts.ink/install.ps1 | iex"
      : "curl -fsSL https://opts.ink/install.sh | sh";

  return (
    <>
      <PageHeader
        description="Install the CLI, authenticate this machine, and open your first tunnel."
        eyebrow="Get connected"
        title="CLI setup"
      />
      <div className="setup-grid">
        <Card className="setup-steps">
          <div className="setup-step">
            <span>1</span>
            <div>
              <h2>Install OpenTunnel</h2>
              <p>Choose your platform and run the installer.</p>
              <div className="platform-tabs">
                {(["macos", "linux", "windows"] as const).map((item) => (
                  <button
                    className={cn(platform === item && "active")}
                    key={item}
                    onClick={() => setPlatform(item)}
                    type="button"
                  >
                    {item === "macos"
                      ? "macOS"
                      : item === "linux"
                        ? "Linux"
                        : "Windows"}
                  </button>
                ))}
              </div>
              <div className="command-block">
                <code>{installCommand}</code>
                <CopyButton value={installCommand} />
              </div>
            </div>
          </div>
          <div className="setup-step">
            <span>2</span>
            <div>
              <h2>Sign in from the terminal</h2>
              <p>A browser window opens so you can securely approve the CLI.</p>
              <div className="command-block">
                <code>opentunnel login</code>
                <CopyButton value="opentunnel login" />
              </div>
            </div>
          </div>
          <div className="setup-step">
            <span>3</span>
            <div>
              <h2>Start a tunnel</h2>
              <p>Point a persistent public domain at your local server.</p>
              <div className="command-block">
                <code>opentunnel up 3000 --domain checkout</code>
                <CopyButton value="opentunnel up 3000 --domain checkout" />
              </div>
            </div>
          </div>
        </Card>
        <Card className="terminal-preview">
          <div className="terminal-bar">
            <i />
            <i />
            <i />
            <span>Terminal</span>
          </div>
          <pre>
            <span>$ opentunnel up 3000 --domain checkout</span>
            {"\n\n"}
            <b>✓ Authenticated as troy@example.com</b>
            {"\n"}
            <b>✓ Connected to OpenTunnel edge</b>
            {"\n\n"}
            <em>Public URL</em>
            {"\n"}
            <a>https://checkout.opts.ink</a>
            {"\n\n"}
            <em>Forwarding</em>
            {"\n"}
            http://127.0.0.1:3000
            {"\n\n"}
            <small>Watching for requests… press Ctrl+C to stop</small>
          </pre>
        </Card>
      </div>
    </>
  );
}

function AccessSection({
  tokens,
  onRevoke,
}: {
  tokens: TokenSummary[];
  onRevoke: (id: string) => void;
}) {
  const [revoking, setRevoking] = useState("");
  async function revoke(id: string) {
    setRevoking(id);
    try {
      await onRevoke(id);
    } finally {
      setRevoking("");
    }
  }

  return (
    <>
      <PageHeader
        description="Review machines with access to your workspace and revoke credentials."
        eyebrow="Security"
        title="Sessions & tokens"
      />
      <Card className="security-notice">
        <ShieldCheck size={21} />
        <div>
          <strong>Your CLI tokens are stored securely</strong>
          <p>
            Token values are never shown here. Revocation takes effect
            immediately and disconnects active tunnels.
          </p>
        </div>
      </Card>
      <Card className="current-session-card">
        <span className="device-icon">
          <Globe2 size={19} />
        </span>
        <div>
          <strong>Current web session</strong>
          <p>
            This browser · Active now · Protected with a secure, HttpOnly
            session cookie
          </p>
        </div>
        <Badge tone="success">Current</Badge>
      </Card>
      <Card className="token-card">
        <div className="card-header">
          <div>
            <h2>Authorized devices</h2>
            <p>{tokens.length} active CLI credentials</p>
          </div>
        </div>
        {tokens.length ? (
          <div className="token-list">
            {tokens.map((token) => (
              <div className="token-row" key={token.id}>
                <span className="device-icon">
                  <Laptop size={19} />
                </span>
                <div>
                  <strong>{token.name}</strong>
                  <span>
                    Last used{" "}
                    {token.lastUsedAt
                      ? formatRelativeTime(token.lastUsedAt)
                      : "never"}{" "}
                    · Created {formatRelativeTime(token.createdAt)}
                  </span>
                </div>
                <Button
                  aria-label={`Revoke ${token.name}`}
                  loading={revoking === token.id}
                  onClick={() => revoke(token.id)}
                  size="sm"
                  variant="danger"
                >
                  <Trash2 size={14} /> Revoke
                </Button>
              </div>
            ))}
          </div>
        ) : (
          <EmptyState
            description="Run opentunnel login to authorize a machine."
            icon={<KeyRound size={20} />}
            title="No authorized devices"
          />
        )}
      </Card>
    </>
  );
}

export function Dashboard({
  user,
  preview,
  onSignOut,
}: {
  user: User;
  preview: boolean;
  onSignOut: () => void;
}) {
  const [section, setSection] = useState<Section>("overview");
  const [domains, setDomains] = useState<Domain[]>(
    preview ? previewDomains : [],
  );
  const [requests, setRequests] = useState<CapturedRequest[]>(
    preview ? previewRequests : [],
  );
  const [tokens, setTokens] = useState<TokenSummary[]>(
    preview ? previewTokens : [],
  );
  const [selectedDomainId, setSelectedDomainId] = useState(
    preview ? (previewDomains[0]?.id ?? "") : "",
  );
  const [selectedRequest, setSelectedRequest] = useState<CapturedRequest>();
  const [streamState, setStreamState] = useState<StreamState>("closed");
  const [loading, setLoading] = useState(!preview);
  const [newDomainOpen, setNewDomainOpen] = useState(false);
  const [mobileNavOpen, setMobileNavOpen] = useState(false);
  const [error, setError] = useState("");

  const selectedDomain = useMemo(
    () =>
      domains.find((domain) => domain.id === selectedDomainId) ?? domains[0],
    [domains, selectedDomainId],
  );
  const visibleRequests = useMemo(
    () =>
      selectedDomain
        ? requests.filter((request) => request.domainId === selectedDomain.id)
        : requests,
    [requests, selectedDomain],
  );

  const loadDashboard = useCallback(async () => {
    if (preview) return;
    setLoading(true);
    setError("");
    try {
      const [loadedDomains, loadedTokens] = await Promise.all([
        api.listDomains(),
        api.listTokens(),
      ]);
      setDomains(loadedDomains);
      setTokens(loadedTokens);
      const domainId = selectedDomainId || loadedDomains[0]?.id;
      if (domainId) {
        setSelectedDomainId(domainId);
        const page = await api.listRequests(domainId);
        setRequests(page.items);
      }
    } catch (caught) {
      setError(
        caught instanceof ApiError
          ? caught.message
          : "Could not load dashboard.",
      );
    } finally {
      setLoading(false);
    }
  }, [preview, selectedDomainId]);

  useEffect(() => {
    const timeout = window.setTimeout(() => void loadDashboard(), 0);
    return () => window.clearTimeout(timeout);
  }, [loadDashboard]);

  useEffect(() => {
    if (preview || !selectedDomainId) return;
    return subscribeToDomain(selectedDomainId, {
      onStateChange: setStreamState,
      onEvent: (event) => {
        if (event.type === "domain.status") {
          setDomains((current) =>
            current.map((domain) =>
              domain.id === event.domain.id ? event.domain : domain,
            ),
          );
          return;
        }
        if (
          event.type === "request.created" ||
          event.type === "request.updated"
        ) {
          setRequests((current) => [
            event.request,
            ...current.filter((item) => item.id !== event.request.id),
          ]);
          setSelectedRequest((current) =>
            current?.id === event.request.id ? event.request : current,
          );
        }
      },
    });
  }, [preview, selectedDomainId]);

  async function signOut() {
    if (!preview) {
      try {
        await api.logout();
      } catch {
        // Clear local UI state even if the API is unavailable.
      }
    }
    onSignOut();
  }

  async function revokeToken(id: string) {
    if (!preview) {
      await api.revokeToken(id);
    }
    setTokens((current) => current.filter((token) => token.id !== id));
  }

  function navigate(next: Section) {
    setSection(next);
    setMobileNavOpen(false);
  }

  const onlineCount = domains.filter(
    (domain) => domain.status === "online",
  ).length;
  const successful = visibleRequests.filter(
    (request) =>
      request.response &&
      request.response.status >= 200 &&
      request.response.status < 400,
  ).length;
  const successRate = visibleRequests.length
    ? Math.round((successful / visibleRequests.length) * 100)
    : 0;

  return (
    <div className="app-shell">
      <aside className={cn("sidebar", mobileNavOpen && "sidebar-open")}>
        <div className="sidebar-top">
          <a className="brand" href="#" onClick={() => navigate("overview")}>
            <span className="brand-mark">
              <RadioTower size={18} />
            </span>
            <span>OpenTunnel</span>
          </a>
          <button
            aria-label="Close navigation"
            className="mobile-close"
            onClick={() => setMobileNavOpen(false)}
            type="button"
          >
            <X size={19} />
          </button>
        </div>
        <nav>
          <p>Workspace</p>
          {navItems.map((item) => (
            <button
              className={cn(section === item.id && "active")}
              key={item.id}
              onClick={() => navigate(item.id)}
              type="button"
            >
              {item.icon}
              {item.label}
              {item.id === "tunnels" && onlineCount ? (
                <span className="nav-count">{onlineCount}</span>
              ) : null}
            </button>
          ))}
        </nav>
        <div className="sidebar-help">
          <span>
            <Zap size={16} />
          </span>
          <strong>Need a hand?</strong>
          <p>Read the quickstart or explore CLI commands.</p>
          <a href="#docs">
            View documentation <ArrowUpRight size={13} />
          </a>
        </div>
        <button className="account-button" type="button">
          <span>{user.email.slice(0, 1).toUpperCase()}</span>
          <div>
            <strong>{user.email.split("@")[0]}</strong>
            <small>{user.email}</small>
          </div>
          <MoreHorizontal size={17} />
        </button>
      </aside>
      {mobileNavOpen ? (
        <button
          aria-label="Close navigation"
          className="mobile-scrim"
          onClick={() => setMobileNavOpen(false)}
          type="button"
        />
      ) : null}

      <div className="app-main">
        <div className="topbar">
          <button
            aria-label="Open navigation"
            className="mobile-menu"
            onClick={() => setMobileNavOpen(true)}
            type="button"
          >
            <Menu size={20} />
          </button>
          <div className="topbar-status">
            <span className="pulse-dot" />
            All systems operational
          </div>
          <div className="topbar-actions">
            {preview ? <Badge tone="blue">Preview data</Badge> : null}
            <button aria-label="Settings" type="button">
              <Settings size={18} />
            </button>
            <button aria-label="Sign out" onClick={signOut} type="button">
              <LogOut size={18} />
            </button>
          </div>
        </div>

        <main className="content">
          {error ? (
            <div className="error-banner">
              <span>{error}</span>
              <Button onClick={loadDashboard} size="sm" variant="secondary">
                <RefreshCw size={14} /> Retry
              </Button>
            </div>
          ) : null}

          {section === "overview" ? (
            <>
              <PageHeader
                action={
                  <Button onClick={() => setNewDomainOpen(true)}>
                    <Plus size={16} /> New domain
                  </Button>
                }
                description="Monitor tunnel health, inspect traffic, and keep localhost connected."
                eyebrow="Saturday, July 18"
                title={`Good afternoon, ${user.email.split("@")[0]}`}
              />
              <div className="stats-grid">
                <StatCard
                  detail={
                    <span className="positive">
                      <ArrowUpRight size={13} /> {onlineCount} active now
                    </span>
                  }
                  icon={<RadioTower size={18} />}
                  label="Active tunnels"
                  value={loading ? "—" : String(onlineCount)}
                />
                <StatCard
                  detail="Across selected tunnel"
                  icon={<Activity size={18} />}
                  label="Requests today"
                  value={loading ? "—" : String(visibleRequests.length)}
                />
                <StatCard
                  detail="p50 response time"
                  icon={<Gauge size={18} />}
                  label="Avg. latency"
                  value={
                    visibleRequests.length
                      ? `${Math.round(
                          visibleRequests.reduce(
                            (total, request) =>
                              total + (request.response?.durationMs ?? 0),
                            0,
                          ) / visibleRequests.length,
                        )} ms`
                      : "—"
                  }
                />
                <StatCard
                  detail="2xx and 3xx responses"
                  icon={<ShieldCheck size={18} />}
                  label="Success rate"
                  value={`${successRate}%`}
                />
              </div>

              <div className="overview-grid">
                <Card className="tunnels-card">
                  <div className="card-header">
                    <div>
                      <h2>Your tunnels</h2>
                      <p>Persistent domains and their current session.</p>
                    </div>
                    <button onClick={() => navigate("tunnels")} type="button">
                      View all <ArrowUpRight size={14} />
                    </button>
                  </div>
                  {loading ? (
                    <div className="card-skeleton">
                      <Skeleton />
                      <Skeleton />
                    </div>
                  ) : domains.length ? (
                    <div className="tunnel-list">
                      {domains.slice(0, 3).map((domain) => (
                        <TunnelCard
                          domain={domain}
                          key={domain.id}
                          onSelect={() => setSelectedDomainId(domain.id)}
                          selected={domain.id === selectedDomain?.id}
                        />
                      ))}
                    </div>
                  ) : (
                    <EmptyState
                      action={
                        <Button
                          onClick={() => setNewDomainOpen(true)}
                          size="sm"
                        >
                          <Plus size={14} /> Reserve domain
                        </Button>
                      }
                      description="Reserve a persistent URL for your first tunnel."
                      icon={<Globe2 size={20} />}
                      title="No domains yet"
                    />
                  )}
                </Card>
                <Card className="quickstart-card">
                  <div className="quickstart-icon">
                    <Terminal size={20} />
                  </div>
                  <h2>Start a tunnel</h2>
                  <p>Run this command from your project directory.</p>
                  <div className="command-block compact">
                    <code>opentunnel up 3000</code>
                    <CopyButton value="opentunnel up 3000" />
                  </div>
                  <button onClick={() => navigate("setup")} type="button">
                    CLI setup guide <ArrowUpRight size={14} />
                  </button>
                </Card>
              </div>
              <RequestList
                loading={loading}
                onSelect={setSelectedRequest}
                requests={visibleRequests}
                selectedId={selectedRequest?.id}
              />
            </>
          ) : null}

          {section === "tunnels" ? (
            <>
              <PageHeader
                action={
                  <Button onClick={() => setNewDomainOpen(true)}>
                    <Plus size={16} /> New domain
                  </Button>
                }
                description="Manage reserved domains and inspect their active tunnel sessions."
                eyebrow="Traffic"
                title="Tunnels"
              />
              <div className="tunnel-detail-grid">
                <Card className="domain-sidebar-card">
                  <div className="card-header">
                    <div>
                      <h2>Domains</h2>
                      <p>{domains.length} reserved</p>
                    </div>
                  </div>
                  <div className="tunnel-list">
                    {domains.map((domain) => (
                      <TunnelCard
                        domain={domain}
                        key={domain.id}
                        onSelect={() => setSelectedDomainId(domain.id)}
                        selected={domain.id === selectedDomain?.id}
                      />
                    ))}
                  </div>
                </Card>
                {selectedDomain ? (
                  <Card className="domain-detail-card">
                    <div className="domain-detail-head">
                      <div>
                        <span className="domain-big-icon">
                          <Globe2 size={22} />
                        </span>
                        <div>
                          <h2>{selectedDomain.hostname}</h2>
                          <Badge
                            tone={
                              selectedDomain.status === "online"
                                ? "success"
                                : "neutral"
                            }
                          >
                            <i
                              className={cn(
                                "status-dot",
                                selectedDomain.status,
                              )}
                            />
                            {selectedDomain.status}
                          </Badge>
                        </div>
                      </div>
                      <Button
                        onClick={() =>
                          window.open(`https://${selectedDomain.hostname}`)
                        }
                        size="sm"
                        variant="secondary"
                      >
                        Open URL <ExternalLink size={14} />
                      </Button>
                    </div>
                    <div className="domain-metrics">
                      <div>
                        <span>Local target</span>
                        <strong>
                          {selectedDomain.status === "online"
                            ? "localhost:3000"
                            : "Not connected"}
                        </strong>
                      </div>
                      <div>
                        <span>Live stream</span>
                        <strong>
                          {preview
                            ? "Preview"
                            : streamState === "open"
                              ? "Connected"
                              : "Reconnecting"}
                        </strong>
                      </div>
                      <div>
                        <span>Created</span>
                        <strong>
                          {new Date(
                            selectedDomain.createdAt,
                          ).toLocaleDateString()}
                        </strong>
                      </div>
                    </div>
                    {selectedDomain.status === "offline" ? (
                      <div className="offline-callout">
                        <Server size={19} />
                        <div>
                          <strong>This tunnel is offline</strong>
                          <p>
                            Start it with{" "}
                            <code>
                              opentunnel up 3000 --domain {selectedDomain.slug}
                            </code>
                          </p>
                        </div>
                        <CopyButton
                          value={`opentunnel up 3000 --domain ${selectedDomain.slug}`}
                        />
                      </div>
                    ) : (
                      <div className="online-callout">
                        <span className="pulse-dot" />
                        Connected and forwarding requests
                      </div>
                    )}
                  </Card>
                ) : (
                  <EmptyState
                    description="Reserve a domain to view its tunnel status."
                    icon={<Globe2 size={20} />}
                    title="No tunnel selected"
                  />
                )}
              </div>
              <RequestList
                loading={loading}
                onSelect={setSelectedRequest}
                requests={visibleRequests}
                selectedId={selectedRequest?.id}
              />
            </>
          ) : null}

          {section === "setup" ? <SetupSection /> : null}
          {section === "access" ? (
            <AccessSection onRevoke={revokeToken} tokens={tokens} />
          ) : null}
        </main>
      </div>

      {newDomainOpen ? (
        <NewDomainDialog
          domains={domains}
          onClose={() => setNewDomainOpen(false)}
          onCreated={(domain) => {
            setDomains((current) => [domain, ...current]);
            setSelectedDomainId(domain.id);
          }}
          preview={preview}
        />
      ) : null}
      {selectedRequest ? (
        <>
          <button
            aria-label="Close request inspector"
            className="inspector-scrim"
            onClick={() => setSelectedRequest(undefined)}
            type="button"
          />
          <RequestInspector
            online={selectedDomain?.status === "online"}
            onClose={() => setSelectedRequest(undefined)}
            preview={preview}
            request={selectedRequest}
          />
        </>
      ) : null}
    </div>
  );
}
