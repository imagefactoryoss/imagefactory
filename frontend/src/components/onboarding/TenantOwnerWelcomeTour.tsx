import React, { useEffect, useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { profileService } from '@/services/profileService'
import { useAuthStore } from '@/store/auth'

type TenantGroup = {
    role_type?: string
}

interface TenantOwnerWelcomeTourProps {
    groups?: TenantGroup[]
}

const TOUR_VERSION = 'v1'
const TOUR_PREF_KEY = `tenant_owner_welcome_${TOUR_VERSION}_seen_at`

const TenantOwnerWelcomeTour: React.FC<TenantOwnerWelcomeTourProps> = ({ groups }) => {
    const [isOpen, setIsOpen] = useState(false)
    const [step, setStep] = useState(0)
    const [saving, setSaving] = useState(false)
    const user = useAuthStore((state) => state.user)
    const preferences = useAuthStore((state) => state.preferences)
    const setPreferences = useAuthStore((state) => state.setPreferences)
    const userId = user?.id

    const isTenantOwner = useMemo(
        () => (groups || []).some((group) => group?.role_type === 'owner' || group?.role_type === 'tenant_admin'),
        [groups]
    )
    const isSystemAdmin = useMemo(
        () => (groups || []).some((group) => group?.role_type === 'system_administrator'),
        [groups]
    )

    const hasSeenTour = useMemo(() => {
        const onboarding = preferences?.onboarding_tours
        if (!onboarding || typeof onboarding !== 'object') return false
        const seenAt = (onboarding as Record<string, any>)[TOUR_PREF_KEY]
        return typeof seenAt === 'string' && seenAt.trim().length > 0
    }, [preferences])

    useEffect(() => {
        if (!userId || !isTenantOwner || isSystemAdmin) {
            setIsOpen(false)
            return
        }
        if (!hasSeenTour) {
            setStep(0)
            setIsOpen(true)
        } else {
            setIsOpen(false)
        }
    }, [hasSeenTour, isSystemAdmin, isTenantOwner, userId])

    const markSeenAndClose = async () => {
        if (saving) {
            return
        }
        const nextPreferences: Record<string, any> = {
            ...(preferences || {}),
            onboarding_tours: {
                ...(((preferences?.onboarding_tours as Record<string, any>) || {})),
                [TOUR_PREF_KEY]: new Date().toISOString(),
            },
        }
        setSaving(true)
        try {
            const updatedProfile = await profileService.updateProfile({ preferences: nextPreferences })
            setPreferences(updatedProfile.preferences || nextPreferences)
            setIsOpen(false)
        } catch {
            setPreferences(nextPreferences)
            setIsOpen(false)
        } finally {
            setSaving(false)
        }
    }

    if (!isOpen) return null

    const steps = [
        {
            title: 'Welcome To Image Factory',
            body: 'Build and ship container images with traceability, security evidence, and policy controls in one workflow.',
            visual: (
                <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-gradient-to-br from-blue-50 via-cyan-50 to-emerald-50 dark:from-slate-800 dark:via-slate-800 dark:to-slate-700 p-4">
                    <div className="grid grid-cols-3 gap-2">
                        <div className="h-16 rounded bg-blue-200/80 dark:bg-blue-900/50" />
                        <div className="h-16 rounded bg-cyan-200/80 dark:bg-cyan-900/50" />
                        <div className="h-16 rounded bg-emerald-200/80 dark:bg-emerald-900/50" />
                    </div>
                    <div className="mt-3 h-3 w-2/3 rounded bg-slate-300 dark:bg-slate-600" />
                    <div className="mt-2 h-3 w-1/2 rounded bg-slate-200 dark:bg-slate-700" />
                </div>
            ),
        },
        {
            title: 'Connect Project Sources',
            body: 'Create a project, add a Git source, and set repository auth. You can then drive builds from `image-factory.yaml` in your repo.',
            visual: (
                <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-4">
                    <div className="flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-slate-200">
                        <span className="inline-flex h-6 w-6 items-center justify-center rounded bg-violet-100 text-violet-700 dark:bg-violet-900/40 dark:text-violet-300">1</span>
                        Project
                    </div>
                    <div className="mt-2 h-2 w-full rounded bg-slate-200 dark:bg-slate-700" />
                    <div className="mt-4 flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-slate-200">
                        <span className="inline-flex h-6 w-6 items-center justify-center rounded bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300">2</span>
                        Source + Auth
                    </div>
                    <div className="mt-2 h-2 w-4/5 rounded bg-slate-200 dark:bg-slate-700" />
                </div>
            ),
        },
        {
            title: 'Run Builds With Evidence',
            body: 'Start builds and monitor logs in real time. After completion, review layers, SBOM, and vulnerability evidence.',
            visual: (
                <div className="rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 p-4">
                    <div className="h-24 rounded border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800 p-2 space-y-2">
                        <div className="h-2 w-3/4 rounded bg-emerald-300/80 dark:bg-emerald-700/70" />
                        <div className="h-2 w-2/3 rounded bg-blue-300/80 dark:bg-blue-700/70" />
                        <div className="h-2 w-1/2 rounded bg-amber-300/80 dark:bg-amber-700/70" />
                    </div>
                    <div className="mt-3 grid grid-cols-3 gap-2 text-xs">
                        <div className="rounded bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300 px-2 py-1 text-center">Layers</div>
                        <div className="rounded bg-cyan-50 text-cyan-700 dark:bg-cyan-900/30 dark:text-cyan-300 px-2 py-1 text-center">SBOM</div>
                        <div className="rounded bg-rose-50 text-rose-700 dark:bg-rose-900/30 dark:text-rose-300 px-2 py-1 text-center">Vulns</div>
                    </div>
                </div>
            ),
        },
    ]

    const current = steps[step]
    const isLastStep = step === steps.length - 1

    return (
        <div className="fixed inset-0 z-[120] flex items-center justify-center p-4">
            <div className="absolute inset-0 bg-slate-900/60" onClick={() => { void markSeenAndClose() }} />
            <div className="relative w-full max-w-2xl rounded-xl border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900 shadow-2xl overflow-hidden">
                <div className="p-5 sm:p-6">
                    <div className="flex items-start justify-between gap-4">
                        <div>
                            <p className="text-xs font-semibold uppercase tracking-wide text-blue-600 dark:text-blue-400">Getting Started</p>
                            <h2 className="mt-1 text-xl font-semibold text-slate-900 dark:text-slate-100">{current.title}</h2>
                            <p className="mt-2 text-sm text-slate-600 dark:text-slate-300">{current.body}</p>
                        </div>
                        <button
                            type="button"
                            onClick={() => { void markSeenAndClose() }}
                            className="rounded-md px-2 py-1 text-sm text-slate-500 hover:bg-slate-100 dark:text-slate-300 dark:hover:bg-slate-800"
                        >
                            Skip
                        </button>
                    </div>

                    <div className="mt-4">{current.visual}</div>

                    <div className="mt-4 flex items-center justify-between">
                        <div className="flex items-center gap-2">
                            {steps.map((_, idx) => (
                                <span
                                    key={idx}
                                    className={`h-2.5 w-2.5 rounded-full ${idx === step ? 'bg-blue-600 dark:bg-blue-400' : 'bg-slate-300 dark:bg-slate-700'}`}
                                />
                            ))}
                        </div>

                        <div className="flex items-center gap-2">
                            {step > 0 && (
                                <button
                                    type="button"
                                    onClick={() => setStep((s) => Math.max(0, s - 1))}
                                    className="rounded-md border border-slate-300 dark:border-slate-600 px-3 py-2 text-sm text-slate-700 dark:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800"
                                >
                                    Back
                                </button>
                            )}
                            {!isLastStep ? (
                                <button
                                    type="button"
                                    onClick={() => setStep((s) => Math.min(steps.length - 1, s + 1))}
                                    className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700"
                                >
                                    Next
                                </button>
                            ) : (
                                <div className="flex items-center gap-2">
                                    <Link
                                        to="/projects/new"
                                        onClick={() => { void markSeenAndClose() }}
                                        className="rounded-md border border-emerald-300 dark:border-emerald-700 px-3 py-2 text-sm font-medium text-emerald-700 dark:text-emerald-300 hover:bg-emerald-50 dark:hover:bg-emerald-900/20"
                                    >
                                        Create Project
                                    </Link>
                                    <button
                                        type="button"
                                        onClick={() => { void markSeenAndClose() }}
                                        className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700"
                                    >
                                        Finish
                                    </button>
                                </div>
                            )}
                        </div>
                    </div>
                </div>
            </div>
        </div>
    )
}

export default TenantOwnerWelcomeTour
