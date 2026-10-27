import path from 'node:path'
import babel from '@rolldown/plugin-babel'
import tailwindcss from '@tailwindcss/vite'
import react, { reactCompilerPreset } from '@vitejs/plugin-react'
import { defineConfig, loadEnv } from 'vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    plugins: [
      react(),
      babel({ presets: [reactCompilerPreset()] }),
      tailwindcss(),
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
        // OIDC runtime config written by zitadel-setup
        '/config.json': {
          target: env.VITE_API_PROXY_TARGET || 'http://localhost',
          changeOrigin: true,
        },
      },
    },
  }
})
