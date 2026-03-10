import { expect, test } from '@playwright/test';
import { loginAsAdmin, verifyToastNotification } from './helpers';

test.describe('Project Management (E2E)', () => {
    test.beforeEach(async ({ page }) => {
        await loginAsAdmin(page);
    });

    test('should list projects or show empty state', async ({ page }) => {
        await page.goto('/projects');
        await page.waitForLoadState('networkidle');

        const rows = page.locator('table tbody tr');
        const count = await rows.count().catch(() => 0);

        if (count > 0) {
            expect(count).toBeGreaterThan(0);
        } else {
            const empty = page.locator('text=No projects yet.').first();
            expect(await empty.isVisible({ timeout: 3000 }).catch(() => false)).toBe(true);
        }
    });

    test('should create a project draft, save basics and show it in the list', async ({ page }) => {
        await page.goto('/projects');
        await page.waitForLoadState('networkidle');

        // Ensure tenant context exists (safe no-op if already set)
        const contextButton = page.locator('button:has-text("Tenant Context"), button:has-text("Tenant Context •"), button:has-text("Tenant")').first();
        if (await contextButton.isVisible().catch(() => false)) {
            await contextButton.click().catch(() => { });
            const firstContext = page.locator('div[role="dialog"] button').first();
            if (await firstContext.isVisible().catch(() => false)) {
                await firstContext.click().catch(() => { });
                await page.waitForLoadState('networkidle');
            }
        }

        // Open create modal
        const createBtn = page.locator('button:has-text("Create Project"), button:has-text("Create First Project")').first();
        await expect(createBtn).toBeVisible({ timeout: 5000 });
        await createBtn.click();

        // Modal should appear and draft should be created
        await expect(page.locator('text=Create Project')).toBeVisible({ timeout: 10000 });
        const nameInput = page.locator('#name').first();
        await expect(nameInput).toBeVisible({ timeout: 5000 });

        const projectName = `e2e-project-${Date.now()}`;
        await nameInput.fill(projectName);

        const slugInput = page.locator('#slug').first();
        expect((await slugInput.inputValue()).length).toBeGreaterThan(0);

        // Save basics (this will finalize the draft in the create flow)
        const saveBtn = page.locator('button:has-text("Save Changes")').first();
        await expect(saveBtn).toBeVisible({ timeout: 3000 });
        await saveBtn.click();

        // Expect success toast / saved indicator
        await verifyToastNotification(page, /Project updated successfully/i);

        // Close modal
        const closeBtn = page.locator('button:has-text("Close")').first();
        if (await closeBtn.isVisible().catch(() => false)) {
            await closeBtn.click();
        } else {
            const doneBtn = page.locator('button:has-text("Done")').first();
            if (await doneBtn.isVisible().catch(() => false)) {
                await doneBtn.click();
            }
        }

        // Project should appear in list
        await page.waitForLoadState('networkidle');
        const projectRow = page.locator(`text=${projectName}`).first();
        await expect(projectRow).toBeVisible({ timeout: 5000 });

        // Cleanup (best-effort): remove created project via API
        try {
            const resp = await page.request.get(`http://localhost:8080/api/v1/projects?search=${encodeURIComponent(projectName)}`);
            if (resp.ok()) {
                const body = await resp.json();
                const found = body?.projects?.[0] || body?.data?.[0];
                if (found?.id) {
                    await page.request.delete(`http://localhost:8080/api/v1/projects/${found.id}`);
                }
            }
        } catch (e) {
            // best-effort cleanup only
        }
    });
});
