import { defineConfig } from "vite"
import { devtools } from "@tanstack/devtools-vite"
import { tanstackStart } from "@tanstack/react-start/plugin/vite"
import viteReact from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

const config = defineConfig({
  resolve: { tsconfigPaths: true },
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
      prerender: { concurrency: 1 },
    }),
    viteReact(),
  ],
})

export default config
