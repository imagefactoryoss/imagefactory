import { buildService } from '@/services/buildService'
import { projectService } from '@/services/projectService'
import { useTenantStore } from '@/store/tenant'
import { BuildConfig, BuildType, InfrastructureType, Project, RegistryType, SBOMTool, ScanTool, SecretManagerType } from '@/types'
import React, { Suspense, lazy, useEffect, useState } from 'react'
import toast from 'react-hot-toast'
import { Link, useNavigate } from 'react-router-dom'

const BuildMethodStep = lazy(() => import('./steps/BuildMethodStep'))
const ConfigurationStep = lazy(() => import('./steps/ConfigurationStep'))
const ToolSelectionStep = lazy(() => import('./steps/ToolSelectionStep'))
const ValidationStep = lazy(() => import('./steps/ValidationStep'))

// Types for wizard state
export interface WizardState {
    currentStep: number
    selectedProject?: Project
    buildName: string
    buildDescription: string
    buildMethod?: BuildType
    selectedTools: {
        sbom?: SBOMTool
        scan?: ScanTool
        registry?: RegistryType
        secrets?: SecretManagerType
    }
    buildConfig: Partial<BuildConfig>
    infrastructureType?: InfrastructureType
    infrastructureProviderId: string | null
    validationErrors: Record<string, string>
    isSubmitting: boolean
}

const STEPS = [
    { id: 1, title: 'Build Method', description: 'Choose your build approach' },
    { id: 2, title: 'Tool Selection', description: 'Select security and registry tools' },
    { id: 3, title: 'Configuration', description: 'Configure build settings' },
    { id: 4, title: 'Validation', description: 'Review and create build' }
]

const stepLoadingFallback = (
    <div className="flex min-h-[320px] items-center justify-center rounded-lg border border-slate-200 bg-slate-50 p-6 dark:border-slate-700 dark:bg-slate-900/50">
        <div className="text-center">
            <div className="mx-auto mb-3 h-8 w-8 animate-spin rounded-full border-b-2 border-blue-600"></div>
            <p className="text-sm text-slate-600 dark:text-slate-300">Loading wizard step...</p>
        </div>
    </div>
)

const isDockerfileProvided = (dockerfile?: BuildConfig['dockerfile']): boolean => {
    if (!dockerfile) return false
    if (typeof dockerfile === 'string') {
        return dockerfile.trim().length > 0
    }
    if (dockerfile.source === 'path') {
        return !!dockerfile.path?.trim()
    }
    if (dockerfile.source === 'content' || dockerfile.source === 'upload') {
        return !!dockerfile.content?.trim()
    }
    return false
}

