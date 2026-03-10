import { Page, expect } from '@playwright/test';

/**
 * Test Helper Functions for E2E Tests
 * Provides common functionality for authentication, navigation, and data creation
 */

/**
 * Login as admin user
 * This will handle LDAP or local authentication
 */
export async function loginAsAdmin(page: Page) {
    // Navigate to login page
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    // Check if already logged in
    const url = page.url();
    if (url.includes('/admin') || url.includes('/dashboard')) {
        // Already logged in
        return;
    }

    // If LDAP is configured, we need to handle LDAP login
    // Otherwise, use a test account that exists in the database

    // Try to find email input
    const emailInput = page.locator('input[type="email"], input[placeholder*="email" i]').first();
    const passwordInput = page.locator('input[type="password"], input[placeholder*="password" i]').first();

    // Check if login form is visible
    const isLoginFormVisible = await emailInput.isVisible({ timeout: 5000 }).catch(() => false);

    if (isLoginFormVisible) {
        // Use test credentials from seed data
        // These are LDAP users created in migrations
        await emailInput.fill('admin@imagefactory.local');
        await passwordInput.fill('password');

        // Find and click login button
        const loginButton = page.locator('button[type="submit"], button:has-text("Login"), button:has-text("Sign in")').first();
        await loginButton.click();

        // Wait for redirect to admin or dashboard
        await page.waitForNavigation({ waitUntil: 'networkidle' });

        // Verify we're logged in
        const loggedInUrl = page.url();
        const isLoggedIn = loggedInUrl.includes('/admin') || loggedInUrl.includes('/dashboard');
        expect(isLoggedIn).toBe(true);
    }
}

/**
 * Navigate to admin users page
 */
export async function goToUsersPage(page: Page) {
    // Ensure we're logged in
    const url = page.url();
    if (!url.includes('/admin')) {
        await loginAsAdmin(page);
    }

    // Navigate to users page
    await page.goto('/admin/users');
    await page.waitForLoadState('networkidle');

    // Wait for users table or list to appear
    const usersList = page.locator('table, [role="table"], tbody').first();
    await usersList.waitFor({ timeout: 5000 }).catch(() => {
        // If table not found, page might be loading
        // Give it more time
    });
}

/**
 * Navigate to admin tenants page
 */
export async function goToTenantsPage(page: Page) {
    const url = page.url();
    if (!url.includes('/admin')) {
        await loginAsAdmin(page);
    }

    await page.goto('/admin/tenants');
    await page.waitForLoadState('networkidle');

    // Wait for tenants table or list
    const tenantsList = page.locator('table, [role="table"], tbody').first();
    await tenantsList.waitFor({ timeout: 5000 }).catch(() => {
        // OK if not found, give it more time
    });
}

/**
 * Navigate to admin roles page
 */
export async function goToRolesPage(page: Page) {
    const url = page.url();
    if (!url.includes('/admin')) {
        await loginAsAdmin(page);
    }

    await page.goto('/admin/access/roles');
    await page.waitForLoadState('networkidle');

    // Wait for roles grid or list
    const rolesList = page.locator('[role="article"], .grid, [class*="grid"]').first();
    await rolesList.waitFor({ timeout: 5000 }).catch(() => {
        // OK if not found
    });
}

/**
 * Create a test user via API
 */
export async function createTestUser(page: Page, data: {
    email: string;
    firstName: string;
    lastName: string;
    password: string;
    status?: string;
}) {
    const response = await page.request.post('http://localhost:8080/api/v1/users', {
        data: {
            email: data.email,
            first_name: data.firstName,
            last_name: data.lastName,
            password: data.password,
            status: data.status || 'active',
        },
        headers: {
            'Content-Type': 'application/json',
        },
    });

    if (!response.ok()) {
        throw new Error(`Failed to create user: ${response.status()}`);
    }

    return response.json();
}

/**
 * Create a test tenant via API
 */
export async function createTestTenant(page: Page, data: {
    name: string;
    slug: string;
    description?: string;
    status?: string;
}) {
    const response = await page.request.post('http://localhost:8080/api/v1/tenants', {
        data: {
            name: data.name,
            slug: data.slug,
            description: data.description || '',
            status: data.status || 'active',
        },
        headers: {
            'Content-Type': 'application/json',
        },
    });

    if (!response.ok()) {
        throw new Error(`Failed to create tenant: ${response.status()}`);
    }

    return response.json();
}

/**
 * Create a test role via API
 */
export async function createTestRole(page: Page, data: {
    name: string;
    description?: string;
    permissions: string[];
}) {
    const response = await page.request.post('http://localhost:8080/api/v1/roles', {
        data: {
            name: data.name,
            description: data.description || '',
            permissions: data.permissions,
        },
        headers: {
            'Content-Type': 'application/json',
        },
    });

    if (!response.ok()) {
        throw new Error(`Failed to create role: ${response.status()}`);
    }

    return response.json();
}

