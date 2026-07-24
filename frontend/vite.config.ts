import { defineConfig } from "vite"
import { devtools } from "@tanstack/devtools-vite"
import { tanstackStart } from "@tanstack/react-start/plugin/vite"
import viteReact from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

const config = defineConfig({
  resolve: { tsconfigPaths: true },
  preview: {
    host: "127.0.0.1",
  },
  plugins: [
    devtools(),
    tailwindcss(),
    tanstackStart({
      spa: {
        enabled: true,
        maskPath: "/login",
        prerender: {
          outputPath: "/_shell.html",
          crawlLinks: false,
          retryCount: 0,
        },
      },
      // Explicit concurrency avoids hang when os.cpus() is empty (some CI/sandboxes).
      // Prerender the landing page so crawlers get real HTML + meta tags.
      prerender: {
        concurrency: 1,
        enabled: true,
        crawlLinks: false,
        autoStaticPathsDiscovery: false,
      },
      pages: [
        {
          path: "/",
          prerender: { enabled: true, outputPath: "/index.html" },
        },
      ],
    }),
    viteReact(),
  ],
})

export default config
