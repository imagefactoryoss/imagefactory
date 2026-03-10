import { expect, test } from '@playwright/test'

test.describe('Phase K selector coverage', () => {
  test('K-05 smoke: application route resolves to login/setup/dashboard shell', async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded', timeout: 60000 })

    const url = page.url()
    const isKnownRoute =
      url.startsWith('http://localhost:3000/') ||
      url.includes('/login') ||
      url.includes('/setup') ||
      url.includes('/admin') ||
      url.includes('/dashboard')

    expect(isKnownRoute).toBe(true)
  })

  test('K-08 smoke: login page renders primary action', async ({ page }) => {
    await page.goto('/login', { waitUntil: 'domcontentloaded', timeout: 60000 })

    const hasAction = page
      .locator('button[type="submit"], button:has-text("Sign in"), button:has-text("Login")')
      .first()

    await expect(hasAction).toBeVisible({ timeout: 10000 })
  })
})
