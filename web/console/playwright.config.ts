import { defineConfig, devices } from '@playwright/test'

/**
 * Playwright E2E configuration for VC Console.
 *
 * Tests run against the Vite dev server (auto-started).
 * Usage:
 *   npx playwright test              # run all E2E tests
 *   npx playwright test --headed     # run with browser visible
 *   npx playwright test --ui         # run with Playwright UI
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  timeout: 30_000,

  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure'
  },

  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] }
    }
  ],

  /* Start the Vite dev server before tests */
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: !process.env.CI,
    timeout: 30_000
  }
})
