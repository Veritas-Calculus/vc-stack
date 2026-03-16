import { test, expect } from '@playwright/test'

/**
 * E2E tests for the login form: detailed interaction flows including
 * form validation, keyboard navigation, and error display.
 */
test.describe('Login Form Interactions', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test('submit via Enter key works', async ({ page }) => {
    await page.getByLabel('Username').fill('testuser')
    await page.getByLabel('Password').fill('testpass')
    await page.getByLabel('Password').press('Enter')

    // Should attempt login and show error (no backend)
    await expect(page.locator('.text-red-500')).toBeVisible({ timeout: 10000 })
  })

  test('Tab navigates through form elements in order', async ({ page }) => {
    // Focus should start on username
    await expect(page.getByLabel('Username')).toBeFocused()

    // Tab to password
    await page.keyboard.press('Tab')
    await expect(page.getByLabel('Password')).toBeFocused()

    // Tab to sign-in button
    await page.keyboard.press('Tab')
    await expect(page.getByRole('button', { name: 'Sign in' })).toBeFocused()
  })

  test('password field masks input', async ({ page }) => {
    const passwordField = page.getByLabel('Password')
    await expect(passwordField).toHaveAttribute('type', 'password')
  })

  test('form preserves input on failed submission', async ({ page }) => {
    await page.getByLabel('Username').fill('myuser')
    await page.getByLabel('Password').fill('mypass')
    await page.getByRole('button', { name: 'Sign in' }).click()

    // After error, username should still contain the entered value
    await expect(page.locator('.text-red-500')).toBeVisible({ timeout: 10000 })
    await expect(page.getByLabel('Username')).toHaveValue('myuser')
  })

  test('multiple rapid submissions do not cause errors', async ({ page }) => {
    const errors: string[] = []
    page.on('pageerror', (err) => errors.push(err.message))

    await page.getByLabel('Username').fill('user')
    await page.getByLabel('Password').fill('pass')

    // Rapid-fire clicks
    await page.getByRole('button', { name: 'Sign in' }).click()
    await page.getByRole('button', { name: 'Sign in' }).click()
    await page.getByRole('button', { name: 'Sign in' }).click()

    await page.waitForTimeout(2000)
    const realErrors = errors.filter((e) => !e.includes('ResizeObserver'))
    expect(realErrors).toHaveLength(0)
  })
})