const BuildCreationWizard: React.FC = () => {
    const navigate = useNavigate()
    const { selectedTenantId } = useTenantStore()
    const searchParams = new URLSearchParams(window.location.search)
    const projectId = searchParams.get('projectId')
    const cloneFrom = searchParams.get('cloneFrom')
    const baseImage = searchParams.get('baseImage')
    const sourceImageRef = searchParams.get('sourceImageRef')

    const [wizardState, setWizardState] = useState<WizardState>({
        currentStep: 1,
        buildName: '',
        buildDescription: '',
        selectedTools: {},
        buildConfig: {},
        infrastructureType: undefined,
        infrastructureProviderId: null,
        validationErrors: {},
        isSubmitting: false
    })

    const [projects, setProjects] = useState<Project[]>([])
    const [loadingProjects, setLoadingProjects] = useState(false)
    const [tenantNamespaceConflict, setTenantNamespaceConflict] = useState<{
        message: string
        providerId: string | null
    } | null>(null)
    const [quarantineArtifactConflict, setQuarantineArtifactConflict] = useState<{
        message: string
        imageRef: string | null
        tenantId: string | null
    } | null>(null)
    const [cloneProjectId, setCloneProjectId] = useState<string | null>(null)
    const [cloneHydrated, setCloneHydrated] = useState(false)

    // Load projects on mount
    useEffect(() => {
        loadProjects()
    }, [selectedTenantId])

    // Pre-select project if projectId is provided
    useEffect(() => {
        if (projectId && projects.length > 0) {
            const selectedProject = projects.find(p => p.id === projectId)
            if (selectedProject) {
                updateWizardState({ selectedProject })
            }
        }
    }, [projectId, projects])

    // Hydrate wizard from an existing build when cloneFrom is provided.
    useEffect(() => {
        if (!cloneFrom || cloneHydrated) return
        let isMounted = true

        const loadCloneSource = async () => {
            try {
                const sourceBuild = await buildService.getBuild(cloneFrom)
                if (!isMounted) return

                const sourceManifest = sourceBuild.manifest || ({} as any)
                const sourceConfig = (sourceManifest.buildConfig || {}) as Partial<BuildConfig>
                const sourceDescription = (sourceManifest as any).description || ''

                setCloneProjectId(sourceBuild.projectId || null)

                updateWizardState({
                    currentStep: 1,
                    buildName: sourceManifest.name ? `${sourceManifest.name} (Clone)` : 'Cloned Build',
                    buildDescription: sourceDescription,
                    buildMethod: sourceManifest.type,
                    selectedTools: {
                        sbom: sourceConfig.sbomTool,
                        scan: sourceConfig.scanTool,
                        registry: sourceConfig.registryType,
                        secrets: sourceConfig.secretManagerType,
                    },
                    buildConfig: {
                        ...sourceConfig,
                    },
                    infrastructureType: (sourceManifest as any).infrastructureType || (sourceManifest as any).infrastructure_type,
                    infrastructureProviderId: (sourceManifest as any).infrastructureProviderId || (sourceManifest as any).infrastructure_provider_id || null,
                    validationErrors: {},
                })

                toast.success('Loaded clone source into build wizard')
                setCloneHydrated(true)
            } catch (error: any) {
                if (!isMounted) return
                toast.error(error?.message || 'Failed to load clone source build')
                setCloneHydrated(true)
            }
        }

        loadCloneSource()
        return () => {
            isMounted = false
        }
    }, [cloneFrom, cloneHydrated])

    // Select clone source project once projects are loaded.
    useEffect(() => {
        if (!cloneProjectId || projects.length === 0) return
        const selectedProject = projects.find((p) => p.id === cloneProjectId)
        if (selectedProject) {
            updateWizardState({ selectedProject })
        }
    }, [cloneProjectId, projects])

    // Apply released-artifact build prefill only when not cloning.
    useEffect(() => {
        if (cloneFrom) return
        const trimmedBaseImage = (baseImage || '').trim()
        if (!trimmedBaseImage) return
        setWizardState((prev) => {
            if ((prev.buildConfig?.baseImage || '').trim()) {
                return prev
            }
            return {
                ...prev,
                buildMethod: prev.buildMethod || 'container',
                buildConfig: {
                    ...prev.buildConfig,
                    baseImage: trimmedBaseImage,
                    variables: {
                        ...(prev.buildConfig?.variables || {}),
                        released_source_image_ref: (sourceImageRef || '').trim(),
                    },
                },
            }
        })
    }, [cloneFrom, baseImage, sourceImageRef])

    const loadProjects = async () => {
        if (!selectedTenantId) {
            toast.error('No tenant selected')
            return
        }
        try {
            setLoadingProjects(true)
            const response = await projectService.getProjects({
                page: 1,
                limit: 50,
                status: ['active'],
                tenantId: selectedTenantId
            })
            setProjects(response.data)
        } catch (error) {
            toast.error('Failed to load projects')

        } finally {
            setLoadingProjects(false)
        }
    }

    const updateWizardState = (updates: Partial<WizardState>) => {
        const isInfrastructureChange =
            Object.prototype.hasOwnProperty.call(updates, 'infrastructureType') ||
            Object.prototype.hasOwnProperty.call(updates, 'infrastructureProviderId')
        if (isInfrastructureChange) {
            setTenantNamespaceConflict(null)
        }

        setWizardState(prev => {
            const next = { ...prev, ...updates }

            // Clear stale infra validation error when user changes selection.
            // Don't interfere when validation is explicitly setting validationErrors.
            if (isInfrastructureChange && !Object.prototype.hasOwnProperty.call(updates, 'validationErrors')) {
                const { infrastructureProvider, ...rest } = next.validationErrors || {}
                next.validationErrors = rest
            }

            return next
        })
    }

    const nextStep = () => {
        if (wizardState.currentStep < STEPS.length) {
            updateWizardState({ currentStep: wizardState.currentStep + 1 })
        }
    }

    const prevStep = () => {
        if (wizardState.currentStep > 1) {
            updateWizardState({ currentStep: wizardState.currentStep - 1 })
        }
    }

    const validateCurrentStep = (): boolean => {
        const errors: Record<string, string> = {}

        switch (wizardState.currentStep) {
            case 1: // Build Method
                if (!wizardState.selectedProject) {
                    errors.project = 'Project selection is required to proceed'
                }
                if (!wizardState.buildName.trim()) {
                    errors.buildName = 'Build name is required (must not be empty)'
                } else if (wizardState.buildName.trim().length < 3) {
                    errors.buildName = 'Build name must be at least 3 characters'
                } else if (wizardState.buildName.trim().length > 100) {
                    errors.buildName = 'Build name must be less than 100 characters'
                }
                if (!wizardState.buildMethod) {
                    errors.buildMethod = 'Build method selection is required'
                }
                break

            case 2: // Tool Selection
                if (!wizardState.selectedTools.sbom) {
                    errors.sbom = 'SBOM generation tool is required'
                }
                if (!wizardState.selectedTools.scan) {
                    errors.scan = 'Security scanning tool is required'
                }
                if (!wizardState.selectedTools.registry) {
                    errors.registry = 'Registry backend is required to store built images'
                }
                if (!wizardState.selectedTools.secrets) {
                    errors.secrets = 'Secret manager is required for handling credentials'
                }
                break

            case 3: // Configuration
                // Build method-specific validation
                if (!wizardState.infrastructureType) {
                    errors.infrastructureProvider = 'Infrastructure selection is required'
                } else if (!wizardState.infrastructureProviderId) {
                    errors.infrastructureProvider = 'Infrastructure provider selection is required'
                }
                if (!wizardState.buildConfig?.sourceId) {
                    errors.sourceId = 'Source selection is required'
                }
                if ((wizardState.buildConfig?.refPolicy || 'source_default') === 'fixed' && !wizardState.buildConfig?.fixedRef?.trim()) {
                    errors.fixedRef = 'Fixed ref is required when ref policy is fixed'
                }
                if (wizardState.buildMethod === 'kaniko') {
                    if (!isDockerfileProvided(wizardState.buildConfig?.dockerfile)) {
                        errors.dockerfile = 'Dockerfile path or inline content is required for Kaniko builds'
                    }
                    if (!wizardState.buildConfig?.buildContext?.trim()) {
                        errors.buildContext = 'Build context is required for Kaniko builds'
                    }
                    if (!wizardState.buildConfig?.registryRepo?.trim()) {
                        errors.registryRepo = 'Registry repository is required for Kaniko builds'
                    }
                    if (!wizardState.buildConfig?.registryAuthId?.trim()) {
                        errors.registryAuth = 'Registry authentication is required for Kaniko builds'
                    }
                } else if (wizardState.buildMethod === 'packer') {
                    if (!wizardState.buildConfig?.packerTemplate?.trim()) {
                        errors.packerTemplate = 'Packer template is required for Packer builds'
                    }
                } else if (wizardState.buildMethod === 'paketo') {
                    if (!wizardState.buildConfig?.paketoConfig?.builder?.trim()) {
                        errors.paketoBuilder = 'Buildpack builder selection is required for Paketo builds'
                    }
                    if (!wizardState.buildConfig?.registryAuthId?.trim()) {
                        errors.registryAuth = 'Registry authentication is required for Paketo builds'
                    }
                } else if (wizardState.buildMethod === 'buildx') {
                    if (!isDockerfileProvided(wizardState.buildConfig?.dockerfile)) {
                        errors.dockerfile = 'Dockerfile path or inline content is required for Buildx builds'
                    }
                    if (!wizardState.buildConfig?.buildContext?.trim()) {
                        errors.buildContext = 'Build context is required for Buildx builds'
                    }
                    if (!wizardState.buildConfig?.registryRepo?.trim()) {
                        errors.registryRepo = 'Registry repository is required for Buildx builds'
                    }
                    if (!wizardState.buildConfig?.registryAuthId?.trim()) {
                        errors.registryAuth = 'Registry authentication is required for Buildx builds'
                    }
                } else if (wizardState.buildMethod === 'container') {
                    if (!isDockerfileProvided(wizardState.buildConfig?.dockerfile)) {
                        errors.dockerfile = 'Dockerfile path or inline content is required for container builds'
                    }
                    if (!wizardState.buildConfig?.buildContext?.trim()) {
                        errors.buildContext = 'Build context is required for container builds'
                    }
                    if (!wizardState.buildConfig?.registryRepo?.trim()) {
                        errors.registryRepo = 'Registry repository is required for container builds'
                    }
                    if (!wizardState.buildConfig?.registryAuthId?.trim()) {
                        errors.registryAuth = 'Registry authentication is required for container builds'
                    }
                } else if (wizardState.buildMethod === 'nix') {
                    if (!wizardState.buildConfig?.nixExpression?.trim() && !wizardState.buildConfig?.flakeUri?.trim()) {
                        errors.nixExpression = 'Either Nix expression or Flake URI is required for Nix builds'
                    }
                } else {
                    // Default/container builds
                    if (!wizardState.buildConfig?.baseImage?.trim()) {
                        errors.baseImage = 'Base image is required (e.g., ubuntu:22.04, python:3.11)'
                    }
                    if (!wizardState.buildConfig?.instructions?.length) {
                        errors.instructions = 'At least one build instruction is required'
                    }
                }
                break

            case 4: // Validation
                // Final validation before submission
                break
        }

        updateWizardState({ validationErrors: errors })

        // Show error toast if there are validation errors
        if (Object.keys(errors).length > 0) {
            const errorMessages = Object.values(errors)
            const firstError = errorMessages[0]
            toast.error(firstError)
        }

        return Object.keys(errors).length === 0
    }

    const handleNext = () => {
        if (validateCurrentStep()) {
            nextStep()
        }
    }

    const handleCreateBuild = async () => {
        if (!wizardState.selectedProject || !wizardState.buildMethod) {
            toast.error('Missing required configuration')
            return
        }

        if (!selectedTenantId) {
            toast.error('No tenant selected')
            return
        }

        try {
            updateWizardState({ isSubmitting: true })
            setTenantNamespaceConflict(null)
            setQuarantineArtifactConflict(null)

            const buildConfig: BuildConfig = {
                buildType: wizardState.buildMethod,
                sbomTool: wizardState.selectedTools.sbom!,
                scanTool: wizardState.selectedTools.scan!,
                registryType: wizardState.selectedTools.registry!,
                secretManagerType: wizardState.selectedTools.secrets!,
                ...wizardState.buildConfig
            }

            const buildRequest = {
                tenantId: selectedTenantId,
                projectId: wizardState.selectedProject.id,
                manifest: {
                    name: wizardState.buildName,
                    description: wizardState.buildDescription,
                    type: wizardState.buildMethod,
                    baseImage: '', // Will be set based on build type
                    instructions: [],
                    environment: {},
                    tags: wizardState.buildConfig?.tags || [],
                    metadata: {},
                    infrastructure_type: wizardState.infrastructureType || undefined,
                    infrastructure_provider_id: wizardState.infrastructureProviderId || undefined,
                    buildConfig
                }
            }

            const build = await buildService.createBuild(buildRequest)
            const buildId = (build as any)?.id || (build as any)?.build_id

            toast.success('Build created successfully!')
            if (!buildId) {
                toast.error('Build created but response did not include an id.')
                return
            }

            navigate(`/builds/${buildId}`, { replace: true })
            return
        } catch (error: any) {
            const errorStatus = error?.response?.status
            const errorMessage = error?.response?.data?.error || error?.message || 'Failed to create build'
            const errorCode = error?.response?.data?.code
            const errorDetails = error?.response?.data?.details

            // Prefer structured error payloads for tenant-namespace conflicts; fall back to message regex for older servers
            const isStructuredTenantNamespaceConflict = errorStatus === 409 && errorCode === 'tenant_namespace_not_prepared'
            const isDetailsTenantNamespaceConflict = errorStatus === 409 && errorDetails?.prepare_status === 'missing'
            const isLegacyMessageMatch = /tenant namespace|namespace not provisioned|not prepared for this tenant|tenant namespace is not ready/i.test(errorMessage)
            const isTenantNamespaceConflict = isStructuredTenantNamespaceConflict || isDetailsTenantNamespaceConflict || (errorStatus === 409 && isLegacyMessageMatch)
            const isQuarantineArtifactConflict = errorStatus === 409 && errorCode === 'quarantine_artifact_not_released'

            if (isTenantNamespaceConflict) {
                // Prefer structured namespace + provider fields when present
                const ns = errorDetails?.namespace
                const providerId = errorDetails?.provider_id || wizardState.infrastructureProviderId
                const message = ns ? `Tenant namespace ${ns} is not prepared for this tenant.` : errorMessage
                setTenantNamespaceConflict({
                    message,
                    providerId,
                })
            }
            if (isQuarantineArtifactConflict) {
                const imageRef = errorDetails?.image_ref || null
                const tenantId = errorDetails?.tenant_id || selectedTenantId || null
                setQuarantineArtifactConflict({
                    message: errorMessage,
                    imageRef,
                    tenantId,
                })
                toast.error('Build blocked: quarantine artifact is not released for this tenant')
            } else {
                toast.error(error.message || 'Failed to create build')
            }
        } finally {
            updateWizardState({ isSubmitting: false })
        }
    }

    const renderCurrentStep = () => {
        switch (wizardState.currentStep) {
            case 1:
                return (
                    <BuildMethodStep
                        wizardState={wizardState}
                        projects={projects}
                        loadingProjects={loadingProjects}
                        onUpdate={updateWizardState}
                    />
                )
            case 2:
                return (
                    <ToolSelectionStep
                        wizardState={wizardState}
                        onUpdate={updateWizardState}
                    />
                )
            case 3:
                return (
                    <ConfigurationStep
                        wizardState={wizardState}
                        onUpdate={updateWizardState}
                    />
                )
            case 4:
                return (
                    <ValidationStep
                        wizardState={wizardState}
                        onCreateBuild={handleCreateBuild}
                    />
                )
            default:
                return null
        }
    }

    return (
        <div className="max-w-4xl mx-auto px-4 py-6 sm:px-6 lg:px-8">
            {/* Header */}
            <div className="mb-8">
                <h1 className="text-3xl font-bold text-slate-900 dark:text-white">
                    {cloneFrom ? 'Clone Build' : 'Create New Build'}
                </h1>
                <p className="mt-2 text-lg text-slate-600 dark:text-slate-400">
                    {cloneFrom
                        ? 'Review and adjust the cloned configuration before creating a new build.'
                        : 'Set up a multi-tool container build with security scanning and registry integration'}
                </p>
            </div>

            {tenantNamespaceConflict && (
                <div className="mb-6 rounded-lg border border-amber-300 dark:border-amber-700 bg-amber-50 dark:bg-amber-900/20 p-4">
                    <h3 className="text-sm font-semibold text-amber-900 dark:text-amber-200">
                        Tenant Namespace Not Prepared
                    </h3>
                    <p className="mt-1 text-sm text-amber-800 dark:text-amber-300">
                        This build is blocked until tenant namespace provisioning succeeds.
                    </p>
                    <p className="mt-1 text-xs text-amber-700 dark:text-amber-300 whitespace-pre-wrap">
                        {tenantNamespaceConflict.message}
                    </p>
                    <div className="mt-2 flex flex-wrap items-center gap-3 text-sm">
                        <span className="text-amber-900 dark:text-amber-200">
                            System administrator action: run <span className="font-semibold">Prepare Tenant Namespace</span> on the provider details page.
                        </span>
                        {tenantNamespaceConflict.providerId && (
                            <Link
                                to={`/admin/infrastructure/providers/${tenantNamespaceConflict.providerId}`}
                                className="font-medium text-blue-700 hover:text-blue-800 dark:text-blue-300 dark:hover:text-blue-200 underline"
                            >
                                Open Provider Details
                            </Link>
                        )}
                    </div>
                </div>
            )}
            {quarantineArtifactConflict && (
                <div className="mb-6 rounded-lg border border-red-300 dark:border-red-700 bg-red-50 dark:bg-red-900/20 p-4">
                    <h3 className="text-sm font-semibold text-red-900 dark:text-red-200">
                        Quarantine Approval Required
                    </h3>
                    <p className="mt-1 text-sm text-red-800 dark:text-red-300">
                        This build is blocked because it references an unreleased quarantine artifact.
                    </p>
                    <p className="mt-1 text-xs text-red-700 dark:text-red-300 whitespace-pre-wrap">
                        {quarantineArtifactConflict.message}
                    </p>
                    <div className="mt-2 flex flex-wrap items-center gap-3 text-sm">
                        {quarantineArtifactConflict.imageRef && (
                            <span className="text-red-900 dark:text-red-200">
                                Artifact: <span className="font-semibold">{quarantineArtifactConflict.imageRef}</span>
                            </span>
                        )}
                        <Link
                            to="/quarantine/releases"
                            className="font-medium text-blue-700 hover:text-blue-800 dark:text-blue-300 dark:hover:text-blue-200 underline"
                        >
                            Open Released Artifacts
                        </Link>
                    </div>
                </div>
            )}

            {/* Progress Indicator */}
            <div className="mb-8">
                <div className="flex items-center justify-between">
                    {STEPS.map((step, index) => (
                        <React.Fragment key={step.id}>
                            <div className="flex flex-col items-center">
                                <div className={`w-10 h-10 rounded-full flex items-center justify-center text-sm font-medium ${wizardState.currentStep > step.id
                                    ? 'bg-green-500 text-white'
                                    : wizardState.currentStep === step.id
                                        ? 'bg-blue-500 text-white'
                                        : 'bg-slate-200 text-slate-600 dark:bg-slate-700 dark:text-slate-400'
                                    }`}>
                                    {wizardState.currentStep > step.id ? '✓' : step.id}
                                </div>
                                <div className="mt-2 text-center">
                                    <div className={`text-sm font-medium ${wizardState.currentStep >= step.id
                                        ? 'text-slate-900 dark:text-white'
                                        : 'text-slate-500 dark:text-slate-400'
                                        }`}>
                                        {step.title}
                                    </div>
                                    <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                                        {step.description}
                                    </div>
                                </div>
                            </div>
                            {index < STEPS.length - 1 && (
                                <div className={`flex-1 h-px mx-4 mt-5 ${wizardState.currentStep > step.id
                                    ? 'bg-green-500'
                                    : 'bg-slate-200 dark:bg-slate-700'
                                    }`} />
                            )}
                        </React.Fragment>
                    ))}
                </div>
            </div>

            {/* Step Title and Description */}
            <div className="mb-6 pb-4 border-b border-slate-200 dark:border-slate-700">
                <h2 className="text-2xl font-bold text-slate-900 dark:text-white">
                    {STEPS[wizardState.currentStep - 1]?.title}
                </h2>
                <p className="mt-2 text-slate-600 dark:text-slate-400">
                    {STEPS[wizardState.currentStep - 1]?.description}
                </p>
            </div>

            {/* Error Summary */}
            {Object.keys(wizardState.validationErrors).length > 0 && (
                <div className="mb-6 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4">
                    <div className="flex items-start">
                        <div className="flex-shrink-0">
                            <svg className="h-5 w-5 text-red-500" viewBox="0 0 20 20" fill="currentColor">
                                <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
                            </svg>
                        </div>
                        <div className="ml-3">
                            <h3 className="text-sm font-medium text-red-800 dark:text-red-200">
                                {Object.keys(wizardState.validationErrors).length === 1
                                    ? 'Please fix the following error:'
                                    : `Please fix the following ${Object.keys(wizardState.validationErrors).length} errors:`}
                            </h3>
                            <ul className="mt-2 list-disc list-inside space-y-1">
                                {Object.entries(wizardState.validationErrors).map(([key, message]) => (
                                    <li key={key} className="text-sm text-red-700 dark:text-red-300">
                                        {message}
                                    </li>
                                ))}
                            </ul>
                        </div>
                    </div>
                </div>
            )}

            {/* Step Content */}
            <div className="bg-white dark:bg-slate-800 rounded-lg shadow-sm border border-slate-200 dark:border-slate-700 p-6 mb-6">
                <Suspense fallback={stepLoadingFallback}>
                    {renderCurrentStep()}
                </Suspense>
            </div>

            {/* Navigation */}
            <div className="flex justify-between items-center">
                <button
                    type="button"
                    onClick={prevStep}
                    disabled={wizardState.currentStep === 1}
                    className="px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                    Previous
                </button>

                <div className="flex gap-3">
                    <button
                        type="button"
                        onClick={() => navigate('/builds')}
                        className="px-4 py-2 border border-slate-300 dark:border-slate-600 rounded-md text-sm font-medium text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                    >
                        Cancel
                    </button>

                    {wizardState.currentStep < STEPS.length ? (
                        <button
                            type="button"
                            onClick={handleNext}
                            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                        >
                            Next
                        </button>
                    ) : (
                        <button
                            type="button"
                            onClick={handleCreateBuild}
                            disabled={wizardState.isSubmitting}
                            className="px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center gap-2"
                        >
                            {wizardState.isSubmitting && (
                                <svg className="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor">
                                    <circle className="opacity-25" cx="12" cy="12" r="10" strokeWidth="4" />
                                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                                </svg>
                            )}
                            {wizardState.isSubmitting ? 'Creating Build...' : 'Create Build'}
                        </button>
                    )}
                </div>
            </div>
        </div>
    )
}

export default BuildCreationWizard
