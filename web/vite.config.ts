import { createHash } from 'node:crypto'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import babel from '@rolldown/plugin-babel'
import tailwindcss from '@tailwindcss/vite'
import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import { defineConfig, loadEnv, type Plugin } from 'vite'

/**
 * dist/sw.js from sw.js: the service worker that opens the app without coverage,
 * with this build's files to keep and a version that changes with them.
 */
function serviceWorker(): Plugin {
  return {
    name: 'librequenda-service-worker',
    apply: 'build',
    generateBundle(_, bundle) {
      const files = Object.keys(bundle)
        .filter((file) => file.startsWith('assets/'))
        .map((file) => `/${file}`)
        .sort()
      const precache = ['/', '/manifest.webmanifest', '/favicon.svg', '/icon-192.png', ...files]
      const version = createHash('sha256').update(files.join('\n')).digest('hex').slice(0, 12)
      const source = readFileSync(path.resolve(import.meta.dirname, 'sw.js'), 'utf8')
        .replace('__VERSION__', version)
        .replace('__PRECACHE__', JSON.stringify(precache))
      this.emitFile({ type: 'asset', fileName: 'sw.js', source })
    },
  }
}

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    plugins: [
      react(),
      babel({ presets: [reactCompilerPreset()] }),
      tailwindcss(),
      serviceWorker(),
    ],
    resolve: {
      alias: {
        '@': path.resolve(import.meta.dirname, './src'),
      },
    },
    server: {
      proxy: {
        // /api/* -> Caddy (docker compose), which strips /api and forwards to the Go API
        '/api': {
          target: env.VITE_API_PROXY_TARGET || 'http://localhost',
          changeOrigin: true,
        },
        // the voice line (voice/): its config, its session and the call's WebSocket
        '/voice': {
          target: env.VITE_VOICE_PROXY_TARGET || env.VITE_API_PROXY_TARGET || 'http://localhost',
          changeOrigin: true,
          ws: true,
        },
        // OIDC runtime config written by zitadel-setup
        '/config.json': {
          target: env.VITE_API_PROXY_TARGET || 'http://localhost',
          changeOrigin: true,
        },
      },
    },
  }
})
