import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('App admin route contract', () => {
  it('registers admin quarantine and on-demand scan workspaces', () => {
    const appPath = resolve(process.cwd(), 'src', 'App.tsx')
    const source = readFileSync(appPath, 'utf8')

    expect(source).toContain('<Route path="images/scans" element={<OnDemandScansPage />} />')
    expect(source).toContain('<Route path="quarantine/requests" element={<QuarantineReviewWorkbenchPage mode="requests" />} />')
    expect(source).toContain('<Route path="quarantine/requests/:requestId" element={<QuarantineRequestDetailPage scope="admin" />} />')
    expect(source).toContain('<Route path="quarantine/review" element={<QuarantineReviewWorkbenchPage mode="requests" />} />')
  })

  it('registers dedicated reviewer queue routes', () => {
    const appPath = resolve(process.cwd(), 'src', 'App.tsx')
    const source = readFileSync(appPath, 'utf8')

    expect(source).toContain('path="/reviewer/*"')
    expect(source).toContain('<Route path="dashboard" element={<ReviewerDashboardPage />} />')
    expect(source).toContain('<Route path="quarantine/requests" element={<ReviewerRequestsPage />} />')
    expect(source).toContain('<Route path="quarantine/requests/:requestId" element={<QuarantineRequestDetailPage scope="admin" />} />')
    expect(source).toContain('<Route path="epr/approvals" element={<ReviewerEprApprovalsPage />} />')
    expect(source).toContain('<Navigate to="/reviewer/dashboard" replace />')
  })
})
