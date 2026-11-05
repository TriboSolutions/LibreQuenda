import { defineConfig, devices } from '@playwright/test'

// Accessibility checks (WCAG 2.1): the real app in Chromium, the API mocked (see e2e/mocks.ts).
// Run with `pnpm test:a11y`; CI installs the browser with `pnpm exec playwright install --with-deps chromium`.
const PORT = 5179

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['github'], ['list']] : 'list',
  use: {
    baseURL: `http://localhost:${PORT}`,
    locale: 'gl-ES',
    timezoneId: 'Europe/Madrid',
    trace: 'retain-on-failure',
  },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'] }, grepInvert: /@reflow/ },
    // 1.4.10 Reflow: 320 CSS px wide
    { name: 'reflow-320', use: { ...devices['Desktop Chrome'], viewport: { width: 320, height: 640 } }, grep: /@reflow/ },
  ],
  // the production build: the dev server re-optimises lazy dependencies (recharts) and reloads mid-test
  webServer: {
    // built with a version, as in a deployment: the tests on a newer API's version need one
    command: `VITE_APP_VERSION=e2e pnpm exec vite build --logLevel warn && pnpm exec vite preview --port ${PORT} --strictPort`,
    port: PORT,
    reuseExistingServer: !process.env.CI,
  },
})
