import { test, expect } from '@playwright/test'

test.describe('Login Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test('shows login form with title', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Sign in to VC Console' })).toBeVisible()
  })

  test('has username and password fields', async ({ page }) => {
    await expect(page.getByLabel('Username')).toBeVisible()
    await expect(page.getByLabel('Password')).toBeVisible()
  })

  test('has a Sign in button', async ({ page }) => {
    await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible()
  })

  test('shows error on empty submission', async ({ page }) => {
    await page.getByRole('button', { name: 'Sign in' }).click()
    await expect(page.getByText('Please enter username and password')).toBeVisible()
  })

  test('shows error on invalid credentials', async ({ page }) => {
    await page.getByLabel('Username').fill('baduser')
    await page.getByLabel('Password').fill('badpass')
    await page.getByRole('button', { name: 'Sign in' }).click()

    // Should show an error (could be network error or 401)
    await expect(page.locator('.text-red-500')).toBeVisible({ timeout: 10000 })
  })

  test('username field is auto-focused', async ({ page }) => {
    const username = page.getByLabel('Username')
    await expect(username).toBeFocused()
  })

  test('logo is visible', async ({ page }) => {
    await expect(page.getByRole('img', { name: 'logo' })).toBeVisible()
  })
})
