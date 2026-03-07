import { test, expect } from '@playwright/test'

test.describe('Accessibility & SEO', () => {
    test('page has correct title', async ({ page }) => {
        await page.goto('/')
        await expect(page).toHaveTitle(/VC Console/)
    })

    test('login page has proper heading hierarchy', async ({ page }) => {
        await page.goto('/')
        const h1s = page.locator('h1')
        await expect(h1s).toHaveCount(1)
    })

    test('form inputs have associated labels', async ({ page }) => {
        await page.goto('/')
        // Labels must point to the correct inputs
        const usernameLabel = page.locator('label[for="username"]')
        const passwordLabel = page.locator('label[for="password"]')
        await expect(usernameLabel).toBeVisible()
        await expect(passwordLabel).toBeVisible()
    })

    test('interactive elements are keyboard accessible', async ({ page }) => {
        await page.goto('/')
        // Tab through form elements
        await page.keyboard.press('Tab')
        const activeElem = page.locator(':focus')
        // Either username input or some other focusable element should be focused
        await expect(activeElem).toBeVisible()
    })
})

test.describe('Theme & Styling', () => {
    test('page loads without visual errors', async ({ page }) => {
        await page.goto('/')
        // Check no JavaScript errors in console
        const errors: string[] = []
        page.on('pageerror', (err) => errors.push(err.message))
        await page.waitForLoadState('networkidle')
        expect(errors.filter((e) => !e.includes('ResizeObserver'))).toHaveLength(0)
    })

    test('login card is centered on page', async ({ page }) => {
        await page.goto('/')
        const form = page.locator('form')
        const box = await form.boundingBox()
        const viewport = page.viewportSize()
        if (box && viewport) {
            // Form should be roughly centered horizontally
            const centerX = box.x + box.width / 2
            expect(centerX).toBeGreaterThan(viewport.width * 0.3)
            expect(centerX).toBeLessThan(viewport.width * 0.7)
        }
    })
})
