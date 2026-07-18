"use client";

import { Check } from "lucide-react";

import { Button, Card } from "@/components/ui";

export function DeviceApproved({ onContinue }: { onContinue: () => void }) {
  return (
    <main className="auth-shell">
      <Card className="approval-success">
        <span className="success-orb">
          <Check size={28} />
        </span>
        <p className="eyebrow">Device approved</p>
        <h1>You’re all set</h1>
        <p>
          Return to your terminal. OpenTunnel will finish signing in
          automatically.
        </p>
        <Button onClick={onContinue} variant="secondary">
          Open dashboard
        </Button>
      </Card>
    </main>
  );
}