/**
 * Wait for element with flexible selectors
 */
export async function waitForElement(page: Page, selectors: string[]) {
    for (const selector of selectors) {
        try {
            const element = page.locator(selector).first();
            await element.waitFor({ timeout: 3000 });
            return element;
        } catch (e) {
            // Try next selector
        }
    }

    throw new Error(`Could not find any element with selectors: ${selectors.join(', ')}`);
}

/**
 * Get user table rows
 */
export async function getUserTableRows(page: Page) {
    // Try multiple possible table selectors
    const tableSelectors = [
        'table tbody tr',
        'tbody tr',
        '[role="row"]',
        '.user-row',
        'tr[data-testid*="user"]',
    ];

    for (const selector of tableSelectors) {
        const rows = page.locator(selector);
        const count = await rows.count().catch(() => 0);
        if (count > 0) {
            return rows;
        }
    }

    throw new Error('Could not find user table rows');
}

/**
 * Get tenant table rows
 */
export async function getTenantTableRows(page: Page) {
    const tableSelectors = [
        'table tbody tr',
        'tbody tr',
        '[role="row"]',
        '.tenant-row',
        'tr[data-testid*="tenant"]',
    ];

    for (const selector of tableSelectors) {
        const rows = page.locator(selector);
        const count = await rows.count().catch(() => 0);
        if (count > 0) {
            return rows;
        }
    }

    throw new Error('Could not find tenant table rows');
}

/**
 * Get role cards
 */
export async function getRoleCards(page: Page) {
    const cardSelectors = [
        '[role="article"]',
        '.role-card',
        'div[class*="rounded-lg"][class*="shadow"]',
        'div.bg-white.dark\\:bg-slate-950',
    ];

    for (const selector of cardSelectors) {
        const cards = page.locator(selector);
        const count = await cards.count().catch(() => 0);
        if (count > 0) {
            return cards;
        }
    }

    throw new Error('Could not find role cards');
}

/**
 * Toggle dark mode
 */
export async function toggleDarkMode(page: Page) {
    // Look for theme toggle button
    const toggleButton = page.locator('button[title*="theme" i], button[aria-label*="theme" i], button:has-text("🌙"), button:has-text("☀️")').first();

    if (await toggleButton.isVisible().catch(() => false)) {
        await toggleButton.click();
        await page.waitForTimeout(500); // Wait for animation
    }
}

/**
 * Verify dark mode is active
 */
export async function isDarkModeActive(page: Page): Promise<boolean> {
    const html = page.locator('html');
    const isDark = await html.evaluate((el) => {
        return el.classList.contains('dark') ||
            getComputedStyle(el).backgroundColor === 'rgb(15, 23, 42)'; // slate-950
    });
    return isDark;
}

/**
 * Click refresh button
 */
export async function clickRefreshButton(page: Page) {
    const refreshButton = page.locator('button[title*="refresh" i], button[aria-label*="refresh" i], button:has-text("↻")').first();

    if (await refreshButton.isVisible().catch(() => false)) {
        await refreshButton.click();
        await page.waitForLoadState('networkidle');
    }
}

/**
 * Verify toast notification
 */
export async function verifyToastNotification(page: Page, message: string | RegExp) {
    const toast = page.locator(`text=${message}`);
    await expect(toast).toBeVisible({ timeout: 5000 });
}

/**
 * Login as an arbitrary user (LDAP or local)
 * - `useLdap` should be true for LDAP users (default for seeded LDAP users)
 */
export async function loginAsUser(page: Page, email: string, password = 'password', useLdap = true) {
    await page.goto('/login');
    await page.waitForLoadState('networkidle');

    const emailInput = page.locator('input[name="email"], input[placeholder*="Email address"]').first();
    const passwordInput = page.locator('input[name="password"]').first();

    await emailInput.fill(email);
    await passwordInput.fill(password);

    if (useLdap) {
        const ldapCheckbox = page.locator('input#use_ldap');
        if (await ldapCheckbox.isVisible().catch(() => false)) {
            await ldapCheckbox.check().catch(() => { });
        }
    }

    const submit = page.locator('button[type="submit"], button:has-text("Sign in"), button:has-text("Login")').first();
    await submit.click();

    // Wait for a navigation away from login (or for a dashboard/admin route)
    await page.waitForNavigation({ waitUntil: 'networkidle' }).catch(() => { });
    const url = page.url();
    expect(url).not.toContain('/login');
}

export default {
    loginAsAdmin,
    loginAsUser,
    goToUsersPage,
    goToTenantsPage,
    goToRolesPage,
    createTestUser,
    createTestTenant,
    createTestRole,
    waitForElement,
    getUserTableRows,
    getTenantTableRows,
    getRoleCards,
    toggleDarkMode,
    isDarkModeActive,
    clickRefreshButton,
    verifyToastNotification,
};

