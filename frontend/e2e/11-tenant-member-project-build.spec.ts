import { expect, test } from '@playwright/test';
import {
    goToTenantsPage,
    loginAsAdmin,
    loginAsUser,
    verifyToastNotification,
} from './helpers';

test.describe('Tenant → Owner → Member → Project → Build (E2E)', () => {
    const externalTenantSearch = 'Engineering'; // stable external tenant used in other tests
    const ownerEmail = 'alice.johnson@imgfactory.com';
    const memberEmail = 'bob.smith@imgfactory.com';

    test('full lifecycle: admin onboards tenant, owner manages tenant and member, member creates build', async ({ page }) => {
        // 1) Admin onboards a tenant and selects an LDAP user as owner
        await loginAsAdmin(page);
        await goToTenantsPage(page);

        const addBtn = page.locator('button:has-text("Add Tenant"), button:has-text("+ Add")').first();
        await expect(addBtn).toBeVisible({ timeout: 5000 });
        await addBtn.click();

        // Wait for tenant selection modal
        await expect(page.locator('text=/Select Tenant to Onboard/i, text=/Select Tenant/i').first()).toBeVisible({ timeout: 5000 });

        // Search & pick an external tenant (fall back to first if not found)
        const tenantSearch = page.locator('input[placeholder*="Type to search"], input[placeholder*="search"]').first();
        if (await tenantSearch.isVisible().catch(() => false)) {
            await tenantSearch.fill(externalTenantSearch);
            await page.waitForTimeout(400);
        }

        // Prefer the named tenant if visible, otherwise pick the first tenant button
        const tenantButtonByName = page.locator(`button:has-text("${externalTenantSearch}")`).first();
        if (await tenantButtonByName.isVisible().catch(() => false)) {
            await tenantButtonByName.click();
        } else {
            const firstTenantBtn = page.locator('div[role="dialog"] button').first();
            await firstTenantBtn.click();
        }

        // Select LDAP user as tenant admin (owner)
        const adminEmailInput = page.getByLabel('Tenant Administrator Email (LDAP Search)');
        await adminEmailInput.fill('alice'); // trigger LDAP lookup (>=3 chars)
        await page.waitForSelector(`text=${ownerEmail}`, { timeout: 5000 });
        await page.getByText(ownerEmail).click();

        // Create tenant (use defaults for quotas)
        const createBtn = page.getByRole('button', { name: 'Create Tenant' }).first();
        await expect(createBtn).toBeVisible({ timeout: 3000 });
        await createBtn.click();

        // Confirm onboarding success
        await verifyToastNotification(page, /Tenant onboarded successfully/i);

        // Resolve tenant id via API so we can use tenant-scoped URLs later
        const tenantsResp = await page.request.get(`http://localhost:8080/api/v1/tenants?search=${encodeURIComponent(externalTenantSearch)}&limit=1`);
        const tenantsBody = await tenantsResp.json();
        const tenantId = tenantsBody?.data?.[0]?.id;
        expect(tenantId).toBeTruthy();

        // 2) Owner logs in and adds a member to their tenant with role Developer
        // clear session and login as owner (LDAP)
        await page.context().clearCookies();
        await page.evaluate(() => localStorage.clear());
        await loginAsUser(page, ownerEmail, 'password', true);

        // Navigate directly to Members for the newly onboarded tenant
        await page.goto(`/members?tenantId=${tenantId}`);
        await page.waitForLoadState('networkidle');

        // Invite LDAP user (existing user) and add as Developer
        const inviteBtn = page.locator('button:has-text("Invite Member")').first();
        await expect(inviteBtn).toBeVisible({ timeout: 5000 });
        await inviteBtn.click();

        const emailField = page.getByLabel('Email Address *').first();
        await emailField.fill('bob'); // trigger LDAP search
        await page.waitForSelector(`text=${memberEmail}`, { timeout: 5000 });
        await page.getByText(memberEmail).click();

        // Ensure role 'Developer' is selected (or select it)
        const developerRadio = page.getByRole('radio', { name: /Developer/i }).first();
        if (await developerRadio.isVisible().catch(() => false)) {
            await developerRadio.check();
        }

        // Add existing LDAP user to tenant
        const addExistingBtn = page.getByRole('button', { name: 'Add Existing User' }).first();
        await expect(addExistingBtn).toBeVisible({ timeout: 5000 });
        await addExistingBtn.click();

        // Verify member added toast
        await verifyToastNotification(page, new RegExp(memberEmail.split('@')[0], 'i'));

        // 3) Owner creates a project
        await page.goto('/projects');
        await page.waitForLoadState('networkidle');

        const createProjectBtn = page.locator('button:has-text("Create Project"), button:has-text("Create First Project")').first();
        await expect(createProjectBtn).toBeVisible({ timeout: 5000 });
        await createProjectBtn.click();

        const projectName = `e2e-owned-project-${Date.now()}`;
        const nameInput = page.locator('#name').first();
        await expect(nameInput).toBeVisible({ timeout: 5000 });
        await nameInput.fill(projectName);

        const saveBasics = page.locator('button:has-text("Save Changes")').first();
        await saveBasics.click();

        await verifyToastNotification(page, /Project updated successfully/i);

        // Close modal/done
        const doneBtn = page.locator('button:has-text("Done"), button:has-text("Close")').first();
        if (await doneBtn.isVisible().catch(() => false)) await doneBtn.click();

        // Ensure project appears in list and open its detail page
        await page.waitForLoadState('networkidle');
        const projectRow = page.locator(`text=${projectName}`).first();
        await expect(projectRow).toBeVisible({ timeout: 5000 });
        await projectRow.click();
        await page.waitForLoadState('networkidle');

        // Grab projectId from URL
        const projectUrl = page.url();
        const projectIdMatch = projectUrl.match(/\/projects\/([-0-9a-fA-F]{8,})/);
        expect(projectIdMatch).not.toBeNull();
        const projectId = projectIdMatch![1];

        // 4) Owner creates repository auth and registry auth for the project
        // Repository Auth
        await page.getByText('Repository Authentication').first().scrollIntoViewIfNeeded();
        await page.getByRole('button', { name: 'Add Authentication' }).click();
        await page.locator('#repository-auth-name').fill('e2e-repo-auth');
        // Choose Token auth (default) and provide token
        await page.locator('#token').fill('repo-token-123');
        await page.getByRole('button', { name: 'Create Authentication' }).click();
        await verifyToastNotification(page, /Repository authentication created successfully/i);

        // Registry Auth (project-scoped)
        await page.getByText('Registry Authentication').first().scrollIntoViewIfNeeded();
        await page.getByRole('button', { name: 'Add Registry Auth' }).click();

        await page.locator('#registry-auth-name').fill('e2e-registry-auth');
        await page.locator('#registry-host').fill('ghcr.io');
        // token auth type is default
        await page.locator('#registry-token').fill('registry-token-abc');
        await page.getByRole('button', { name: 'Create Authentication' }).click();

        await verifyToastNotification(page, /Registry authentication created successfully/i);

        // 5) Member logs in and creates a build for the project
        await page.context().clearCookies();
        await page.evaluate(() => localStorage.clear());
        await loginAsUser(page, memberEmail, 'password', true);

        // Ensure member has tenant context (navigate to builds create page for project)
        // Stub infrastructure/provider endpoints and build create to keep this test deterministic
        const mockProviderId = '22222222-2222-2222-2222-222222222222';

        await page.route('**/api/v1/infrastructure/providers/available', async (route) => {
            await route.fulfill({
                status: 200,
                contentType: 'application/json',
                body: JSON.stringify({
                    providers: [{
                        id: mockProviderId,
                        tenant_id: tenantId,
                        is_global: true,
                        provider_type: 'kubernetes',
                        name: 'mock-k8s',
                        display_name: 'Mock Kubernetes',
                        status: 'online',
                        is_schedulable: true,
                        config: { tekton_enabled: true },
                    }]
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
                    reason: 'mock',
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
                status: 201,
                contentType: 'application/json',
                body: JSON.stringify({ build: { id: 'mock-build-1', status: 'created' } }),
            });
        });

        // Open build creation page for the project
        await page.goto(`/builds/create?projectId=${projectId}`);
        await page.waitForLoadState('networkidle');

        // Step 1 — name & method
        const buildName = `e2e-build-${Date.now()}`;
        await page.getByPlaceholder('e.g., web-app-production-v1.2.3').fill(buildName);
        await page.getByText('Kaniko - Dockerfile Builds').first().click();
        await page.getByRole('button', { name: 'Next' }).click();

        // Step 2 — tools
        await page.getByText('Syft').first().click();
        await page.getByText('Trivy').first().click();
        await page.getByText('S3').first().click();
        await page.getByText('HashiCorp Vault').first().click();
        await page.getByRole('button', { name: 'Next' }).click();

        // Step 3 — configuration
        // Fill registry repo and select registry auth we created earlier
        await page.getByPlaceholder('123456789.dkr.ecr.us-east-1.amazonaws.com/my-app').fill('ghcr.io/mock-org/mock-app');
        await page.getByRole('button', { name: /Infrastructure/ }).first().click();
        await page.getByText('Kubernetes Cluster').first().click();
        await page.getByRole('button', { name: 'Next' }).click();

        // Step 4 — validation & create
        await page.getByRole('button', { name: 'Create Build' }).first().click();

        // Success (we stubbed the POST) — look for navigation or success toast
        await verifyToastNotification(page, /created/i);
    });
});
