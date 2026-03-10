import { expect, test } from '@playwright/test';
import { loginAsAdmin, verifyToastNotification } from './helpers';

// E2E: Provider -> Tenant Namespace Prepare flow
test.describe('Provider - Tenant Namespace Prepare (E2E)', () => {
    test.beforeEach(async ({ page }) => {
        await loginAsAdmin(page);
    });

    test('admin can trigger tenant namespace prepare from provider detail and see status', async ({ page }) => {
        // 1) Resolve a tenant context (API requires X-Tenant-ID) then create a Kubernetes provider via API
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
                config: {
                    tekton_install_mode: 'image_factory_installer',
                    tekton_profile_version: 'v1'
                },
                capabilities: ['tekton'],
                bootstrap_mode: 'image_factory_managed',
                credential_scope: 'namespace_admin'
            }
        });

        expect(createResp.ok()).toBeTruthy();
        const body = await createResp.json();
        const providerId = body.provider?.id;
        expect(providerId).toBeTruthy();

        // 2) Open provider detail page
        await page.goto(`/admin/infrastructure/${providerId}`);
        await page.waitForLoadState('networkidle');

        // Ensure the Tenant Namespace Provisioning panel is visible
        const panel = page.locator('text=Tenant Namespace Provisioning').first();
        await expect(panel).toBeVisible({ timeout: 5000 });

        // 3) Ensure a tenant context exists in the header (ContextSwitcher will already show one for seeded admin)
        const contextButton = page.locator('button:has-text("Tenant Context"), button:has-text("Tenant Context •")').first();
        if (await contextButton.isVisible().catch(() => false)) {
            // open the context switcher and pick the first available context if not already set
            await contextButton.click().catch(() => { });
            // click the first context option if it exists (safe no-op if already set)
            const firstContext = page.locator('div[role="dialog"] button').first();
            if (await firstContext.isVisible().catch(() => false)) {
                await firstContext.click().catch(() => { });
                await page.waitForLoadState('networkidle');
            }
        }

        // 4) Click Prepare Tenant Namespace
        const prepareBtn = page.locator('button:has-text("Prepare Tenant Namespace")').first();
        await expect(prepareBtn).toBeVisible({ timeout: 5000 });
        await prepareBtn.click();

        // 5) Verify toast / notification shown
        await verifyToastNotification(page, /Tenant namespace preparation (started|prepared)/i);

        // 6) Open "View Details" and assert status appears in drawer
        const viewDetails = page.locator('button:has-text("View Details")').first();
        await viewDetails.click();

        // Drawer should show a status value (pending/running/succeeded/failed)
        const statusBadge = page.locator('text=Status').locator('..').locator('span').first();
        await expect(statusBadge).toBeVisible({ timeout: 5000 });

        const statusText = (await statusBadge.textContent())?.trim().toLowerCase() || '';
        expect(['not prepared', 'pending', 'running', 'succeeded', 'failed']).toContain(statusText);

        // 7) Validate stream panel and step log are visible in drawer
        await expect(page.locator('text=Live Stream').first()).toBeVisible({ timeout: 5000 });
        const streamState = page.locator('text=connected, text=polling fallback').first();
        await expect(streamState).toBeVisible({ timeout: 5000 });
        await expect(page.locator('text=Provisioning Steps').first()).toBeVisible({ timeout: 5000 });
        await expect(page.locator('text=Tekton API preflight').first()).toBeVisible({ timeout: 5000 });
        await expect(page.locator('text=Runtime RBAC applied').first()).toBeVisible({ timeout: 5000 });
        await expect(page.locator('text=Tekton assets applied').first()).toBeVisible({ timeout: 5000 });

        // Clean up: attempt to delete provider (best-effort)
        await page.request.delete(`http://localhost:8080/api/v1/admin/infrastructure/providers/${providerId}`, { headers: { 'X-Tenant-ID': tenantId } });
    });

    test('build create 409 shows tenant namespace banner with provider deep-link', async ({ page }) => {
        const mockProjectId = '11111111-1111-1111-1111-111111111111';
        const mockProviderId = '22222222-2222-2222-2222-222222222222';

        await page.route('**/api/v1/projects**', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    projects: [{
                        id: mockProjectId,
                        name: 'Mock Project',
                        tenant_id: '33333333-3333-3333-3333-333333333333',
                        status: 'active',
                    }],
                    total_count: 1,
                    limit: 20,
                    offset: 0,
                }),
            });
        });

        await page.route('**/api/v1/settings/tools', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    build_methods: { kaniko: true },
                    sbom_tools: { syft: true, trivy: true },
                    scan_tools: { trivy: true },
                    registry_types: { s3: true, harbor: true },
                    secret_managers: { vault: true },
                }),
            });
        });

        await page.route('**/api/v1/registry-auth**', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({ registry_auth: [] }),
            });
        });

        await page.route('**/api/v1/infrastructure/providers/available', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    providers: [{
                        id: mockProviderId,
                        tenant_id: '33333333-3333-3333-3333-333333333333',
                        is_global: true,
                        provider_type: 'kubernetes',
                        name: 'mock-k8s',
                        display_name: 'Mock Kubernetes',
                        status: 'online',
                        is_schedulable: true,
                        config: { tekton_enabled: true },
                    }],
                }),
            });
        });

        await page.route('**/api/v1/builds/infrastructure-recommendation', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    recommended_infrastructure: 'kubernetes',
                    confidence: 0.95,
                    reason: 'Mock recommendation',
                    alternatives: [],
                    timestamp: new Date().toISOString(),
                }),
            });
        });

        await page.route('**/api/v1/builds', async (route) => {
            if (route.request().method() !== 'POST') {
                await route.fallback();
                return;
            }
            await route.fulfill({
                status: 409,
                contentType: 'application/json',
                body: JSON.stringify({
                    error: 'selected infrastructure provider is not prepared for this tenant (namespace not provisioned yet)',
                    code: 'tenant_namespace_not_prepared',
                    details: {
                        provider_id: mockProviderId,
                        tenant_id: '33333333-3333-3333-3333-333333333333',
                        prepare_status: 'missing',
                        namespace: 'image-factory-33333333',
                    },
                }),
            });
        });

        await page.goto(`/builds/create?projectId=${mockProjectId}`);
        await page.waitForLoadState('networkidle');

        await page.getByPlaceholder('e.g., web-app-production-v1.2.3').fill(`e2e-build-${Date.now()}`);
        await page.getByText('Kaniko - Dockerfile Builds').first().click();
        await page.getByRole('button', { name: 'Next' }).click();

        await page.getByText('Syft').first().click();
        await page.getByText('Trivy').first().click();
        await page.getByText('S3').first().click();
        await page.getByText('HashiCorp Vault').first().click();
        await page.getByRole('button', { name: 'Next' }).click();

        await page.getByPlaceholder('123456789.dkr.ecr.us-east-1.amazonaws.com/my-app').fill('123456789012.dkr.ecr.us-east-1.amazonaws.com/mock-app');
        await page.getByRole('button', { name: /Infrastructure/ }).first().click();
        await page.getByText('Kubernetes Cluster').first().click();
        await page.getByRole('button', { name: 'Next' }).click();

        await page.getByRole('button', { name: 'Create Build' }).first().click();
        await expect(page.getByText('Tenant Namespace Not Prepared')).toBeVisible({ timeout: 5000 });

        // Confirm the banner shows the namespace from the structured 409 details
        await expect(page.getByText('image-factory-33333333')).toBeVisible({ timeout: 5000 });

        const providerLink = page.getByRole('link', { name: 'Open Provider Details' }).first();
        await expect(providerLink).toHaveAttribute('href', `/admin/infrastructure/providers/${mockProviderId}`);
    });
});
