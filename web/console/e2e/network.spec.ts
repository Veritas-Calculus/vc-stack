import { test, expect } from '@playwright/test'

/**
 * E2E tests verifying network-related route guards and page rendering.
 */
test.describe('Network Pages', () => {
  test('networks route redirects to login', async ({ page }) => {
    await page.goto('/network/networks')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('VPC route redirects to login', async ({ page }) => {
    await page.goto('/network/vpc')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('security groups route redirects to login', async ({ page }) => {
    await page.goto('/network/security-groups')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('floating IPs route redirects to login', async ({ page }) => {
    await page.goto('/network/floating-ips')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('no JavaScript errors on network redirect', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.goto('/network/networks')
    await page.waitForLoadState('networkidle')

    const realErrors = errors.filter((e) => !e.includes('ResizeObserver'))
    expect(realErrors).toHaveLength(0)
  })
})
