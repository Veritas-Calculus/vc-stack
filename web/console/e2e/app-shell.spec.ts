import { test, expect } from '@playwright/test'

/**
 * E2E tests verifying that the application shell handles edge cases:
 * - Unknown routes → 404 or redirect
 * - Multiple rapid navigations → no crashes
 * - Session management UI
 * - API endpoint health (when backend is absent, should fail gracefully)
 */
test.describe('Application Shell', () => {
  test('unknown route shows 404 or redirects to login', async ({ page }) => {
    await page.goto('/this-route-does-not-exist')
    // Should either show login (auth redirect) or a 404 page
    const loginHeading = page.getByRole('heading', {
      name: 'Sign in to VC Console'
    })
    const notFoundText = page.getByText('404')
    const notFoundAlt = page.getByText('Not Found')

    const isLogin = await loginHeading.isVisible().catch(() => false)
    const is404 = await notFoundText.isVisible().catch(() => false)
    const isNotFound = await notFoundAlt.isVisible().catch(() => false)

    expect(isLogin || is404 || isNotFound).toBeTruthy()
  })

  test('no unhandled JS errors during rapid navigation', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    // Rapidly navigate through multiple routes
    await page.goto('/dashboard')
    await page.goto('/compute/instances')
    await page.goto('/network/networks')
    await page.goto('/storage/volumes')
    await page.goto('/')

    await page.waitForLoadState('networkidle')

    const realErrors = errors.filter((e) => !e.includes('ResizeObserver'))
    expect(realErrors).toHaveLength(0)
  })

  test('page responds within reasonable time', async ({ page }) => {
    const start = Date.now()
    await page.goto('/')
    await page.waitForLoadState('domcontentloaded')
    const loadTime = Date.now() - start

    // Page should load within 10 seconds even on slow CI
    expect(loadTime).toBeLessThan(10000)
  })

  test('viewport renders correctly on mobile', async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 })
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()

    // Login form should still be visible on mobile
    await expect(page.getByLabel('Username')).toBeVisible()
    await expect(page.getByLabel('Password')).toBeVisible()
  })

  test('viewport renders correctly on tablet', async ({ page }) => {
    await page.setViewportSize({ width: 768, height: 1024 })
    await page.goto('/')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })
})
