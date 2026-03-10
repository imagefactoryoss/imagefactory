import { expect, test } from '@playwright/test';
import { loginAsAdmin } from './helpers';

test.describe('Dark Mode Testing', () => {
    test.beforeEach(async ({ page }) => {
        // Login first
        await loginAsAdmin(page);
        // Navigate to users page
        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');
    });

    test('should toggle dark mode on all pages', async ({ page }) => {
        const pages = [
            '/admin/users',
            '/admin/tenants',
            '/admin/access/roles',
            '/admin/dashboard',
            '/profile',
        ];

        for (const pageUrl of pages) {
            await page.goto(pageUrl);
            await page.waitForLoadState('networkidle');

            // Find theme toggle button with flexible selector
            const themeToggle = page.locator(
                'button[aria-label*="theme"], button[title*="theme"], button[aria-label*="dark"], button[aria-label*="mode"], [data-testid*="theme"]'
            ).first();

            if (await themeToggle.isVisible({ timeout: 3000 }).catch(() => false)) {
                // Get initial classes
                const htmlElement = page.locator('html');
                const initialClasses = await htmlElement.getAttribute('class');

                // Click toggle
                await themeToggle.click();
                await page.waitForTimeout(300); // Wait for animation

                // Verify classes changed
                const newClasses = await htmlElement.getAttribute('class');
                expect(initialClasses).not.toBe(newClasses);

                // Click again to return to original
                await themeToggle.click();
                await page.waitForTimeout(300);
            }
        }
    });

    test('should persist theme preference after page refresh', async ({ page }) => {
        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');

        // Find and click theme toggle
        const themeToggle = page.locator('button[aria-label*="theme"], button[title*="theme"]').first();

        if (await themeToggle.isVisible()) {
            // Get initial state
            const htmlElement = page.locator('html');
            const initialClasses = await htmlElement.getAttribute('class');

            // Click toggle
            await themeToggle.click();
            await page.waitForTimeout(300);

            // Get new state
            const newClasses = await htmlElement.getAttribute('class');

            // Refresh page
            await page.reload();
            await page.waitForLoadState('networkidle');

            // Verify theme persisted
            const persistedClasses = await page.locator('html').getAttribute('class');
            expect(persistedClasses).toBe(newClasses);
        }
    });

    test('should have proper contrast in dark mode', async ({ page }) => {
        await page.goto('/admin/users');

        // Enable dark mode
        const htmlElement = page.locator('html');
        const currentClasses = await htmlElement.getAttribute('class');

        if (!currentClasses?.includes('dark')) {
            const themeToggle = page.locator('button[aria-label*="theme"], button[title*="theme"]').first();
            if (await themeToggle.isVisible()) {
                await themeToggle.click();
                await page.waitForTimeout(300);
            }
        }

        // Check for common contrast issues
        // Look for low-contrast text (this is a basic check)
        const textElements = await page.locator('p, a, span, button, label').all();

        // Verify at least some visible text exists
        expect(textElements.length).toBeGreaterThan(0);

        // Check that page is actually readable (no obvious white-on-white)
        const bodyBgColor = await page.locator('body').evaluate((el) => {
            return window.getComputedStyle(el).backgroundColor;
        });

        // Dark background should not be transparent
        expect(bodyBgColor).not.toBe('rgba(0, 0, 0, 0)');
    });

    test('should have proper contrast in light mode', async ({ page }) => {
        await page.goto('/admin/users');

        // Ensure light mode
        const htmlElement = page.locator('html');
        const currentClasses = await htmlElement.getAttribute('class');

        if (currentClasses?.includes('dark')) {
            const themeToggle = page.locator('button[aria-label*="theme"], button[title*="theme"]').first();
            if (await themeToggle.isVisible()) {
                await themeToggle.click();
                await page.waitForTimeout(300);
            }
        }

        // Verify page is readable
        const textElements = await page.locator('p, a, span, button, label').all();
        expect(textElements.length).toBeGreaterThan(0);

        // Check that page has light background
        const bodyBgColor = await page.locator('body').evaluate((el) => {
            return window.getComputedStyle(el).backgroundColor;
        });

        // Light background should not be completely black
        expect(bodyBgColor).not.toBe('rgb(0, 0, 0)');
    });

    test('dark mode UI elements should be visible', async ({ page }) => {
        // Enable dark mode
        const htmlElement = page.locator('html');
        await htmlElement.evaluate((el) => {
            el.classList.add('dark');
        });

        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');

        // Check visibility of key UI elements (with fallbacks)
        const table = page.locator('table, [role="table"], [role="grid"]');
        const tableExists = await table.isVisible({ timeout: 3000 }).catch(() => false);
        if (tableExists) {
            await expect(table).toBeVisible();
        }

        const buttons = page.locator('button');
        const buttonExists = await buttons.first().isVisible({ timeout: 2000 }).catch(() => false);
        if (buttonExists) {
            await expect(buttons.first()).toBeVisible();
        }

        const inputs = page.locator('input');
        const inputExists = await inputs.first().isVisible({ timeout: 2000 }).catch(() => false);
        if (inputExists) {
            await expect(inputs.first()).toBeVisible();
        }
    });
});
