export const SITE_URL = "https://opts.ink"
export const SITE_NAME = "OpenTunnel"
export const SITE_TAGLINE = "Localhost, publicly reachable."
export const DEFAULT_DESCRIPTION =
  "OpenTunnel gives your local apps persistent public domains—perfect for webhooks, live demos, and mobile testing. No ports to open, no router to fight, no traffic left unseen."

const OG_IMAGE_PATH = "/og-image.png"

type PageSeoOptions = {
  title: string
  description?: string
  path?: string
  image?: string
  type?: "website" | "article"
  noIndex?: boolean
}

function absoluteUrl(path: string): string {
  if (path.startsWith("http://") || path.startsWith("https://")) {
    return path
  }
  return `${SITE_URL}${path.startsWith("/") ? path : `/${path}`}`
}

export function pageTitle(title?: string): string {
  if (!title || title === SITE_NAME) {
    return `${SITE_NAME} — ${SITE_TAGLINE}`
  }
  return `${title} · ${SITE_NAME}`
}

export function seoHead({
  title,
  description = DEFAULT_DESCRIPTION,
  path = "/",
  image = OG_IMAGE_PATH,
  type = "website",
  noIndex = false,
}: PageSeoOptions) {
  const fullTitle = pageTitle(title)
  const canonical = absoluteUrl(path)
  const imageUrl = absoluteUrl(image)

  return {
    meta: [
      { title: fullTitle },
      { name: "description", content: description },
      { name: "application-name", content: SITE_NAME },
      {
        name: "robots",
        content: noIndex
          ? "noindex, nofollow"
          : "index, follow, max-image-preview:large",
      },
      { property: "og:site_name", content: SITE_NAME },
      { property: "og:type", content: type },
      { property: "og:title", content: fullTitle },
      { property: "og:description", content: description },
      { property: "og:url", content: canonical },
      { property: "og:image", content: imageUrl },
      { property: "og:locale", content: "en_US" },
      { name: "twitter:card", content: "summary_large_image" },
      { name: "twitter:title", content: fullTitle },
      { name: "twitter:description", content: description },
      { name: "twitter:image", content: imageUrl },
    ],
    links: [{ rel: "canonical", href: canonical }],
  }
}

export function softwareApplicationJsonLd() {
  return {
    type: "application/ld+json" as const,
    children: JSON.stringify({
      "@context": "https://schema.org",
      "@type": "SoftwareApplication",
      name: SITE_NAME,
      applicationCategory: "DeveloperApplication",
      operatingSystem: "macOS, Linux, Windows",
      description: DEFAULT_DESCRIPTION,
      url: SITE_URL,
      image: absoluteUrl(OG_IMAGE_PATH),
      offers: {
        "@type": "Offer",
        price: "0",
        priceCurrency: "USD",
      },
      license: "https://opensource.org/licenses/MIT",
      codeRepository: "https://github.com/optunnel/opentunnel",
    }),
  }
}

export function websiteJsonLd() {
  return {
    type: "application/ld+json" as const,
    children: JSON.stringify({
      "@context": "https://schema.org",
      "@type": "WebSite",
      name: SITE_NAME,
      url: SITE_URL,
      description: DEFAULT_DESCRIPTION,
    }),
  }
}
