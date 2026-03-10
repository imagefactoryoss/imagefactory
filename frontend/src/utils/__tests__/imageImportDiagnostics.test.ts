import { describe, expect, it } from 'vitest'
import { getImportDiagnostic, getImportProgressLabel, getImportRemediationHint } from '@/utils/imageImportDiagnostics'
import type { ImageImportRequest } from '@/types'

const baseRequest: ImageImportRequest = {
  id: '00000000-0000-0000-0000-000000000001',
  tenant_id: '00000000-0000-0000-0000-000000000002',
  requested_by_user_id: '00000000-0000-0000-0000-000000000003',
  request_type: 'quarantine',
  source_registry: 'ghcr.io',
  source_image_ref: 'ghcr.io/acme/tool:1.0.0',
  status: 'approved',
  created_at: '2026-03-04T00:00:00Z',
  updated_at: '2026-03-04T00:00:00Z',
  sync_state: 'awaiting_dispatch',
}

describe('image import diagnostics', () => {
  it('shows provider guidance for awaiting dispatch state', () => {
    const diagnostic = getImportDiagnostic(baseRequest)
    expect(diagnostic.title).toBe('Awaiting Dispatch')
    expect(diagnostic.message).toContain('tekton_enabled=true')
    expect(diagnostic.message).toContain('quarantine_dispatch_enabled=true')
  })

  it('prefers execution state label when available', () => {
    const label = getImportProgressLabel({
      ...baseRequest,
      execution_state: 'evidence_pending',
    })
    expect(label).toBe('Evidence Pending')
  })

  it('returns remediation hint for dispatcher unavailable failures', () => {
    const hint = getImportRemediationHint({
      ...baseRequest,
      status: 'failed',
      sync_state: 'dispatch_failed',
      failure_class: 'dispatch',
      failure_code: 'dispatcher_unavailable',
    })
    expect(hint).toContain('quarantine_dispatch_enabled=true')
  })

  it('returns remediation hint for evidence-incomplete release blockers', () => {
    const hint = getImportRemediationHint({
      ...baseRequest,
      status: 'success',
      release_state: 'release_blocked',
      release_blocker_reason: 'evidence_incomplete',
    })
    expect(hint).toContain('Capture complete policy, SBOM, and vulnerability evidence')
  })

  it('shows awaiting approval message for pending requests', () => {
    const diagnostic = getImportDiagnostic({
      ...baseRequest,
      status: 'pending',
      sync_state: 'awaiting_approval',
      execution_state: 'awaiting_approval',
    })
    expect(diagnostic.title).toBe('Awaiting Approval')
    expect(diagnostic.message).toContain('pending reviewer approval')
  })
})
