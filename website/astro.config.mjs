import { defineConfig, fontProviders } from "astro/config";
import starlight from "@astrojs/starlight";
import sitemap from "@astrojs/sitemap";

import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  site: "https://errorfamily.lars.software",

  compressHTML: true,

  prefetch: {
    prefetchAll: false,
    defaultStrategy: "hover",
  },

  fonts: [
    {
      provider: fontProviders.google(),
      name: "Space Grotesk",
      cssVariable: "--font-space-grotesk",
      weights: [300, 400, 500, 600, 700],
      styles: ["normal"],
      subsets: ["latin"],
      fallbacks: ["sans-serif"],
    },
    {
      provider: fontProviders.fontsource(),
      name: "JetBrains Mono",
      cssVariable: "--font-jetbrains-mono",
      weights: [400, 500, 600, 700],
      styles: ["normal"],
      subsets: ["latin"],
      fallbacks: ["monospace"],
    },
  ],

  integrations: [
    sitemap(),
    starlight({
      title: "go-error-family",
      favicon: "/favicon.svg",
      customCss: ["./src/styles/starlight.css"],
      expressiveCode: {
        themes: ["github-light", "github-dark"],
        frames: {
          showCopyToClipboardButton: true,
        },
      },
      sidebar: [
        {
          label: "Getting Started",
          items: [
            { label: "Installation", slug: "getting-started/installation" },
            { label: "Quick Start", slug: "getting-started/quick-start" },
          ],
        },
        {
          label: "Guides",
          items: [
            { label: "Classification", slug: "guides/classification" },
            { label: "Custom Error Types", slug: "guides/error-types" },
            { label: "HTTP & CLI Boundaries", slug: "guides/http-and-cli" },
            { label: "Diagnostics", slug: "guides/diagnostics" },
            { label: "Performance", slug: "guides/benchmarks" },
            { label: "Twelve-Factor Logs", slug: "guides/twelve-factor-logs" },
          ],
        },
        {
          label: "API Reference",
          items: [
            { label: "Public API", slug: "api-reference" },
            {
              label: "Full API on pkg.go.dev",
              link: "https://pkg.go.dev/github.com/larsartmann/go-error-family",
            },
          ],
        },
        {
          label: "Community",
          items: [
            { label: "Changelog", slug: "changelog" },
            { label: "Contributing", slug: "contributing" },
            { label: "Related Tools", slug: "related-tools" },
          ],
        },
      ],
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/LarsArtmann/go-error-family",
        },
      ],
      head: [
        {
          tag: "meta",
          attrs: {
            name: "description",
            content:
              "Structured error protocol for Go — behavioral classification (Transient/Rejection/Conflict/Corruption/Infrastructure), exit codes, HTTP status mapping, and diagnostic rules.",
          },
        },
      ],
    }),
  ],

  vite: {
    plugins: [tailwindcss()],
  },
});
