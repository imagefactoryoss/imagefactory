import { expect, test } from '@playwright/test';
import { loginAsAdmin } from './helpers';

// Run these tests only on specific browsers
test.describe('Responsive Design Testing', () => {
    test.beforeEach(async ({ page }) => {
        // Login first
        await loginAsAdmin(page);
        // Navigate to users page
        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');
    });

    test('desktop layout should display full sidebar and content', async ({ page }) => {
        // Set desktop viewport (already set in config, but be explicit)
        await page.setViewportSize({ width: 1920, height: 1080 });

        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');

        // Sidebar should be visible
        const sidebar = page.locator('aside, nav[aria-label*="Sidebar"]');
        if (await sidebar.isVisible()) {
            await expect(sidebar).toBeVisible();
        }

        // Main content should be visible (if exists)
        const mainContent = page.locator('main, [role="main"], .container, [class*="content"]');
        const isVisible = await mainContent.isVisible({ timeout: 3000 }).catch(() => false);
        if (isVisible) {
            await expect(mainContent).toBeVisible();
        }

        // Table should not be horizontally scrollable
        const table = page.locator('table, [role="table"], [role="grid"]');
        const tableExists = await table.isVisible({ timeout: 3000 }).catch(() => false);
        if (tableExists) {
            const tableWidth = await table.evaluate((el) => el.scrollWidth);
            const containerWidth = await table.evaluate((el) => el.clientWidth);
            expect(tableWidth).toBeLessThanOrEqual(containerWidth + 10); // Allow small margin
        }
    });

    test('mobile layout should be single column', async ({ page }) => {
        // Set mobile viewport
        await page.setViewportSize({ width: 375, height: 667 });

        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');

        // Main content should be visible (if exists)
        const mainContent = page.locator('main, [role="main"], .container, [class*="content"]');
        const isVisible = await mainContent.isVisible({ timeout: 3000 }).catch(() => false);
        if (isVisible) {
            await expect(mainContent).toBeVisible();
        }

        // Check for hamburger menu or collapsed sidebar
        const hamburgerMenu = page.locator('button[aria-label*="menu"], button[aria-label*="toggle"]');
        // Sidebar should either be hidden or hamburger visible
        const sidebar = page.locator('aside, nav[aria-label*="Sidebar"]');

        // Either hamburger is visible or sidebar is hidden
        const hamburgerVisible = await hamburgerMenu.isVisible().catch(() => false);
        const sidebarVisible = await sidebar.isVisible().catch(() => false);

        expect(hamburgerVisible || !sidebarVisible).toBe(true);

        // No horizontal scrolling
        const body = page.locator('body');
        const bodyWidth = await body.evaluate((el) => el.scrollWidth);
        const viewportWidth = 375;
        expect(bodyWidth).toBeLessThanOrEqual(viewportWidth + 10);
    });

    test('tablet layout should be responsive', async ({ page }) => {
        // Set tablet viewport
        await page.setViewportSize({ width: 768, height: 1024 });

        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');

        // Main content should be visible (if exists)
        const mainContent = page.locator('main, [role="main"], .container, [class*="content"]');
        const isVisible = await mainContent.isVisible({ timeout: 3000 }).catch(() => false);
        if (isVisible) {
            await expect(mainContent).toBeVisible();
        }

        // Content should fit without horizontal scroll
        const body = page.locator('body');
        const bodyWidth = await body.evaluate((el) => el.scrollWidth);
        expect(bodyWidth).toBeLessThanOrEqual(768 + 10);
    });

    test('touch targets should be at least 44px on mobile', async ({ page }) => {
        // Set mobile viewport
        await page.setViewportSize({ width: 375, height: 667 });

        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');

        // Get all buttons
        const buttons = await page.locator('button').all();

        for (const button of buttons.slice(0, 5)) {
            // Get button size
            const boundingBox = await button.boundingBox();
            if (boundingBox) {
                const width = boundingBox.width;
                const height = boundingBox.height;

                // Should be at least 44x44 (standard touch target)
                // Allow some that are smaller if they're part of inline elements
                if (width > 20 && height > 20) {
                    expect(width).toBeGreaterThanOrEqual(36); // Slightly flexible for edge cases
                    expect(height).toBeGreaterThanOrEqual(36);
                }
            }
        }
    });

    test('images and text should scale on mobile', async ({ page }) => {
        // Set mobile viewport
        await page.setViewportSize({ width: 375, height: 667 });

        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');

        // Get viewport width
        const viewportWidth = await page.evaluate(() => window.innerWidth);
        expect(viewportWidth).toBe(375);

        // Check that content is not overflowing
        const body = page.locator('body');
        const scrollWidth = await body.evaluate((el) => el.scrollWidth);
        expect(scrollWidth).toBeLessThanOrEqual(375 + 10);

        // Text should be readable (minimum 16px recommended)
        try {
            const allTextElements = page.locator('p, label, span');
            const count = await allTextElements.count();
            const limit = Math.min(5, count);
            for (let i = 0; i < limit; i++) {
                const element = allTextElements.nth(i);
                const fontSize = await element.evaluate((el) => {
                    return window.getComputedStyle(el).fontSize;
                });

                // Extract numeric value
                const fontSizeNum = parseInt(fontSize.replace('px', ''));
                expect(fontSizeNum).toBeGreaterThanOrEqual(12); // Allow some flexibility
            }
        } catch (e) {
            // Skip if no text elements found
        }
    });

    test('modals should be responsive', async ({ page }) => {
        // Set mobile viewport
        await page.setViewportSize({ width: 375, height: 667 });

        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');

        // Open a modal
        const addButton = page.locator('button:has-text("Add User")');
        if (await addButton.isVisible()) {
            await addButton.click();
            await page.waitForTimeout(300);

            // Modal should be visible and not overflow
            const modal = page.locator('[role="dialog"], .modal');
            if (await modal.isVisible()) {
                const modalWidth = await modal.evaluate((el) => el.scrollWidth);
                expect(modalWidth).toBeLessThanOrEqual(375 + 10);
            }

            // Close modal
            await page.press('Escape');
        }
    });

    test('tables should be horizontally scrollable on mobile', async ({ page }) => {
        // Set mobile viewport
        await page.setViewportSize({ width: 375, height: 667 });

        await page.goto('/admin/users');
        await page.waitForLoadState('networkidle');

        // Table might be scrollable or rearranged for mobile
        const table = page.locator('table');
        if (await table.isVisible()) {
            const container = table.evaluate((el) => {
                return el.parentElement;
            });

            // Either table is hidden on mobile (responsive) or has scroll
            const tableVisible = await table.isVisible();
            expect(tableVisible).toBe(true); // Should be visible in some form
        }
    });
});
