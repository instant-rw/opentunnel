import {
  BookOpenIcon,
  BugIcon,
  GithubLogoIcon,
  HandHeartIcon,
} from "@phosphor-icons/react"
import type { Icon } from "@phosphor-icons/react"

export const GITHUB_URL = "https://github.com/optunnel/opentunnel"
export const DOCS_URL = "https://github.com/optunnel/opentunnel/tree/main/docs"
export const CONTRIBUTE_URL =
  "https://github.com/optunnel/opentunnel/blob/main/CONTRIBUTING.md"
export const ISSUES_URL = "https://github.com/optunnel/opentunnel/issues"

export type ExternalLink = {
  title: string
  href: string
  description: string
  icon: Icon
}

export const externalLinks: ExternalLink[] = [
  {
    title: "Documentation",
    href: DOCS_URL,
    description: "Guides, CLI reference, and self-hosting",
    icon: BookOpenIcon,
  },
  {
    title: "GitHub",
    href: GITHUB_URL,
    description: "Browse the source and star the repo",
    icon: GithubLogoIcon,
  },
  {
    title: "Contribute",
    href: CONTRIBUTE_URL,
    description: "Open a PR and help shape OpenTunnel",
    icon: HandHeartIcon,
  },
  {
    title: "Report an issue",
    href: ISSUES_URL,
    description: "File a bug or request a feature",
    icon: BugIcon,
  },
]
