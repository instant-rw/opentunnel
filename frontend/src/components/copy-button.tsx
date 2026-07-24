import { CheckIcon, CopyIcon } from "@phosphor-icons/react"
import { useState } from "react"

import { Button } from "@/components/ui/button"

export function CopyButton({
  value,
  label = "Copy",
}: {
  value: string
  label?: string
}) {
  const [copied, setCopied] = useState(false)

  async function copy() {
    await navigator.clipboard.writeText(value)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1600)
  }

  return (
    <Button
      aria-label="Copy to clipboard"
      onClick={copy}
      size="sm"
      type="button"
      variant="outline"
    >
      {copied ? <CheckIcon /> : <CopyIcon />}
      {copied ? "Copied" : label}
    </Button>
  )
}
