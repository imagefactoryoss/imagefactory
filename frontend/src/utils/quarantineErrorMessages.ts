import { ImageImportApiError } from '@/services/imageImportService'

export const mapQuarantineImportErrorMessage = (err: unknown, fallbackMessage: string): string => {
    if (err instanceof ImageImportApiError) {
        if (err.code === 'tenant_capability_not_entitled') {
            return 'This tenant is not entitled for quarantine requests. Contact your tenant administrator to enable the capability.'
        }
        if (err.code === 'epr_registration_required') {
            return 'EPR registration is required before requesting quarantine import. Verify the EPR ID is active for this tenant and retry.'
        }
        if (err.code === 'import_not_retryable') {
            return 'This request is not currently retryable. Refresh to confirm latest status or create a new request.'
        }
        if (err.code === 'retry_backoff_active') {
            const retryAfter = Number(err.details?.retry_after_seconds || 0)
            if (retryAfter > 0) {
                return `Retry is temporarily blocked by backoff policy. Try again in about ${retryAfter} seconds.`
            }
            return 'Retry is temporarily blocked by backoff policy. Wait briefly and try again.'
        }
        if (err.code === 'retry_attempt_limit_reached') {
            const maxAttempts = Number(err.details?.max_attempts || 0)
            if (maxAttempts > 0) {
                return `Retry limit reached (${maxAttempts} attempts). Clone the request or create a new one after remediation.`
            }
            return 'Retry limit reached for this request. Clone the request or create a new one after remediation.'
        }
        if (err.code === 'release_not_eligible') {
            const blocker = String(err.details?.release_blocker_reason || '').trim().toLowerCase()
            if (blocker === 'evidence_incomplete') {
                return 'Release is blocked because required evidence is incomplete. Ensure policy snapshot, scan summary, SBOM summary, and image digest are present.'
            }
            if (blocker === 'evidence_stale') {
                return 'Release is blocked because evidence is stale. Re-run quarantine processing to generate fresh evidence before release.'
            }
            if (blocker === 'policy_not_passed') {
                return 'Release is blocked because policy decision did not pass. Review policy results and remediate findings.'
            }
            return 'Release is currently not eligible. Review release blocker details in the request and retry after remediation.'
        }
        if (err.code === 'import_not_withdrawable') {
            return 'This request can only be withdrawn while pending review.'
        }
        if (err.code === 'not_found') {
            return 'The quarantine request could not be found. Refresh and retry.'
        }
        return err.message || fallbackMessage
    }

    if (err instanceof Error && err.message) {
        return err.message
    }
    return fallbackMessage
}
