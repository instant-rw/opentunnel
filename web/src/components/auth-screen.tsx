"use client";

import {
  ArrowRight,
  Check,
  Eye,
  EyeOff,
  KeyRound,
  RadioTower,
  ShieldCheck,
} from "lucide-react";
import { useState, type FormEvent } from "react";

import { DeviceApproved } from "@/components/device-approved";
import { Button, Input } from "@/components/ui";
import { api, ApiError, type User } from "@/lib/api";
import { clearUserCodeFromUrl } from "@/lib/device-auth";

type AuthMode = "login" | "register";

export function AuthScreen({
  userCode,
  onAuthenticated,
  onPreview,
}: {
  userCode?: string;
  onAuthenticated: (user: User) => void;
  onPreview: () => void;
}) {
  const [mode, setMode] = useState<AuthMode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [approvedUser, setApprovedUser] = useState<User>();

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setLoading(true);

    try {
      if (mode === "register") {
        await api.register(email, password);
        await api.login(email, password);
      } else {
        await api.login(email, password);
      }
      const user = await api.me();
      if (userCode) {
        await api.approveDevice(userCode);
        clearUserCodeFromUrl();
        setApprovedUser(user);
      } else {
        onAuthenticated(user);
      }
    } catch (caught) {
      setError(
        caught instanceof ApiError
          ? caught.message
          : "Something went wrong. Please try again.",
      );
    } finally {
      setLoading(false);
    }
  }

  if (approvedUser) {
    return (
      <DeviceApproved onContinue={() => onAuthenticated(approvedUser)} />
    );
  }

  return (
    <main className="auth-shell">
      <section className="auth-brand-panel">
        <a className="brand brand-light" href="#" aria-label="OpenTunnel home">
          <span className="brand-mark">
            <RadioTower size={18} />
          </span>
          <span>OpenTunnel</span>
        </a>
        <div className="auth-pitch">
          <p className="eyebrow">Local, but live</p>
          <h1>Bring localhost into the open.</h1>
          <p>
            Secure public URLs, instant request inspection, and painless webhook
            debugging—without touching your router.
          </p>
          <ul className="auth-benefits">
            <li>
              <Check size={16} /> Persistent, memorable domains
            </li>
            <li>
              <Check size={16} /> End-to-end request visibility
            </li>
            <li>
              <Check size={16} /> One-command tunnel setup
            </li>
          </ul>
        </div>
        <p className="auth-quote">
          “The missing devtool between a webhook and localhost.”
        </p>
      </section>

      <section className="auth-form-panel">
        <div className="auth-form-wrap">
          {userCode ? (
            <div className="approval-banner">
              <ShieldCheck size={20} />
              <div>
                <strong>Approve CLI sign-in</strong>
                <span>
                  Code <kbd>{userCode.toUpperCase()}</kbd>
                </span>
              </div>
            </div>
          ) : null}
          <div className="auth-heading">
            <span className="mobile-brand">
              <RadioTower size={18} /> OpenTunnel
            </span>
            <h2>{mode === "login" ? "Welcome back" : "Create your account"}</h2>
            <p>
              {mode === "login"
                ? "Sign in to manage your tunnels."
                : "Start exposing localhost in under a minute."}
            </p>
          </div>

          <form onSubmit={submit}>
            <label>
              Email
              <Input
                autoComplete="email"
                onChange={(event) => setEmail(event.target.value)}
                placeholder="you@example.com"
                required
                type="email"
                value={email}
              />
            </label>
            <label>
              <span className="label-row">
                Password
                {mode === "login" ? (
                  <a href="#reset">Forgot password?</a>
                ) : null}
              </span>
              <span className="password-field">
                <Input
                  autoComplete={
                    mode === "login" ? "current-password" : "new-password"
                  }
                  minLength={8}
                  onChange={(event) => setPassword(event.target.value)}
                  placeholder="At least 8 characters"
                  required
                  type={showPassword ? "text" : "password"}
                  value={password}
                />
                <button
                  aria-label={showPassword ? "Hide password" : "Show password"}
                  onClick={() => setShowPassword((visible) => !visible)}
                  type="button"
                >
                  {showPassword ? <EyeOff size={17} /> : <Eye size={17} />}
                </button>
              </span>
            </label>
            {error ? <p className="form-error">{error}</p> : null}
            <Button className="auth-submit" loading={loading} type="submit">
              {userCode
                ? "Sign in & approve"
                : mode === "login"
                  ? "Sign in"
                  : "Create account"}
              <ArrowRight size={16} />
            </Button>
          </form>

          <p className="auth-switch">
            {mode === "login"
              ? "New to OpenTunnel?"
              : "Already have an account?"}
            <button
              onClick={() =>
                setMode((current) =>
                  current === "login" ? "register" : "login",
                )
              }
              type="button"
            >
              {mode === "login" ? "Create an account" : "Sign in"}
            </button>
          </p>
          <button className="preview-link" onClick={onPreview} type="button">
            <KeyRound size={14} /> Preview the dashboard
          </button>
        </div>
        <p className="legal">
          By continuing, you agree to the Terms and Privacy Policy.
        </p>
      </section>
    </main>
  );
}
