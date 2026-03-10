import { expect, test } from "@playwright/test";
import { loginAsAdmin, verifyToastNotification } from "./helpers";

type AuthContext = {
  token: string;
  tenantId: string;
};

async function readAuthContext(page: any): Promise<AuthContext> {
  const data = await page.evaluate(() => {
    const authRaw = localStorage.getItem("auth-storage");
    const tenantRaw = localStorage.getItem("tenant-context-store");
    const auth = authRaw ? JSON.parse(authRaw) : {};
    const tenant = tenantRaw ? JSON.parse(tenantRaw) : {};
    const token = auth?.state?.token || "";
    const tenantId = tenant?.state?.selectedTenantId || "";
    return { token, tenantId };
  });
  return data as AuthContext;
}

test.describe("Build Notifications (E2E)", () => {
  test.beforeEach(async ({ page }) => {
    await loginAsAdmin(page);
  });

  test("project trigger matrix persists and notification center renders in-app feed", async ({
    page,
  }) => {
    await page.goto("/projects");
    await page.waitForLoadState("networkidle");

    let targetProjectName = `e2e-notify-${Date.now()}`;
    const createBtn = page
      .locator(
        'button:has-text("Create Project"), button:has-text("Create First Project")',
      )
      .first();

    if (await createBtn.isVisible().catch(() => false)) {
      await createBtn.click();
      await expect(page.locator("text=Create Project")).toBeVisible({
        timeout: 10000,
      });
      await page.locator("#name").first().fill(targetProjectName);
      await page
        .locator('button:has-text("Save Changes")')
        .first()
        .click();
      await verifyToastNotification(page, /Project updated successfully/i);
      await page.locator('button:has-text("Close")').first().click().catch(() => {});
      await page.waitForLoadState("networkidle");
    } else {
      const firstProjectCell = page.locator("table tbody tr td").first();
      await expect(firstProjectCell).toBeVisible({ timeout: 8000 });
      targetProjectName = (await firstProjectCell.textContent())?.trim() || targetProjectName;
    }

    const projectRow = page.locator(`text=${targetProjectName}`).first();
    await expect(projectRow).toBeVisible({ timeout: 10000 });
    await projectRow.click();
    await page.waitForLoadState("networkidle");

    const projectUrl = page.url();
    const projectIdMatch = projectUrl.match(/\/projects\/([-0-9a-fA-F]{8,})/);
    expect(projectIdMatch).not.toBeNull();
    const projectId = projectIdMatch![1];

    await page.locator('button:has-text("Notifications")').first().click();
    await expect(
      page.locator("text=Build Notification Trigger Matrix"),
    ).toBeVisible({ timeout: 10000 });

    const bn004Row = page.locator("tr", { hasText: "BN-004" }).first();
    await expect(bn004Row).toBeVisible({ timeout: 8000 });

    const enabledCheckbox = bn004Row.locator('input[type="checkbox"]').first();
    if (!(await enabledCheckbox.isChecked())) {
      await enabledCheckbox.check();
    }

    const inAppCheckbox = bn004Row.locator('label:has-text("In-app") input[type="checkbox"]');
    if (!(await inAppCheckbox.isChecked())) {
      await inAppCheckbox.check();
    }
    const emailCheckbox = bn004Row.locator('label:has-text("Email") input[type="checkbox"]');
    if (!(await emailCheckbox.isChecked())) {
      await emailCheckbox.check();
    }

    await bn004Row.locator("select").first().selectOption("custom_users");
    const addButtons = bn004Row.locator('button:has-text("Add")');
    if ((await addButtons.count()) > 0) {
      await addButtons.first().click();
    }

    await page.locator('button:has-text("Save Preferences")').click();
    await verifyToastNotification(page, /saved/i);

    const { token, tenantId } = await readAuthContext(page);
    expect(token.length).toBeGreaterThan(10);
    expect(tenantId.length).toBeGreaterThan(0);

    const triggersResp = await page.request.get(
      `http://localhost:8080/api/v1/projects/${projectId}/notification-triggers`,
      {
        headers: {
          Authorization: `Bearer ${token}`,
          "X-Tenant-ID": tenantId,
        },
      },
    );
    expect(triggersResp.ok()).toBeTruthy();
    const triggerBody = await triggersResp.json();
    const bn004 = (triggerBody?.preferences || []).find(
      (pref: any) => pref.trigger_id === "BN-004",
    );
    expect(bn004).toBeTruthy();
    expect(bn004.enabled).toBeTruthy();
    expect(bn004.channels || []).toEqual(
      expect.arrayContaining(["in_app", "email"]),
    );
    expect(bn004.recipient_policy).toBe("custom_users");

    const mockBuildId = "00000000-0000-0000-0000-000000000123";
    await page.route("**/api/v1/notifications/unread-count**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ unread_count: 1 }),
      });
    });
    await page.route("**/api/v1/notifications?**", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          notifications: [
            {
              id: "00000000-0000-0000-0000-000000000900",
              title: "Build Failed",
              message: "Build failed: simulated for E2E",
              notification_type: "build_failed",
              is_read: false,
              created_at: new Date().toISOString(),
              related_resource_type: "build",
              related_resource_id: mockBuildId,
            },
          ],
          total: 1,
          unread_count: 1,
        }),
      });
    });
    await page.route("**/api/v1/notifications/*/read", async (route) => {
      await route.fulfill({ status: 200, contentType: "application/json", body: "{}" });
    });

    await page.getByLabel("Open notifications").click();
    await expect(page.locator("text=Build Failed")).toBeVisible({ timeout: 5000 });
    await page.locator("button", { hasText: "Build Failed" }).first().click();
    await expect(page).toHaveURL(new RegExp(`/builds/${mockBuildId}$`), {
      timeout: 8000,
    });
  });
});

