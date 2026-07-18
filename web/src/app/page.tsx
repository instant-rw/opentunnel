"use client";

import { RadioTower } from "lucide-react";
import { useEffect, useState } from "react";

import { AuthScreen } from "@/components/auth-screen";
import { Dashboard } from "@/components/dashboard";
import { api, ApiError, type User } from "@/lib/api";
import { previewUser } from "@/lib/fixtures";

type AppState =
  | { status: "loading" }
  | { status: "anonymous"; userCode?: string }
  | { status: "authenticated"; user: User; preview: boolean };

export default function HomePage() {
  const [state, setState] = useState<AppState>({ status: "loading" });

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const userCode = params.get("user_code") ?? undefined;

    api
      .me()
      .then((user) =>
        setState({ status: "authenticated", user, preview: false }),
      )
      .catch((error: unknown) => {
        if (error instanceof ApiError && error.status === 0 && !userCode) {
          setState({
            status: "authenticated",
            user: previewUser,
            preview: true,
          });
          return;
        }
        setState({ status: "anonymous", userCode });
      });
  }, []);

  if (state.status === "loading") {
    return (
      <main className="app-loading">
        <span className="brand-mark">
          <RadioTower size={20} />
        </span>
        <strong>OpenTunnel</strong>
        <span className="loading-line" />
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

  return (
    <Dashboard
      onSignOut={() => setState({ status: "anonymous" })}
      preview={state.preview}
      user={state.user}
    />
  );
}
