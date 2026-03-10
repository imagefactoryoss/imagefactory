import { expect, test } from '@playwright/test'
import { loginAsUser } from './helpers'

type AuthContext = {
  token: string
  tenantId: string
}

const API_BASE = 'http://localhost:8080/api/v1'

async function readAuthContext(page: any): Promise<AuthContext> {
  return page.evaluate(() => {
    const authRaw = localStorage.getItem('auth-storage')
    const tenantRaw = localStorage.getItem('tenant-context-store')
    const auth = authRaw ? JSON.parse(authRaw) : {}
    const tenant = tenantRaw ? JSON.parse(tenantRaw) : {}
    return {
      token: auth?.state?.token || '',
      tenantId: tenant?.state?.selectedTenantId || '',
    }
  }) as Promise<AuthContext>
}

async function resetSession(page: any) {
  await page.context().clearCookies()
  await page.evaluate(() => localStorage.clear())
}

async function apiPost(page: any, auth: AuthContext, path: string, data?: unknown) {
  return page.request.post(`${API_BASE}${path}`, {
    data,
    headers: {
      Authorization: `Bearer ${auth.token}`,
      'X-Tenant-ID': auth.tenantId,
    },
  })
}

test.describe('Quarantine Happy Path (J-01)', () => {
  test('tenant submit -> reviewer approve/release -> tenant consume to build wizard', async ({ page }) => {
    const runKey = `${Date.now()}`
    const eprRecordId = `EPR-J01-${runKey}`
    const sourceImageRef = `library/alpine:j01-${runKey}`
    const sourceRegistry = 'registry-1.docker.io'
    const projectName = `j01-project-${runKey}`
    const projectSlug = `j01-project-${runKey}`.toLowerCase()

    // 1) Tenant prepares EPR + quarantine import request via authenticated API
    await loginAsUser(page, 'alice.johnson@imagefactory.local', 'password', true)
    const tenantAuth = await readAuthContext(page)
    expect(tenantAuth.token.length).toBeGreaterThan(10)
    expect(tenantAuth.tenantId.length).toBeGreaterThan(10)

    const projectResp = await apiPost(page, tenantAuth, '/projects', {
      tenant_id: tenantAuth.tenantId,
      name: projectName,
      slug: projectSlug,
      visibility: 'private',
    })
    expect(projectResp.ok()).toBeTruthy()
    const projectPayload = await projectResp.json()
    const projectId = projectPayload?.data?.id
    expect(projectId).toBeTruthy()

    const eprResp = await apiPost(page, tenantAuth, '/epr/registration-requests', {
      epr_record_id: eprRecordId,
      product_name: `J01 Product ${runKey}`,
      technology_name: 'Alpine',
      business_justification: 'J-01 e2e happy path automation',
    })
    expect(eprResp.ok()).toBeTruthy()
    const eprBody = await eprResp.json()
    const eprRequestId = eprBody?.data?.id
    expect(eprRequestId).toBeTruthy()

    await resetSession(page)

    // 2) Security reviewer approves EPR
    await loginAsUser(page, 'bob.smith@imagefactory.local', 'password', true)
    const reviewerAuth = await readAuthContext(page)
    expect(reviewerAuth.token.length).toBeGreaterThan(10)

    const approveEprResp = await apiPost(page, reviewerAuth, `/admin/epr/registration-requests/${eprRequestId}/approve`, {
      reason: 'Approved by J-01 automation',
    })
    expect(approveEprResp.ok()).toBeTruthy()

    await resetSession(page)

    // 3) Tenant submits quarantine request
    await loginAsUser(page, 'alice.johnson@imagefactory.local', 'password', true)
    const tenantAuthAfterApproval = await readAuthContext(page)
    const importResp = await apiPost(page, tenantAuthAfterApproval, '/images/import-requests', {
      epr_record_id: eprRecordId,
      source_registry: sourceRegistry,
      source_image_ref: sourceImageRef,
    })
    expect(importResp.ok()).toBeTruthy()
    const importBody = await importResp.json()
    const importRequestId = importBody?.data?.id
    expect(importRequestId).toBeTruthy()

    await resetSession(page)

    // 4) Security reviewer approves + releases quarantine request
    await loginAsUser(page, 'bob.smith@imagefactory.local', 'password', true)
    const reviewerAuthAfterImport = await readAuthContext(page)
    const approveImportResp = await apiPost(page, reviewerAuthAfterImport, `/images/import-requests/${importRequestId}/approve`)
    expect(approveImportResp.ok()).toBeTruthy()

    const releaseResp = await apiPost(page, reviewerAuthAfterImport, `/images/import-requests/${importRequestId}/release`)
    expect(releaseResp.ok()).toBeTruthy()

    await resetSession(page)

    // 5) Tenant consumes released artifact through UI drawer and lands in build wizard with prefill
    await loginAsUser(page, 'alice.johnson@imagefactory.local', 'password', true)
    await page.goto('/quarantine/releases')
    await page.waitForLoadState('networkidle')

    await expect(page.locator(`text=${sourceImageRef}`).first()).toBeVisible({ timeout: 15000 })
    await expect(page.locator('span:has-text("Ready")').first()).toBeVisible({ timeout: 15000 })

    const row = page.locator('tr', { hasText: sourceImageRef }).first()
    await row.getByRole('button', { name: 'Use in Project' }).click()

    await expect(page.locator('text=Use Released Artifact in Project').first()).toBeVisible({ timeout: 10000 })
    await page.selectOption('select', projectId)
    await page.getByRole('button', { name: 'Open Build Wizard' }).click()

    await expect(page).toHaveURL(new RegExp(`/builds/new\\?`), { timeout: 10000 })
    await expect(page).toHaveURL(new RegExp(`projectId=${projectId}`), { timeout: 10000 })
    await expect(page).toHaveURL(new RegExp(`baseImage=`), { timeout: 10000 })
  })

  test('branch path: tenant withdraws pending request and clones into prefilled form (J-02)', async ({ page }) => {
    const runKey = `${Date.now()}`
    const eprRecordId = `EPR-J02-${runKey}`
    const sourceImageRef = `library/alpine:j02-${runKey}`
    const sourceRegistry = 'registry-1.docker.io'

    await loginAsUser(page, 'alice.johnson@imagefactory.local', 'password', true)
    const tenantAuth = await readAuthContext(page)

    const eprResp = await apiPost(page, tenantAuth, '/epr/registration-requests', {
      epr_record_id: eprRecordId,
      product_name: `J02 Product ${runKey}`,
      technology_name: 'Alpine',
      business_justification: 'J-02 withdraw + clone branch-path automation',
    })
    expect(eprResp.ok()).toBeTruthy()
    const eprBody = await eprResp.json()
    const eprRequestId = eprBody?.data?.id
    expect(eprRequestId).toBeTruthy()

    await resetSession(page)
    await loginAsUser(page, 'bob.smith@imagefactory.local', 'password', true)
    const reviewerAuth = await readAuthContext(page)
    const approveEprResp = await apiPost(page, reviewerAuth, `/admin/epr/registration-requests/${eprRequestId}/approve`, {
      reason: 'Approved by J-02 automation',
    })
    expect(approveEprResp.ok()).toBeTruthy()

    await resetSession(page)
    await loginAsUser(page, 'alice.johnson@imagefactory.local', 'password', true)
    const tenantAuthAfterApproval = await readAuthContext(page)

    const importResp = await apiPost(page, tenantAuthAfterApproval, '/images/import-requests', {
      epr_record_id: eprRecordId,
      source_registry: sourceRegistry,
      source_image_ref: sourceImageRef,
    })
    expect(importResp.ok()).toBeTruthy()

    await page.goto('/quarantine/requests')
    await page.waitForLoadState('networkidle')
    const row = page.locator('div', { hasText: sourceImageRef }).first()
    await expect(row).toBeVisible({ timeout: 15000 })

    await row.getByRole('button', { name: 'Withdraw' }).click()
    await expect(page.locator('text=Withdraw Quarantine Request')).toBeVisible({ timeout: 10000 })
    await page.getByRole('button', { name: 'Withdraw' }).last().click()
    await expect(page.locator('text=Quarantine request withdrawn')).toBeVisible({ timeout: 10000 })

    await expect(page.locator('text=failed').first()).toBeVisible({ timeout: 15000 })
    await row.getByRole('button', { name: 'Clone' }).click()

    await expect(page.locator('text=Create Quarantine Request')).toBeVisible({ timeout: 10000 })
    await expect(page.locator('#epr-record-id')).toHaveValue(eprRecordId, { timeout: 10000 })
    await expect(page.locator('#source-registry')).toHaveValue(sourceRegistry, { timeout: 10000 })
    await expect(page.locator('#source-image-ref')).toHaveValue(sourceImageRef, { timeout: 10000 })
  })
})
