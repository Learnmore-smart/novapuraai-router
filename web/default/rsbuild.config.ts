import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { defineConfig, loadEnv } from '@rsbuild/core'
import { pluginReact } from '@rsbuild/plugin-react'
import { pluginTailwindcss } from '@rsbuild/plugin-tailwindcss'
import { tanstackRouter } from '@tanstack/router-plugin/rspack'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig(({ envMode }) => {
  const env = loadEnv({ mode: envMode, prefixes: ['VITE_'] })
  // Go API (embedded static UI on PORT, usually 3000). Rsbuild proxies /api here.
  const serverUrl =
    process.env.VITE_REACT_APP_SERVER_URL ||
    env.rawPublicVars.VITE_REACT_APP_SERVER_URL ||
    'http://localhost:3000'

  // Dedicated frontend dev port — must NOT share Go's PORT (default 3000).
  // Opening :3000 serves the embedded `web/default/dist` snapshot (needs rebuild).
  // Opening this port serves live Rsbuild HMR.
  const devPort = Number(
    process.env.VITE_DEV_PORT || process.env.RSBUILD_PORT || 5173
  )

  const isProd = envMode === 'production'
  // Windows + non-ASCII paths often miss native FS events; poll is more reliable.
  const useWatchPolling =
    process.env.CHOKIDAR_USEPOLLING === '1' ||
    process.env.WATCHPACK_POLLING === 'true' ||
    process.platform === 'win32'

  const devProxy = Object.fromEntries(
    (['/api', '/mj', '/pg'] as const).map((key) => [
      key,
      { target: serverUrl, changeOrigin: true },
    ])
  ) as Record<string, { target: string; changeOrigin: boolean }>

  return {
    plugins: [pluginReact(), pluginTailwindcss({ optimize: false })],
    // Rsbuild 2: replaces deprecated `performance.chunkSplit` (RSPack 2 aligned)
    splitChunks: {
      preset: 'default',
      cacheGroups: {
        'vendor-react': {
          test: /node_modules[\\/](react|react-dom)[\\/]/,
          name: 'vendor-react',
          chunks: 'all',
          priority: 0,
          enforce: true,
        },
        'vendor-ui-primitives': {
          test: /node_modules[\\/](@base-ui|@radix-ui)[\\/]/,
          name: 'vendor-ui-primitives',
          chunks: 'all',
          priority: 0,
          enforce: true,
        },
        'vendor-tanstack': {
          test: /node_modules[\\/]@tanstack[\\/]/,
          name: 'vendor-tanstack',
          chunks: 'all',
          priority: 0,
          enforce: true,
        },
      },
    },
    source: {
      entry: {
        index: './src/main.tsx',
      },
    },
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    html: {
      template: './index.html',
    },
    // Keep the browser stable during development; refresh manually when needed.
    dev: {
      hmr: false,
      liveReload: false,
    },
    server: {
      host: '0.0.0.0',
      port: isProd ? undefined : devPort,
      // Fail loudly if 5173 is taken instead of silently moving ports.
      strictPort: !isProd,
      proxy: devProxy,
    },
    output: {
      // Production optimizations
      minify: isProd,
      target: 'web',
      distPath: {
        root: 'dist',
      },
      copy: [
        {
          // Ensure root /favicon.ico is the multi-size brand pack
          // (public/logo/novapuraai_router_favicon → web/default/public).
          from: './public/favicon.ico',
          to: 'favicon.ico',
        },
      ],
      // Rely on Rsbuild default legalComments ("linked" → per-chunk *.LICENSE.txt) in all modes.
      // Do not set "none" in production: that strips minifier-preserved third-party notices and
      // extracted license files, which some distributions require for open-source compliance.
    },
    performance: {
      // Remove console in production
      removeConsole: isProd ? ['log'] : false,
      // Persistent Rspack cache: dev restarts reuse the previous compilation
      // instead of cold-compiling every module. Kept off for production so
      // release builds stay fully reproducible. If dev output ever looks
      // stale, delete web/default/node_modules/.cache and restart.
      buildCache: !isProd,
    },
    tools: {
      rspack: {
        plugins: [
          tanstackRouter({
            target: 'react',
            // Dev: avoid per-route async chunks (reduces white flash on navigation + faster HMR feedback).
            // Prod: keep route-based code splitting.
            autoCodeSplitting: isProd,
          }),
        ],
        module: {
          rules: [
            // Official docs markdown is imported as raw UTF-8 strings.
            {
              test: /\.md$/,
              type: 'asset/source',
            },
          ],
        },
        // Reliable file watching on Windows (and when explicitly requested).
        ...(!isProd && useWatchPolling
          ? {
              watchOptions: {
                poll: 1000,
                aggregateTimeout: 300,
                ignored: /[\\/]node_modules[\\/]/,
              },
            }
          : {}),
      },
    },
  }
})
