import { test, expect } from '@playwright/test'

/**
 * E2E tests verifying that authenticated routes (compute pages)
 * properly redirect unauthenticated users and that the login → error
 * flow works correctly for compute-related pages.
 */
test.describe('Compute Pages', () => {
  test('instances route redirects to login', async ({ page }) => {
    await page.goto('/compute/instances')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('SSH keys route redirects to login', async ({ page }) => {
    await page.goto('/compute/ssh-keys')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('deep nested compute route redirects gracefully', async ({ page }) => {
    await page.goto('/compute/instances/some-uuid')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('no JavaScript errors on compute redirect', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/compute/instances')
    await page.waitForLoadState('networkidle')

    const realErrors = errors.filter((e) => !e.includes('ResizeObserver'))
    expect(realErrors).toHaveLength(0)
  })
})
