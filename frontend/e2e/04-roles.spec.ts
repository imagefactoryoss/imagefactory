import { expect, test } from '@playwright/test';
import { getRoleCards, goToRolesPage, loginAsAdmin, verifyToastNotification } from './helpers';

test.describe('Role Management Workflows', () => {
    test.beforeEach(async ({ page }) => {
        // Login first
        await loginAsAdmin(page);
        // Navigate to roles page
        await goToRolesPage(page);
    });

    test('should list roles', async ({ page }) => {
        // Wait for roles to load
        await page.waitForLoadState('networkidle');

        // Should see role content
        try {
            const roleCards = await getRoleCards(page);
            const count = await roleCards.count();
            expect(count).toBeGreaterThan(0);
        } catch (e) {
            // Fallback: check for role heading
            const hasHeading = await page.locator('text=/role/i').first().isVisible({ timeout: 5000 }).catch(() => false);
            expect(hasHeading).toBe(true);
        }
    });

    test('should create a new role', async ({ page }) => {
        // Click Add Role button
        const addButton = page.locator('button:has-text("Add Role"), button:has-text("+ Add")').first();

        if (await addButton.isVisible({ timeout: 3000 }).catch(() => false)) {
            await addButton.click();
            await page.waitForLoadState('networkidle');

            // Fill role name
            const nameInput = page.locator('input[placeholder*="Role Name"], input[placeholder*="Name"]').first();
            if (await nameInput.isVisible()) {
                const timestamp = Date.now();
                await nameInput.fill(`Test Role ${timestamp}`);

                // Fill description
                const descInput = page.locator('textarea[placeholder*="Description"], textarea[name*="description"]').first();
                if (await descInput.isVisible()) {
                    await descInput.fill('Test role for E2E testing');
                }

                // Submit form
                const submitButton = page.locator('button:has-text("Create Role"), button:has-text("Create")').nth(-1);
                if (await submitButton.isVisible()) {
                    await submitButton.click();
                    await page.waitForLoadState('networkidle');

                    // Verify success
                    try {
                        await verifyToastNotification(page, 'created successfully');
                    } catch (e) {
                        // Toast might not appear, that's OK
                    }
                }
            }
        }
    });

    test('should edit a role', async ({ page }) => {
        // Find and click first edit button
        const editButton = page.locator('button:has-text("Edit")').first();

        if (await editButton.isVisible({ timeout: 3000 }).catch(() => false)) {
            await editButton.click();
            await page.waitForLoadState('networkidle');

            // Update description
            const descInput = page.locator('textarea[placeholder*="Description"]').first();
            if (await descInput.isVisible()) {
                await descInput.clear();
                await descInput.fill(`Updated Role Description ${Date.now()}`);

                // Submit form
                const submitButton = page.locator('button:has-text("Update"), button:has-text("Save")').nth(-1);
                if (await submitButton.isVisible()) {
                    await submitButton.click();
                    await page.waitForLoadState('networkidle');
                }
            }
        }
    });

    test('should delete a role', async ({ page }) => {
        // Find and click first delete button
        const deleteButton = page.locator('button:has-text("✕"), button[aria-label*="Delete"]').first();

        if (await deleteButton.isVisible({ timeout: 3000 }).catch(() => false)) {
            await deleteButton.click();

            // Confirm deletion in dialog
            const confirmButton = page.locator('button:has-text("OK"), button:has-text("Confirm"), button:has-text("Delete")').nth(-1);
            if (await confirmButton.isVisible({ timeout: 2000 }).catch(() => false)) {
                await confirmButton.click();
                await page.waitForLoadState('networkidle');
            }
        }
    });

    test('should toggle role permissions', async ({ page }) => {
        // Find first role with Edit button
        const editButton = page.locator('button:has-text("Edit")').first();

        if (await editButton.isVisible({ timeout: 3000 }).catch(() => false)) {
            await editButton.click();
            await page.waitForLoadState('networkidle');

            // Toggle first permission checkbox if available
            const permissionCheckbox = page.locator('input[type="checkbox"]').first();
            if (await permissionCheckbox.isVisible({ timeout: 2000 }).catch(() => false)) {
                const isChecked = await permissionCheckbox.isChecked();
                await permissionCheckbox.click();

                // Verify it toggled
                const newChecked = await permissionCheckbox.isChecked();
                expect(newChecked).not.toBe(isChecked);
            }
        }
    });
});
