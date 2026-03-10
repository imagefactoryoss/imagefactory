import { expect, test } from '@playwright/test';
import { loginAsAdmin } from './helpers';

test.describe('Admin — Infrastructure Providers (E2E)', () => {
    test.beforeEach(async ({ page }) => {
        await loginAsAdmin(page);
    });

    test('should show provider created via API in the providers list and allow editing', async ({ page }) => {
        // 1) Create provider via API (admin endpoint)
        const tenantsResp = await page.request.get('http://localhost:8080/api/v1/tenants?limit=1');
        const tenantList = await tenantsResp.json();
        const tenantId = tenantList?.data?.[0]?.id;
        if (!tenantId) throw new Error('no tenant available for test');

        const providerName = `e2e-k8s-provider-${Date.now()}`;
        const createResp = await page.request.post('http://localhost:8080/api/v1/admin/infrastructure/providers', {
            headers: { 'X-Tenant-ID': tenantId },
            data: {
                provider_type: 'kubernetes',
                name: providerName,
                display_name: providerName,
                config: { tekton_install_mode: 'image_factory_installer', tekton_profile_version: 'v1' },
                capabilities: ['tekton'],
                bootstrap_mode: 'image_factory_managed',
                credential_scope: 'namespace_admin'
            }
        });
        expect(createResp.ok()).toBeTruthy();
        const body = await createResp.json();
        const providerId = body.provider?.id;
        expect(providerId).toBeTruthy();

        // 2) Open providers page and verify provider is visible
        await page.goto('/admin/infrastructure');
        await page.waitForLoadState('networkidle');
        const card = page.locator(`div:has-text("${providerName}")`).first();
        await expect(card).toBeVisible({ timeout: 5000 });

        // 3) Click Edit on the provider card
        const editBtn = page.locator(`div:has-text("${providerName}") button[title="Edit Provider"]`).first();
        await expect(editBtn).toBeVisible({ timeout: 3000 });
        await editBtn.click();

        // 4) Update the Display Name in the modal
        await expect(page.locator('text=Edit Provider')).toBeVisible({ timeout: 5000 });
        const newDisplay = `${providerName} (updated)`;
        await page.getByLabel('Display Name').fill(newDisplay);

        // 5) Submit the form
        await page.getByRole('button', { name: 'Update Provider' }).click();

        // 6) Confirm success message appears and updated name is shown
        await expect(page.locator('text=Provider updated successfully!')).toBeVisible({ timeout: 5000 });
        await expect(page.locator(`div:has-text("${newDisplay}")`).first()).toBeVisible({ timeout: 5000 });

        // Cleanup
        await page.request.delete(`http://localhost:8080/api/v1/admin/infrastructure/providers/${providerId}`, { headers: { 'X-Tenant-ID': tenantId } });
    });

    test('should delete provider via UI confirmation', async ({ page }) => {
        // Seed provider via API
        const tenantsResp = await page.request.get('http://localhost:8080/api/v1/tenants?limit=1');
        const tenantList = await tenantsResp.json();
        const tenantId = tenantList?.data?.[0]?.id;
        if (!tenantId) throw new Error('no tenant available for test');

        const providerName = `e2e-k8s-provider-delete-${Date.now()}`;
        const createResp = await page.request.post('http://localhost:8080/api/v1/admin/infrastructure/providers', {
            headers: { 'X-Tenant-ID': tenantId },
            data: {
                provider_type: 'kubernetes',
                name: providerName,
                display_name: providerName,
                config: { tekton_install_mode: 'image_factory_installer', tekton_profile_version: 'v1' },
                capabilities: ['tekton'],
                bootstrap_mode: 'image_factory_managed',
                credential_scope: 'namespace_admin'
            }
        });
        expect(createResp.ok()).toBeTruthy();
        const body = await createResp.json();
        const providerId = body.provider?.id;
        expect(providerId).toBeTruthy();

        // Open providers page
        await page.goto('/admin/infrastructure');
        await page.waitForLoadState('networkidle');

        // Click Delete and accept confirm dialog
        const deleteBtn = page.locator(`div:has-text("${providerName}") button[title="Delete Provider"]`).first();
        await expect(deleteBtn).toBeVisible({ timeout: 5000 });

        page.once('dialog', dialog => dialog.accept());
        await deleteBtn.click();

        // Expect the provider to be removed from list
        await expect(page.locator(`text=${providerName}`)).toHaveCount(0, { timeout: 5000 });
    });
});
