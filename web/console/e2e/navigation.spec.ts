import { test, expect } from '@playwright/test'

test.describe('Navigation & Guards', () => {
  test('unauthenticated user is redirected to login', async ({ page }) => {
    await page.goto('/dashboard')
    // Should redirect to login page
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('unauthenticated user cannot access instances page', async ({ page }) => {
    await page.goto('/compute/instances')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('unauthenticated user cannot access networks page', async ({ page }) => {
    await page.goto('/network/networks')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('unauthenticated user cannot access storage page', async ({ page }) => {
    await page.goto('/storage/volumes')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('unauthenticated user cannot access IAM page', async ({ page }) => {
    await page.goto('/iam/users')
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })
})
