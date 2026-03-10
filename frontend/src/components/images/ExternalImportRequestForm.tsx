import {
    CreateImageImportRequestInput,
    imageImportService,
} from '@/services/imageImportService'
import { eprRegistrationService } from '@/services/eprRegistrationService'
import { registryAuthClient } from '@/api/registryAuthClient'
import api from '@/services/api'
import type { ImageImportRequest } from '@/types'
import type { RegistryAuth } from '@/types/registryAuth'
import { mapQuarantineImportErrorMessage } from '@/utils/quarantineErrorMessages'
import { AlertTriangle, ShieldAlert } from 'lucide-react'
import React, { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'

interface ExternalImportRequestFormProps {
    onCreated: (item: ImageImportRequest) => void
    initialValues?: Partial<CreateImageImportRequestInput>
}

type FormValues = CreateImageImportRequestInput
type EPRRuntimeMode = 'error' | 'deny' | 'allow'
type EPRPolicyLoadState = 'loaded' | 'fallback'
type RegistryAccessMode = 'public' | 'private'

const defaultValues: FormValues = {
    eprRecordId: '',
    sourceRegistry: '',
    sourceImageRef: '',
    registryAuthId: '',
}

const ExternalImportRequestForm: React.FC<ExternalImportRequestFormProps> = ({ onCreated, initialValues }) => {
    const [values, setValues] = useState<FormValues>(defaultValues)
    const [submitting, setSubmitting] = useState(false)
    const [formError, setFormError] = useState<string | null>(null)
    const [fieldErrors, setFieldErrors] = useState<Partial<Record<keyof FormValues, string>>>({})
    const [eprRuntimeMode, setEPRRuntimeMode] = useState<EPRRuntimeMode>('error')
    const [eprPolicyLoadState, setEPRPolicyLoadState] = useState<EPRPolicyLoadState>('loaded')
    const [registryAccessMode, setRegistryAccessMode] = useState<RegistryAccessMode>('public')
    const [registryAuthOptions, setRegistryAuthOptions] = useState<RegistryAuth[]>([])
    const [registryAuthLoading, setRegistryAuthLoading] = useState(false)
    const [registryAuthError, setRegistryAuthError] = useState<string | null>(null)
    const [approvedEprIDs, setApprovedEprIDs] = useState<string[]>([])
    const [approvedEprLoading, setApprovedEprLoading] = useState(false)
    const [approvedEprLoadError, setApprovedEprLoadError] = useState<string | null>(null)

    useEffect(() => {
        let active = true
        const loadEPRPolicy = async () => {
            try {
                const response = await api.get('/settings/epr-registration')
                const mode = String(response?.data?.runtime_error_mode || '').toLowerCase()
                if (!active) {
                    return
                }
                if (mode === 'deny' || mode === 'allow' || mode === 'error') {
                    setEPRRuntimeMode(mode)
                    setEPRPolicyLoadState('loaded')
                } else {
                    setEPRRuntimeMode('error')
                    setEPRPolicyLoadState('fallback')
                }
            } catch {
                if (active) {
                    setEPRRuntimeMode('error')
                    setEPRPolicyLoadState('fallback')
                }
            }
        }
        loadEPRPolicy()
        return () => {
            active = false
        }
    }, [])

    useEffect(() => {
        setValues({
            eprRecordId: initialValues?.eprRecordId?.trim() || '',
            sourceRegistry: initialValues?.sourceRegistry?.trim() || '',
            sourceImageRef: initialValues?.sourceImageRef?.trim() || '',
            registryAuthId: initialValues?.registryAuthId?.trim() || '',
        })
        setRegistryAccessMode(initialValues?.registryAuthId ? 'private' : 'public')
        setFieldErrors({})
        setFormError(null)
    }, [initialValues])

    useEffect(() => {
        let active = true
        const loadRegistryAuth = async () => {
            try {
                setRegistryAuthLoading(true)
                setRegistryAuthError(null)
                const response = await registryAuthClient.listRegistryAuth(undefined, true)
                if (!active) {
                    return
                }
                setRegistryAuthOptions(response.registry_auth || [])
            } catch {
                if (!active) {
                    return
                }
                setRegistryAuthOptions([])
                setRegistryAuthError('Failed to load registry authentications')
            } finally {
                if (active) {
                    setRegistryAuthLoading(false)
                }
            }
        }
        loadRegistryAuth()
        return () => {
            active = false
        }
    }, [])

    useEffect(() => {
        let active = true
        const loadApprovedEprIDs = async () => {
            try {
                setApprovedEprLoading(true)
                setApprovedEprLoadError(null)
                const rows = await eprRegistrationService.listTenantRequests({
                    status: 'approved',
                    limit: 200,
                    page: 1,
                })
                if (!active) {
                    return
                }
                const uniqueIDs = Array.from(new Set(rows.map((row) => row.epr_record_id).filter(Boolean)))
                setApprovedEprIDs(uniqueIDs)
            } catch {
                if (!active) {
                    return
                }
                setApprovedEprIDs([])
                setApprovedEprLoadError('Failed to load approved EPR IDs')
            } finally {
                if (active) {
                    setApprovedEprLoading(false)
                }
            }
        }
        loadApprovedEprIDs()
        return () => {
            active = false
        }
    }, [])

    const normalizedEprRecordID = values.eprRecordId.trim().toLowerCase()
    const isValidApprovedEpr = useMemo(() => {
        if (!normalizedEprRecordID) {
            return false
        }
        return approvedEprIDs.some((item) => item.trim().toLowerCase() === normalizedEprRecordID)
    }, [approvedEprIDs, normalizedEprRecordID])

    const isFormValid = useMemo(() => {
        return (
            isValidApprovedEpr &&
            values.sourceRegistry.trim().length > 0 &&
            values.sourceImageRef.trim().length > 0
        )
    }, [values, isValidApprovedEpr])

    const updateField = (key: keyof FormValues, value: string) => {
        setValues((prev) => ({ ...prev, [key]: value }))
        setFieldErrors((prev) => ({ ...prev, [key]: undefined }))
        setFormError(null)
    }

    const validate = (): boolean => {
        const nextErrors: Partial<Record<keyof FormValues, string>> = {}
        if (!values.eprRecordId.trim()) {
            nextErrors.eprRecordId = 'EPR record is required.'
        } else if (!isValidApprovedEpr) {
            nextErrors.eprRecordId = 'Select a valid approved EPR record ID from the tenant list.'
        }
        if (!values.sourceRegistry.trim()) {
            nextErrors.sourceRegistry = 'Source registry is required.'
        }
        if (!values.sourceImageRef.trim()) {
            nextErrors.sourceImageRef = 'Source image reference is required.'
        }
        if (registryAccessMode === 'private' && !values.registryAuthId?.trim()) {
            nextErrors.registryAuthId = 'Registry auth is required for private registries.'
        }
        setFieldErrors(nextErrors)
        return Object.keys(nextErrors).length === 0
    }

    const eprPostureLabel = eprRuntimeMode === 'allow' ? 'Permissive' : 'Strict'
    const eprRuntimeOutcome =
        eprRuntimeMode === 'allow'
            ? 'If the EPR integration is temporarily unavailable, the request can still be admitted.'
            : eprRuntimeMode === 'deny'
                ? 'If the EPR integration is temporarily unavailable, the request is denied as not registered.'
                : 'If the EPR integration is temporarily unavailable, the request fails and you must retry after service recovery.'
    const eprBusinessOutcome =
        eprRuntimeMode === 'allow'
            ? 'EPR not-found/inactive checks still deny the request when the service responds.'
            : 'EPR not-found/inactive checks deny the request.'

    const handleSubmit = async (event: React.FormEvent) => {
        event.preventDefault()
        if (!validate()) {
            return
        }

        setSubmitting(true)
        setFormError(null)
        try {
            const created = await imageImportService.createImportRequest({
                eprRecordId: values.eprRecordId.trim(),
                sourceRegistry: values.sourceRegistry.trim(),
                sourceImageRef: values.sourceImageRef.trim(),
                registryAuthId: registryAccessMode === 'private' ? values.registryAuthId?.trim() || undefined : undefined,
            })
            onCreated(created)
            setValues(defaultValues)
            setFieldErrors({})
        } catch (err) {
            setFormError(mapQuarantineImportErrorMessage(err, 'Failed to create quarantine import request.'))
        } finally {
            setSubmitting(false)
        }
    }

    return (
        <div className="bg-white dark:bg-slate-800 shadow rounded-lg p-6">
            <h3 className="text-lg font-medium text-slate-900 dark:text-white mb-1 flex items-center">
                <ShieldAlert className="w-5 h-5 mr-2 text-blue-600 dark:text-blue-400" />
                Request Quarantine Import
            </h3>
            <p className="text-xs text-slate-600 dark:text-slate-300 mb-4">
                Provide EPR registration and source image details. Approval and policy checks run before quarantine admission.
            </p>
            <div className="mb-4 rounded-md border border-amber-300 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
                <p className="font-medium">
                    EPR runtime policy mode: <span className="font-semibold">{eprRuntimeMode}</span>
                </p>
                <p className="mt-1">
                    Effective posture: <span className="font-semibold">{eprPostureLabel}</span>
                </p>
                <p className="mt-1">
                    {eprRuntimeOutcome}
                </p>
                <p className="mt-1">
                    {eprBusinessOutcome}
                </p>
                {eprPolicyLoadState === 'fallback' ? (
                    <p className="mt-2 rounded border border-amber-400/70 bg-amber-100/70 px-2 py-1 text-[11px] text-amber-900 dark:border-amber-700 dark:bg-amber-950/40 dark:text-amber-200">
                        Could not load tenant EPR policy. Showing strict fallback posture (`error`) until settings are reachable.
                    </p>
                ) : null}
            </div>

            {formError ? (
                <div className="mb-4 flex items-start gap-2 rounded-md border border-rose-200 bg-rose-50 p-3 text-xs text-rose-800 dark:border-rose-800 dark:bg-rose-950/30 dark:text-rose-200">
                    <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
                    <span>{formError}</span>
                </div>
            ) : null}

            <form onSubmit={handleSubmit} className="space-y-3">
                <div>
                    <label htmlFor="epr-record-id" className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                        EPR Record ID
                    </label>
                    <input
                        id="epr-record-id"
                        value={values.eprRecordId}
                        onChange={(e) => updateField('eprRecordId', e.target.value)}
                        list="approved-epr-ids"
                        placeholder={approvedEprLoading ? 'Loading approved EPR IDs...' : 'Type to search approved EPR IDs'}
                        className="w-full rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30"
                    />
                    <datalist id="approved-epr-ids">
                        {approvedEprIDs.map((id) => (
                            <option key={id} value={id} />
                        ))}
                    </datalist>
                    {approvedEprLoadError ? (
                        <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{approvedEprLoadError}</p>
                    ) : null}
                    {!approvedEprLoadError && !approvedEprLoading && !isValidApprovedEpr && values.eprRecordId.trim() ? (
                        <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
                            EPR ID not found in approved records. Select a valid approved EPR ID to continue.
                        </p>
                    ) : null}
                    {fieldErrors.eprRecordId ? (
                        <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{fieldErrors.eprRecordId}</p>
                    ) : null}
                </div>

                <div>
                    <label htmlFor="source-registry" className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Source Registry
                    </label>
                    <input
                        id="source-registry"
                        value={values.sourceRegistry}
                        onChange={(e) => updateField('sourceRegistry', e.target.value)}
                        placeholder="eg: registry-1.docker.io"
                        disabled={!isValidApprovedEpr}
                        className="w-full rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30"
                    />
                    {fieldErrors.sourceRegistry ? (
                        <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{fieldErrors.sourceRegistry}</p>
                    ) : null}
                </div>

                <div>
                    <label htmlFor="source-image-ref" className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Source Image Ref
                    </label>
                    <input
                        id="source-image-ref"
                        value={values.sourceImageRef}
                        onChange={(e) => updateField('sourceImageRef', e.target.value)}
                        placeholder="eg: library/nginx:1.27-alpine"
                        disabled={!isValidApprovedEpr}
                        className="w-full rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30"
                    />
                    {fieldErrors.sourceImageRef ? (
                        <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{fieldErrors.sourceImageRef}</p>
                    ) : null}
                </div>

                <div>
                    <label htmlFor="registry-access-mode" className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Registry Access
                    </label>
                    <select
                        id="registry-access-mode"
                        value={registryAccessMode}
                        onChange={(e) => {
                            const mode = e.target.value as RegistryAccessMode
                            setRegistryAccessMode(mode)
                            if (mode === 'public') {
                                updateField('registryAuthId', '')
                            }
                        }}
                        disabled={!isValidApprovedEpr}
                        className="w-full rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30"
                    >
                        <option value="public">Public registry (no auth needed)</option>
                        <option value="private">Private registry (auth required)</option>
                    </select>
                </div>

                {registryAccessMode === 'private' ? (
                    <div>
                        <label htmlFor="registry-auth-id" className="block text-xs font-medium text-slate-700 dark:text-slate-300 mb-1">
                            Registry Auth
                        </label>
                        <select
                            id="registry-auth-id"
                            value={values.registryAuthId || ''}
                            onChange={(e) => updateField('registryAuthId', e.target.value)}
                            disabled={registryAuthLoading || !isValidApprovedEpr}
                            className="w-full rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm bg-white dark:bg-slate-700 text-slate-900 dark:text-white focus:border-blue-500 focus:ring-2 focus:ring-blue-500/30 disabled:opacity-60"
                        >
                            <option value="">{registryAuthLoading ? 'Loading registry auth...' : 'Select registry authentication'}</option>
                            {registryAuthOptions.map((auth) => (
                                <option key={auth.id} value={auth.id}>
                                    {auth.name} ({auth.scope})
                                </option>
                            ))}
                        </select>
                        {fieldErrors.registryAuthId ? (
                            <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{fieldErrors.registryAuthId}</p>
                        ) : null}
                        {registryAuthError ? (
                            <p className="mt-1 text-xs text-rose-700 dark:text-rose-300">{registryAuthError}</p>
                        ) : null}
                        {!registryAuthLoading && registryAuthOptions.length === 0 ? (
                            <p className="mt-1 text-xs text-amber-700 dark:text-amber-300">
                                No registry auth found. <Link to="/settings/auth" className="underline font-medium">Create one first</Link>.
                            </p>
                        ) : null}
                    </div>
                ) : null}

                <button
                    type="submit"
                    disabled={submitting || !isFormValid}
                    className="w-full inline-flex items-center justify-center rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:bg-slate-400 dark:disabled:bg-slate-600"
                >
                    {submitting ? 'Submitting...' : isFormValid ? 'Create Request' : 'Create Request'}
                </button>
            </form>
        </div>
    )
}

export default ExternalImportRequestForm
