CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id uuid PRIMARY KEY,
    email text NOT NULL,
    password_hash text NOT NULL,
    email_verified_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_normalized CHECK (email = lower(email))
);
CREATE UNIQUE INDEX users_email_unique ON users (lower(email));

CREATE TABLE web_sessions (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash bytea NOT NULL UNIQUE,
    csrf_hash bytea NOT NULL,
    expires_at timestamptz NOT NULL,
    last_seen_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX web_sessions_user_id_idx ON web_sessions(user_id);
CREATE INDEX web_sessions_expires_at_idx ON web_sessions(expires_at);

CREATE TABLE cli_tokens (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name text NOT NULL DEFAULT 'OpenTunnel CLI',
    token_hash bytea NOT NULL UNIQUE,
    last_used_at timestamptz,
    revoked_at timestamptz,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX cli_tokens_user_id_idx ON cli_tokens(user_id);

CREATE TYPE device_authorization_status AS ENUM ('pending', 'approved', 'consumed');
CREATE TABLE device_authorizations (
    id uuid PRIMARY KEY,
    device_code_hash bytea NOT NULL UNIQUE,
    user_code text NOT NULL UNIQUE,
    status device_authorization_status NOT NULL DEFAULT 'pending',
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    interval_seconds integer NOT NULL,
    last_polled_at timestamptz,
    expires_at timestamptz NOT NULL,
    approved_at timestamptz,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX device_authorizations_expires_at_idx ON device_authorizations(expires_at);

CREATE TABLE domains (
    id uuid PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slug text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT domains_slug_format CHECK (slug ~ '^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$')
);
CREATE INDEX domains_user_id_idx ON domains(user_id);

CREATE TABLE tunnel_sessions (
    id uuid PRIMARY KEY,
    domain_id uuid NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    cli_token_id uuid NOT NULL REFERENCES cli_tokens(id) ON DELETE CASCADE,
    connected_at timestamptz NOT NULL DEFAULT now(),
    last_heartbeat_at timestamptz NOT NULL DEFAULT now(),
    disconnected_at timestamptz
);
CREATE UNIQUE INDEX tunnel_sessions_one_active_per_domain
    ON tunnel_sessions(domain_id) WHERE disconnected_at IS NULL;

CREATE TABLE requests (
    id uuid PRIMARY KEY,
    domain_id uuid NOT NULL REFERENCES domains(id) ON DELETE CASCADE,
    method text NOT NULL,
    path text NOT NULL,
    query text NOT NULL DEFAULT '',
    request_headers jsonb NOT NULL DEFAULT '[]',
    request_body bytea,
    request_body_size bigint,
    request_body_truncated boolean NOT NULL DEFAULT false,
    response_status integer,
    response_headers jsonb,
    response_body bytea,
    response_body_size bigint,
    response_body_truncated boolean NOT NULL DEFAULT false,
    duration_ms bigint,
    received_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX requests_domain_received_idx ON requests(domain_id, received_at DESC, id DESC);

CREATE TYPE replay_status AS ENUM ('queued', 'running', 'succeeded', 'failed');
CREATE TABLE replay_attempts (
    id uuid PRIMARY KEY,
    request_id uuid NOT NULL REFERENCES requests(id) ON DELETE CASCADE,
    status replay_status NOT NULL DEFAULT 'queued',
    error text,
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz
);
CREATE INDEX replay_attempts_request_id_idx ON replay_attempts(request_id);
