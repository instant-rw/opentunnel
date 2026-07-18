"use client";

import { RadioTower } from "lucide-react";
import { useEffect, useState } from "react";

import { AuthScreen } from "@/components/auth-screen";
import { Dashboard } from "@/components/dashboard";
import { DeviceApproved } from "@/components/device-approved";
import { Button, Card } from "@/components/ui";
import { api, ApiError, type User } from "@/lib/api";
import { clearUserCodeFromUrl } from "@/lib/device-auth";
import { previewUser } from "@/lib/fixtures";

type AppState =
  | { status: "loading" }
  | { status: "anonymous"; userCode?: string }
  | { status: "approving"; user: User }
  | { status: "approved"; user: User }
  | { status: "approve_failed"; user: User; message: string }
  | { status: "authenticated"; user: User; preview: boolean };

export function AppShell() {
  const [state, setState] = useState<AppState>({ status: "loading" });

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const userCode = params.get("user_code") ?? undefined;
    let cancelled = false;

    (async () => {
      try {
        const user = await api.me();
        if (cancelled) {
          return;
        }
        if (!userCode) {
          setState({ status: "authenticated", user, preview: false });
          return;
        }

        setState({ status: "approving", user });
        try {
          await api.approveDevice(userCode);
          if (cancelled) {
            return;
          }
          clearUserCodeFromUrl();
          setState({ status: "approved", user });
        } catch (caught) {
          if (cancelled) {
            return;
          }
          setState({
            status: "approve_failed",
            user,
            message:
              caught instanceof ApiError
                ? caught.message
                : "Could not approve this device.",
          });
        }
      } catch (error: unknown) {
        if (cancelled) {
          return;
        }
        if (error instanceof ApiError && error.status === 0 && !userCode) {
          setState({
            status: "authenticated",
            user: previewUser,
            preview: true,
          });
          return;
        }
        setState({ status: "anonymous", userCode });
      }
    })();

    return () => {
      cancelled = true;
    };
  }, []);

  if (state.status === "loading" || state.status === "approving") {
    return (
      <main className="app-loading">
        <span className="brand-mark">
          <RadioTower size={20} />
        </span>
        <strong>OpenTunnel</strong>
        <span className="loading-line" />
        {state.status === "approving" ? (
          <p className="eyebrow">Approving device…</p>
        ) : null}
      </main>
    );
  }

  if (state.status === "anonymous") {
    return (
      <AuthScreen
        onAuthenticated={(user) =>
          setState({ status: "authenticated", user, preview: false })
        }
        onPreview={() =>
          setState({
            status: "authenticated",
            user: previewUser,
            preview: true,
          })
        }
        userCode={state.userCode}
      />
    );
  }

  if (state.status === "approved") {
    return (
      <DeviceApproved
        onContinue={() =>
          setState({
            status: "authenticated",
            user: state.user,
            preview: false,
          })
        }
      />
    );
  }

  if (state.status === "approve_failed") {
    return (
      <main className="auth-shell">
        <Card className="approval-success">
          <p className="eyebrow">Approval failed</p>
          <h1>Couldn’t approve this device</h1>
          <p>{state.message}</p>
          <Button
            onClick={() =>
              setState({
                status: "authenticated",
                user: state.user,
                preview: false,
              })
            }
            variant="secondary"
          >
            Open dashboard
          </Button>
        </Card>
      </main>
    );
  }

  return (
    <Dashboard
      onSignOut={() => setState({ status: "anonymous" })}
      preview={state.preview}
      user={state.user}
    />
  );
}
