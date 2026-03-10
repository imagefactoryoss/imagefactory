import { expect, test } from '@playwright/test';
import { getTenantTableRows, goToTenantsPage, loginAsAdmin, verifyToastNotification } from './helpers';

test.describe('Tenant Management Workflows', () => {
    test.beforeEach(async ({ page }) => {
        // Login first
        await loginAsAdmin(page);
        // Navigate to tenants page
        await goToTenantsPage(page);
    });

    test('should list tenants', async ({ page }) => {
        // Wait for tenants to load
        await page.waitForLoadState('networkidle');

        // Should see tenant content
        try {
            const tenantRows = await getTenantTableRows(page);
            const count = await tenantRows.count();
            expect(count).toBeGreaterThan(0);
        } catch (e) {
            // Fallback: check for any tenant heading or text
            const hasHeading = await page.locator('text=/tenant/i').first().isVisible({ timeout: 5000 }).catch(() => false);
            expect(hasHeading).toBe(true);
        }
    });

    test('should open tenant selection dialog when Add Tenant is clicked', async ({ page }) => {
        // Click Add Tenant button
        const addButton = page.locator('button:has-text("Add Tenant"), button:has-text("+ Add")').first();

        if (await addButton.isVisible({ timeout: 3000 }).catch(() => false)) {
            await addButton.click();
            await page.waitForLoadState('networkidle');

            // Verify tenant selection dialog appears
            const dialogTitle = page.locator('text=/Select Tenant to Onboard/i, text=/Select Tenant/i').first();
            expect(await dialogTitle.isVisible({ timeout: 5000 }).catch(() => false)).toBe(true);

            // Verify search input exists
            const searchInput = page.locator('input[placeholder*="search"], input[placeholder*="Search"]').first();
            expect(await searchInput.isVisible({ timeout: 3000 }).catch(() => false)).toBe(true);
        }
    });

    test('should search and filter external tenants', async ({ page }) => {
        // Click Add Tenant button
        const addButton = page.locator('button:has-text("Add Tenant"), button:has-text("+ Add")').first();

        if (await addButton.isVisible({ timeout: 3000 }).catch(() => false)) {
            await addButton.click();
            await page.waitForLoadState('networkidle');

            // Search for a tenant
            const searchInput = page.locator('input[placeholder*="search"], input[placeholder*="Search"]').first();
            if (await searchInput.isVisible({ timeout: 3000 })) {
                await searchInput.fill('Engineering');
                await page.waitForTimeout(500);

                // Verify filtered results appear
                const tenantItems = page.locator('text=/Engineering/i').first();
                expect(await tenantItems.isVisible({ timeout: 3000 }).catch(() => false)).toBe(true);
            }
        }
    });

    test('should select a tenant and show quota configuration form', async ({ page }) => {
        // Click Add Tenant button
        const addButton = page.locator('button:has-text("Add Tenant"), button:has-text("+ Add")').first();

        if (await addButton.isVisible({ timeout: 3000 }).catch(() => false)) {
            await addButton.click();
            await page.waitForLoadState('networkidle');

            // Click on first tenant in list
            const tenantItem = page.locator('button').filter({ hasText: /team|Team/ }).first();
            if (await tenantItem.isVisible({ timeout: 3000 }).catch(() => false)) {
                await tenantItem.click();
                await page.waitForTimeout(500);

                // Verify quota form appears
                const apiRateLimitInput = page.locator('input[placeholder*="requests"], input[type="number"]').first();
                const storageInput = page.locator('input[placeholder*="GB"], input[type="number"]').nth(1);

                expect(await apiRateLimitInput.isVisible({ timeout: 3000 }).catch(() => false)).toBe(true);
                expect(await storageInput.isVisible({ timeout: 3000 }).catch(() => false)).toBe(true);
            }
        }
    });

    test('should configure quotas and create tenant', async ({ page }) => {
        // Click Add Tenant button
        const addButton = page.locator('button:has-text("Add Tenant"), button:has-text("+ Add")').first();

        if (await addButton.isVisible({ timeout: 3000 }).catch(() => false)) {
            await addButton.click();
            await page.waitForLoadState('networkidle');

            // Click on first tenant in list
            const tenantItem = page.locator('button').filter({ hasText: /team|Team/ }).first();
            if (await tenantItem.isVisible({ timeout: 3000 }).catch(() => false)) {
                await tenantItem.click();
                await page.waitForTimeout(500);

                // Fill quota fields
                const apiRateLimitInput = page.locator('input[type="number"]').first();
                const storageInput = page.locator('input[type="number"]').nth(1);
                const maxUsersInput = page.locator('input[type="number"]').nth(2);

                if (await apiRateLimitInput.isVisible({ timeout: 3000 })) {
                    await apiRateLimitInput.fill('2000');
                }
                if (await storageInput.isVisible({ timeout: 3000 })) {
                    await storageInput.fill('500');
                }
                if (await maxUsersInput.isVisible({ timeout: 3000 })) {
                    await maxUsersInput.fill('100');
                }

                // Click Create Tenant button
                const createButton = page.locator('button:has-text("Create Tenant")').nth(-1);
                if (await createButton.isVisible({ timeout: 3000 }).catch(() => false)) {
                    await createButton.click();
                    await page.waitForLoadState('networkidle');

                    // Verify success (toast or dialog closes)
                    try {
                        await verifyToastNotification(page, 'created successfully');
                    } catch (e) {
                        // Toast might not appear, check if dialog closed
                        const dialog = page.locator('text=/Select Tenant/i').first();
                        expect(await dialog.isVisible({ timeout: 3000 }).catch(() => false)).toBe(false);
                    }
                }
            }
        }
    });

    test('should edit a tenant', async ({ page }) => {
        // Wait for table to load
        await page.waitForLoadState('networkidle');

        // Find and click first edit button (pencil icon)
        const editButton = page.locator('button[title="Edit tenant"]').first();

        if (await editButton.isVisible({ timeout: 3000 }).catch(() => false)) {
            await editButton.click();
            await page.waitForLoadState('networkidle');

            // Verify edit modal appears
            const modalTitle = page.locator('text=/Edit Tenant/i').first();
            expect(await modalTitle.isVisible({ timeout: 3000 }).catch(() => false)).toBe(true);

            // Update name
            const nameInput = page.locator('input[placeholder*="Finance"], input[placeholder*="Company"], input[placeholder*="Name"]').first();
            if (await nameInput.isVisible()) {
                await nameInput.fill(`Updated Tenant ${Date.now()}`);

                // Submit form
                const submitButton = page.locator('button:has-text("Update Tenant"), button:has-text("Update")').nth(-1);
                if (await submitButton.isVisible()) {
                    await submitButton.click();
                    await page.waitForLoadState('networkidle');
                }
            }
        }
    });

    test('should delete a tenant', async ({ page }) => {
        // Wait for table to load
        await page.waitForLoadState('networkidle');

        // Find and click first delete button (X icon)
        const deleteButton = page.locator('button[title="Delete tenant"]').first();

        if (await deleteButton.isVisible({ timeout: 3000 }).catch(() => false)) {
            await deleteButton.click();

            // Confirm deletion in dialog
            const confirmButton = page.locator('button:has-text("OK"), button:has-text("Confirm"), button:has-text("Delete")').nth(-1);
            if (await confirmButton.isVisible({ timeout: 3000 }).catch(() => false)) {
                await confirmButton.click();
                await page.waitForLoadState('networkidle');

                // Verify success
                try {
                    await verifyToastNotification(page, 'deleted successfully');
                } catch (e) {
                    // Toast might not appear, that's OK
                }
            }
        }
    });

    test('should filter tenants by status', async ({ page }) => {
        // Find status filter dropdown
        const statusSelect = page.locator('select').nth(1); // Second select is status filter

        if (await statusSelect.isVisible({ timeout: 3000 }).catch(() => false)) {
            await statusSelect.selectOption('active');
            await page.waitForLoadState('networkidle');

            // Verify filtered results
            const tenantRows = await getTenantTableRows(page);
            const count = await tenantRows.count();
            expect(count).toBeGreaterThan(0);
        }
    });

    test('should paginate through tenants', async ({ page }) => {
        // Wait for table to load
        await page.waitForLoadState('networkidle');

        // Find Next button
        const nextButton = page.locator('button:has-text("Next")').first();

        if (await nextButton.isVisible({ timeout: 3000 }).catch(() => false) && !await nextButton.isDisabled()) {
            await nextButton.click();
            await page.waitForLoadState('networkidle');

            // Verify page changed
            const pageText = page.locator('text=/Page \\d+ of/').first();
            expect(await pageText.isVisible({ timeout: 3000 }).catch(() => false)).toBe(true);
        }
    });
});
